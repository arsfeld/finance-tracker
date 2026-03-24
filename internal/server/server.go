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
	Analysis  *api.AnalysisRunHandler
	Scheduler *scheduler.Scheduler
}

// New creates a new Server and registers all routes.
func New(db *database.DB, cfg *config.Config, sched *scheduler.Scheduler) *Server {
	events := api.NewEventHub()

	accountStore := store.NewAccountStore(db.Read, db.Write)
	txnStore := store.NewTransactionStore(db.Read, db.Write)
	catStore := store.NewCategoryStore(db.Read, db.Write)
	analysisStore := store.NewAnalysisStore(db.Read, db.Write)
	syncLogStore := store.NewSyncLogStore(db.Read, db.Write)
	budgetStore := store.NewBudgetStore(db.Read, db.Write)

	syncHandler := api.NewSyncHandler(cfg, accountStore, txnStore, catStore, syncLogStore, sched, events)
	analysisRunHandler := api.NewAnalysisRunHandler(cfg, txnStore, accountStore, catStore, analysisStore, budgetStore, sched, events)

	s := &Server{
		db:        db,
		mux:       http.NewServeMux(),
		Events:    events,
		Sync:      syncHandler,
		Analysis:  analysisRunHandler,
		Scheduler: sched,
	}

	// Health
	s.mux.Handle("GET /api/health", api.NewHealthHandler())

	// Events (SSE)
	s.mux.Handle("GET /api/events", events)

	// Dashboard
	dashboard := api.NewDashboardHandler(cfg, txnStore, analysisStore)
	s.mux.HandleFunc("GET /api/dashboard", dashboard.Get)

	// Billing periods
	billingPeriods := api.NewBillingPeriodsHandler(cfg)
	s.mux.HandleFunc("GET /api/billing-periods", billingPeriods.List)

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
	catHandler := api.NewCategoryHandler(catStore, txnStore, events)
	s.mux.HandleFunc("GET /api/categories", catHandler.List)
	s.mux.HandleFunc("GET /api/categories/unique", catHandler.ListUnique)
	s.mux.HandleFunc("POST /api/categories/exclude", catHandler.SetExcluded)
	s.mux.HandleFunc("GET /api/categories/summary", catHandler.Summary)

	// Analytics
	analyticsHandler := api.NewAnalyticsHandler(cfg, txnStore)
	s.mux.HandleFunc("GET /api/analytics/category-trend", analyticsHandler.CategoryTrend)
	s.mux.HandleFunc("GET /api/analytics/daily-totals", analyticsHandler.DailyTotals)
	s.mux.HandleFunc("GET /api/analytics/top-merchants", analyticsHandler.TopMerchants)

	// Analyses
	analysisHandler := api.NewAnalysisHandler(analysisStore)
	s.mux.HandleFunc("GET /api/analyses", analysisHandler.List)
	s.mux.HandleFunc("GET /api/analyses/latest", analysisHandler.GetLatest)
	s.mux.HandleFunc("GET /api/analyses/{id}", analysisHandler.GetByID)
	s.mux.HandleFunc("POST /api/analyses/run", analysisRunHandler.TriggerAnalysis)

	// Sync
	s.mux.HandleFunc("POST /api/sync", syncHandler.TriggerSync)
	s.mux.HandleFunc("GET /api/sync/status", syncHandler.GetStatus)
	s.mux.HandleFunc("GET /api/sync/log", syncHandler.GetLog)

	// Settings (read-only from .env, test notification)
	settingsHandler := api.NewSettingsHandler(cfg)
	s.mux.HandleFunc("GET /api/settings", settingsHandler.Get)
	s.mux.HandleFunc("POST /api/settings/test-notification", settingsHandler.TestNotification)

	// Budgets
	budgetHandler := api.NewBudgetHandler(budgetStore, txnStore, cfg)
	s.mux.HandleFunc("POST /api/budgets", budgetHandler.Upsert)
	s.mux.HandleFunc("DELETE /api/budgets/{category}", budgetHandler.Delete)
	s.mux.HandleFunc("GET /api/budgets/status", budgetHandler.Status)

	// Chat
	chatHandler := api.NewChatHandler(cfg, txnStore, accountStore, catStore, analysisStore, budgetStore, events)
	s.mux.HandleFunc("POST /api/chat", chatHandler.Handle)

	// SPA fallback: must be last (least specific pattern).
	// In dev mode (VITE_DEV_URL set), proxies to Vite for hot-reload.
	// In production, serves static files from FrontendDir.
	s.mux.Handle("/", spaHandler(cfg.FrontendDir, cfg.ViteDevURL))

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
