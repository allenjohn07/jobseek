package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// Item is one editable piece of text (a bullet, the summary, a skills line).
// Its ID is stable, so AI suggestions can target an item rather than a position.
type Item struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Basics struct {
	Name     string `json:"name"`
	Headline string `json:"headline"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Links    string `json:"links"` // free text, e.g. "github.com/you · linkedin.com/in/you"
}

type Experience struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Role    string `json:"role"`
	Start   string `json:"start"`
	End     string `json:"end"`
	Bullets []Item `json:"bullets"`
}

type Education struct {
	ID     string `json:"id"`
	School string `json:"school"`
	Degree string `json:"degree"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

type Project struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Link    string `json:"link"`
	Bullets []Item `json:"bullets"`
}

// ResumeContent is the whole resume as data. Slices are always non-nil so JSON has [] not null.
type ResumeContent struct {
	Document   map[string]any `json:"document,omitempty"`
	Basics     Basics         `json:"basics"`
	Summary    Item           `json:"summary"`
	Experience []Experience   `json:"experience"`
	Education  []Education    `json:"education"`
	Skills     []Item         `json:"skills"`
	Projects   []Project      `json:"projects"`
}

// Resume is a stored resume. JobID is nil for the base resume.
type Resume struct {
	Revision  string        `json:"revision"`
	ID        int64         `json:"id"`
	JobID     *int64        `json:"jobId"`
	Content   ResumeContent `json:"content"`
	SessionID string        `json:"sessionId"`
	UpdatedAt string        `json:"updatedAt"`
}

type Proposal struct {
	ID         int64  `json:"id"`
	ResumeID   int64  `json:"resumeId"`
	Op         string `json:"op"` // replace | insert_after | delete
	ItemID     string `json:"itemId"`
	BeforeText string `json:"beforeText"`
	AfterText  string `json:"afterText"`
	Reason     string `json:"reason"`
	Status     string `json:"status"` // pending | accepted | rejected
	CreatedAt  string `json:"createdAt"`
}

const (
	opReplace         = "replace"
	opInsertAfter     = "insert_after"
	opDelete          = "delete"
	opReplaceDocument = "replace_document"
)

var (
	// errConflict: the request is valid but can't be applied to the current state
	// (proposal already resolved, or the item's text changed since it was proposed).
	errConflict = errors.New("conflict")
	// errInvalid: the request itself is malformed (unknown op or item, empty text).
	errInvalid = errors.New("invalid")
)

const resumeSchema = `
CREATE TABLE IF NOT EXISTS resumes (
  id         INTEGER PRIMARY KEY,
  job_id     INTEGER UNIQUE,
  content    TEXT NOT NULL,
  session_id TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS proposals (
  id          INTEGER PRIMARY KEY,
  resume_id   INTEGER NOT NULL,
  op          TEXT NOT NULL,
  item_id     TEXT NOT NULL,
  before_text TEXT NOT NULL DEFAULT '',
  after_text  TEXT NOT NULL DEFAULT '',
  reason      TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'pending',
  created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the OS random source failing is unrecoverable
	}
	return hex.EncodeToString(b)
}

func fillID(id *string) {
	if *id == "" {
		*id = newID()
	}
}

// normalize makes sure every slice is non-nil and every item has an ID.
func (c *ResumeContent) normalize() {
	c.indexDocument()
	if c.Experience == nil {
		c.Experience = []Experience{}
	}
	if c.Education == nil {
		c.Education = []Education{}
	}
	if c.Skills == nil {
		c.Skills = []Item{}
	}
	if c.Projects == nil {
		c.Projects = []Project{}
	}
	c.Summary.ID = "summary" // fixed ID: there is exactly one summary
	for i := range c.Skills {
		fillID(&c.Skills[i].ID)
	}
	for i := range c.Experience {
		e := &c.Experience[i]
		fillID(&e.ID)
		if e.Bullets == nil {
			e.Bullets = []Item{}
		}
		for j := range e.Bullets {
			fillID(&e.Bullets[j].ID)
		}
	}
	for i := range c.Education {
		fillID(&c.Education[i].ID)
	}
	for i := range c.Projects {
		p := &c.Projects[i]
		fillID(&p.ID)
		if p.Bullets == nil {
			p.Bullets = []Item{}
		}
		for j := range p.Bullets {
			fillID(&p.Bullets[j].ID)
		}
	}
}

// itemLists returns every list of Items in the resume, so lookups work the same everywhere.
func (c *ResumeContent) itemLists() []*[]Item {
	lists := []*[]Item{&c.Skills}
	for i := range c.Experience {
		lists = append(lists, &c.Experience[i].Bullets)
	}
	for i := range c.Projects {
		lists = append(lists, &c.Projects[i].Bullets)
	}
	return lists
}

// findItem locates an item by ID. list/idx are nil/-1 for the summary, which isn't in a list.
func (c *ResumeContent) findItem(id string) (item *Item, list *[]Item, idx int, ok bool) {
	if id == "summary" {
		return &c.Summary, nil, -1, true
	}
	for _, l := range c.itemLists() {
		for i := range *l {
			if (*l)[i].ID == id {
				return &(*l)[i], l, i, true
			}
		}
	}
	if c.Document != nil {
		if node := findDocumentNode(c.Document, "item:"+id); node != nil {
			return &Item{ID: id, Text: docText(node)}, nil, -1, true
		}
	}
	return nil, nil, -1, false
}

// apply performs one proposal's change on the content.
func (c *ResumeContent) apply(p Proposal) error {
	if p.Op == opReplaceDocument {
		return c.applyDocumentProposal(p)
	}
	item, list, idx, ok := c.findItem(p.ItemID)
	if !ok {
		return fmt.Errorf("%w: item %q no longer exists", errConflict, p.ItemID)
	}
	if item.Text != p.BeforeText {
		return fmt.Errorf("%w: item changed since this was suggested", errConflict)
	}
	if c.Document != nil && list == nil && p.ItemID != "summary" {
		if !patchDocument(c.Document, p, newID()) {
			return fmt.Errorf("%w: document item no longer exists", errConflict)
		}
		return nil
	}
	addedID := ""
	switch p.Op {
	case opReplace:
		item.Text = p.AfterText
	case opInsertAfter:
		if list == nil {
			return fmt.Errorf("%w: cannot insert after the summary", errInvalid)
		}
		added := Item{ID: newID(), Text: p.AfterText}
		addedID = added.ID
		*list = append((*list)[:idx+1], append([]Item{added}, (*list)[idx+1:]...)...)
	case opDelete:
		if list == nil {
			return fmt.Errorf("%w: cannot delete the summary", errInvalid)
		}
		*list = append((*list)[:idx], (*list)[idx+1:]...)
	default:
		return fmt.Errorf("%w: unknown op %q", errInvalid, p.Op)
	}
	if !patchDocument(c.Document, p, addedID) {
		return fmt.Errorf("%w: this item was removed from the document", errConflict)
	}
	return nil
}

func emptyContent() ResumeContent {
	c := ResumeContent{}
	c.normalize()
	return c
}

func scanResume(row interface{ Scan(...any) error }) (Resume, error) {
	var r Resume
	var raw string
	var jobID sql.NullInt64
	if err := row.Scan(&r.ID, &jobID, &raw, &r.SessionID, &r.UpdatedAt); err != nil {
		return r, err
	}
	if jobID.Valid {
		r.JobID = &jobID.Int64
	}
	if err := json.Unmarshal([]byte(raw), &r.Content); err != nil {
		return r, err
	}
	r.Content.normalize()
	r.Revision = fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
	return r, nil
}

const resumeCols = `id, job_id, content, session_id, updated_at`

func getResume(db *sql.DB, id int64) (Resume, error) {
	return scanResume(db.QueryRow(`SELECT `+resumeCols+` FROM resumes WHERE id = ?`, id))
}

// getBaseResume returns the base resume, creating an empty one the first time.
func getBaseResume(db *sql.DB) (Resume, error) {
	r, err := scanResume(db.QueryRow(`SELECT ` + resumeCols + ` FROM resumes WHERE job_id IS NULL`))
	if !errors.Is(err, sql.ErrNoRows) {
		return r, err
	}
	raw, _ := json.Marshal(emptyContent())
	res, err := db.Exec(`INSERT INTO resumes (job_id, content) VALUES (NULL, ?)`, string(raw))
	if err != nil {
		return Resume{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Resume{}, err
	}
	return getResume(db, id)
}

// getJobResume returns the resume tailored for a job, or sql.ErrNoRows if none exists yet.
func getJobResume(db *sql.DB, jobID int64) (Resume, error) {
	return scanResume(db.QueryRow(`SELECT `+resumeCols+` FROM resumes WHERE job_id = ?`, jobID))
}

// createJobResume copies the base resume for a job. It returns sql.ErrNoRows if the job
// doesn't exist and errConflict if the job already has a resume.
func createJobResume(db *sql.DB, jobID int64) (Resume, error) {
	if _, err := getJob(db, jobID); err != nil {
		return Resume{}, err
	}
	if _, err := getJobResume(db, jobID); err == nil {
		return Resume{}, errConflict
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Resume{}, err
	}
	base, err := getBaseResume(db)
	if err != nil {
		return Resume{}, err
	}
	raw, err := json.Marshal(base.Content)
	if err != nil {
		return Resume{}, err
	}
	res, err := db.Exec(`INSERT INTO resumes (job_id, content) VALUES (?, ?)`, jobID, string(raw))
	if err != nil {
		return Resume{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Resume{}, err
	}
	return getResume(db, id)
}

func saveResumeContent(db *sql.DB, id int64, c ResumeContent) (Resume, error) {
	if err := validateDocument(c.Document); err != nil {
		return Resume{}, err
	}
	c.normalize()
	raw, err := json.Marshal(c)
	if err != nil {
		return Resume{}, err
	}
	res, err := db.Exec(`UPDATE resumes SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(raw), id)
	if err != nil {
		return Resume{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Resume{}, sql.ErrNoRows
	}
	return getResume(db, id)
}

func saveSessionID(db *sql.DB, id int64, sessionID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE resumes SET session_id = ? WHERE id = ?`, sessionID, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO settings (key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, fmt.Sprintf("agent-session-version:%d", id), agentSessionVersion()); err != nil {
		return err
	}
	return tx.Commit()
}

// createProposal records a pending suggestion. It never changes the resume itself:
// only acceptProposal does, and only the user can call that.
func createProposal(db *sql.DB, resumeID int64, op, itemID, text, reason string) (Proposal, error) {
	r, err := getResume(db, resumeID)
	if err != nil {
		return Proposal{}, err
	}
	item, list, _, ok := r.Content.findItem(itemID)
	if !ok {
		return Proposal{}, fmt.Errorf("%w: no item with id %q", errInvalid, itemID)
	}
	switch op {
	case opReplace, opInsertAfter:
		if text == "" {
			return Proposal{}, fmt.Errorf("%w: text is required for %s", errInvalid, op)
		}
		if op == opInsertAfter && list == nil && item.ID == "summary" {
			return Proposal{}, fmt.Errorf("%w: cannot insert after the summary", errInvalid)
		}
	case opDelete:
		if list == nil && item.ID == "summary" {
			return Proposal{}, fmt.Errorf("%w: cannot delete the summary", errInvalid)
		}
		text = ""
	default:
		return Proposal{}, fmt.Errorf("%w: op must be replace, insert_after or delete", errInvalid)
	}
	// If the same change is already waiting for review, return it instead of stacking a duplicate
	// (the AI often re-suggests the same edit across chat turns).
	var existing int64
	err = db.QueryRow(`SELECT id FROM proposals WHERE resume_id = ? AND op = ? AND item_id = ? AND before_text = ? AND after_text = ? AND status = 'pending'`,
		resumeID, op, itemID, item.Text, text).Scan(&existing)
	if err == nil {
		return getProposal(db, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Proposal{}, err
	}
	res, err := db.Exec(`INSERT INTO proposals (resume_id, op, item_id, before_text, after_text, reason)
	                     VALUES (?, ?, ?, ?, ?, ?)`, resumeID, op, itemID, item.Text, text, reason)
	if err != nil {
		return Proposal{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Proposal{}, err
	}
	return getProposal(db, id)
}

const proposalCols = `id, resume_id, op, item_id, before_text, after_text, reason, status, created_at`

func scanProposal(row interface{ Scan(...any) error }) (Proposal, error) {
	var p Proposal
	err := row.Scan(&p.ID, &p.ResumeID, &p.Op, &p.ItemID, &p.BeforeText, &p.AfterText, &p.Reason, &p.Status, &p.CreatedAt)
	return p, err
}

func getProposal(db *sql.DB, id int64) (Proposal, error) {
	return scanProposal(db.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE id = ?`, id))
}

// listProposals returns a resume's proposals, newest first.
func listProposals(db *sql.DB, resumeID int64) ([]Proposal, error) {
	rows, err := db.Query(`SELECT `+proposalCols+` FROM proposals WHERE resume_id = ? ORDER BY id DESC`, resumeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Proposal{}
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// resolveProposal accepts (applies) or rejects a pending proposal in one transaction.
// Accepting fails with errConflict if the proposal was already resolved or the item changed.
func resolveProposal(db *sql.DB, id int64, accept bool) (Proposal, error) {
	tx, err := db.Begin()
	if err != nil {
		return Proposal{}, err
	}
	defer tx.Rollback() // no-op after a successful Commit

	p, err := scanProposal(tx.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE id = ?`, id))
	if err != nil {
		return Proposal{}, err
	}
	if p.Status != "pending" {
		return Proposal{}, fmt.Errorf("%w: proposal already %s", errConflict, p.Status)
	}
	status := "rejected"
	if accept {
		status = "accepted"
		r, err := scanResume(tx.QueryRow(`SELECT `+resumeCols+` FROM resumes WHERE id = ?`, p.ResumeID))
		if err != nil {
			return Proposal{}, err
		}
		if err := r.Content.apply(p); err != nil {
			return Proposal{}, err
		}
		raw, err := json.Marshal(r.Content)
		if err != nil {
			return Proposal{}, err
		}
		if _, err := tx.Exec(`UPDATE resumes SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(raw), r.ID); err != nil {
			return Proposal{}, err
		}
	}
	if _, err := tx.Exec(`UPDATE proposals SET status = ? WHERE id = ?`, status, id); err != nil {
		return Proposal{}, err
	}
	if err := tx.Commit(); err != nil {
		return Proposal{}, err
	}
	p.Status = status
	return p, nil
}

// updateResumeFromAI rejects stale snapshots and changes every section atomically.
func updateResumeFromAI(db *sql.DB, id int64, revision string, content ResumeContent) (Resume, error) {
	var before string
	if err := db.QueryRow(`SELECT content FROM resumes WHERE id = ?`, id).Scan(&before); err != nil {
		return Resume{}, err
	}
	if revision == "" || revision != fmt.Sprintf("%x", sha256.Sum256([]byte(before))) {
		return Resume{}, fmt.Errorf("%w: resume changed; read it again before editing", errConflict)
	}
	if err := validateDocument(content.Document); err != nil {
		return Resume{}, err
	}
	var previous ResumeContent
	if err := json.Unmarshal([]byte(before), &previous); err != nil {
		return Resume{}, err
	}
	if previous.Document != nil && content.Document == nil {
		return Resume{}, fmt.Errorf("%w: preserve content.document and edit the document directly", errInvalid)
	}
	if previous.Document != nil && reflect.DeepEqual(previous.Document, content.Document) {
		oldIndex, newIndex := previous, content
		oldIndex.Document = nil
		newIndex.Document = nil
		oldIndex.normalize()
		newIndex.normalize()
		if !reflect.DeepEqual(oldIndex, newIndex) {
			return Resume{}, fmt.Errorf("%w: edit content.document directly; the other fields are an index of that document", errInvalid)
		}
	}
	content.normalize()
	raw, err := json.Marshal(content)
	if err != nil {
		return Resume{}, err
	}
	result, err := db.Exec(`UPDATE resumes SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND content = ?`, string(raw), id, before)
	if err != nil {
		return Resume{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return Resume{}, fmt.Errorf("%w: resume changed; read it again before editing", errConflict)
	}
	return getResume(db, id)
}
