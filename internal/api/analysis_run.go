package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/ledger"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/models"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/store"
)

// AnalysisRunHandler handles on-demand and scheduled analysis runs.
type AnalysisRunHandler struct {
	cfg           *config.Config
	txnStore      *store.TransactionStore
	acctStore     *store.AccountStore
	catStore      *store.CategoryStore
	snapshotStore *store.SnapshotStore
	settingsStore *store.SettingsStore
	analysisStore *store.AnalysisStore
	budgetStore   *store.BudgetStore
	scheduler     *scheduler.Scheduler
	events        *EventHub
}

func NewAnalysisRunHandler(
	cfg *config.Config,
	txns *store.TransactionStore,
	accts *store.AccountStore,
	cats *store.CategoryStore,
	snapshots *store.SnapshotStore,
	settings *store.SettingsStore,
	analyses *store.AnalysisStore,
	budgets *store.BudgetStore,
	sched *scheduler.Scheduler,
	events *EventHub,
) *AnalysisRunHandler {
	return &AnalysisRunHandler{
		cfg:           cfg,
		txnStore:      txns,
		acctStore:     accts,
		catStore:      cats,
		snapshotStore: snapshots,
		settingsStore: settings,
		analysisStore: analyses,
		budgetStore:   budgets,
		scheduler:     sched,
		events:        events,
	}
}

func (h *AnalysisRunHandler) TriggerAnalysis(w http.ResponseWriter, r *http.Request) {
	if !h.scheduler.TryAcquire("analysis") {
		WriteError(w, http.StatusConflict, "OP_IN_PROGRESS",
			"Another operation is running: "+h.scheduler.CurrentJob())
		return
	}

	go func() {
		defer h.scheduler.Release()
		h.runAnalysis(context.Background())
	}()

	WriteJSON(w, http.StatusAccepted, Response{Data: map[string]string{"status": "started"}})
}

// AnalysisPrompt is an assembled analysis prompt and what it was built from.
type AnalysisPrompt struct {
	Text       string
	Start, End time.Time
	Charges    []models.DBTransaction
}

// BuildPrompt assembles the analysis prompt from the database without calling
// the LLM. cmd/promptdump uses it to tune the prompt against a copy of
// production.
func (h *AnalysisRunHandler) BuildPrompt(ctx context.Context, now time.Time) (*AnalysisPrompt, error) {
	billingDay := h.cfg.BillingDay
	start, end, err := billing.CalculateDateRange(models.DateRangeTypeCurrentAndLastMonth, nil, nil, billingDay)
	if err != nil {
		return nil, fmt.Errorf("date range: %w", err)
	}
	periods := llmclient.CyclePeriods(start, end, billingDay)

	accounts, err := h.acctStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("accounts: %w", err)
	}
	snapshots, err := h.snapshotStore.ListByCard(ctx)
	if err != nil {
		return nil, fmt.Errorf("balance snapshots: %w", err)
	}
	txns, err := h.txnStore.GetForPeriodAllAccounts(ctx, ledger.FetchFrom(snapshots, start).Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("transactions: %w", err)
	}
	rawPatterns, err := h.settingsStore.Get(ctx, ledger.PaymentPatternsSettingKey)
	if err != nil {
		return nil, fmt.Errorf("payment patterns: %w", err)
	}
	patterns, err := ledger.ParsePaymentPatterns(rawPatterns)
	if err != nil {
		return nil, fmt.Errorf("payment patterns: %w", err)
	}
	excluded, _ := h.catStore.ExcludedCategoryNames(ctx)

	report := ledger.Build(periods, accounts, txns, snapshots, patterns, excluded)

	// Only the cards being analyzed matter to the model, and only through their
	// current account: a superseded ID is dead by definition. Without this the
	// old TD account would be reported stale forever.
	analyzed := make(map[string]bool)
	for _, a := range ledger.AnalyzedCards(accounts) {
		analyzed[a.ID] = true
	}
	stale, err := h.acctStore.StaleConnections(ctx, now, StaleConnectionThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check for stale connections")
	}
	drifted, err := h.acctStore.UnreconciledAccounts(ctx, h.cfg.BalanceDriftThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reconcile account balances")
	}
	stale, drifted = filterHealth(stale, drifted, func(id string) bool { return analyzed[id] })

	budgets, _ := h.budgetStore.GetAll(ctx)

	text := llmclient.GeneratePrompt(llmclient.PromptInput{
		Periods: report.Periods, Charges: report.Charges,
		Stale: stale, Drifted: drifted, Budgets: budgets,
		BillingDay: billingDay, Now: now,
	})
	return &AnalysisPrompt{Text: text, Start: start, End: end, Charges: report.Charges}, nil
}

func (h *AnalysisRunHandler) runAnalysis(ctx context.Context) {
	h.events.Broadcast("analysis_started", `{"status":"running"}`)

	billingDay := h.cfg.BillingDay
	dateRangeType := models.DateRangeTypeCurrentAndLastMonth

	built, err := h.BuildPrompt(ctx, time.Now().UTC())
	if err != nil {
		log.Error().Err(err).Msg("Failed to build analysis prompt")
		h.events.Broadcast("analysis_error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return
	}
	prompt, startDate, endDate, txns := built.Text, built.Start, built.End, built.Charges

	if h.cfg.OpenRouterURL == "" || h.cfg.OpenRouterAPIKey == "" || h.cfg.OpenRouterModel == "" {
		log.Error().Msg("OpenRouter not configured")
		h.events.Broadcast("analysis_error", `{"error":"OpenRouter not configured. Set OPENROUTER_URL, OPENROUTER_API_KEY, and OPENROUTER_MODEL in .env"}`)
		return
	}

	llm := llmclient.NewClient(h.cfg.OpenRouterURL, h.cfg.OpenRouterAPIKey, h.cfg.OpenRouterModel)

	// Call LLM with retry.
	response, err := llmclient.RetryWithBackoff(func() (string, error) {
		content, model, err := llm.Analyze(llmclient.SystemPrompt, prompt, true)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s\n\n---\n*Generated by %s*", content, model), nil
	}, 3, 2, "LLM analysis")

	status := "success"
	if err != nil {
		log.Error().Err(err).Msg("LLM analysis failed")
		response = fmt.Sprintf("Analysis failed: %s", err.Error())
		status = "error"
	}

	// Store result.
	_, storeErr := h.analysisStore.Create(ctx, models.Analysis{
		PeriodStart:   startDate.Unix(),
		PeriodEnd:     endDate.Unix(),
		BillingDay:    billingDay,
		DateRangeType: string(dateRangeType),
		ResponseText:  response,
		IsMultiPeriod: true,
		Status:        status,
	})
	if storeErr != nil {
		log.Error().Err(storeErr).Msg("Failed to store analysis result")
	}

	if status == "success" {
		h.events.Broadcast("analysis_complete", `{"status":"success"}`)
		log.Info().Msg("Analysis complete")

		// Send notifications.
		h.sendNotifications(response, txns)
	} else {
		h.events.Broadcast("analysis_error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
	}
}

func (h *AnalysisRunHandler) sendNotifications(message string, txns []models.DBTransaction) {
	dispatcher := newDispatcher(h.cfg)

	channels, err := dispatcher.Send(message, txns, "info")
	if err != nil {
		log.Error().Err(err).Msg("Failed to send analysis notifications")
	} else if len(channels) > 0 {
		log.Info().Strs("channels", channels).Msg("Notifications sent")
	}
}

// RunAnalysis is called by the scheduler for periodic analysis.
func (h *AnalysisRunHandler) RunAnalysis() {
	if !h.scheduler.TryAcquire("scheduled_analysis") {
		log.Warn().Msg("Skipping scheduled analysis: another operation is running")
		return
	}
	defer h.scheduler.Release()
	h.runAnalysis(context.Background())
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// Suppress unused import.
var _ = time.Now
