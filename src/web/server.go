package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/jobs"
	"finance_tracker/src/internal/services"
	"finance_tracker/src/providers"
	"finance_tracker/src/providers/simplefin"
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
	
	// Initialize category service and handler
	var categoryHandler *handlers.CategoryHandler
	if s.config.Client != nil {
		categoryService := services.NewCategoryService(s.config.Client)
		categoryHandler = handlers.NewCategoryHandler(categoryService)
	}

	// Initialize AI service and handler
	var aiHandler *handlers.AIHandler
	if openRouterKey := os.Getenv("OPENROUTER_API_KEY"); openRouterKey != "" && s.config.Client != nil {
		openRouterURL := os.Getenv("OPENROUTER_URL")
		modelsStr := os.Getenv("OPENROUTER_MODELS")
		var models []string
		if modelsStr != "" {
			models = strings.Split(modelsStr, ",")
		}
		
		aiService := services.NewAIService(s.config.Client, openRouterKey, openRouterURL, models)
		aiHandler = handlers.NewAIHandler(aiService)
		log.Info().Msg("AI service initialized successfully")
	} else {
		log.Warn().Msg("AI service disabled - OPENROUTER_API_KEY not configured")
	}
	
	// Initialize River job client and handlers
	var jobHandlers *handlers.RiverJobHandler
	
	// Try to initialize database connection for River jobs
	// This is optional - if it fails, job endpoints will be disabled
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		log.Info().Msg("Initializing database connection for job management")
		
		// Create database connections
		sqlxDB, err := createSQLXConnection()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to create SQLX connection - job endpoints disabled")
		} else {
			defer sqlxDB.Close()
			
			dbPool, err := createPgxPool()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to create pgx pool - job endpoints disabled")
			} else {
				defer dbPool.Close()
				
				// Initialize services
				jobService := services.NewJobService(sqlxDB)
				syncService := services.NewSyncService(sqlxDB, jobService)
				
				// Initialize crypto service
				cryptoService, err := services.NewCryptoService()
				if err != nil {
					log.Warn().Err(err).Msg("Failed to create crypto service - job endpoints disabled")
				} else {
					// Initialize transaction repository for categorization jobs
					transactionRepo := services.NewTransactionRepository(s.config.Client)
					// Initialize providers (optional)
					financialProviders := make(map[string]providers.FinancialProvider)
					if bridgeURL := os.Getenv("SIMPLEFIN_BRIDGE_URL"); bridgeURL != "" {
						sfProvider := simplefin.NewSimpleFin()
						financialProviders["simplefin"] = sfProvider
						syncService.RegisterProvider("simplefin", sfProvider)
						log.Info().Msg("SimpleFin provider registered for job management")
					}
					
					// Initialize River job client
					riverClient, err := jobs.NewRiverJobClient(dbPool, syncService, financialProviders, cryptoService, transactionRepo)
					if err != nil {
						log.Warn().Err(err).Msg("Failed to create River job client - job endpoints disabled")
					} else {
						syncService.SetRiverClient(riverClient)
						jobHandlers = handlers.NewRiverJobHandler(riverClient)
						log.Info().Msg("River job client initialized successfully")
					}
				}
			}
		}
	} else {
		log.Info().Msg("DATABASE_URL not set - job endpoints disabled")
	}

	// Setup routes
	s.setupRoutes(r, pageHandlers, apiHandlers, authHandlers, orgHandlers, jobHandlers, categoryHandler, aiHandler, authMiddleware)

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
	categoryHandler *handlers.CategoryHandler,
	aiHandler *handlers.AIHandler,
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
			// Enable sync endpoint if River job client is available
			if jobHandlers != nil {
				r.Post("/{id}/sync", jobHandlers.CreateSyncJob)
			}
		})

		// Enable job endpoints if River job client is available
		if jobHandlers != nil {
			// Job endpoints
			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", jobHandlers.ListJobs)
				r.Get("/stats", jobHandlers.GetJobStats)
				r.Get("/health", jobHandlers.HealthCheck)
				r.Get("/{id}", jobHandlers.GetJob)
				r.Post("/{id}/cancel", jobHandlers.CancelJob)
			})

			// Analysis job endpoints
			r.Route("/analysis", func(r chi.Router) {
				r.Post("/jobs", jobHandlers.CreateAnalysisJob)
			})

			// Maintenance job endpoints
			r.Route("/maintenance", func(r chi.Router) {
				r.Post("/jobs", jobHandlers.CreateMaintenanceJob)
			})

			// Worker endpoints
			r.Route("/workers", func(r chi.Router) {
				r.Get("/", jobHandlers.ListWorkers)
				r.Get("/stats", jobHandlers.GetWorkerStats)
			})

			// Queue endpoints
			r.Route("/queues", func(r chi.Router) {
				r.Get("/", jobHandlers.GetQueues)
			})
		}

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
		r.Get("/categories", pageHandlers.CategoriesPage)
	})

	// Register category API routes
	if categoryHandler != nil {
		categoryHandler.RegisterRoutes(r)
	}

	// Register AI API routes (with auth middleware)
	if aiHandler != nil {
		r.Group(func(r chi.Router) {
			if authMiddleware != nil {
				r.Use(authMiddleware.RequireAuth)
				r.Use(authMiddleware.RequireOrganization)
			}
			aiHandler.RegisterRoutes(r)
		})
	}

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
// createSQLXConnection creates a SQLX database connection
func createSQLXConnection() (*sqlx.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}
	
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	return db, nil
}

// createPgxPool creates a pgx connection pool for River
func createPgxPool() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	return pool, nil
}
