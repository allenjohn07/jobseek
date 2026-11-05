package main

import (
	"database/sql"
	"encoding/json" // convert Go structs <-> JSON
	"errors"        // NEW: lets us check which error we got (errors.Is)
	"net/http"
	"strconv" // NEW: converts strings to numbers ("42" -> 42)
)

// App holds things our handlers need to share, like the database.
type App struct {
	db *sql.DB // * = a pointer: we share one database handle, we don't copy it
}

// writeJSON is a small helper so every handler sends JSON the same way.
// "any" means v can be a value of any type.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json") // tell the browser this is JSON
	w.WriteHeader(status)                              // HTTP status code: 200 OK, 201 Created, etc.
	json.NewEncoder(w).Encode(v)                       // convert v to JSON and write it into the response
}

// (a *App) makes this a "method" on App, so it can use a.db.
// GET /api/jobs: return all jobs.
func (a *App) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := listJobs(a.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500 = "server broke"
		return                                                     // stop here; don't run the line below
	}
	writeJSON(w, http.StatusOK, jobs) // 200 + the list as JSON
}

// POST /api/jobs: create a job from the JSON in the request body.
func (a *App) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var j Job
	// Read the request body (JSON) and fill in j's fields, matched by the json tags.
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest) // 400 = "client sent something wrong"
		return
	}
	if j.Company == "" || j.Title == "" { // || means "or"
		http.Error(w, "company and title are required", http.StatusBadRequest)
		return
	}
	id, err := createJob(a.db, j)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Read the job back so the response has the stored values (default status, createdAt).
	created, err := getJob(a.db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created) // 201 = "created", and send back the saved job
}

// parseID reads the {id} part of the URL and turns it into a number.
// The three handlers below all need it, so it lives in one place.
func parseID(r *http.Request) (int64, error) {
	// r.PathValue("id") returns the text matched by {id} in the route, e.g. "7" for /api/jobs/7.
	// ParseInt(text, 10, 64) = parse as base-10, fitting in 64 bits. "abc" gives an error.
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// GET /api/jobs/{id}: return one job.
func (a *App) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest) // 400: e.g. /api/jobs/abc
		return
	}
	j, err := getJob(a.db, id)
	if errors.Is(err, sql.ErrNoRows) { // errors.Is checks whether err is that specific error
		http.Error(w, "job not found", http.StatusNotFound) // 404: valid id, but no such job
		return
	}
	if err != nil { // any other error means something actually broke
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
	writeJSON(w, http.StatusOK, j)
}

// PUT /api/jobs/{id}: replace a job's fields with the JSON sent.
func (a *App) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var j Job
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil { // same decoding as in create
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if j.Company == "" || j.Title == "" { // same validation as in create
		http.Error(w, "company and title are required", http.StatusBadRequest)
		return
	}
	if j.Status == "" {
		j.Status = "wishlist"
	}
	j.ID = id // trust the id in the URL, not whatever might be in the JSON body

	err = updateJob(a.db, j) // "=" not ":=" because err already exists in this function
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Read the job back from the database so the response includes everything,
	// including fields the client didn't send, like createdAt.
	updated, err := getJob(a.db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DELETE /api/jobs/{id}: remove a job.
func (a *App) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	err = deleteJob(a.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204 = "done, and there's nothing to send back"
}
