package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"httpserver/internal/store"
)

// do sends a request through the router and returns the recorded response.
func do(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestTaskLifecycle(t *testing.T) {
	mux := New(store.New()).Routes("")

	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantBody                 string // substring expected in the response
	}{
		{"health", "GET", "/health", "", 200, `"ok"`},
		{"empty list", "GET", "/tasks", "", 200, "[]"},
		{"create", "POST", "/tasks", `{"title":"learn Go"}`, 201, `"title":"learn Go"`},
		{"create missing title", "POST", "/tasks", `{"title":"  "}`, 400, "title is required"},
		{"create bad json", "POST", "/tasks", `{bad`, 400, "invalid JSON"},
		{"create unknown field", "POST", "/tasks", `{"nope":1}`, 400, "unknown field"},
		{"get", "GET", "/tasks/1", "", 200, `"id":1`},
		{"get bad id", "GET", "/tasks/abc", "", 400, "positive integer"},
		{"get missing", "GET", "/tasks/99", "", 404, "not found"},
		{"mark done", "PATCH", "/tasks/1", `{"done":true}`, 200, `"done":true`},
		{"rename keeps done", "PATCH", "/tasks/1", `{"title":"master Go"}`, 200, `"done":true`},
		{"delete", "DELETE", "/tasks/1", "", 204, ""},
		{"delete again", "DELETE", "/tasks/1", "", 404, "not found"},
		{"wrong method", "PUT", "/tasks", "", 405, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, tc.method, tc.path, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body, tc.wantBody)
			}
		})
	}
}
