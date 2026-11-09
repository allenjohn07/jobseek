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

	app := &App{db: db, baseURL: "http://" + listenAddr} // & = make a pointer to a new App holding our db

	// Bind to loopback only: the app holds personal data and has no login.
	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(listenAddr, app.routes()))
}

// routes builds the router. It lives in its own function so tests can use the real routing.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	// Go 1.22+ lets you put the HTTP method in the pattern.
	// The same URL can go to different handlers depending on GET vs POST.
	mux.HandleFunc("GET /api/jobs", a.handleListJobs)
	mux.HandleFunc("POST /api/jobs", a.handleCreateJob)
	mux.HandleFunc("GET /api/jobs/{id}", a.handleGetJob)    // {id} is a wildcard: matches /api/jobs/1, /api/jobs/42, ...
	mux.HandleFunc("PUT /api/jobs/{id}", a.handleUpdateJob) // same URL, different method = different handler
	mux.HandleFunc("DELETE /api/jobs/{id}", a.handleDeleteJob)

	mux.HandleFunc("POST /api/resume/import", a.handleImportResume)
	mux.HandleFunc("GET /api/resume/base", a.handleGetBaseResume)
	mux.HandleFunc("GET /api/jobs/{id}/resume", a.handleGetJobResume)
	mux.HandleFunc("POST /api/jobs/{id}/resume", a.handleCreateJobResume)
	mux.HandleFunc("PUT /api/resumes/{id}", a.handleUpdateResume)
	mux.HandleFunc("GET /api/resumes/{id}/proposals", a.handleListProposals)
	mux.HandleFunc("POST /api/proposals/{id}/accept", a.handleResolveProposal(true))
	mux.HandleFunc("POST /api/proposals/{id}/reject", a.handleResolveProposal(false))
	mux.HandleFunc("GET /api/mcp/info", a.handleMCPInfo)
	mux.HandleFunc("POST /api/jobtweak/chat", a.handleChat)
	mux.Handle("/mcp", a.mcpHandler()) // MCP clients (Claude Code, Claude Desktop) connect here
	return localOnly(mux)
}
