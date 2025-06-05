package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/jobs"
	"finance_tracker/src/internal/services"
	"finance_tracker/src/providers"
	"finance_tracker/src/providers/simplefin"
)

// WorkerConfig holds configuration for the worker
type WorkerConfig struct {
	EnvFile     string
	Environment string
	Queues      []string
	Concurrency int
}

// workerCmd represents the River worker command
func workerCmd() *cobra.Command {
	var cfg WorkerConfig

	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Start the River job worker",
		Long:  `Start the River job worker to process background sync and analysis jobs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorker(cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.EnvFile, "env-file", ".env", "Path to environment file")
	cmd.Flags().StringVar(&cfg.Environment, "environment", "development", "Environment (development/production)")
	cmd.Flags().StringSliceVar(&cfg.Queues, "queues", []string{"sync", "analysis", "maintenance", "default", "high_priority", "categorization"}, "Queues to process")
	cmd.Flags().IntVar(&cfg.Concurrency, "concurrency", 5, "Number of concurrent jobs to process")

	return cmd
}

func runWorker(cfg WorkerConfig) error {
	// Initialize logger
	initLogger(cfg.Environment == "development")

	log.Info().Str("version", GetVersion()).Msg("🚀 Starting Finaro Worker")

	// Load settings
	settings, err := NewSettings(cfg.EnvFile)
	if err != nil {
		return fmt.Errorf("error loading settings: %w", err)
	}

	// Initialize Supabase client (optional for now)
	supabaseClient, err := config.NewSupabaseClient()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Supabase client - continuing without it")
	}

	// Create PostgreSQL connection for database operations
	sqlxDB, err := createSQLXConnection()
	if err != nil {
		return fmt.Errorf("failed to create SQLX connection: %w", err)
	}
	defer sqlxDB.Close()

	// Get pgxpool from the same connection string for River
	dbPool, err := createPgxPool()
	if err != nil {
		return fmt.Errorf("failed to create pgx pool: %w", err)
	}
	defer dbPool.Close()

	// Initialize services
	jobService := services.NewJobService(sqlxDB)
	syncService := services.NewSyncService(sqlxDB, jobService)
	
	// Initialize crypto service
	cryptoService, err := services.NewCryptoService()
	if err != nil {
		return fmt.Errorf("failed to create crypto service: %w", err)
	}
	
	// Initialize transaction repository for categorization jobs
	var transactionRepo *services.TransactionRepository
	if supabaseClient != nil {
		transactionRepo = services.NewTransactionRepository(supabaseClient)
		log.Info().Msg("Transaction repository initialized for categorization jobs")
	} else {
		log.Warn().Msg("Transaction repository not available - categorization jobs will be disabled")
	}

	// Initialize providers
	financialProviders := make(map[string]providers.FinancialProvider)
	if settings.SimplefinBridgeURL != "" {
		sfProvider := simplefin.NewSimpleFin()
		financialProviders["simplefin"] = sfProvider
		syncService.RegisterProvider("simplefin", sfProvider)
		log.Info().Msg("SimpleFin provider registered")
	}

	// Initialize River job client
	riverClient, err := jobs.NewRiverJobClient(dbPool, syncService, financialProviders, cryptoService, transactionRepo)
	if err != nil {
		return fmt.Errorf("failed to create River job client: %w", err)
	}

	// Set the river client back into sync service
	syncService.SetRiverClient(riverClient)

	// Start the worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().
		Strs("queues", cfg.Queues).
		Int("concurrency", cfg.Concurrency).
		Msg("Starting River worker")

	err = riverClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start River worker: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info().Msg("🎯 Worker started successfully - waiting for jobs")
	log.Info().Msg("Press Ctrl+C to stop gracefully")

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("🛑 Shutdown signal received, stopping worker gracefully...")

	// Cancel context to stop the worker
	cancel()

	// Stop the River client
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	err = riverClient.Stop(shutdownCtx)
	if err != nil {
		log.Error().Err(err).Msg("Error during worker shutdown")
		return err
	}

	log.Info().Msg("✅ Worker stopped gracefully")
	return nil
}

// createSQLXConnection creates a direct SQLX connection for services
func createSQLXConnection() (*sqlx.DB, error) {
	// Get connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// createPgxPool creates a pgx connection pool for River
func createPgxPool() (*pgxpool.Pool, error) {
	// Get connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Test the connection
	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}