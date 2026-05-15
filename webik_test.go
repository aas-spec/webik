package webik

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestServer creates a Server pointing at a temporary directory populated
// with a small set of static files.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()

	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.html", "<html>index</html>")
	write("style.css", "body{}")
	write("app.js", "var x=1;")
	write("sub/page.html", "<html>sub</html>")

	return &Server{
		workDir:  dir,
		sitePath: ".",
	}
}

func TestServeIndex(t *testing.T) {
	s := newTestServer(t)

	paths := []string{"/", "", "/noop", "/some/route"}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, "/placeholder", nil)
		req.URL.Path = p
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("path %q: status = %d, want %d", p, rr.Code, http.StatusOK)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("path %q: Content-Type = %q, want text/html", p, ct)
		}
		if got := rr.Body.String(); got != "<html>index</html>" {
			t.Errorf("path %q: body = %q, want index.html content", p, got)
		}
	}
}

func TestServeStaticFiles(t *testing.T) {
	s := newTestServer(t)

	tests := []struct {
		path     string
		wantCT   string
		wantBody string
	}{
		{"/style.css", "text/css", "body{}"},
		{"/app.js", "javascript", "var x=1;"},
		{"/sub/page.html", "text/html", "<html>sub</html>"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", tt.path, rr.Code, http.StatusOK)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, tt.wantCT) {
			t.Errorf("%s: Content-Type = %q, want %q", tt.path, ct, tt.wantCT)
		}
		if got := rr.Body.String(); got != tt.wantBody {
			t.Errorf("%s: body = %q, want %q", tt.path, got, tt.wantBody)
		}
	}
}

func TestServeNotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/missing.html", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/index.html", nil)
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /index.html: status = %d, want %d", method, rr.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestPathTraversal(t *testing.T) {
	s := newTestServer(t)

	// Directly set a crafted path to bypass URL parsing normalization.
	req := httptest.NewRequest(http.MethodGet, "/placeholder.css", nil)
	req.URL.Path = "/sub/../../../etc/passwd.css"

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("path traversal: status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestProxyRouting(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend:" + r.URL.Path))
	}))
	defer backend.Close()

	s := newTestServer(t)
	s.sourceAPIRoute = "/api"
	s.targetAPIRoute = backend.URL

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); body != "backend:/users" {
		t.Errorf("body = %q, want %q", body, "backend:/users")
	}
}

func TestProxyNotTriggeredWithoutSourceRoute(t *testing.T) {
	s := newTestServer(t)
	// sourceAPIRoute is empty — all requests should be served as files.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNonAPIPathNotProxied(t *testing.T) {
	s := newTestServer(t)
	s.sourceAPIRoute = "/api"
	s.targetAPIRoute = "http://127.0.0.1:19999" // intentionally unreachable

	// A non-API path must be served from the file system, not proxied.
	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNew(t *testing.T) {
	s, err := New(":8080", "dist", "http://backend", "/api")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.port != ":8080" {
		t.Errorf("port = %q, want %q", s.port, ":8080")
	}
	if s.sitePath != "dist" {
		t.Errorf("sitePath = %q, want %q", s.sitePath, "dist")
	}
	if s.targetAPIRoute != "http://backend" {
		t.Errorf("targetAPIRoute = %q, want %q", s.targetAPIRoute, "http://backend")
	}
	if s.sourceAPIRoute != "/api" {
		t.Errorf("sourceAPIRoute = %q, want %q", s.sourceAPIRoute, "/api")
	}
	if s.workDir == "" {
		t.Error("workDir must not be empty")
	}
}
