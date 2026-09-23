// Command promptdump prints the analysis prompt for a database without calling
// the LLM, to tune the prompt against real data. It migrates and seeds the
// database it opens, so point it at a copy:
//
//	DB_PATH=/path/to/copy.db ENV_FILE=/dev/null go run ./cmd/promptdump
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"finance_tracker/internal/api"
	"finance_tracker/internal/config"
	"finance_tracker/internal/database"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/store"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx := context.Background()

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

	accts := store.NewAccountStore(db.Read, db.Write)
	snaps := store.NewSnapshotStore(db.Read, db.Write)
	settings := store.NewSettingsStore(db.Read, db.Write)
	if err := store.InitCardLedger(ctx, accts, snaps, settings); err != nil {
		log.Fatal().Err(err).Msg("Failed to prepare card ledger")
	}

	h := api.NewAnalysisRunHandler(cfg,
		store.NewTransactionStore(db.Read, db.Write), accts,
		store.NewCategoryStore(db.Read, db.Write), snaps, settings,
		store.NewAnalysisStore(db.Read, db.Write), store.NewBudgetStore(db.Read, db.Write),
		nil, nil)

	built, err := h.BuildPrompt(ctx, time.Now().UTC())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build prompt")
	}
	fmt.Println("=== SYSTEM ===")
	fmt.Println(llmclient.SystemPrompt)
	fmt.Println("=== USER ===")
	fmt.Println(built.Text)
}
