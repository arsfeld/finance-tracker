package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/config"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/notify"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/simplefin"
	"finance_tracker/internal/store"
)

type SyncHandler struct {
	cfg       *config.Config
	accounts  *store.AccountStore
	txns      *store.TransactionStore
	cats      *store.CategoryStore
	syncLog   *store.SyncLogStore
	scheduler *scheduler.Scheduler
	events    *EventHub
}

func NewSyncHandler(
	cfg *config.Config,
	accounts *store.AccountStore,
	txns *store.TransactionStore,
	cats *store.CategoryStore,
	syncLog *store.SyncLogStore,
	sched *scheduler.Scheduler,
	events *EventHub,
) *SyncHandler {
	return &SyncHandler{
		cfg:       cfg,
		accounts:  accounts,
		txns:      txns,
		cats:      cats,
		syncLog:   syncLog,
		scheduler: sched,
		events:    events,
	}
}

func (h *SyncHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if !h.scheduler.TryAcquire("sync") {
		WriteError(w, http.StatusConflict, "SYNC_IN_PROGRESS",
			"A sync or analysis operation is already running: "+h.scheduler.CurrentJob())
		return
	}

	go func() {
		defer h.scheduler.Release()
		h.runSync(context.Background())
	}()

	WriteJSON(w, http.StatusAccepted, Response{Data: map[string]string{"status": "started"}})
}

func (h *SyncHandler) runSync(ctx context.Context) {
	h.events.Broadcast("sync_started", `{"status":"running"}`)

	logID, err := h.syncLog.Create(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create sync log")
		return
	}

	client := simplefin.NewClient(h.cfg.SimplefinBridgeURL)
	end := time.Now().UTC()
	start := simplefin.ClampToAPILimit(end.AddDate(0, -3, 0), end)

	txnAdded, txnUpdated, apiErrors, err := client.FetchAndStore(ctx, start, end, h.accounts, h.txns)
	if err != nil {
		apiErrJSON, _ := json.Marshal(apiErrors)
		h.syncLog.Complete(ctx, logID, "error", 0, 0, err.Error(), string(apiErrJSON))
		h.events.Broadcast("sync_error", `{"error":"`+err.Error()+`"}`)
		return
	}

	status := "success"
	if len(apiErrors) > 0 {
		status = "partial"
	}

	apiErrJSON, _ := json.Marshal(apiErrors)
	h.syncLog.Complete(ctx, logID, status, txnAdded, txnUpdated, "", string(apiErrJSON))
	h.events.Broadcast("sync_complete", `{"status":"`+status+`","transactions":`+
		fmt.Sprintf("%d", txnAdded)+`}`)

	log.Info().
		Int("added", txnAdded).Int("updated", txnUpdated).
		Int("api_errors", len(apiErrors)).Msg("Sync complete")

	h.alertOnStaleConnections(ctx, end, apiErrors)

	// Run categorization on fetched transactions.
	h.runCategorization(ctx, start, end)
}

// StaleConnectionThreshold is how long an included account may go without a
// balance refresh before it is treated as a broken connection. SimpleFin
// refreshes balances on every successful sync, so three days spans a missed
// sync without crying wolf over a quiet card.
const StaleConnectionThreshold = 72 * time.Hour

// alertOnStaleConnections notifies when an included account is not reporting in full.
//
// This runs after the upserts so balance dates are current. Without it a
// de-authorized connection is recorded only as a "partial" row in sync_log,
// which is how the TD card went ten days unnoticed while every report showed
// spending falling.
func (h *SyncHandler) alertOnStaleConnections(ctx context.Context, now time.Time, apiErrors []string) {
	stale, err := h.accounts.StaleConnections(ctx, now, StaleConnectionThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check for stale connections")
		return
	}

	drifted, err := h.accounts.UnreconciledAccounts(ctx, h.cfg.BalanceDriftThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reconcile account balances")
		return
	}

	message := notify.SyncHealthAlert(stale, drifted, apiErrors, now)
	if message == "" {
		return
	}

	log.Warn().
		Int("stale_accounts", len(stale)).
		Int("unreconciled_accounts", len(drifted)).
		Msg("Accounts are not reporting in full")
	h.events.Broadcast("sync_stale", fmt.Sprintf(`{"stale":%d,"unreconciled":%d}`, len(stale), len(drifted)))

	dispatcher := newDispatcher(h.cfg)
	if _, err := dispatcher.Send(message, nil, "warning"); err != nil {
		log.Error().Err(err).Msg("Failed to send stale connection alert")
	}
}

func (h *SyncHandler) runCategorization(ctx context.Context, start, end time.Time) {
	if h.cfg.OpenRouterURL == "" || h.cfg.OpenRouterAPIKey == "" || h.cfg.OpenRouterModel == "" {
		log.Debug().Msg("Skipping categorization: OpenRouter not configured in .env")
		return
	}

	txns, err := h.txns.GetForPeriod(ctx, start.Unix(), end.Unix())
	if err != nil || len(txns) == 0 {
		return
	}

	client := llmclient.NewClient(h.cfg.OpenRouterURL, h.cfg.OpenRouterAPIKey, h.cfg.OpenRouterModel)
	if err := llmclient.CategorizeTransactions(ctx, client, h.cats, txns); err != nil {
		log.Error().Err(err).Msg("Categorization failed")
	} else {
		h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	}
}

// RunSync is called by the scheduler for periodic syncs.
func (h *SyncHandler) RunSync() {
	if !h.scheduler.TryAcquire("scheduled_sync") {
		log.Warn().Msg("Skipping scheduled sync: another operation is running")
		return
	}
	defer h.scheduler.Release()
	h.runSync(context.Background())
}

func (h *SyncHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	current := h.scheduler.CurrentJob()
	WriteData(w, map[string]string{"current_job": current})
}

func (h *SyncHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.syncLog.List(r.Context(), 20)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, entries)
}
