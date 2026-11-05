package main

import (
	"log"
	"net/http"
)

func main() {
	db, err := openDB("jobs.db") // creates jobs.db next to the program if it doesn't exist
	if err != nil {
		log.Fatal(err) // can't run without a database, so print the error and exit
	}
	defer db.Close() // close the database when main exits

	app := &App{db: db} // & = make a pointer to a new App holding our db

	mux := http.NewServeMux()
	// Go 1.22+ lets you put the HTTP method in the pattern.
	// The same URL can go to different handlers depending on GET vs POST.
	mux.HandleFunc("GET /api/jobs", app.handleListJobs)
	mux.HandleFunc("POST /api/jobs", app.handleCreateJob)
	mux.HandleFunc("GET /api/jobs/{id}", app.handleGetJob)    // {id} is a wildcard: matches /api/jobs/1, /api/jobs/42, ...
	mux.HandleFunc("PUT /api/jobs/{id}", app.handleUpdateJob) // same URL, different method = different handler
	mux.HandleFunc("DELETE /api/jobs/{id}", app.handleDeleteJob)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
