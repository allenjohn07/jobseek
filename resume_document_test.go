package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const richFixture = `{"type":"doc","content":[{"type":"heading","attrs":{"level":1,"resumeKey":"basics.name"},"content":[{"type":"text","text":"Allen"}]},{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Summary"}]},{"type":"paragraph","attrs":{"resumeKey":"item:summary"},"content":[{"type":"text","text":"Updated summary","marks":[{"type":"bold"}]}]},{"type":"resumeEntry","attrs":{"entryId":"exp1"},"content":[{"type":"paragraph","attrs":{"resumeKey":"experience:exp1:role"},"content":[{"type":"text","text":"Developer"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","attrs":{"resumeKey":"item:bullet1"},"content":[{"type":"text","text":"Built APIs","marks":[{"type":"italic"}]}]}]}]}]},{"type":"paragraph","content":[{"type":"text","text":"A new freely typed section"}]}]}`

func richContent(t *testing.T) ResumeContent {
	t.Helper()
	c := emptyContent()
	c.Experience = []Experience{{ID: "exp1", Role: "Old role", Bullets: []Item{{ID: "bullet1", Text: "Old bullet"}}}}
	if err := json.Unmarshal([]byte(richFixture), &c.Document); err != nil {
		t.Fatal(err)
	}
	return c
}
func TestRichDocumentPersistenceAndIndex(t *testing.T) {
	_, app := newTestServer(t)
	base, _ := getBaseResume(app.db)
	saved, err := saveResumeContent(app.db, base.ID, richContent(t))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Content.Summary.Text != "Updated summary" || saved.Content.Experience[0].Role != "Developer" || saved.Content.Experience[0].Bullets[0].Text != "Built APIs" {
		t.Fatalf("index not synchronized: %+v", saved.Content)
	}
	raw, _ := json.Marshal(saved.Content.Document)
	if !strings.Contains(string(raw), `"italic"`) || !strings.Contains(string(raw), "A new freely typed section") {
		t.Fatal("formatting or free content lost")
	}
	// Full document edits from AI are canonical and survive the same persistence path.
	next := saved.Content
	children := docChildren(next.Document)
	children[0].(map[string]any)["content"] = []any{map[string]any{"type": "text", "text": "Allen John"}}
	updated, err := updateResumeFromAI(app.db, saved.ID, saved.Revision, next)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content.Basics.Name != "Allen John" {
		t.Fatal("AI document update not indexed")
	}
	legacy := updated.Content
	legacy.Document = nil
	if _, err := updateResumeFromAI(app.db, saved.ID, updated.Revision, legacy); err == nil {
		t.Fatal("AI must not discard a rich document")
	}
}
func TestProposalsUpdateRichDocument(t *testing.T) {
	for _, op := range []string{opReplace, opInsertAfter, opDelete} {
		t.Run(op, func(t *testing.T) {
			_, app := newTestServer(t)
			base, _ := getBaseResume(app.db)
			saved, err := saveResumeContent(app.db, base.ID, richContent(t))
			if err != nil {
				t.Fatal(err)
			}
			proposal, err := createProposal(app.db, saved.ID, op, "bullet1", "Improved APIs", "Reason")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = resolveProposal(app.db, proposal.ID, true); err != nil {
				t.Fatal(err)
			}
			updated, _ := getResume(app.db, saved.ID)
			raw, _ := json.Marshal(updated.Content.Document)
			if !strings.Contains(string(raw), "A new freely typed section") {
				t.Fatal("unrelated document content lost")
			}
			if op == opDelete {
				if strings.Contains(string(raw), "Built APIs") {
					t.Fatal("bullet not deleted from document")
				}
			} else if !strings.Contains(string(raw), "Improved APIs") {
				t.Fatal("change not applied to document")
			}
			if op == opReplace && !strings.Contains(string(raw), `"italic"`) {
				t.Fatal("formatting lost on replacement")
			}
		})
	}
}
func TestRichDocumentValidation(t *testing.T) {
	for _, raw := range []string{`{"type":"script"}`, `{"type":"doc","content":"oops"}`, `{"type":"doc","content":[{"type":"text","text":""}]}`} {
		var doc map[string]any
		json.Unmarshal([]byte(raw), &doc)
		if validateDocument(doc) == nil {
			t.Fatalf("accepted invalid document %s", raw)
		}
	}
}

func TestProposalsCanEditNewDocumentBlocks(t *testing.T) {
	_, app := newTestServer(t)
	base, _ := getBaseResume(app.db)
	c := richContent(t)
	children := docChildren(c.Document)
	children = append(children, map[string]any{"type": "paragraph", "attrs": map[string]any{"resumeKey": "item:document-extra"}, "content": []any{map[string]any{"type": "text", "text": "New achievement"}}})
	c.Document["content"] = children
	saved, err := saveResumeContent(app.db, base.ID, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{opReplace, opInsertAfter, opDelete} {
		p, err := createProposal(app.db, saved.ID, op, "document-extra", "Stronger achievement", "Job fit")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = resolveProposal(app.db, p.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	final, _ := getResume(app.db, saved.ID)
	if findDocumentNode(final.Content.Document, "item:document-extra") != nil {
		t.Fatal("generic document paragraph was not deleted")
	}
	if !strings.Contains(docText(final.Content.Document), "Stronger achievement") {
		t.Fatal("inserted document paragraph was lost")
	}
}

func TestAIEditingLegacyIndexCannotSilentlyIgnoreDocument(t *testing.T) {
	_, app := newTestServer(t)
	base, _ := getBaseResume(app.db)
	saved, err := saveResumeContent(app.db, base.ID, richContent(t))
	if err != nil {
		t.Fatal(err)
	}
	next := saved.Content
	next.Basics.Name = "Different name"
	if _, err := updateResumeFromAI(app.db, saved.ID, saved.Revision, next); err == nil {
		t.Fatal("AI must edit the canonical document rather than only its index")
	}
}
