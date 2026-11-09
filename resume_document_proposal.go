package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// A whole-document proposal shares the existing review lifecycle. Snapshots are
// kept in before_text/after_text so old databases need no schema migration.
func createDocumentProposal(db *sql.DB, id int64, revision string, document map[string]any, reason string) (Proposal, error) {
	current, err := getResume(db, id)
	if err != nil {
		return Proposal{}, err
	}
	if revision == "" || revision != current.Revision {
		return Proposal{}, fmt.Errorf("%w: reread the resume before proposing document changes", errConflict)
	}
	if document == nil {
		return Proposal{}, fmt.Errorf("%w: document is required", errInvalid)
	}
	if err := validateDocument(document); err != nil {
		return Proposal{}, err
	}
	before, err := json.Marshal(current.Content)
	if err != nil {
		return Proposal{}, err
	}
	next := current.Content
	next.Document = document
	next.normalize()
	after, err := json.Marshal(next)
	if err != nil {
		return Proposal{}, err
	}
	if string(before) == string(after) {
		return Proposal{}, fmt.Errorf("%w: the proposed document has no changes", errInvalid)
	}
	var existing int64
	err = db.QueryRow(`SELECT id FROM proposals WHERE resume_id=? AND op=? AND before_text=? AND after_text=? AND status='pending'`, id, opReplaceDocument, string(before), string(after)).Scan(&existing)
	if err == nil {
		return getProposal(db, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Proposal{}, err
	}
	result, err := db.Exec(`INSERT INTO proposals (resume_id,op,item_id,before_text,after_text,reason) VALUES (?,?,'',?,?,?)`, id, opReplaceDocument, string(before), string(after), reason)
	if err != nil {
		return Proposal{}, err
	}
	proposalID, err := result.LastInsertId()
	if err != nil {
		return Proposal{}, err
	}
	return getProposal(db, proposalID)
}

func (c *ResumeContent) applyDocumentProposal(p Proposal) error {
	before, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if string(before) != p.BeforeText {
		return fmt.Errorf("%w: resume changed since this document was suggested; generate a fresh suggestion", errConflict)
	}
	var next ResumeContent
	if err := json.Unmarshal([]byte(p.AfterText), &next); err != nil {
		return err
	}
	if next.Document == nil {
		return fmt.Errorf("%w: proposed document is missing", errInvalid)
	}
	if err := validateDocument(next.Document); err != nil {
		return err
	}
	next.normalize()
	*c = next
	return nil
}

// Legacy resumes get the same editable document representation used by the UI.
// This is only a tool response until an edit is applied or accepted.
func documentFromContent(c ResumeContent) map[string]any {
	if c.Document != nil {
		return c.Document
	}
	paragraph := func(text, key string) map[string]any {
		node := map[string]any{"type": "paragraph", "content": []any{}}
		if key != "" {
			node["attrs"] = map[string]any{"resumeKey": key}
		}
		if text != "" {
			node["content"] = []any{map[string]any{"type": "text", "text": text}}
		}
		return node
	}
	heading := func(text string, level int, key string) map[string]any {
		n := paragraph(text, key)
		n["type"] = "heading"
		attrs := docAttrs(n)
		if attrs == nil {
			attrs = map[string]any{}
		}
		attrs["level"] = level
		n["attrs"] = attrs
		return n
	}
	bullets := func(items []Item) map[string]any {
		nodes := []any{}
		for _, item := range items {
			nodes = append(nodes, map[string]any{"type": "listItem", "content": []any{paragraph(item.Text, "item:"+item.ID)}})
		}
		return map[string]any{"type": "bulletList", "content": nodes}
	}
	entry := func(id string, nodes []any) map[string]any {
		return map[string]any{"type": "resumeEntry", "attrs": map[string]any{"entryId": id}, "content": nodes}
	}
	nodes := []any{heading(c.Basics.Name, 1, "basics.name")}
	if c.Basics.Headline != "" {
		nodes = append(nodes, paragraph(c.Basics.Headline, "basics.headline"))
	}
	for _, field := range []struct{ key, text string }{{"email", c.Basics.Email}, {"phone", c.Basics.Phone}, {"location", c.Basics.Location}, {"links", c.Basics.Links}} {
		if field.text != "" {
			nodes = append(nodes, paragraph(field.text, "basics."+field.key))
		}
	}
	if c.Summary.Text != "" {
		nodes = append(nodes, heading("Summary", 2, ""), paragraph(c.Summary.Text, "item:summary"))
	}
	if len(c.Experience) > 0 {
		nodes = append(nodes, heading("Experience", 2, ""))
		for _, e := range c.Experience {
			prefix := "experience:" + e.ID + ":"
			children := []any{heading(e.Role, 3, prefix+"role"), paragraph(e.Company, prefix+"company"), paragraph(e.Start, prefix+"start"), paragraph(e.End, prefix+"end")}
			if len(e.Bullets) > 0 {
				children = append(children, bullets(e.Bullets))
			}
			nodes = append(nodes, entry(e.ID, children))
		}
	}
	if len(c.Projects) > 0 {
		nodes = append(nodes, heading("Projects", 2, ""))
		for _, e := range c.Projects {
			prefix := "projects:" + e.ID + ":"
			children := []any{heading(e.Name, 3, prefix+"name"), paragraph(e.Link, prefix+"link")}
			if len(e.Bullets) > 0 {
				children = append(children, bullets(e.Bullets))
			}
			nodes = append(nodes, entry(e.ID, children))
		}
	}
	if len(c.Education) > 0 {
		nodes = append(nodes, heading("Education", 2, ""))
		for _, e := range c.Education {
			prefix := "education:" + e.ID + ":"
			nodes = append(nodes, entry(e.ID, []any{heading(e.School, 3, prefix+"school"), paragraph(e.Degree, prefix+"degree"), paragraph(e.Start, prefix+"start"), paragraph(e.End, prefix+"end")}))
		}
	}
	if len(c.Skills) > 0 {
		nodes = append(nodes, heading("Skills", 2, ""))
		for _, item := range c.Skills {
			nodes = append(nodes, paragraph(item.Text, "item:"+item.ID))
		}
	}
	return map[string]any{"type": "doc", "content": nodes}
}
