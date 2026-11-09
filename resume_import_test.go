package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestImportResumeDraftDoesNotSave(t *testing.T) {
	h, a := newTestServer(t)
	base, _ := getBaseResume(a.db)
	a.agent = &fakeRunner{run: func(_ int, req agentRequest, emit func(agentEvent)) error {
		if !req.Import || req.Resume || req.SessionID != "" {
			t.Fatalf("not an isolated import: %+v", req)
		}
		emit(agentEvent{Type: "text", Text: `{"basics":{"name":"Allen"},"summary":{"text":"Backend developer"},"skills":[{"id":"same","text":"Go"},{"id":"same","text":"SQL"}]}`})
		return nil
	}}
	rec := do(t, h, "POST", "/api/resume/import", `{"text":"Allen, Backend developer, Go, SQL"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d: %s", rec.Code, rec.Body.String())
	}
	var c ResumeContent
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	if c.Basics.Name != "Allen" || c.Skills[0].ID == c.Skills[1].ID || c.Experience == nil {
		t.Fatalf("invalid draft: %+v", c)
	}
	saved, _ := getBaseResume(a.db)
	if saved.ID != base.ID || saved.Content.Basics.Name != "" {
		t.Fatal("import changed saved base")
	}
}

func TestImportResumeRejectsInvalidInputAndOutput(t *testing.T) {
	for _, body := range []string{`{`, `{"text":"  "}`, `{"text":"` + strings.Repeat("x", (100<<10)+1) + `"}`} {
		h, a := newTestServer(t)
		a.agent = &fakeRunner{run: func(_ int, _ agentRequest, _ func(agentEvent)) error { t.Fatal("unexpected AI call"); return nil }}
		if rec := do(t, h, "POST", "/api/resume/import", body); rec.Code != 400 {
			t.Fatalf("got %d", rec.Code)
		}
	}
	for _, output := range []string{`garbage`, `{}`, `{"basics":{},"summary":{}}`, `{"basics":{"name":"Allen"},"summary":"wrong type"}`} {
		h, a := newTestServer(t)
		a.agent = &fakeRunner{run: func(_ int, _ agentRequest, emit func(agentEvent)) error {
			emit(agentEvent{Type: "text", Text: output})
			return nil
		}}
		if rec := do(t, h, "POST", "/api/resume/import", `{"text":"Resume"}`); rec.Code != 502 {
			t.Fatalf("output %q: got %d", output, rec.Code)
		}
	}
	h, a := newTestServer(t)
	a.agent = &fakeRunner{run: func(_ int, _ agentRequest, _ func(agentEvent)) error { return errors.New("Claude sign-in required") }}
	if rec := do(t, h, "POST", "/api/resume/import", `{"text":"Resume"}`); rec.Code != 502 || !strings.Contains(rec.Body.String(), "sign-in") {
		t.Fatal(rec.Body.String())
	}
}
