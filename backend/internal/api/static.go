package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves a single-page application from an fs.FS, falling back to
// index.html for any path that does not match a built asset (client-side
// routing). API paths are never reached here because the mux matches /api/
// routes first.
func SPAHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			// Unknown path → serve the SPA shell for client-side routing.
			serveIndex(w, dist)
			return
		}
		// Long-cache fingerprinted assets, but never the HTML shell.
		if strings.HasSuffix(p, ".html") {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.Contains(p, "/immutable/") || strings.Contains(p, "/_app/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, dist fs.FS) {
	b, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}
