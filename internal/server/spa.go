package server

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// spaHandler serves static files from dir, falling back to index.html for SPA routes.
func spaHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't serve API routes from the SPA handler.
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly.
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath != "" {
			if _, err := os.Stat(dir + "/" + cleanPath); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to index.html for SPA client-side routing.
		indexPath := dir + "/index.html"
		if _, err := os.Stat(indexPath); err != nil {
			// No index.html — frontend not built yet.
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<!DOCTYPE html><html><body>
				<h1>Finance Tracker</h1>
				<p>Frontend not built yet. Run <code>cd web && npm run build</code> first, or use the Vite dev server at <a href="http://localhost:5173">:5173</a>.</p>
				<p><a href="/api/health">API Health Check</a></p>
			</body></html>`))
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}

// Suppress unused import.
var _ fs.FS
