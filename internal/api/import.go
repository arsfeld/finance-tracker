package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/config"
	"finance_tracker/internal/importer"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/models"
	"finance_tracker/internal/ofx"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/store"
)

const maxImportBytes = 10 << 20

// ImportHandler accepts statement files and writes them into an existing
// account.
type ImportHandler struct {
	cfg       *config.Config
	accounts  *store.AccountStore
	txns      *store.TransactionStore
	cats      *store.CategoryStore
	snapshots *store.SnapshotStore
	syncLog   *store.SyncLogStore
	scheduler *scheduler.Scheduler
	events    *EventHub
	importer  *importer.Service
}

func NewImportHandler(
	cfg *config.Config,
	accounts *store.AccountStore,
	txns *store.TransactionStore,
	cats *store.CategoryStore,
	snapshots *store.SnapshotStore,
	syncLog *store.SyncLogStore,
	sched *scheduler.Scheduler,
	events *EventHub,
) *ImportHandler {
	return &ImportHandler{
		cfg:       cfg,
		accounts:  accounts,
		txns:      txns,
		cats:      cats,
		snapshots: snapshots,
		syncLog:   syncLog,
		scheduler: sched,
		events:    events,
		importer:  importer.New(accounts, txns, snapshots),
	}
}

type importStatementInfo struct {
	AccountID        string   `json:"account_id"`
	AccountType      string   `json:"account_type,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Start            int64    `json:"start,omitempty"`
	End              int64    `json:"end,omitempty"`
	LedgerBalance    *float64 `json:"ledger_balance,omitempty"`
	BalanceDate      int64    `json:"balance_date,omitempty"`
	TransactionCount int      `json:"transaction_count"`
}

type importRow struct {
	Date        int64   `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	FITID       string  `json:"fitid,omitempty"`
	ExistingID  string  `json:"existing_id,omitempty"`
}

type importCandidate struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	OrgName   string `json:"org_name"`
	Score     int    `json:"score"`
}

type importPreviewResponse struct {
	Filename   string              `json:"filename"`
	Statement  importStatementInfo `json:"statement"`
	Candidates []importCandidate   `json:"candidates"`
	Account    *models.DBAccount   `json:"account,omitempty"`
	NewCount   int                 `json:"new_count"`
	Duplicates int                 `json:"duplicate_count"`
	Updated    int                 `json:"updated_count"`
	Rows       []importRow         `json:"rows"`
	Warnings   []string            `json:"warnings,omitempty"`
}

type importResultResponse struct {
	LogID          int64            `json:"log_id"`
	Account        models.DBAccount `json:"account"`
	Added          int              `json:"added"`
	Updated        int              `json:"updated"`
	Duplicates     int              `json:"duplicates"`
	BalanceApplied bool             `json:"balance_applied"`
	Warnings       []string         `json:"warnings,omitempty"`
}

// Preview parses a statement and reports what an import would do.
func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	stmt, filename, accountID, ok := h.readStatement(w, r)
	if !ok {
		return
	}

	all, err := h.accounts.List(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	candidates := importer.Candidates(all, stmt.AccountID)
	resp := importPreviewResponse{
		Filename:   filename,
		Statement:  statementInfo(stmt),
		Candidates: candidateInfo(candidates),
		Rows:       []importRow{},
	}

	if accountID == "" {
		if best, found := importer.BestCandidate(candidates); found {
			accountID = best.ID
		}
	}
	if accountID == "" {
		resp.Warnings = h.importWarnings(nil, candidates, "")
		WriteData(w, resp)
		return
	}

	plan, err := h.importer.Plan(r.Context(), accountID, stmt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "ACCOUNT_NOT_FOUND", err.Error())
		return
	}
	resp.Account = &plan.Account
	resp.NewCount = plan.NewCount
	resp.Duplicates = plan.DuplicateCount
	resp.Updated = plan.UpdatedCount
	resp.Warnings = h.importWarnings(plan.Warnings, candidates, plan.Account.ID)
	resp.Rows = rowInfo(plan.Rows)
	WriteData(w, resp)
}

// Commit imports a statement into the chosen account.
func (h *ImportHandler) Commit(w http.ResponseWriter, r *http.Request) {
	stmt, filename, accountID, ok := h.readStatement(w, r)
	if !ok {
		return
	}
	if accountID == "" {
		WriteError(w, http.StatusBadRequest, "ACCOUNT_REQUIRED", "Choose the account this statement belongs to.")
		return
	}

	if h.scheduler != nil {
		if !h.scheduler.TryAcquire("import") {
			WriteError(w, http.StatusConflict, "IMPORT_IN_PROGRESS",
				"A sync or analysis is already running: "+h.scheduler.CurrentJob())
			return
		}
		defer h.scheduler.Release()
	}

	plan, err := h.importer.Plan(r.Context(), accountID, stmt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "ACCOUNT_NOT_FOUND", err.Error())
		return
	}

	// Log the row the import actually lands on, not the superseded row the
	// user may have selected from the account list.
	logID, err := h.syncLog.CreateEntry(r.Context(), "import", filename, plan.Account.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	candidates := importer.Candidates([]models.DBAccount{plan.Account}, stmt.AccountID)

	result, err := h.importer.Commit(r.Context(), accountID, stmt)
	if err != nil {
		status, added, updated := "error", 0, 0
		if result != nil {
			added, updated = result.Added, result.Updated
			if added+updated > 0 {
				status = "partial"
			}
		}
		h.syncLog.Complete(context.Background(), logID, status, added, updated, err.Error(), "[]")
		WriteError(w, http.StatusInternalServerError, "IMPORT_FAILED", err.Error())
		return
	}

	warnings := h.importWarnings(result.Warnings, candidates, result.Account.ID)
	h.syncLog.Complete(context.Background(), logID, "success", result.Added, result.Updated, "", "[]")
	h.events.Broadcast("sync_complete", fmt.Sprintf(`{"status":"success","source":"import","transactions":%d}`, result.Added))

	if newRows := newTransactions(plan); len(newRows) > 0 {
		go h.categorize(newRows)
	}

	WriteData(w, importResultResponse{
		LogID:          logID,
		Account:        result.Account,
		Added:          result.Added,
		Updated:        result.Updated,
		Duplicates:     result.Duplicates,
		BalanceApplied: result.BalanceApplied,
		Warnings:       warnings,
	})
}

func (h *ImportHandler) readStatement(w http.ResponseWriter, r *http.Request) (*ofx.Statement, string, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
	if err := r.ParseMultipartForm(maxImportBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "The statement is larger than 10 MB.")
			return nil, "", "", false
		}
		WriteError(w, http.StatusBadRequest, "BAD_FILE", "Could not read the uploaded file: "+err.Error())
		return nil, "", "", false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_FILE", "Choose an OFX, QFX, QBO or ASO statement to upload.")
		return nil, "", "", false
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_FILE", "Could not read the uploaded file: "+err.Error())
		return nil, "", "", false
	}

	stmt, err := ofx.Parse(bytes.NewReader(data))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_FILE", err.Error())
		return nil, "", "", false
	}
	return stmt, header.Filename, r.FormValue("account_id"), true
}

// importWarnings adds the cross-cutting cautions the importer service cannot
// know about: whether SimpleFIN is still configured, and whether the chosen
// account actually resembles the statement's account number.
func (h *ImportHandler) importWarnings(warnings []string, candidates []importer.Candidate, accountID string) []string {
	seen := make(map[string]bool, len(warnings))
	for _, warning := range warnings {
		seen[warning] = true
	}
	add := func(message string) {
		if !seen[message] {
			seen[message] = true
			warnings = append(warnings, message)
		}
	}

	if h.cfg != nil && h.cfg.SimplefinBridgeURL != "" {
		add("SimpleFIN is still configured; a later sync may overwrite this statement balance or duplicate these transactions.")
	}
	if accountID != "" {
		matched := false
		for _, candidate := range candidates {
			if candidate.Account.ID == accountID {
				matched = true
				break
			}
		}
		if !matched {
			add("The statement's account number does not match this account's name; confirm the destination account.")
		}
	}
	return warnings
}

func (h *ImportHandler) categorize(rows []models.DBTransaction) {
	if h.cfg == nil || h.cfg.OpenRouterURL == "" || h.cfg.OpenRouterAPIKey == "" || h.cfg.OpenRouterModel == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := llmclient.NewClient(h.cfg.OpenRouterURL, h.cfg.OpenRouterAPIKey, h.cfg.OpenRouterModel)
	if err := llmclient.CategorizeTransactions(ctx, client, h.cats, rows); err != nil {
		log.Error().Err(err).Msg("Categorization after import failed")
		return
	}
	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
}

func statementInfo(stmt *ofx.Statement) importStatementInfo {
	info := importStatementInfo{
		AccountID:        stmt.AccountID,
		AccountType:      stmt.AccountType,
		Currency:         stmt.Currency,
		LedgerBalance:    stmt.LedgerBalance,
		TransactionCount: len(stmt.Transactions),
	}
	if !stmt.Start.IsZero() {
		info.Start = stmt.Start.Unix()
	}
	if !stmt.End.IsZero() {
		info.End = stmt.End.Unix()
	}
	if !stmt.BalanceDate.IsZero() {
		info.BalanceDate = stmt.BalanceDate.Unix()
	}
	return info
}

func candidateInfo(candidates []importer.Candidate) []importCandidate {
	out := make([]importCandidate, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, importCandidate{
			AccountID: c.Account.ID,
			Name:      c.Account.Name,
			OrgName:   c.Account.OrgName,
			Score:     c.Score,
		})
	}
	return out
}

func rowInfo(rows []importer.Row) []importRow {
	out := make([]importRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, importRow{
			Date:        row.Transaction.Posted.Unix(),
			Description: rowDescription(row.Transaction),
			Amount:      row.Transaction.Amount,
			Status:      string(row.Status),
			FITID:       row.Transaction.FITID,
			ExistingID:  row.ExistingID,
		})
	}
	return out
}

func newTransactions(plan *importer.Plan) []models.DBTransaction {
	var out []models.DBTransaction
	for _, row := range plan.Rows {
		if row.Status != importer.StatusNew {
			continue
		}
		out = append(out, models.DBTransaction{
			ID:          row.ID,
			AccountID:   plan.Account.ID,
			Description: rowDescription(row.Transaction),
			Amount:      row.Transaction.Amount,
			Posted:      row.Transaction.Posted.Unix(),
		})
	}
	return out
}

func rowDescription(tx ofx.Transaction) string {
	if tx.Name != "" {
		return tx.Name
	}
	if tx.Memo != "" {
		return tx.Memo
	}
	return "Unknown"
}
