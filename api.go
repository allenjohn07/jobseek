package main

import (
	"database/sql"
	"encoding/json" // convert Go structs <-> JSON
	"net/http"
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
		return // stop here; don't run the line below
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
	j.ID = id                              // fill in the id the database assigned
	writeJSON(w, http.StatusCreated, j)    // 201 = "created", and send back the saved job
}