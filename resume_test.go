package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return v
}

// seedBase saves a base resume with one job entry and returns it with its assigned IDs.
func seedBase(t *testing.T, h http.Handler) Resume {
	t.Helper()
	base := decode[Resume](t, do(t, h, "GET", "/api/resume/base", ""))
	body := `{"basics":{"name":"Allen"},"summary":{"text":"Backend dev"},
	  "experience":[{"company":"Acme","role":"Dev","bullets":[{"text":"Built APIs"},{"text":"Wrote tests"}]}],
	  "skills":[{"text":"Go, SQL"}]}`
	rec := do(t, h, "PUT", "/api/resumes/"+strconv.FormatInt(base.ID, 10), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("save base = %d %s", rec.Code, rec.Body)
	}
	return decode[Resume](t, rec)
}

func TestBaseResumeStartsEmptyAndSavesWithIDs(t *testing.T) {
	h, _ := newTestServer(t)
	empty := decode[Resume](t, do(t, h, "GET", "/api/resume/base", ""))
	if empty.JobID != nil || empty.Content.Experience == nil || empty.Content.Skills == nil {
		t.Fatalf("empty base should have no job and non-nil slices: %+v", empty)
	}
	saved := seedBase(t, h)
	b := saved.Content.Experience[0].Bullets
	if len(b) != 2 || b[0].ID == "" || b[0].ID == b[1].ID || saved.Content.Summary.ID != "summary" {
		t.Fatalf("items should get unique IDs, got %+v", saved.Content)
	}
	// Saving again with the IDs echoed back must keep them (the UI does this on every edit).
	again, _ := json.Marshal(saved.Content)
	got := decode[Resume](t, do(t, h, "PUT", "/api/resumes/"+strconv.FormatInt(saved.ID, 10), string(again)))
	if got.Content.Experience[0].Bullets[0].ID != b[0].ID {
		t.Fatal("IDs changed on re-save")
	}
	if rec := do(t, h, "PUT", "/api/resumes/999", `{}`); rec.Code != http.StatusNotFound {
		t.Fatalf("missing resume = %d, want 404", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/resumes/"+strconv.FormatInt(saved.ID, 10), `{nope`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d, want 400", rec.Code)
	}
}

func TestJobResumeIsCopyOfBase(t *testing.T) {
	h, _ := newTestServer(t)
	base := seedBase(t, h)
	do(t, h, "POST", "/api/jobs", `{"company":"Beta","title":"Dev"}`)

	if rec := do(t, h, "GET", "/api/jobs/1/resume", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("before create = %d, want 404", rec.Code)
	}
	rec := do(t, h, "POST", "/api/jobs/1/resume", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", rec.Code, rec.Body)
	}
	copyR := decode[Resume](t, rec)
	if copyR.JobID == nil || *copyR.JobID != 1 || copyR.ID == base.ID || copyR.Content.Summary.Text != "Backend dev" {
		t.Fatalf("unexpected copy %+v", copyR)
	}
	if rec := do(t, h, "POST", "/api/jobs/1/resume", ""); rec.Code != http.StatusConflict {
		t.Fatalf("second create = %d, want 409", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/jobs/999/resume", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing job = %d, want 404", rec.Code)
	}

	// Editing the copy must not touch the base.
	c := copyR.Content
	c.Summary.Text = "Tailored"
	raw, _ := json.Marshal(c)
	do(t, h, "PUT", "/api/resumes/"+strconv.FormatInt(copyR.ID, 10), string(raw))
	if b := decode[Resume](t, do(t, h, "GET", "/api/resume/base", "")); b.Content.Summary.Text != "Backend dev" {
		t.Fatalf("base changed to %q", b.Content.Summary.Text)
	}
}

func TestProposalLifecycle(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	rid := strconv.FormatInt(r.ID, 10)
	bullet := r.Content.Experience[0].Bullets[0]

	propose := func(op, itemID, text string) (Proposal, error) {
		return createProposal(app.db, r.ID, op, itemID, text, "because")
	}

	// Proposing changes nothing.
	p, err := propose(opReplace, bullet.ID, "Built Go APIs serving 1M requests")
	if err != nil || p.Status != "pending" || p.BeforeText != "Built APIs" {
		t.Fatalf("propose: %+v %v", p, err)
	}
	if got := getContentOf(t, app, r.ID).Experience[0].Bullets[0].Text; got != "Built APIs" {
		t.Fatalf("proposal changed the resume to %q", got)
	}

	// Accept applies it.
	pid := strconv.FormatInt(p.ID, 10)
	if rec := do(t, h, "POST", "/api/proposals/"+pid+"/accept", ""); rec.Code != http.StatusOK {
		t.Fatalf("accept = %d %s", rec.Code, rec.Body)
	}
	if got := getContentOf(t, app, r.ID).Experience[0].Bullets[0].Text; got != "Built Go APIs serving 1M requests" {
		t.Fatalf("after accept: %q", got)
	}
	// A resolved proposal can't be resolved again.
	for _, action := range []string{"accept", "reject"} {
		if rec := do(t, h, "POST", "/api/proposals/"+pid+"/"+action, ""); rec.Code != http.StatusConflict {
			t.Fatalf("%s twice = %d, want 409", action, rec.Code)
		}
	}

	// Reject leaves the resume alone.
	p2, _ := propose(opReplace, bullet.ID, "Something else")
	if rec := do(t, h, "POST", "/api/proposals/"+strconv.FormatInt(p2.ID, 10)+"/reject", ""); rec.Code != http.StatusOK {
		t.Fatalf("reject = %d", rec.Code)
	}
	if got := getContentOf(t, app, r.ID).Experience[0].Bullets[0].Text; got != "Built Go APIs serving 1M requests" {
		t.Fatalf("reject changed resume: %q", got)
	}

	// Stale: the user edits the item after the suggestion was made -> 409, edit survives.
	p3, _ := propose(opReplace, bullet.ID, "AI rewrite")
	cc := getContentOf(t, app, r.ID)
	cc.Experience[0].Bullets[0].Text = "My manual edit"
	do(t, h, "PUT", "/api/resumes/"+rid, mustJSON(t, cc))
	if rec := do(t, h, "POST", "/api/proposals/"+strconv.FormatInt(p3.ID, 10)+"/accept", ""); rec.Code != http.StatusConflict {
		t.Fatalf("stale accept = %d, want 409", rec.Code)
	}
	if got := getContentOf(t, app, r.ID).Experience[0].Bullets[0].Text; got != "My manual edit" {
		t.Fatalf("stale accept overwrote manual edit: %q", got)
	}

	if rec := do(t, h, "POST", "/api/proposals/999/accept", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing proposal = %d, want 404", rec.Code)
	}
	list := decode[[]Proposal](t, do(t, h, "GET", "/api/resumes/"+rid+"/proposals", ""))
	if len(list) != 3 || list[0].ID != p3.ID {
		t.Fatalf("expected 3 proposals newest first, got %+v", list)
	}
}

func TestProposalInsertAndDelete(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	first := r.Content.Experience[0].Bullets[0]

	ins, err := createProposal(app.db, r.ID, opInsertAfter, first.ID, "Cut deploy time 40%", "add impact")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveProposal(app.db, ins.ID, true); err != nil {
		t.Fatal(err)
	}
	b := getContentOf(t, app, r.ID).Experience[0].Bullets
	if len(b) != 3 || b[1].Text != "Cut deploy time 40%" || b[1].ID == "" || b[2].Text != "Wrote tests" {
		t.Fatalf("insert_after wrong: %+v", b)
	}

	del, _ := createProposal(app.db, r.ID, opDelete, first.ID, "ignored", "not relevant")
	if del.AfterText != "" {
		t.Fatalf("delete should clear text, got %q", del.AfterText)
	}
	if _, err := resolveProposal(app.db, del.ID, true); err != nil {
		t.Fatal(err)
	}
	if b := getContentOf(t, app, r.ID).Experience[0].Bullets; len(b) != 2 || b[0].Text != "Cut deploy time 40%" {
		t.Fatalf("delete wrong: %+v", b)
	}
}

func TestCreateProposalValidation(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	bullet := r.Content.Experience[0].Bullets[0].ID
	tests := []struct {
		name, op, item, text string
	}{
		{"unknown item", opReplace, "nope", "x"},
		{"unknown op", "rewrite", bullet, "x"},
		{"empty replace text", opReplace, bullet, ""},
		{"empty insert text", opInsertAfter, bullet, ""},
		{"insert after summary", opInsertAfter, "summary", "x"},
		{"delete summary", opDelete, "summary", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := createProposal(app.db, r.ID, tt.op, tt.item, tt.text, ""); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	// The summary itself can be rewritten.
	if _, err := createProposal(app.db, r.ID, opReplace, "summary", "New summary", ""); err != nil {
		t.Fatalf("summary replace: %v", err)
	}
}

func TestDeleteJobRemovesItsResume(t *testing.T) {
	h, app := newTestServer(t)
	seedBase(t, h)
	do(t, h, "POST", "/api/jobs", `{"company":"Beta","title":"Dev"}`)
	jr := decode[Resume](t, do(t, h, "POST", "/api/jobs/1/resume", ""))
	if _, err := createProposal(app.db, jr.ID, opReplace, "summary", "x", ""); err != nil {
		t.Fatal(err)
	}

	if rec := do(t, h, "DELETE", "/api/jobs/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
	var resumes, proposals int
	app.db.QueryRow(`SELECT COUNT(*) FROM resumes WHERE job_id IS NOT NULL`).Scan(&resumes)
	app.db.QueryRow(`SELECT COUNT(*) FROM proposals`).Scan(&proposals)
	if resumes != 0 || proposals != 0 {
		t.Fatalf("leftovers: %d resumes, %d proposals", resumes, proposals)
	}
	if _, err := getBaseResume(app.db); err != nil {
		t.Fatalf("base resume must survive: %v", err)
	}
}

func getContentOf(t *testing.T, app *App, id int64) ResumeContent {
	t.Helper()
	r, err := getResume(app.db, id)
	if err != nil {
		t.Fatal(err)
	}
	return r.Content
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestIdenticalPendingProposalIsNotDuplicated(t *testing.T) {
	h, app := newTestServer(t)
	r := seedBase(t, h)
	id := r.Content.Experience[0].Bullets[0].ID

	a, _ := createProposal(app.db, r.ID, opReplace, id, "Built Go APIs", "first")
	b, err := createProposal(app.db, r.ID, opReplace, id, "Built Go APIs", "again")
	if err != nil || a.ID != b.ID {
		t.Fatalf("expected the same proposal back, got %d and %d (%v)", a.ID, b.ID, err)
	}
	c, _ := createProposal(app.db, r.ID, opReplace, id, "Built Go APIs at scale", "different")
	if c.ID == a.ID {
		t.Fatal("a different suggestion must be stored separately")
	}
	// Once rejected, the same suggestion may be proposed again.
	if _, err := resolveProposal(app.db, a.ID, false); err != nil {
		t.Fatal(err)
	}
	d, _ := createProposal(app.db, r.ID, opReplace, id, "Built Go APIs", "retry")
	if d.ID == a.ID {
		t.Fatal("a resolved proposal must not be reused")
	}
	if ps, _ := listProposals(app.db, r.ID); len(ps) != 3 {
		t.Fatalf("expected 3 proposals, got %d", len(ps))
	}
}

func TestAIUpdateAllSectionsAndRejectStaleRevision(t *testing.T) {
	_, app := newTestServer(t)
	base, err := getBaseResume(app.db)
	if err != nil {
		t.Fatal(err)
	}
	next := base.Content
	next.Basics.Name = "Allen"
	next.Basics.Email = "allen@example.com"
	next.Experience = []Experience{{Company: "Acme", Role: "Engineer", Bullets: []Item{{Text: "Built services"}}}}
	next.Education = []Education{{School: "Example University", Degree: "BSc"}}
	next.Projects = []Project{{Name: "JobSeek", Bullets: []Item{{Text: "Built a job tracker"}}}}
	saved, err := updateResumeFromAI(app.db, base.ID, base.Revision, next)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Content.Basics.Email != next.Basics.Email || saved.Content.Experience[0].Company != "Acme" || saved.Content.Education[0].Degree != "BSc" || saved.Content.Projects[0].Name != "JobSeek" {
		t.Fatalf("incomplete update: %+v", saved.Content)
	}
	if saved.Revision == base.Revision || saved.Content.Experience[0].ID == "" {
		t.Fatal("revision or item IDs missing")
	}
	if _, err = updateResumeFromAI(app.db, base.ID, base.Revision, base.Content); !errors.Is(err, errConflict) {
		t.Fatalf("expected stale conflict: %v", err)
	}
	fresh, _ := getResume(app.db, base.ID)
	if fresh.Content.Basics.Name != "Allen" {
		t.Fatal("stale update overwrote resume")
	}
}
