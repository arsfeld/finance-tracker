package server

import (
	"net/http"

	"finance_tracker/internal/api"
	"finance_tracker/internal/config"
	"finance_tracker/internal/database"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/store"
)

// Server holds the HTTP server dependencies and routes.
type Server struct {
	db        *database.DB
	mux       *http.ServeMux
	Events    *api.EventHub
	Sync      *api.SyncHandler
	Scheduler *scheduler.Scheduler
}

// New creates a new Server and registers all routes.
func New(db *database.DB, cfg *config.Config, sched *scheduler.Scheduler) *Server {
	events := api.NewEventHub()

	accountStore := store.NewAccountStore(db.Read, db.Write)
	txnStore := store.NewTransactionStore(db.Read, db.Write)
	catStore := store.NewCategoryStore(db.Read, db.Write)
	analysisStore := store.NewAnalysisStore(db.Read, db.Write)
	settingsStore := store.NewSettingsStore(db.Read, db.Write)
	filterStore := store.NewFilterStore(db.Read, db.Write)
	syncLogStore := store.NewSyncLogStore(db.Read, db.Write)

	syncHandler := api.NewSyncHandler(cfg, accountStore, txnStore, syncLogStore, sched, events)

	s := &Server{
		db:        db,
		mux:       http.NewServeMux(),
		Events:    events,
		Sync:      syncHandler,
		Scheduler: sched,
	}

	// Health
	s.mux.Handle("GET /api/health", api.NewHealthHandler())

	// Events (SSE)
	s.mux.Handle("GET /api/events", events)

	// Dashboard
	dashboard := api.NewDashboardHandler(txnStore, analysisStore, settingsStore)
	s.mux.HandleFunc("GET /api/dashboard", dashboard.Get)

	// Transactions
	txnHandler := api.NewTransactionHandler(txnStore, catStore, events)
	s.mux.HandleFunc("GET /api/transactions", txnHandler.List)
	s.mux.HandleFunc("GET /api/transactions/export", txnHandler.ExportCSV)
	s.mux.HandleFunc("GET /api/transactions/{id}", txnHandler.GetByID)
	s.mux.HandleFunc("PATCH /api/transactions/{id}/category", txnHandler.OverrideCategory)

	// Accounts
	acctHandler := api.NewAccountHandler(accountStore)
	s.mux.HandleFunc("GET /api/accounts", acctHandler.List)
	s.mux.HandleFunc("PATCH /api/accounts/{id}", acctHandler.UpdateInclusion)

	// Categories
	catHandler := api.NewCategoryHandler(catStore, txnStore)
	s.mux.HandleFunc("GET /api/categories", catHandler.List)
	s.mux.HandleFunc("GET /api/categories/summary", catHandler.Summary)

	// Analyses
	analysisHandler := api.NewAnalysisHandler(analysisStore)
	s.mux.HandleFunc("GET /api/analyses", analysisHandler.List)
	s.mux.HandleFunc("GET /api/analyses/latest", analysisHandler.GetLatest)
	s.mux.HandleFunc("GET /api/analyses/{id}", analysisHandler.GetByID)

	// Sync
	s.mux.HandleFunc("POST /api/sync", syncHandler.TriggerSync)
	s.mux.HandleFunc("GET /api/sync/status", syncHandler.GetStatus)
	s.mux.HandleFunc("GET /api/sync/log", syncHandler.GetLog)

	// Settings
	settingsHandler := api.NewSettingsHandler(settingsStore, cfg)
	s.mux.HandleFunc("GET /api/settings", settingsHandler.Get)
	s.mux.HandleFunc("PATCH /api/settings", settingsHandler.Update)
	s.mux.HandleFunc("POST /api/settings/test-notification", settingsHandler.TestNotification)

	// Filters
	filterHandler := api.NewFilterHandler(filterStore)
	s.mux.HandleFunc("GET /api/filters", filterHandler.List)
	s.mux.HandleFunc("POST /api/filters", filterHandler.Create)
	s.mux.HandleFunc("PATCH /api/filters/{id}", filterHandler.Update)
	s.mux.HandleFunc("DELETE /api/filters/{id}", filterHandler.Delete)

	// SPA fallback: must be last (least specific pattern).
	s.mux.Handle("GET /", spaHandler("web/dist"))

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
