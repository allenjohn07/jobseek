package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bearer adds the Authorization header to every request an MCP client makes.
type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// mcpSession starts the real app on a test server and connects a real MCP client to it.
func mcpSession(t *testing.T) (*mcp.ClientSession, *App) {
	t.Helper()
	h, app := newTestServer(t)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	tok, err := mcpToken(app.db)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   srv.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearer{tok}},
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, app
}

// call runs a tool and decodes its structured output into out (when out is non-nil).
// It returns the tool result so callers can check IsError.
func call(t *testing.T, cs *mcp.ClientSession, name string, args any, out any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: protocol error: %v", name, err)
	}
	if out != nil && !res.IsError {
		raw, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s: bad output %s: %v", name, raw, err)
		}
	}
	return res
}

func TestMCPRequiresToken(t *testing.T) {
	h, _ := newTestServer(t)
	srv := httptest.NewServer(h)
	defer srv.Close()
	for name, auth := range map[string]string{"none": "", "wrong": "Bearer nope", "not bearer": "Basic abc"} {
		t.Run(name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{}`))
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", res.StatusCode)
			}
		})
	}
}

func TestMCPTokenIsStable(t *testing.T) {
	_, app := newTestServer(t)
	a, err := mcpToken(app.db)
	b, _ := mcpToken(app.db)
	if err != nil || len(a) != 64 || a != b {
		t.Fatalf("token should be 64 hex chars and stable: %q %q %v", a, b, err)
	}
}

// The AI can directly edit a resume; suggestion acceptance remains a UI action.
func TestMCPToolSet(t *testing.T) {
	cs, _ := mcpSession(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	want := []string{"list_jobs", "get_job", "get_resume", "create_job_resume", "propose_edit", "list_proposals", "update_resume"}
	if len(got) != len(want) {
		t.Fatalf("tools = %v, want exactly %v", got, want)
	}
	for _, n := range want {
		if !got[n] {
			t.Fatalf("missing tool %s in %v", n, got)
		}
	}
}

func TestMCPTailoringFlow(t *testing.T) {
	cs, app := mcpSession(t)
	h := app.routes()
	base := seedBase(t, h)
	do(t, h, "POST", "/api/jobs", `{"company":"Beta","title":"Go Engineer","description":"Needs Go and APIs"}`)

	var jobs listJobsOutput
	call(t, cs, "list_jobs", map[string]any{}, &jobs)
	if len(jobs.Jobs) != 1 || jobs.Jobs[0].Title != "Go Engineer" {
		t.Fatalf("list_jobs = %+v", jobs)
	}

	var job Job
	call(t, cs, "get_job", map[string]any{"job_id": 1}, &job)
	if job.Description != "Needs Go and APIs" {
		t.Fatalf("get_job description = %q", job.Description)
	}

	// A job's resume must be created before it can be read.
	if res := call(t, cs, "get_resume", map[string]any{"job_id": 1}, nil); !res.IsError {
		t.Fatal("get_resume for a job without a resume should be an error")
	}
	var jr Resume
	call(t, cs, "create_job_resume", map[string]any{"job_id": 1}, &jr)
	if jr.JobID == nil || *jr.JobID != 1 || jr.ID == base.ID {
		t.Fatalf("create_job_resume = %+v", jr)
	}
	if res := call(t, cs, "create_job_resume", map[string]any{"job_id": 1}, nil); !res.IsError {
		t.Fatal("second create_job_resume should be an error")
	}

	var got Resume
	call(t, cs, "get_resume", map[string]any{"job_id": 1}, &got)
	bullet := got.Content.Experience[0].Bullets[0]
	if bullet.ID == "" || bullet.Text != "Built APIs" {
		t.Fatalf("get_resume bullet = %+v", bullet)
	}
	var baseOut Resume
	call(t, cs, "get_resume", map[string]any{}, &baseOut)
	if baseOut.ID != base.ID {
		t.Fatalf("omitting job_id should return the base resume, got id %d", baseOut.ID)
	}

	// Proposing records a suggestion and changes nothing.
	var p Proposal
	call(t, cs, "propose_edit", map[string]any{
		"resume_id": jr.ID, "op": "replace", "item_id": bullet.ID,
		"text": "Built Go APIs", "reason": "Job asks for Go",
	}, &p)
	if p.Status != "pending" || p.BeforeText != "Built APIs" || p.AfterText != "Built Go APIs" {
		t.Fatalf("propose_edit = %+v", p)
	}
	if text := getContentOf(t, app, jr.ID).Experience[0].Bullets[0].Text; text != "Built APIs" {
		t.Fatalf("propose_edit changed the resume to %q", text)
	}

	var list listProposalsOutput
	call(t, cs, "list_proposals", map[string]any{"resume_id": jr.ID}, &list)
	if len(list.Proposals) != 1 || list.Proposals[0].ID != p.ID {
		t.Fatalf("list_proposals = %+v", list)
	}

	// Bad suggestions come back as tool errors the AI can read and correct.
	for name, args := range map[string]map[string]any{
		"unknown item": {"resume_id": jr.ID, "op": "replace", "item_id": "nope", "text": "x", "reason": ""},
		"unknown op":   {"resume_id": jr.ID, "op": "rewrite", "item_id": bullet.ID, "text": "x", "reason": ""},
		"empty text":   {"resume_id": jr.ID, "op": "replace", "item_id": bullet.ID, "text": "", "reason": ""},
	} {
		if res := call(t, cs, "propose_edit", args, nil); !res.IsError {
			t.Fatalf("%s should be a tool error", name)
		}
	}
}

func TestMCPInfoEndpoint(t *testing.T) {
	h, app := newTestServer(t)
	info := decode[map[string]string](t, do(t, h, "GET", "/api/mcp/info", ""))
	tok, _ := mcpToken(app.db)
	if info["token"] != tok || info["url"] != "http://127.0.0.1:8080/mcp" ||
		!strings.Contains(info["command"], "claude mcp add --transport http jobseek http://127.0.0.1:8080/mcp") ||
		!strings.Contains(info["command"], tok) {
		t.Fatalf("info = %+v", info)
	}
}

func TestMCPDirectResumeEdit(t *testing.T) {
	cs, app := mcpSession(t)
	base := seedBase(t, app.routes())
	do(t, app.routes(), "POST", "/api/jobs", `{"company":"Acme","title":"Engineer"}`)
	var jobResume Resume
	call(t, cs, "create_job_resume", map[string]any{"job_id": 1}, &jobResume)
	next := jobResume.Content
	next.Basics.Headline = "Backend Engineer"
	next.Experience[0].Role = "Senior Developer"
	next.Education = []Education{{School: "Example University", Degree: "BSc"}}
	var saved Resume
	result := call(t, cs, "update_resume", map[string]any{"resume_id": jobResume.ID, "expected_revision": jobResume.Revision, "content": next}, &saved)
	if result.IsError {
		t.Fatalf("update failed: %+v", result)
	}
	if saved.Content.Basics.Headline != "Backend Engineer" || saved.Content.Experience[0].Role != "Senior Developer" || saved.Content.Education[0].Degree != "BSc" {
		t.Fatalf("content not updated: %+v", saved)
	}
	unchanged, _ := getBaseResume(app.db)
	if unchanged.Revision != base.Revision {
		t.Fatal("tailoring changed the base resume")
	}
	stale := call(t, cs, "update_resume", map[string]any{"resume_id": jobResume.ID, "expected_revision": jobResume.Revision, "content": jobResume.Content}, nil)
	if !stale.IsError {
		t.Fatal("stale edit should fail")
	}
}
