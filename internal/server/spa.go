package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// spaHandler returns a handler that either reverse-proxies to Vite (dev) or serves static files (prod).
func spaHandler(frontendDir, viteDevURL string) http.Handler {
	if viteDevURL != "" {
		return viteProxy(viteDevURL)
	}
	return staticSPA(frontendDir)
}

// viteProxy reverse-proxies all non-API requests to the Vite dev server.
func viteProxy(rawURL string) http.Handler {
	target, err := url.Parse(rawURL)
	if err != nil {
		log.Fatal().Err(err).Str("url", rawURL).Msg("Invalid VITE_DEV_URL")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	log.Info().Str("target", rawURL).Msg("Dev mode: proxying frontend to Vite")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = target.Host
		proxy.ServeHTTP(w, r)
	})
}

// staticSPA serves built files from dir, falling back to index.html for SPA routes.
func staticSPA(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly.
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if cleanPath != "" {
			if _, err := os.Stat(dir + "/" + cleanPath); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to index.html for SPA client-side routing.
		indexPath := dir + "/index.html"
		if _, err := os.Stat(indexPath); err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<!DOCTYPE html><html><body>
				<h1>Finance Tracker</h1>
				<p>Frontend not built. Run <code>cd web && npm run build</code></p>
				<p><a href="/api/health">API Health</a></p>
			</body></html>`))
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}
