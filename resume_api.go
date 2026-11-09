package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

const maxResumeBytes = 1 << 20 // 1 MB is far more than any resume

// fail maps a store error to an HTTP status.
func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, errConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, errInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GET /api/resume/base: the base resume (created empty on first use).
func (a *App) handleGetBaseResume(w http.ResponseWriter, r *http.Request) {
	res, err := getBaseResume(a.db)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// GET /api/jobs/{id}/resume: the resume tailored for this job (404 if not created yet).
func (a *App) handleGetJobResume(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := getJobResume(a.db, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// POST /api/jobs/{id}/resume: copy the base resume for this job.
func (a *App) handleCreateJobResume(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := createJobResume(a.db, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// PUT /api/resumes/{id}: replace a resume's content.
func (a *App) handleUpdateResume(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var c ResumeContent
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxResumeBytes)).Decode(&c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	res, err := saveResumeContent(a.db, id, c)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// GET /api/resumes/{id}/proposals
func (a *App) handleListProposals(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	ps, err := listProposals(a.db, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

// handleResolveProposal builds the accept or reject handler. Accept is REST-only
// (used by the UI); no MCP tool exposes it, so the AI can't approve its own edits.
func (a *App) handleResolveProposal(accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		p, err := resolveProposal(a.db, id, accept)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
