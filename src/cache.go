package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// LoadCache reads the transaction cache from disk. Returns an empty cache if the file doesn't exist.
func LoadCache(path string) (*TransactionCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Str("path", path).Msg("Cache file not found, starting with empty cache")
			return &TransactionCache{
				Accounts:     make(map[string]CachedAccount),
				Transactions: make(map[string]CachedTransaction),
			}, nil
		}
		return nil, err
	}

	var cache TransactionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if cache.Accounts == nil {
		cache.Accounts = make(map[string]CachedAccount)
	}
	if cache.Transactions == nil {
		cache.Transactions = make(map[string]CachedTransaction)
	}

	log.Info().
		Int("accounts", len(cache.Accounts)).
		Int("transactions", len(cache.Transactions)).
		Str("last_fetched", cache.LastFetched).
		Msg("📦 Loaded transaction cache")

	return &cache, nil
}

// SaveCache writes the transaction cache to disk with indentation.
func SaveCache(path string, cache *TransactionCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	log.Info().
		Int("accounts", len(cache.Accounts)).
		Int("transactions", len(cache.Transactions)).
		Str("path", path).
		Msg("📦 Saved transaction cache")

	return nil
}

// MergeTransactions upserts fresh account/transaction data into the cache.
// Account metadata is always updated. Transactions are upserted by ID.
func MergeTransactions(cache *TransactionCache, accounts []Account) *TransactionCache {
	now := time.Now().UTC()
	cache.LastFetched = now.Format(time.RFC3339)
	cachedAt := now.Unix()

	newTxCount := 0
	updatedTxCount := 0

	for _, acct := range accounts {
		// Update account metadata
		currency := ""
		if acct.Currency != nil {
			currency = *acct.Currency
		}
		cache.Accounts[acct.ID] = CachedAccount{
			ID:          acct.ID,
			Name:        acct.Name,
			Balance:     acct.Balance,
			BalanceDate: acct.BalanceDate,
			Currency:    currency,
		}

		// Upsert transactions
		for _, tx := range acct.Transactions {
			_, exists := cache.Transactions[tx.ID]
			pending := false
			if tx.Pending != nil {
				pending = *tx.Pending
			}
			cache.Transactions[tx.ID] = CachedTransaction{
				ID:           tx.ID,
				AccountID:    acct.ID,
				Description:  tx.Description,
				Amount:       tx.Amount,
				Posted:       tx.Posted,
				TransactedAt: tx.TransactedAt,
				Pending:      pending,
				CachedAt:     cachedAt,
			}
			if exists {
				updatedTxCount++
			} else {
				newTxCount++
			}
		}
	}

	log.Info().
		Int("new_transactions", newTxCount).
		Int("updated_transactions", updatedTxCount).
		Int("total_cached", len(cache.Transactions)).
		Msg("📦 Merged transactions into cache")

	return cache
}

// GetTransactionsForRange reconstructs Account→Transaction structure from the cache
// for transactions within the given date range.
func GetTransactionsForRange(cache *TransactionCache, start, end time.Time) ([]Account, []Transaction) {
	// Group transactions by account
	acctTxns := make(map[string][]Transaction)
	var allTxns []Transaction

	startUnix := start.Unix()
	endUnix := end.Unix()

	for _, ct := range cache.Transactions {
		// Use TransactedAt if available, otherwise Posted
		timestamp := ct.Posted
		if ct.TransactedAt != nil {
			timestamp = *ct.TransactedAt
		}

		if timestamp >= startUnix && timestamp <= endUnix {
			tx := Transaction{
				ID:           ct.ID,
				Description:  ct.Description,
				Amount:       ct.Amount,
				Posted:       ct.Posted,
				TransactedAt: ct.TransactedAt,
			}
			if ct.Pending {
				pending := true
				tx.Pending = &pending
			}
			acctTxns[ct.AccountID] = append(acctTxns[ct.AccountID], tx)
			allTxns = append(allTxns, tx)
		}
	}

	// Build Account objects
	var accounts []Account
	for acctID, txns := range acctTxns {
		ca, ok := cache.Accounts[acctID]
		if !ok {
			continue
		}
		var currency *string
		if ca.Currency != "" {
			c := ca.Currency
			currency = &c
		}
		accounts = append(accounts, Account{
			ID:           ca.ID,
			Name:         ca.Name,
			Balance:      ca.Balance,
			BalanceDate:  ca.BalanceDate,
			Currency:     currency,
			Transactions: txns,
		})
	}

	log.Debug().
		Int("accounts", len(accounts)).
		Int("transactions", len(allTxns)).
		Str("start", start.Format("2006-01-02")).
		Str("end", end.Format("2006-01-02")).
		Msg("Reconstructed accounts from cache for date range")

	return accounts, allTxns
}

// PrunePendingOlderThan removes stale pending transactions older than the cutoff.
func PrunePendingOlderThan(cache *TransactionCache, cutoff time.Time) int {
	cutoffUnix := cutoff.Unix()
	pruned := 0

	for id, ct := range cache.Transactions {
		if ct.Pending && ct.CachedAt < cutoffUnix {
			delete(cache.Transactions, id)
			pruned++
		}
	}

	if pruned > 0 {
		log.Info().
			Int("pruned", pruned).
			Str("cutoff", cutoff.Format("2006-01-02")).
			Msg("📦 Pruned stale pending transactions")
	}

	return pruned
}
