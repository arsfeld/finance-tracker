package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Common response types used across API handlers

// respondWithJSON writes a JSON response with the given status code
func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// respondWithError writes an error response with logging
func respondWithError(w http.ResponseWriter, r *http.Request, code int, message string, err error) {
	if err != nil {
		log.Error().
			Err(err).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Int("status", code).
			Msg(message)
	}
	
	response := map[string]interface{}{
		"error": message,
	}
	respondWithJSON(w, r, code, response)
}

// logAndError logs an error and sends an HTTP error response
func logAndError(w http.ResponseWriter, r *http.Request, message string, code int) {
	log.Error().
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Int("status", code).
		Msg(message)
	http.Error(w, message, code)
}

// parseJSON parses JSON request body into the provided interface
func parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}