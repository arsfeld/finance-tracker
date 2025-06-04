package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/web/handlers"
	webmiddleware "finance_tracker/src/web/middleware"
)

// Server represents the web server
type Server struct {
	httpServer *http.Server
	config     ServerConfig
	inertia    *config.InertiaConfig
}

// ServerConfig holds configuration for the web server
type ServerConfig struct {
	Port          string
	Environment   string
	Client        *config.Client
	IsDevelopment bool
}

// NewServer creates a new web server
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	// Determine Vite host (use same host as the server for development)
	viteHost := "localhost"
	if s.config.IsDevelopment {
		// In development, we'll dynamically determine this per request
		// For now, default to localhost but this will be overridden
		viteHost = "localhost" // Set to your development domain
	}
	
	// Initialize Inertia
	inertia, err := config.NewInertiaConfig(s.config.IsDevelopment, viteHost)
	if err != nil {
		return err
	}
	s.inertia = inertia

	// Set up router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(webmiddleware.ErrorLogger)  // Our custom error logger
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.inertia.Middleware())

	// Initialize auth middleware
	var authMiddleware *auth.Middleware
	if s.config.Client != nil && s.config.Client.Anon != nil {
		authMiddleware = auth.NewMiddleware(s.config.Client.Anon, s.config.Client.Service)
		log.Debug().Msg("Auth middleware initialized successfully")
	} else {
		log.Error().Msg("Failed to initialize auth middleware - Supabase client not available")
	}

	// Initialize handlers
	pageHandlers := handlers.NewInertiaPageHandlers(s.inertia)
	apiHandlers := handlers.NewAPIHandlers(s.config.Client)
	authHandlers := handlers.NewInertiaAuthHandlers(s.config.Client, s.inertia)
	orgHandlers := handlers.NewInertiaOrganizationHandlers(s.config.Client, s.inertia)
	
	// Initialize River job client and handlers
	// For the web server, we'll use a minimal job client setup
	// The full River client is initialized in the worker command
	var jobHandlers *handlers.RiverJobHandler
	// TODO: Initialize proper River job client when database is available

	// Setup routes
	s.setupRoutes(r, pageHandlers, apiHandlers, authHandlers, orgHandlers, jobHandlers, authMiddleware)

	// Server configuration
	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("port", s.config.Port).Str("environment", s.config.Environment).Msg("Server listening")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
	return nil
}

// setupRoutes configures all the application routes
func (s *Server) setupRoutes(
	r chi.Router,
	pageHandlers *handlers.PageHandlers,
	apiHandlers *handlers.APIHandlers,
	authHandlers *handlers.AuthHandlers,
	orgHandlers *handlers.OrganizationHandlers,
	jobHandlers *handlers.RiverJobHandler,
	authMiddleware *auth.Middleware,
) {
	// Public routes
	r.Get("/health", s.healthHandler())
	r.Get("/", pageHandlers.HomePage)
	r.Get("/login", pageHandlers.LoginPage)
	r.Get("/register", pageHandlers.RegisterPage)

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandlers.HandleRegister)
		r.Post("/login", authHandlers.HandleLogin)
		r.Post("/logout", authHandlers.HandleLogout)
	})

	// API routes group (protected)
	r.Route("/api/v1", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}

		// Organization endpoints
		r.Route("/organizations", func(r chi.Router) {
			r.Get("/", orgHandlers.HandleListOrganizations)
			r.Post("/", orgHandlers.HandleCreateOrganization)
			r.Get("/{orgID}", orgHandlers.HandleGetOrganization)
			r.Post("/{orgID}/switch", orgHandlers.HandleSwitchOrganization)

			// Member management
			r.Get("/{orgID}/members", orgHandlers.HandleGetMembers)
			r.Post("/{orgID}/members", orgHandlers.HandleInviteMember)
			r.Put("/{orgID}/members/{userID}", orgHandlers.HandleUpdateMemberRole)
			r.Delete("/{orgID}/members/{userID}", orgHandlers.HandleRemoveMember)
		})

		// Transaction endpoints
		r.Get("/transactions", apiHandlers.HandleGetTransactions)
		r.Get("/transactions/recent", apiHandlers.HandleGetRecentTransactions)
		r.Get("/transactions/{transactionID}", apiHandlers.HandleGetTransactionDetail)
		r.Put("/transactions/{transactionID}/category", apiHandlers.HandleUpdateTransactionCategory)

		// Account endpoints
		r.Get("/accounts", apiHandlers.HandleGetAccounts)

		// Connection endpoints
		r.Route("/connections", func(r chi.Router) {
			r.Get("/", apiHandlers.HandleGetConnections)
			r.Post("/", apiHandlers.HandleCreateConnection)
			r.Delete("/{connectionID}", apiHandlers.HandleDeleteConnection)
			r.Post("/{connectionID}/test", apiHandlers.HandleTestConnection)
			r.Get("/{connectionID}/accounts", apiHandlers.HandleGetConnectionAccounts)
			// TODO: Re-enable when River job client is properly initialized
			// r.Post("/{id}/sync", jobHandlers.CreateSyncJob)
		})

		// TODO: Re-enable job endpoints when River job client is properly initialized
		// Job endpoints
		// r.Route("/jobs", func(r chi.Router) {
		//	r.Get("/", jobHandlers.ListJobs)
		//	r.Get("/stats", jobHandlers.GetJobStats)
		//	r.Get("/{id}", jobHandlers.GetJob)
		//	r.Post("/{id}/cancel", jobHandlers.CancelJob)
		//	r.Post("/{id}/pause", jobHandlers.PauseJob)
		//	r.Post("/{id}/resume", jobHandlers.ResumeJob)
		//	r.Post("/{id}/retry", jobHandlers.RetryJob)
		// })

		// Worker endpoints
		// r.Route("/workers", func(r chi.Router) {
		//	r.Get("/", jobHandlers.ListWorkers)
		//	r.Get("/stats", jobHandlers.GetWorkerStats)
		// })

		// Bank account endpoints
		r.Put("/bank-accounts/{accountID}/status", apiHandlers.HandleUpdateAccountStatus)

		// Dashboard endpoints
		r.Get("/dashboard/stats", apiHandlers.HandleGetDashboardStats)
	})

	// Protected page routes
	r.Route("/dashboard", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}
		r.Get("/", pageHandlers.DashboardPage)
	})

	r.Route("/transactions", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}
		r.Get("/", pageHandlers.TransactionsPage)
	})

	r.Route("/accounts", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}
		r.Get("/", pageHandlers.AccountsPage)
		r.Get("/{accountID}", pageHandlers.AccountDetailPage)
	})

	r.Route("/analytics", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}
		r.Get("/", pageHandlers.AnalyticsPage)
	})

	r.Route("/settings", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
			r.Use(authMiddleware.RequireOrganization)
		}
		r.Get("/", pageHandlers.SettingsPage)
		r.Get("/connections", pageHandlers.ConnectionsPage)
	})

	// Static files for production build
	if !s.config.IsDevelopment {
		fileServer := http.FileServer(http.Dir("./src/web/static"))
		r.Handle("/build/*", http.StripPrefix("/", fileServer))
		r.Handle("/assets/*", http.StripPrefix("/", fileServer))
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string    `json:"status"`
	Environment string    `json:"environment"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Supabase    struct {
		Connected bool   `json:"connected"`
		Error     string `json:"error,omitempty"`
	} `json:"supabase"`
}

// healthHandler returns a health check handler
func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status:      "ok",
			Environment: s.config.Environment,
			Version:     "1.0.0", // TODO: Get actual version
			Timestamp:   time.Now(),
		}

		// Check Supabase connection if client is available
		if s.config.Client != nil && s.config.Client.Anon != nil {
			response.Supabase.Connected = true
		} else {
			response.Supabase.Connected = false
			response.Supabase.Error = "Client not initialized"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error().Err(err).Msg("Failed to encode health response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}