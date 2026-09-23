package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"finance_tracker/internal/config"
	"finance_tracker/internal/database"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/server"
	"finance_tracker/internal/store"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	envFile := ".env"
	if v := os.Getenv("ENV_FILE"); v != "" {
		envFile = v
	}
	cfg, err := config.Load(envFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	if err := database.Migrate(db.Write); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Failure is not fatal, and every step is idempotent, so the next start
	// retries it. Until then: accounts stored before card tracking stay
	// unclassified, so the analysis finds no cards and reports an error rather
	// than a $0 report; without the seeded snapshots the balance history starts
	// at the next sync, and earlier periods fall back to itemized totals,
	// labeled as such; missing payment patterns fall back to the defaults.
	if err := store.InitCardLedger(context.Background(),
		store.NewAccountStore(db.Read, db.Write),
		store.NewSnapshotStore(db.Read, db.Write),
		store.NewSettingsStore(db.Read, db.Write)); err != nil {
		log.Error().Err(err).Msg("Failed to prepare card ledger")
	}

	sched := scheduler.New()

	srv := server.New(db, cfg, sched)

	// Schedule periodic sync + analysis if SimpleFin is configured.
	if cfg.SimplefinBridgeURL != "" {
		if err := sched.AddFunc(cfg.SyncSchedule, func() {
			srv.Sync.RunSync()
			srv.Analysis.RunAnalysis()
		}); err != nil {
			log.Error().Err(err).Str("schedule", cfg.SyncSchedule).Msg("Failed to add sync schedule")
		} else {
			log.Info().Str("schedule", cfg.SyncSchedule).Msg("Scheduled periodic sync + analysis")
		}
	}

	sched.Start()

	httpSrv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.ListenAddr).Msg("Server starting")
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sched.Stop(ctx)
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}
	log.Info().Msg("Shutdown complete")
}
