// Package importer turns a parsed OFX statement into the app's accounts and
// transactions without duplicating history already supplied by SimpleFIN.
package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
	"finance_tracker/internal/ofx"
	"finance_tracker/internal/store"
)

// Status describes what an import would do with one statement row.
type Status string

const (
	StatusNew       Status = "new"
	StatusDuplicate Status = "duplicate"
	StatusUpdated   Status = "updated"
)

// Row is one statement transaction and the decision made about it.
type Row struct {
	Transaction ofx.Transaction
	ID          string
	Status      Status
	ExistingID  string
}

// Plan is a dry run of an import.
type Plan struct {
	Account          models.DBAccount
	Rows             []Row
	NewCount         int
	DuplicateCount   int
	UpdatedCount     int
	StatementBalance *float64
	BalanceDate      time.Time
	Warnings         []string
}

// Result is the outcome of a committed import.
type Result struct {
	Account        models.DBAccount
	Added          int
	Updated        int
	Duplicates     int
	BalanceApplied bool
	Warnings       []string
}

// Service plans and commits statement imports.
type Service struct {
	accounts  *store.AccountStore
	txns      *store.TransactionStore
	snapshots *store.SnapshotStore
}

func New(accounts *store.AccountStore, txns *store.TransactionStore, snapshots *store.SnapshotStore) *Service {
	return &Service{accounts: accounts, txns: txns, snapshots: snapshots}
}

// Plan decides what each statement row would do, without writing anything.
func (s *Service) Plan(ctx context.Context, accountID string, stmt *ofx.Statement) (*Plan, error) {
	current, identityIDs, err := s.resolve(ctx, accountID)
	if err != nil {
		return nil, err
	}

	plan := &Plan{
		Account:          *current,
		StatementBalance: stmt.LedgerBalance,
		BalanceDate:      statementBalanceDate(stmt),
	}
	warned := make(map[string]bool)
	warn := func(message string) {
		if !warned[message] {
			warned[message] = true
			plan.Warnings = append(plan.Warnings, message)
		}
	}
	if stmt.LedgerBalance == nil {
		warn("The file has no ending balance; the account balance was left unchanged.")
	}
	if len(stmt.Transactions) == 0 {
		warn("The file contains no transactions.")
	}
	if stmt.Currency != "" && current.Currency != "" && !strings.EqualFold(stmt.Currency, current.Currency) {
		warn(fmt.Sprintf("The statement is in %s but the account is in %s.", stmt.Currency, current.Currency))
	}
	for _, tx := range stmt.Transactions {
		if tx.FITID == "" {
			warn("One or more transactions have no FITID; a stable ID was derived from the row itself.")
			break
		}
	}

	existing, err := s.existingForRange(ctx, identityIDs, stmt.Transactions)
	if err != nil {
		return nil, err
	}

	// Count occurrences by signature so FITID-less rows get an ID that does
	// not change when an unrelated row is prepended to a later export.
	occurrences := make(map[string]int, len(stmt.Transactions))
	nth := make([]int, len(stmt.Transactions))
	for i, tx := range stmt.Transactions {
		key := matchKey(tx.Posted.Unix(), tx.Amount, description(tx))
		nth[i] = occurrences[key]
		occurrences[key]++
	}

	// A row is identified by its FITID-derived ID first. The signature pool is
	// the fallback for rows SimpleFIN already stored under a TRN- ID, and it
	// must count both kinds of existing row so repeated identical purchases are
	// neither collapsed nor duplicated.
	type existingRow struct {
		txn  models.DBTransaction
		used bool
	}
	byID := make(map[string]*existingRow, len(existing))
	bySignature := make(map[string][]*existingRow)
	for i := range existing {
		row := &existingRow{txn: existing[i]}
		byID[row.txn.ID] = row
		key := matchKey(row.txn.Posted, row.txn.Amount, row.txn.Description)
		bySignature[key] = append(bySignature[key], row)
	}

	identity := ledger.IdentityKey(*current)
	rows := make([]Row, len(stmt.Transactions))
	var pending []int
	for i, tx := range stmt.Transactions {
		id := importID(identity, tx, nth[i])
		rows[i] = Row{Transaction: tx, ID: id}

		if old := byID[id]; old != nil && !old.used {
			old.used = true
			rows[i].ExistingID = old.txn.ID
			if sameTransaction(old.txn, tx) {
				rows[i].Status = StatusDuplicate
				plan.DuplicateCount++
			} else {
				rows[i].Status = StatusUpdated
				plan.UpdatedCount++
			}
			continue
		}
		pending = append(pending, i)
	}

	for _, i := range pending {
		row := &rows[i]
		key := matchKey(row.Transaction.Posted.Unix(), row.Transaction.Amount, description(row.Transaction))
		var match *existingRow
		for _, candidate := range bySignature[key] {
			if !candidate.used {
				match = candidate
				break
			}
		}
		if match != nil {
			match.used = true
			row.ExistingID = match.txn.ID
			row.Status = StatusDuplicate
			plan.DuplicateCount++
			continue
		}
		row.Status = StatusNew
		plan.NewCount++
	}
	plan.Rows = rows

	return plan, nil
}

// Commit writes a plan's new and changed rows, then updates the account balance
// and records a card snapshot.
func (s *Service) Commit(ctx context.Context, accountID string, stmt *ofx.Statement) (*Result, error) {
	plan, err := s.Plan(ctx, accountID, stmt)
	if err != nil {
		return nil, err
	}

	var batch []models.DBTransaction
	for _, row := range plan.Rows {
		if row.Status == StatusDuplicate {
			continue
		}
		batch = append(batch, models.DBTransaction{
			ID:          row.ID,
			AccountID:   plan.Account.ID,
			Description: description(row.Transaction),
			Amount:      row.Transaction.Amount,
			Posted:      row.Transaction.Posted.Unix(),
			Pending:     false,
		})
	}

	added, updated := 0, 0
	if len(batch) > 0 {
		var err error
		added, updated, err = s.txns.UpsertBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("store imported transactions: %w", err)
		}
	}

	// Once the transactions are written, a later failure must still report what
	// landed so the sync log does not claim nothing happened.
	result := &Result{
		Account:    plan.Account,
		Added:      added,
		Updated:    updated,
		Duplicates: plan.DuplicateCount,
		Warnings:   plan.Warnings,
	}

	if stmt.LedgerBalance != nil {
		date := statementBalanceDate(stmt)
		applied, err := s.accounts.UpdateBalance(ctx, plan.Account.ID, *stmt.LedgerBalance, date.Unix())
		if err != nil {
			return result, fmt.Errorf("update account balance: %w", err)
		}
		result.BalanceApplied = applied
		if applied {
			result.Account.Balance = *stmt.LedgerBalance
			result.Account.BalanceDate = date.Unix()
		}
		if applied && plan.Account.IsCreditCard && plan.Account.CardKey != "" {
			if _, err := s.snapshots.Record(ctx, models.BalanceSnapshot{
				CardKey:     plan.Account.CardKey,
				AccountID:   plan.Account.ID,
				Balance:     *stmt.LedgerBalance,
				BalanceDate: date.Unix(),
				Source:      store.SnapshotSourceImport,
			}); err != nil {
				return result, fmt.Errorf("record card balance: %w", err)
			}
		}
	}

	return result, nil
}

// Resolve returns the current account row for a selected row. An import always
// lands on the current row of the identity, even when the user picked a
// superseded duplicate from an old re-auth.
func (s *Service) resolve(ctx context.Context, accountID string) (*models.DBAccount, []string, error) {
	selected, err := s.accounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	if selected == nil {
		return nil, nil, fmt.Errorf("account %s not found", accountID)
	}

	all, err := s.accounts.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	key := ledger.IdentityKey(*selected)
	var group []models.DBAccount
	var ids []string
	for _, a := range all {
		if ledger.IdentityKey(a) != key {
			continue
		}
		group = append(group, a)
		ids = append(ids, a.ID)
	}
	if len(group) == 0 {
		return nil, nil, fmt.Errorf("account %s not found", accountID)
	}
	current := ledger.CurrentByKey(group)[key]
	sort.Strings(ids)
	return &current, ids, nil
}

func (s *Service) existingForRange(ctx context.Context, accountIDs []string, txns []ofx.Transaction) ([]models.DBTransaction, error) {
	if len(txns) == 0 {
		return nil, nil
	}
	min, max := txns[0].Posted.Unix(), txns[0].Posted.Unix()
	for _, tx := range txns[1:] {
		at := tx.Posted.Unix()
		if at < min {
			min = at
		}
		if at > max {
			max = at
		}
	}
	return s.txns.GetForAccountsPeriod(ctx, accountIDs, min-86400, max+86400)
}

func statementBalanceDate(stmt *ofx.Statement) time.Time {
	if !stmt.BalanceDate.IsZero() {
		return stmt.BalanceDate
	}
	if !stmt.End.IsZero() {
		return time.Date(stmt.End.Year(), stmt.End.Month(), stmt.End.Day(), 12, 0, 0, 0, time.UTC)
	}
	if len(stmt.Transactions) > 0 {
		last := stmt.Transactions[0].Posted
		for _, tx := range stmt.Transactions[1:] {
			if tx.Posted.After(last) {
				last = tx.Posted
			}
		}
		return time.Date(last.Year(), last.Month(), last.Day(), 12, 0, 0, 0, time.UTC)
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
}

func importID(identity string, tx ofx.Transaction, occurrence int) string {
	fitid := tx.FITID
	if fitid == "" {
		fitid = fmt.Sprintf("%s|%.2f|%s|%s|%d", tx.Posted.Format("2006-01-02"), tx.Amount, tx.Name, tx.Memo, occurrence)
	}
	sum := sha256.Sum256([]byte(identity + "\x00" + fitid))
	return "IMP-" + hex.EncodeToString(sum[:12])
}

func description(tx ofx.Transaction) string {
	if name := strings.TrimSpace(tx.Name); name != "" {
		return name
	}
	if memo := strings.TrimSpace(tx.Memo); memo != "" {
		return memo
	}
	return "Unknown"
}

func sameTransaction(old models.DBTransaction, tx ofx.Transaction) bool {
	return old.Posted == tx.Posted.Unix() &&
		math.Abs(old.Amount-tx.Amount) < 0.005 &&
		old.Description == description(tx)
}

func matchKey(posted int64, amount float64, desc string) string {
	day := time.Unix(posted, 0).UTC().Format("2006-01-02")
	return fmt.Sprintf("%s|%d|%s", day, int64(math.Round(amount*100)), normalize(desc))
}

func normalize(desc string) string {
	return strings.ToUpper(strings.Join(strings.Fields(desc), " "))
}

// Candidate is an existing account that may own a statement, with a score for
// how strongly its name or card key matches the file's account number.
type Candidate struct {
	Account models.DBAccount
	Score   int
}

var digitRun = regexp.MustCompile(`\d{4,}`)

// Candidates scores current accounts against the statement's account number.
// A name ending in the file's last four digits wins; a four-digit run anywhere
// in the file (such as TD's Visa BIN at the start) is a weaker match.
func Candidates(accounts []models.DBAccount, fileAccountID string) []Candidate {
	fileGroups := digitRun.FindAllString(fileAccountID, -1)
	if len(fileGroups) == 0 {
		return nil
	}
	fileDigits := strings.Join(fileGroups, "")
	last4 := fileDigits
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}

	plain := make([]models.DBAccount, len(accounts))
	copy(plain, accounts)
	current := ledger.CurrentByKey(plain)

	var candidates []Candidate
	for _, a := range current {
		score := 0
		for _, group := range digitRuns(a.Name + " " + a.CardKey) {
			switch {
			case len(group) >= 4 && group[len(group)-4:] == last4:
				score = 100
			case strings.Contains(fileDigits, group):
				if score < 50 {
					score = 50
				}
			}
		}
		if score > 0 {
			candidates = append(candidates, Candidate{Account: a, Score: score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Account.Name < candidates[j].Account.Name
	})
	return candidates
}

// BestCandidate returns the single strongest candidate, or false when the top
// score is shared and the choice needs a person.
func BestCandidate(candidates []Candidate) (models.DBAccount, bool) {
	if len(candidates) == 0 {
		return models.DBAccount{}, false
	}
	if len(candidates) > 1 && candidates[0].Score == candidates[1].Score {
		return models.DBAccount{}, false
	}
	return candidates[0].Account, true
}

func digitRuns(s string) []string {
	runs := digitRun.FindAllString(s, -1)
	seen := make(map[string]bool, len(runs))
	out := runs[:0]
	for _, run := range runs {
		if len(run) > 4 {
			run = run[len(run)-4:]
		}
		if !seen[run] {
			seen[run] = true
			out = append(out, run)
		}
	}
	return out
}
