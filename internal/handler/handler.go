// Package handler wires HTTP routes to the task store.
//
// Routing uses the method-and-path patterns of Go 1.22+ net/http
// (e.g. "GET /tasks/{id}"), so no third-party router is needed.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"httpserver/internal/store"
)

// maxBodyBytes limits request bodies to stop clients sending huge payloads.
const maxBodyBytes = 1 << 20 // 1 MiB

// Handler holds the dependencies the HTTP endpoints need.
type Handler struct {
	store *store.Store
}

// New creates a Handler backed by the given store.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Routes registers every endpoint on a new ServeMux and returns it.
// staticDir is served at "/" (pass "" to disable static files).
func (h *Handler) Routes(staticDir string) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)

	mux.HandleFunc("GET /tasks", h.listTasks)
	mux.HandleFunc("POST /tasks", h.createTask)
	mux.HandleFunc("GET /tasks/{id}", h.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", h.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", h.deleteTask)

	if staticDir != "" {
		mux.Handle("GET /", http.FileServer(http.Dir(staticDir)))
	}
	return mux
}

// health reports that the server is up. Useful for load balancers and Docker.
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// listTasks returns every task.
func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

// getTask returns a single task by ID.
func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.store.Get(id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// createRequest is the JSON body expected by POST /tasks.
type createRequest struct {
	Title string `json:"title"`
}

// createTask adds a new task. Body: {"title": "..."}.
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	writeJSON(w, http.StatusCreated, h.store.Create(req.Title))
}

// updateRequest is the JSON body expected by PATCH /tasks/{id}.
// Pointer fields let us tell "not sent" apart from "sent as empty/false".
type updateRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

// updateTask partially updates a task. Body: {"title": "...", "done": true}.
func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		req.Title = &trimmed
	}
	t, err := h.store.Update(id, req.Title, req.Done)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// deleteTask removes a task and returns 204 No Content.
func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

// parseID reads the {id} path value. On failure it writes a 400 and returns false.
func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// decodeJSON parses the request body into dst, rejecting unknown fields and
// oversized bodies. On failure it writes a 400 and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// writeStoreError maps store errors to HTTP status codes.
func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

// writeError sends {"error": msg} with the given status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
