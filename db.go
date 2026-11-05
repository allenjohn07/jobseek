package main

import (
	"database/sql" //standard interface for the sql

	_ "modernc.org/sqlite" //the way to talk to that sql. It registers a driver named "sqlite" with database/sql. We never call it directly. _ means import only for its initialisation.
)

type Job struct {
	ID        int64  `json:"id"`      // the text in backticks is a "struct tag":
	Company   string `json:"company"` // it tells the JSON encoder what key name to use,
	Title     string `json:"title"`   // so Go's "Company" becomes "company" in JSON
	URL       string `json:"url"`
	Status    string `json:"status"`    // wishlist, applied, interviewing, offer, rejected
	AppliedOn string `json:"appliedOn"` // date as text, e.g. "2026-11-04"
	Notes     string `json:"notes"`
	CreatedAt string `json:"createdAt"`
}

// The SQL that creates our table. Backticks make a multi-line "raw" string.
// IF NOT EXISTS = safe to run every time the app starts.
// NOT NULL DEFAULT ” = every column always has a value, so Go never gets a NULL
// (scanning NULL into a Go string would be an error).
const schema = `
CREATE TABLE IF NOT EXISTS jobs (
  id         INTEGER PRIMARY KEY,
  company    TEXT NOT NULL,
  title      TEXT NOT NULL,
  url        TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'wishlist',
  applied_on TEXT NOT NULL DEFAULT '',
  notes      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// openDB opens (or creates) the database file and makes sure the table exists.
// It returns two things: the database handle and an error. Returning an error is Go's main way of reporting failure.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path) // "sqlite" = the driver name registered by the import above
	if err != nil {                     // the Go pattern: check the error right after every call that can fail
		return nil, err // nil = "nothing"; pass the error up to whoever called us
	}
	if _, err := db.Exec(schema); err != nil { // Exec runs SQL that doesn't return rows
		return nil, err // "_" throws away the first return value (we don't need it)
	}
	return db, nil // success: return the db and "no error"
}

// listJobs returns every job, newest first.
func listJobs(db *sql.DB) ([]Job, error) { // []Job = a "slice" (a growable list) of Jobs
	rows, err := db.Query(`SELECT id, company, title, url, status, applied_on, notes, created_at
	                       FROM jobs ORDER BY created_at DESC`) // Query runs SQL that returns rows
	if err != nil {
		return nil, err
	}
	defer rows.Close() // "defer" = run this when the function exits, however it exits.
	// It guarantees we release the database rows even if we return early.

	jobs := []Job{}   // start with an empty list (not nil), so the JSON output is [] instead of null
	for rows.Next() { // move to the next row; returns false when there are no more
		var j Job // a fresh, empty Job
		// Scan copies each column into the field we point at. & means "the address of",
		// so Scan can fill in our variable. The order must match the SELECT columns.
		if err := rows.Scan(&j.ID, &j.Company, &j.Title, &j.URL, &j.Status, &j.AppliedOn, &j.Notes, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j) // add it to the list (append returns the updated slice)
	}
	return jobs, rows.Err() // rows.Err() reports any error that happened while looping
}

// createJob inserts a job and returns its new ID.
func createJob(db *sql.DB, j Job) (int64, error) {
	if j.Status == "" { // default status if none was given
		j.Status = "wishlist"
	}
	// The ? marks are placeholders. The driver safely fills them in from the arguments that follow.
	// Never build SQL by gluing strings together with user input; that's how SQL injection happens.
	res, err := db.Exec(`INSERT INTO jobs (company, title, url, status, applied_on, notes)
	                     VALUES (?, ?, ?, ?, ?, ?)`,
		j.Company, j.Title, j.URL, j.Status, j.AppliedOn, j.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId() // the id SQLite just assigned to the new row
}
