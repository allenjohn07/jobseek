package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestTranslateStreamLines(t *testing.T) {
	delta := `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hel"}}}`
	whole := `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}`
	tool := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__jobseek__propose_edit","input":{}}]}}`

	t.Run("deltas stream and the final block is not repeated", func(t *testing.T) {
		st := &streamState{}
		got := append(st.translate([]byte(delta)), st.translate([]byte(whole))...)
		if len(got) != 1 || got[0] != (agentEvent{Type: "text", Text: "Hel"}) {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("without deltas the whole block is sent", func(t *testing.T) {
		got := (&streamState{}).translate([]byte(whole))
		if len(got) != 1 || got[0].Text != "Hello" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("a later message after streamed text is not swallowed", func(t *testing.T) {
		st := &streamState{}
		st.translate([]byte(delta))
		st.translate([]byte(whole)) // resets
		got := st.translate([]byte(whole))
		if len(got) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("tool use strips the server prefix", func(t *testing.T) {
		got := (&streamState{}).translate([]byte(tool))
		if len(got) != 1 || got[0] != (agentEvent{Type: "tool", Name: "propose_edit"}) {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("error result is recorded", func(t *testing.T) {
		st := &streamState{}
		st.translate([]byte(`{"type":"result","is_error":true,"result":"Not logged in"}`))
		if st.failure != "Not logged in" {
			t.Fatalf("failure = %q", st.failure)
		}
	})
	t.Run("junk and blank lines are ignored", func(t *testing.T) {
		st := &streamState{}
		if len(st.translate([]byte("not json"))) != 0 || len(st.translate(nil)) != 0 {
			t.Fatal("expected nothing")
		}
	})
}

// fakeRunner records requests and replays scripted behavior.
type fakeRunner struct {
	mu    sync.Mutex
	reqs  []agentRequest
	run   func(call int, req agentRequest, emit func(agentEvent)) error
	block chan struct{} // if set, Run waits for it (to test the busy lock)
}

func (f *fakeRunner) Run(ctx context.Context, req agentRequest, emit func(agentEvent)) error {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	n := len(f.reqs) - 1
	f.mu.Unlock()
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.run(n, req, emit)
}

type sseEvents []agentEvent

func parseSSE(t *testing.T, body string) sseEvents {
	t.Helper()
	var out sseEvents
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		if after, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			var ev agentEvent
			if err := json.Unmarshal([]byte(after), &ev); err != nil {
				t.Fatalf("bad SSE %q: %v", after, err)
			}
			out = append(out, ev)
		}
	}
	return out
}

func chat(t *testing.T, h http.Handler, resumeID int64, msg string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(chatRequest{ResumeID: resumeID, Message: msg})
	return do(t, h, "POST", "/api/jobtweak/chat", string(body))
}

func TestChatStreamsAndKeepsOneSession(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	fr := &fakeRunner{run: func(_ int, _ agentRequest, emit func(agentEvent)) error {
		emit(agentEvent{Type: "text", Text: "Sure."})
		emit(agentEvent{Type: "tool", Name: "propose_edit"})
		return nil
	}}
	app.agent = fr

	rec := chat(t, h, r.ID, "tailor it")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d, type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	evs := parseSSE(t, rec.Body.String())
	if len(evs) != 3 || evs[0].Text != "Sure." || evs[1].Name != "propose_edit" || evs[2].Type != "done" {
		t.Fatalf("events = %+v", evs)
	}

	first := fr.reqs[0]
	if first.Resume || first.SessionID == "" || first.MCPURL != "http://127.0.0.1:8080/mcp" || first.MCPToken == "" {
		t.Fatalf("first request = %+v", first)
	}
	if !strings.Contains(first.Message, "resume_id="+strconv.FormatInt(r.ID, 10)) || !strings.HasSuffix(first.Message, "tailor it") {
		t.Fatalf("message lacks context: %q", first.Message)
	}

	// Second turn continues the same conversation.
	chat(t, h, r.ID, "shorter please")
	second := fr.reqs[1]
	if !second.Resume || second.SessionID != first.SessionID {
		t.Fatalf("second request should resume %q, got %+v", first.SessionID, second)
	}
}

func TestChatContextNamesTheJob(t *testing.T) {
	h, app := newTestServer(t)
	seedBase(t, h)
	do(t, h, "POST", "/api/jobs", `{"company":"Initech","title":"Go Dev"}`)
	jr := decode[Resume](t, do(t, h, "POST", "/api/jobs/1/resume", ""))
	fr := &fakeRunner{run: func(int, agentRequest, func(agentEvent)) error { return nil }}
	app.agent = fr
	chat(t, h, jr.ID, "hi")
	if m := fr.reqs[0].Message; !strings.Contains(m, "job_id=1") || !strings.Contains(m, "Go Dev at Initech") {
		t.Fatalf("message = %q", m)
	}
}

func TestChatValidation(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	app.agent = &fakeRunner{run: func(int, agentRequest, func(agentEvent)) error { return nil }}
	tests := []struct {
		name string
		body string
		want int
	}{
		{"bad json", `{nope`, http.StatusBadRequest},
		{"empty message", `{"resumeId":` + strconv.FormatInt(r.ID, 10) + `,"message":"  "}`, http.StatusBadRequest},
		{"too long", `{"resumeId":` + strconv.FormatInt(r.ID, 10) + `,"message":"` + strings.Repeat("a", maxChatBytes+1) + `"}`, http.StatusBadRequest},
		{"unknown resume", `{"resumeId":999,"message":"hi"}`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(t, h, "POST", "/api/jobtweak/chat", tt.body); rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestChatReportsRunnerErrors(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	app.agent = &fakeRunner{run: func(int, agentRequest, func(agentEvent)) error { return errClaudeMissing }}
	evs := parseSSE(t, chat(t, h, r.ID, "hi").Body.String())
	if len(evs) != 1 || evs[0].Type != "error" || !strings.Contains(evs[0].Message, "isn't installed") {
		t.Fatalf("events = %+v", evs)
	}
	// A failed turn must not record a session that never existed.
	if got, _ := getResume(app.db, r.ID); got.SessionID != "" {
		t.Fatalf("session saved after failure: %q", got.SessionID)
	}
}

func TestChatStartsFreshWhenResumeFails(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	saveSessionID(app.db, r.ID, "old-session")
	fr := &fakeRunner{run: func(call int, req agentRequest, emit func(agentEvent)) error {
		if req.Resume {
			return errors.New("No conversation found with session ID")
		}
		emit(agentEvent{Type: "text", Text: "ok"})
		return nil
	}}
	app.agent = fr
	evs := parseSSE(t, chat(t, h, r.ID, "hi").Body.String())
	if len(fr.reqs) != 2 || fr.reqs[1].Resume || fr.reqs[1].SessionID == "old-session" {
		t.Fatalf("expected a fresh retry, got %+v", fr.reqs)
	}
	if evs[len(evs)-1].Type != "done" {
		t.Fatalf("events = %+v", evs)
	}
	if got, _ := getResume(app.db, r.ID); got.SessionID != fr.reqs[1].SessionID {
		t.Fatalf("saved session %q, want %q", got.SessionID, fr.reqs[1].SessionID)
	}
}

func TestChatRefusesSecondRunOnSameResume(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	fr := &fakeRunner{block: make(chan struct{}), run: func(int, agentRequest, func(agentEvent)) error { return nil }}
	app.agent = fr

	done := make(chan struct{})
	go func() { chat(t, h, r.ID, "first"); close(done) }()
	for { // wait until the first request is inside Run
		fr.mu.Lock()
		n := len(fr.reqs)
		fr.mu.Unlock()
		if n == 1 {
			break
		}
	}
	if rec := chat(t, h, r.ID, "second"); rec.Code != http.StatusConflict {
		t.Fatalf("second chat = %d, want 409", rec.Code)
	}
	close(fr.block)
	<-done
	if rec := chat(t, h, r.ID, "third"); rec.Code != http.StatusOK {
		t.Fatalf("after finishing = %d, want 200", rec.Code)
	}
}

// --- the real subprocess path, using a fake `claude` script ---

const fakeClaude = `#!/bin/sh
out="$JOBSEEK_FAKE_OUT"
printf '%s\n' "$@" > "$out/args"
cat > "$out/stdin"
pwd > "$out/cwd"
while [ $# -gt 0 ]; do
  if [ "$1" = "--mcp-config" ]; then
    cp "$2" "$out/mcp.json"
    echo "$2" > "$out/mcp.path"
    (stat -c %a "$2" 2>/dev/null || stat -f %Lp "$2") > "$out/mcp.mode" # GNU (Linux) first, then BSD (macOS)
  fi
  shift
done
cat "$out/canned"
exit $(cat "$out/exit" 2>/dev/null || echo 0)
`

func fakeCLI(t *testing.T, canned string, exit int) (cliRunner, string) {
	t.Helper()
	out := t.TempDir()
	bin := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(bin, []byte(fakeClaude), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(out, "canned"), []byte(canned), 0o644)
	os.WriteFile(filepath.Join(out, "exit"), []byte(strconv.Itoa(exit)), 0o644)
	t.Setenv("JOBSEEK_FAKE_OUT", out)
	return cliRunner{bin: bin, dir: t.TempDir()}, out
}

func TestCLIRunnerBuildsASafeCommand(t *testing.T) {
	canned := `{"type":"system","subtype":"init"}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Done."}}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Done."}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__jobseek__propose_edit","input":{}}]}}
{"type":"result","subtype":"success","is_error":false,"result":"Done."}
`
	runner, out := fakeCLI(t, canned, 0)
	var got []agentEvent
	err := runner.Run(context.Background(), agentRequest{
		Message: "hello there", SessionID: "sess-1", MCPURL: "http://127.0.0.1:8080/mcp", MCPToken: "secret-token",
	}, func(e agentEvent) { got = append(got, e) })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Text != "Done." || got[1].Name != "propose_edit" {
		t.Fatalf("events = %+v", got)
	}

	read := func(name string) string { b, _ := os.ReadFile(filepath.Join(out, name)); return string(b) }
	args := strings.Split(strings.TrimSpace(read("args")), "\n")
	has := func(a ...string) bool { // consecutive args present
		for i := 0; i+len(a) <= len(args); i++ {
			if strings.Join(args[i:i+len(a)], "\x00") == strings.Join(a, "\x00") {
				return true
			}
		}
		return false
	}
	for _, want := range [][]string{
		{"-p"}, {"--output-format", "stream-json"}, {"--strict-mcp-config"},
		{"--tools", ""}, {"--allowedTools", "mcp__jobseek__*"}, {"--permission-mode", "dontAsk"},
		{"--session-id", "sess-1"},
	} {
		if !has(want...) {
			t.Errorf("missing %v in args %q", want, args)
		}
	}
	if !strings.Contains(read("args"), "--system-prompt\n"+agentSystemPrompt) {
		t.Error("app instructions must replace the default coding-agent prompt")
	}
	if has("--resume", "sess-1") {
		t.Error("a new session must not use --resume")
	}
	// The secret and the message must never appear on the command line.
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "secret-token") || strings.Contains(joined, "hello there") {
		t.Errorf("secret or message leaked into argv: %q", joined)
	}
	if strings.TrimSpace(read("stdin")) != "hello there" {
		t.Errorf("stdin = %q", read("stdin"))
	}
	if !strings.Contains(read("mcp.json"), "Bearer secret-token") || !strings.Contains(read("mcp.json"), "http://127.0.0.1:8080/mcp") {
		t.Errorf("mcp config = %q", read("mcp.json"))
	}
	if mode := strings.TrimSpace(read("mcp.mode")); mode != "600" {
		t.Errorf("mcp config mode = %s, want 600", mode)
	}
	if _, err := os.Stat(strings.TrimSpace(read("mcp.path"))); !os.IsNotExist(err) {
		t.Error("temp MCP config should be deleted after the run")
	}
	if cwd, _ := filepath.EvalSymlinks(strings.TrimSpace(read("cwd"))); cwd == "" {
		t.Error("expected a working directory")
	}
}

func TestCLIRunnerResumesAnExistingSession(t *testing.T) {
	runner, out := fakeCLI(t, `{"type":"result","is_error":false}`+"\n", 0)
	if err := runner.Run(context.Background(), agentRequest{Message: "x", SessionID: "sess-9", Resume: true}, func(agentEvent) {}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "args"))
	if args := string(b); !strings.Contains(args, "--resume\nsess-9") || strings.Contains(args, "--session-id") {
		t.Fatalf("args = %q", args)
	}
}

func TestCLIRunnerErrors(t *testing.T) {
	t.Run("error result with a login hint", func(t *testing.T) {
		runner, _ := fakeCLI(t, `{"type":"result","is_error":true,"result":"Not logged in"}`+"\n", 1)
		err := runner.Run(context.Background(), agentRequest{Message: "x", SessionID: "s"}, func(agentEvent) {})
		if err == nil || !strings.Contains(err.Error(), "Not logged in") || !strings.Contains(err.Error(), "run `claude`") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("non-zero exit with no output", func(t *testing.T) {
		runner, _ := fakeCLI(t, "", 3)
		if err := runner.Run(context.Background(), agentRequest{Message: "x", SessionID: "s"}, func(agentEvent) {}); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("claude not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		err := cliRunner{dir: t.TempDir()}.Run(context.Background(), agentRequest{Message: "x"}, func(agentEvent) {})
		if !errors.Is(err, errClaudeMissing) {
			t.Fatalf("err = %v, want errClaudeMissing", err)
		}
	})
	t.Run("cancelling stops the run quietly", func(t *testing.T) {
		runner, _ := fakeCLI(t, "", 0)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := runner.Run(ctx, agentRequest{Message: "x", SessionID: "s"}, func(agentEvent) {}); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})
}

func TestCLIImportUsesStructuredOutputWithoutResumeTools(t *testing.T) {
	runner, out := fakeCLI(t, `{"type":"assistant","message":{"content":[{"type":"text","text":"Organizing your document"}]}}
{"type":"result","is_error":false,"structured_output":{"basics":{"name":"Allen"},"summary":{"text":"Developer"}}}`, 0)
	var events []agentEvent
	if err := runner.Run(context.Background(), agentRequest{Import: true, Message: "Allen Developer"}, func(ev agentEvent) { events = append(events, ev) }); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || !strings.Contains(events[0].Text, `"name":"Allen"`) {
		t.Fatalf("unexpected output: %+v", events)
	}
	raw, _ := os.ReadFile(filepath.Join(out, "args"))
	args := string(raw)
	if !strings.Contains(args, "--json-schema\n") || strings.Contains(args, "--allowedTools") || strings.Contains(args, "--resume\n") {
		t.Fatalf("unsafe import command: %s", args)
	}
	if !strings.Contains(args, "--tools\n\n") || !strings.Contains(args, `{"mcpServers":{}}`) {
		t.Fatal("import tools must be disabled")
	}
}

func TestChatRefreshesSessionsCreatedWithOldInstructions(t *testing.T) {
	for _, version := range []string{"", "obsolete-tool-contract"} {
		t.Run(version, func(t *testing.T) {
			h, app := newTestServer(t)
			r := seedBase(t, h)
			// Simulate an existing conversation created before capability versioning.
			if _, err := app.db.Exec(`UPDATE resumes SET session_id='old-snapshotted-prompt' WHERE id=?`, r.ID); err != nil {
				t.Fatal(err)
			}
			if version != "" {
				if _, err := app.db.Exec(`INSERT INTO settings(key,value) VALUES (?,?)`, fmt.Sprintf("agent-session-version:%d", r.ID), version); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := getResume(app.db, r.ID)
			pending, err := createProposal(app.db, r.ID, opReplace, "summary", "Existing suggestion", "Keep this")
			if err != nil {
				t.Fatal(err)
			}
			fake := &fakeRunner{run: func(_ int, req agentRequest, emit func(agentEvent)) error {
				emit(agentEvent{Type: "text", Text: "I can reorder the document."})
				return nil
			}}
			app.agent = fake
			events := parseSSE(t, chat(t, h, r.ID, "Reorder the sections for review").Body.String())
			if fake.reqs[0].Resume || fake.reqs[0].SessionID == "old-snapshotted-prompt" {
				t.Fatal("resumed the CLI session with stale system instructions")
			}
			if len(events) < 2 || events[0].Type != "notice" {
				t.Fatalf("missing capability refresh notice: %+v", events)
			}
			after, _ := getResume(app.db, r.ID)
			if before.Revision != after.Revision {
				t.Fatal("session refresh changed resume content")
			}
			proposal, _ := getProposal(app.db, pending.ID)
			if proposal.Status != "pending" {
				t.Fatal("session refresh changed pending suggestions")
			}
			current, err := currentAgentSession(app.db, r.ID)
			if err != nil || !current {
				t.Fatalf("new session version not recorded: %v", err)
			}
			next := parseSSE(t, chat(t, h, r.ID, "Continue").Body.String())
			if !fake.reqs[1].Resume || fake.reqs[1].SessionID != fake.reqs[0].SessionID {
				t.Fatal("current session should continue normally")
			}
			for _, event := range next {
				if event.Type == "notice" {
					t.Fatal("current sessions must not reset repeatedly")
				}
			}
		})
	}
}
