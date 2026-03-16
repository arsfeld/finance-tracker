package server

import (
	"net/http"

	"finance_tracker/internal/api"
	"finance_tracker/internal/database"
)

// Server holds the HTTP server dependencies and routes.
type Server struct {
	db  *database.DB
	mux *http.ServeMux
}

// New creates a new Server and registers all routes.
func New(db *database.DB) *Server {
	s := &Server{
		db:  db,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the http.Handler with middleware applied.
func (s *Server) Handler() http.Handler {
	return Chain(s.mux,
		Recovery(),
		Logging(),
		CORS(),
	)
}

func (s *Server) routes() {
	health := api.NewHealthHandler()

	s.mux.Handle("GET /api/health", health)
}
