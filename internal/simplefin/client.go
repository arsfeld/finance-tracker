package simplefin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

// Client communicates with the SimpleFin Bridge API.
type Client struct {
	bridgeURL string
}

// NewClient creates a new SimpleFin API client.
func NewClient(bridgeURL string) *Client {
	return &Client{bridgeURL: bridgeURL}
}

// FetchAndStore fetches transactions from SimpleFin and upserts them into the store.
// Returns (accounts fetched, api errors, fatal error).
func (c *Client) FetchAndStore(ctx context.Context, startDate, endDate time.Time, accounts *store.AccountStore, transactions *store.TransactionStore) (int, []string, error) {
	fetched, apiErrors, err := c.fetch(startDate, endDate)
	if err != nil {
		return 0, nil, err
	}

	totalAdded := 0
	for _, acct := range fetched {
		orgName := ""
		if acct.Org.Name != nil {
			orgName = *acct.Org.Name
		}
		orgDomain := ""
		if acct.Org.Domain != nil {
			orgDomain = *acct.Org.Domain
		}
		currency := ""
		if acct.Currency != nil {
			currency = *acct.Currency
		}

		if err := accounts.Upsert(ctx, models.DBAccount{
			ID:          acct.ID,
			Name:        acct.Name,
			Balance:     acct.Balance.Float64(),
			BalanceDate: acct.BalanceDate,
			Currency:    currency,
			OrgName:     orgName,
			OrgDomain:   orgDomain,
			IsIncluded:  true,
		}); err != nil {
			return 0, apiErrors, fmt.Errorf("upsert account %s: %w", acct.ID, err)
		}

		var dbTxns []models.DBTransaction
		for _, tx := range acct.Transactions {
			pending := false
			if tx.Pending != nil {
				pending = *tx.Pending
			}
			dbTxns = append(dbTxns, models.DBTransaction{
				ID:           tx.ID,
				AccountID:    acct.ID,
				Description:  tx.Description,
				Amount:       tx.Amount.Float64(),
				Posted:       tx.Posted,
				TransactedAt: tx.TransactedAt,
				Pending:      pending,
			})
		}

		if len(dbTxns) > 0 {
			added, _, err := transactions.UpsertBatch(ctx, dbTxns)
			if err != nil {
				return 0, apiErrors, fmt.Errorf("upsert transactions for account %s: %w", acct.ID, err)
			}
			totalAdded += added
		}
	}

	return totalAdded, apiErrors, nil
}

func (c *Client) fetch(startDate, endDate time.Time) ([]models.Account, []string, error) {
	url := fmt.Sprintf("%s/accounts?start-date=%d&end-date=%d", c.bridgeURL, startDate.Unix(), endDate.Unix())
	log.Debug().Str("url", url).Msg("Fetching transactions from SimpleFin bridge")

	client := &http.Client{Timeout: 120 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accountsResp models.AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResp); err != nil {
		return nil, nil, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info().Int("account_count", len(accountsResp.Accounts)).Msg("Fetched accounts from SimpleFin")

	if len(accountsResp.Errors) > 0 {
		for _, errMsg := range accountsResp.Errors {
			log.Warn().Str("error", errMsg).Msg("SimpleFin API error")
		}
	}

	// Every account the bridge returns is kept. A balance of exactly zero is the
	// normal end state of a paid-off credit card, and discarding it dropped a
	// whole cycle of transactions with it; is_included is the supported way to
	// leave an account out.
	return accountsResp.Accounts, accountsResp.Errors, nil
}

// ClampToAPILimit returns a start date that is at most 90 days before end.
func ClampToAPILimit(start, end time.Time) time.Time {
	maxStart := end.AddDate(0, 0, -90)
	if start.Before(maxStart) {
		return maxStart
	}
	return start
}
