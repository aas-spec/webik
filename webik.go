// Package webik provides a micro web server for serving frontend applications
// (Angular, Vue, React, etc.) with optional API reverse proxy support.
package webik

import (
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	types := map[string]string{
		".css":  "text/css; charset=utf-8",
		".js":   "text/javascript",
		".html": "text/html; charset=utf-8",
	}
	for ext, ct := range types {
		if err := mime.AddExtensionType(ext, ct); err != nil {
			log.Printf("webik: failed to register MIME type for %s: %v", ext, err)
		}
	}
}

// Server serves static files from a directory and proxies API requests
// to a backend service.
type Server struct {
	port           string
	sitePath       string
	sourceAPIRoute string
	targetAPIRoute string
	workDir        string
}

// New returns a Server configured with the given parameters.
// port is the address to listen on (e.g. ":8080").
// sitePath is the directory containing static files, relative to the working directory.
// targetAPIRoute is the backend URL to proxy API requests to.
// sourceAPIRoute is the URL prefix that identifies API requests (e.g. "/api").
func New(port, sitePath, targetAPIRoute, sourceAPIRoute string) (*Server, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Server{
		port:           port,
		sitePath:       sitePath,
		sourceAPIRoute: sourceAPIRoute,
		targetAPIRoute: targetAPIRoute,
		workDir:        workDir,
	}, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.sourceAPIRoute != "" && strings.HasPrefix(r.URL.Path, s.sourceAPIRoute) {
		s.proxy(w, r)
		return
	}
	s.serveFile(w, r)
}

// ListenAndServe starts the HTTP server on the configured port.
// It returns a non-nil error when the server stops.
func (s *Server) ListenAndServe() error {
	log.Printf("webik: listening on %s", s.port)
	return http.ListenAndServe(s.port, s)
}

// ListenAndServe is a convenience wrapper that creates a Server and starts it.
// It calls log.Fatal if the server cannot start or stops unexpectedly.
func ListenAndServe(port, sitePath, targetAPIRoute, sourceAPIRoute string) {
	s, err := New(port, sitePath, targetAPIRoute, sourceAPIRoute)
	if err != nil {
		log.Fatalf("webik: %v", err)
	}
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	ext := filepath.Ext(path)

	// Serve index.html for the root and any path without a file extension
	// so that SPA client-side routing works correctly.
	if path == "" || path == "/" || ext == "" {
		path = "/index.html"
		ext = ".html"
	}

	filePath := filepath.Join(s.workDir, s.sitePath, filepath.FromSlash(path))
	baseDir := filepath.Clean(filepath.Join(s.workDir, s.sitePath))

	if filePath != baseDir && !strings.HasPrefix(filePath, baseDir+string(os.PathSeparator)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(ext))
	if _, err := w.Write(content); err != nil {
		log.Printf("webik: write error: %v", err)
	}
}

func (s *Server) proxy(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(s.targetAPIRoute)
	if err != nil {
		log.Printf("webik: invalid target URL %q: %v", s.targetAPIRoute, err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	orig := rp.Director

	// Strip the source route prefix before the default Director joins
	// target.Path with the incoming request path.
	rp.Director = func(req *http.Request) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, s.sourceAPIRoute)
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, s.sourceAPIRoute)
		}
		orig(req)
	}

	rp.ServeHTTP(w, r)
}
