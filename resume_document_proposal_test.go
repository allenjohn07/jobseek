package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func reorderForTest(t *testing.T, document map[string]any, order []string) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(document)
	var copy map[string]any
	json.Unmarshal(raw, &copy)
	header := []any{}
	groups := map[string][]any{}
	current := ""
	for _, child := range docChildren(copy) {
		node := child.(map[string]any)
		if node["type"] == "heading" && fmt.Sprint(docAttrs(node)["level"]) == "2" {
			current = docText(node)
		}
		if current == "" {
			header = append(header, node)
		} else {
			groups[current] = append(groups[current], node)
		}
	}
	result := header
	for _, name := range order {
		result = append(result, groups[name]...)
	}
	copy["content"] = result
	return copy
}
func sectionOrderForTest(document map[string]any) []string {
	out := []string{}
	for _, child := range docChildren(document) {
		node := child.(map[string]any)
		if node["type"] == "heading" && fmt.Sprint(docAttrs(node)["level"]) == "2" {
			out = append(out, docText(node))
		}
	}
	return out
}
func TestDocumentReorderProposalLifecycle(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		t.Run(fmt.Sprintf("legacy=%v", legacy), func(t *testing.T) {
			cs, app := mcpSession(t)
			base, _ := getBaseResume(app.db)
			c := emptyContent()
			c.Basics.Name = "Allen"
			c.Experience = []Experience{{ID: "exp1", Company: "Acme", Role: "Developer", Bullets: []Item{{ID: "bullet1", Text: "Built APIs"}}}}
			c.Education = []Education{{ID: "edu1", School: "University", Degree: "BSc"}}
			c.Projects = []Project{{ID: "project1", Name: "JobSeek", Bullets: []Item{{ID: "project-bullet", Text: "Built a job tracker"}}}}
			c.Skills = []Item{{ID: "skill1", Text: "Go"}}
			if !legacy {
				c.Document = documentFromContent(c)
			}
			saved, err := saveResumeContent(app.db, base.ID, c)
			if err != nil {
				t.Fatal(err)
			}
			var read Resume
			call(t, cs, "get_resume", map[string]any{}, &read)
			if read.Content.Document == nil {
				t.Fatal("even legacy resumes must expose an editable document to AI")
			}
			order := []string{"Education", "Projects", "Experience", "Skills"}
			next := reorderForTest(t, read.Content.Document, order)
			var proposal Proposal
			result := call(t, cs, "propose_edit", map[string]any{"resume_id": read.ID, "op": "replace_document", "expected_revision": read.Revision, "document": next, "reason": "Put education and projects before experience"}, &proposal)
			if result.IsError {
				t.Fatalf("proposal failed: %+v", result)
			}
			unchanged, _ := getResume(app.db, saved.ID)
			if unchanged.Revision != saved.Revision {
				t.Fatal("proposal applied before review")
			}
			rejected, err := resolveProposal(app.db, proposal.ID, false)
			if err != nil || rejected.Status != "rejected" {
				t.Fatalf("reject: %v", err)
			}
			unchanged, _ = getResume(app.db, saved.ID)
			if unchanged.Revision != saved.Revision {
				t.Fatal("rejection changed content")
			}
			p, err := createDocumentProposal(app.db, saved.ID, saved.Revision, next, "Reorder")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = resolveProposal(app.db, p.ID, true); err != nil {
				t.Fatal(err)
			}
			updated, _ := getResume(app.db, saved.ID)
			if !reflect.DeepEqual(sectionOrderForTest(updated.Content.Document), order) {
				t.Fatalf("wrong section order: %+v", sectionOrderForTest(updated.Content.Document))
			}
			if updated.Content.Experience[0].Bullets[0].Text != "Built APIs" || updated.Content.Education[0].Degree != "BSc" || updated.Content.Projects[0].Bullets[0].Text != "Built a job tracker" {
				t.Fatal("section content lost during reorder")
			}
		})
	}
}
func TestDocumentProposalRejectsStaleAndInvalidChanges(t *testing.T) {
	_, app := newTestServer(t)
	base, _ := getBaseResume(app.db)
	saved, err := saveResumeContent(app.db, base.ID, richContent(t))
	if err != nil {
		t.Fatal(err)
	}
	doc := reorderForTest(t, saved.Content.Document, []string{"Experience", "Summary"})
	doc["content"] = append(docChildren(doc), map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Proposed additional paragraph"}}})
	p, err := createDocumentProposal(app.db, saved.ID, saved.Revision, doc, "Reorder")
	if err != nil {
		t.Fatal(err)
	}
	next := saved.Content
	findDocumentNode(next.Document, "item:summary")["content"] = []any{map[string]any{"type": "text", "text": "A newer manual edit"}}
	fresh, err := saveResumeContent(app.db, saved.ID, next)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = resolveProposal(app.db, p.ID, true); !errors.Is(err, errConflict) {
		t.Fatalf("stale accept should fail: %v", err)
	}
	after, _ := getResume(app.db, saved.ID)
	if after.Revision != fresh.Revision {
		t.Fatal("stale proposal overwrote newer changes")
	}
	if _, err = createDocumentProposal(app.db, saved.ID, saved.Revision, doc, "Stale"); !errors.Is(err, errConflict) {
		t.Fatal("stale proposal generation should fail")
	}
	if _, err = createDocumentProposal(app.db, saved.ID, fresh.Revision, map[string]any{"type": "script"}, "Invalid"); err == nil {
		t.Fatal("invalid document accepted")
	}
}
