package main

import (
	"fmt"
	"strings"
)

// The document is canonical when present. Structured fields remain an index for
// existing tools and suggestions, linked by stable resumeKey attributes.
func docChildren(node map[string]any) []any { children, _ := node["content"].([]any); return children }
func docAttrs(node map[string]any) map[string]any {
	attrs, _ := node["attrs"].(map[string]any)
	return attrs
}
func docText(node map[string]any) string {
	if node["type"] == "text" {
		text, _ := node["text"].(string)
		return text
	}
	if node["type"] == "hardBreak" {
		return "\n"
	}
	var b strings.Builder
	for _, child := range docChildren(node) {
		if n, ok := child.(map[string]any); ok {
			b.WriteString(docText(n))
		}
	}
	return b.String()
}
func validateDocument(doc map[string]any) error {
	if doc == nil {
		return nil
	}
	allowed := map[string]bool{"doc": true, "paragraph": true, "heading": true, "text": true, "bulletList": true, "orderedList": true, "listItem": true, "blockquote": true, "hardBreak": true, "horizontalRule": true, "codeBlock": true, "table": true, "tableRow": true, "tableCell": true, "tableHeader": true, "resumeEntry": true}
	var walk func(map[string]any, int) error
	count := 0
	walk = func(n map[string]any, depth int) error {
		count++
		kind, _ := n["type"].(string)
		if !allowed[kind] || depth > 60 || count > 20000 {
			return fmt.Errorf("%w: unsupported or overly complex resume document", errInvalid)
		}
		if kind == "text" {
			text, ok := n["text"].(string)
			if !ok || text == "" {
				return fmt.Errorf("%w: invalid document text", errInvalid)
			}
		}
		if raw, ok := n["content"]; ok {
			children, valid := raw.([]any)
			if !valid {
				return fmt.Errorf("%w: invalid document content", errInvalid)
			}
			for _, child := range children {
				node, valid := child.(map[string]any)
				if !valid {
					return fmt.Errorf("%w: invalid document node", errInvalid)
				}
				if err := walk(node, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if doc["type"] != "doc" {
		return fmt.Errorf("%w: document must have a doc root", errInvalid)
	}
	return walk(doc, 0)
}
func (c *ResumeContent) indexDocument() {
	if c.Document == nil {
		return
	}
	keys := map[string]string{}
	entries := map[string]bool{}
	var walk func(map[string]any)
	walk = func(n map[string]any) {
		attrs := docAttrs(n)
		if key, ok := attrs["resumeKey"].(string); ok {
			if _, exists := keys[key]; !exists {
				keys[key] = docText(n)
			}
		}
		if id, ok := attrs["entryId"].(string); ok {
			entries[id] = true
		}
		for _, child := range docChildren(n) {
			if node, ok := child.(map[string]any); ok {
				walk(node)
			}
		}
	}
	walk(c.Document)
	c.Basics = Basics{Name: keys["basics.name"], Headline: keys["basics.headline"], Email: keys["basics.email"], Phone: keys["basics.phone"], Location: keys["basics.location"], Links: keys["basics.links"]}
	c.Summary.Text = keys["item:summary"]
	filter := func(items []Item) []Item {
		out := []Item{}
		for _, item := range items {
			if text, ok := keys["item:"+item.ID]; ok {
				item.Text = text
				out = append(out, item)
			}
		}
		return out
	}
	c.Skills = filter(c.Skills)
	experience := []Experience{}
	for _, e := range c.Experience {
		prefix := "experience:" + e.ID + ":"
		_, hasRole := keys[prefix+"role"]
		if !entries[e.ID] && !hasRole {
			continue
		}
		e.Company = keys[prefix+"company"]
		e.Role = keys[prefix+"role"]
		e.Start = keys[prefix+"start"]
		e.End = keys[prefix+"end"]
		e.Bullets = filter(e.Bullets)
		experience = append(experience, e)
	}
	c.Experience = experience
	education := []Education{}
	for _, e := range c.Education {
		prefix := "education:" + e.ID + ":"
		_, hasSchool := keys[prefix+"school"]
		if !entries[e.ID] && !hasSchool {
			continue
		}
		e.School = keys[prefix+"school"]
		e.Degree = keys[prefix+"degree"]
		e.Start = keys[prefix+"start"]
		e.End = keys[prefix+"end"]
		education = append(education, e)
	}
	c.Education = education
	projects := []Project{}
	for _, e := range c.Projects {
		prefix := "projects:" + e.ID + ":"
		_, hasName := keys[prefix+"name"]
		if !entries[e.ID] && !hasName {
			continue
		}
		e.Name = keys[prefix+"name"]
		e.Link = keys[prefix+"link"]
		e.Bullets = filter(e.Bullets)
		projects = append(projects, e)
	}
	c.Projects = projects
}

func patchDocument(doc map[string]any, p Proposal, addedID string) bool {
	if doc == nil {
		return true
	}
	key := "item:" + p.ItemID
	var contains func(map[string]any) bool
	contains = func(n map[string]any) bool {
		if docAttrs(n)["resumeKey"] == key {
			return true
		}
		for _, child := range docChildren(n) {
			if node, ok := child.(map[string]any); ok && docAttrs(node)["resumeKey"] == key {
				return true
			}
		}
		return false
	}
	text := func(value string) []any {
		if value == "" {
			return []any{}
		}
		return []any{map[string]any{"type": "text", "text": value}}
	}
	var walk func(map[string]any) bool
	walk = func(n map[string]any) bool {
		children := docChildren(n)
		for i, child := range children {
			node, ok := child.(map[string]any)
			if !ok {
				continue
			}
			target := docAttrs(node)["resumeKey"] == key || (node["type"] == "listItem" && contains(node))
			if target {
				switch p.Op {
				case opDelete:
					n["content"] = append(children[:i], children[i+1:]...)
				case opInsertAfter:
					paragraph := map[string]any{"type": "paragraph", "attrs": map[string]any{"resumeKey": "item:" + addedID}, "content": text(p.AfterText)}
					var added any = paragraph
					if node["type"] == "listItem" {
						added = map[string]any{"type": "listItem", "content": []any{paragraph}}
					}
					next := append([]any{}, children[:i+1]...)
					next = append(next, added)
					n["content"] = append(next, children[i+1:]...)
				case opReplace:
					var replace func(map[string]any)
					replace = func(target map[string]any) {
						if docAttrs(target)["resumeKey"] == key {
							replacement := text(p.AfterText)
							if len(replacement) > 0 && len(docChildren(target)) > 0 {
								if first, ok := docChildren(target)[0].(map[string]any); ok && first["marks"] != nil {
									replacement[0].(map[string]any)["marks"] = first["marks"]
								}
							}
							target["content"] = replacement
							return
						}
						for _, c := range docChildren(target) {
							if child, ok := c.(map[string]any); ok {
								replace(child)
							}
						}
					}
					replace(node)
				}
				return true
			}
			if walk(node) {
				if (node["type"] == "bulletList" || node["type"] == "orderedList") && len(docChildren(node)) == 0 {
					n["content"] = append(children[:i], children[i+1:]...)
				}
				return true
			}
		}
		return false
	}
	return walk(doc)
}

func findDocumentNode(node map[string]any, key string) map[string]any {
	if docAttrs(node)["resumeKey"] == key {
		return node
	}
	for _, child := range docChildren(node) {
		if n, ok := child.(map[string]any); ok {
			if found := findDocumentNode(n, key); found != nil {
				return found
			}
		}
	}
	return nil
}
