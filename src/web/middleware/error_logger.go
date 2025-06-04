package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	// Capture error responses for logging
	if rw.status >= 400 {
		rw.body.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

// ErrorLogger middleware logs all HTTP errors
func ErrorLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the response writer to capture status and body
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:        0,
			body:          &bytes.Buffer{},
		}
		
		// Defer panic recovery and logging
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("panic", err).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Msg("Panic recovered in HTTP handler")
				
				// Send error response if not already sent
				if wrapped.status == 0 {
					http.Error(wrapped, "Internal server error", http.StatusInternalServerError)
				}
			}
		}()
		
		// Call the next handler
		next.ServeHTTP(wrapped, r)
		
		// Log errors (4xx and 5xx status codes)
		if wrapped.status >= 400 {
			duration := time.Since(start)
			
			logEvent := log.Error().
				Int("status", wrapped.status).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Dur("duration", duration).
				Str("user_agent", r.UserAgent())
			
			// Add user context if available
			if userID := r.Context().Value("user_id"); userID != nil {
				logEvent = logEvent.Str("user_id", userID.(string))
			}
			
			if orgID := r.Context().Value("organization_id"); orgID != nil {
				logEvent = logEvent.Str("organization_id", orgID.(string))
			}
			
			// Include error body for debugging (limit size)
			if wrapped.body.Len() > 0 && wrapped.body.Len() < 1000 {
				logEvent = logEvent.Str("error_body", wrapped.body.String())
			}
			
			logEvent.Msg("HTTP error response")
		}
	})
}

// RequestLogger logs all requests (not just errors)
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request body for debugging POST/PUT requests (be careful with sensitive data)
		var bodyBytes []byte
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:        0,
			body:          &bytes.Buffer{},
		}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		// Choose log level based on status code and log the request
		if wrapped.status >= 500 {
			log.Error().
				Int("status", wrapped.status).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Dur("duration", duration).
				Str("user_agent", r.UserAgent()).
				Msg("HTTP request completed")
		} else if wrapped.status >= 400 {
			log.Warn().
				Int("status", wrapped.status).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Dur("duration", duration).
				Str("user_agent", r.UserAgent()).
				Msg("HTTP request completed")
		} else {
			log.Info().
				Int("status", wrapped.status).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Dur("duration", duration).
				Str("user_agent", r.UserAgent()).
				Msg("HTTP request completed")
		}
	})
}