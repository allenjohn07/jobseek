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
	// Description is the job posting text, used to tailor the resume.
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
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
	if _, err := db.Exec(resumeSchema + settingsSchema); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil // success: return the db and "no error"
}

// hasColumn reports whether table has a column with this name.
func hasColumn(db *sql.DB, table, column string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n)
	return n > 0, err
}

// migrate upgrades databases created by older versions. CREATE TABLE IF NOT EXISTS
// never alters an existing table, so new columns are added here.
func migrate(db *sql.DB) error {
	ok, err := hasColumn(db, "jobs", "description")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := db.Exec(`ALTER TABLE jobs ADD COLUMN description TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}

// listJobs returns every job, newest first.
func listJobs(db *sql.DB) ([]Job, error) { // []Job = a "slice" (a growable list) of Jobs
	rows, err := db.Query(`SELECT id, company, title, url, status, applied_on, notes, description, created_at
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
		if err := rows.Scan(&j.ID, &j.Company, &j.Title, &j.URL, &j.Status, &j.AppliedOn, &j.Notes, &j.Description, &j.CreatedAt); err != nil {
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
	res, err := db.Exec(`INSERT INTO jobs (company, title, url, status, applied_on, notes, description)
	                     VALUES (?, ?, ?, ?, ?, ?, ?)`,
		j.Company, j.Title, j.URL, j.Status, j.AppliedOn, j.Notes, j.Description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId() // the id SQLite just assigned to the new row
}

// getJob returns the single job with this id.
func getJob(db *sql.DB, id int64) (Job, error) {
	var j Job // an empty Job to fill in
	// QueryRow is like Query, but for when you expect at most ONE row.
	// You chain .Scan() directly onto it (the "." at the end of the line continues onto the next).
	err := db.QueryRow(`SELECT id, company, title, url, status, applied_on, notes, description, created_at
	                    FROM jobs WHERE id = ?`, id).
		Scan(&j.ID, &j.Company, &j.Title, &j.URL, &j.Status, &j.AppliedOn, &j.Notes, &j.Description, &j.CreatedAt)
	// If no row matched, Scan returns the special error sql.ErrNoRows.
	// We just pass it up, and the handler will turn it into a 404.
	return j, err
}

// updateJob overwrites every field of an existing job.
func updateJob(db *sql.DB, j Job) error { // returns only an error: nothing else to give back
	res, err := db.Exec(`UPDATE jobs
	                     SET company = ?, title = ?, url = ?, status = ?, applied_on = ?, notes = ?, description = ?
	                     WHERE id = ?`, // WHERE limits the change to one job. Without it, EVERY row would change!
		j.Company, j.Title, j.URL, j.Status, j.AppliedOn, j.Notes, j.Description, j.ID) // the order matches the ?s exactly
	if err != nil {
		return err
	}
	n, err := res.RowsAffected() // how many rows the UPDATE actually changed
	if err != nil {
		return err
	}
	if n == 0 { // 0 rows changed = no job had that id
		return sql.ErrNoRows // reuse the same "not found" error so the handler treats it the same way
	}
	return nil // nil error = success
}

// deleteJob removes a job by id, along with its tailored resume and that resume's proposals.
func deleteJob(db *sql.DB, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	// Resume cleanup is done by hand: SQLite foreign keys are a per-connection setting,
	// which is awkward with database/sql's connection pool.
	if _, err := tx.Exec(`DELETE FROM proposals WHERE resume_id IN (SELECT id FROM resumes WHERE job_id = ?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM resumes WHERE job_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected() // same trick as update: did we actually delete anything?
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows // nothing was deleted, so that id didn't exist
	}
	return tx.Commit()
}
