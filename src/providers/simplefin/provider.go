package simplefin

import (
	"context"
	"time"

	"finance_tracker/src/providers"
)

// Provider implements the provider.Provider interface for SimpleFin
type Provider struct {
	// Add any SimpleFin-specific configuration here
}

// NewProvider creates a new SimpleFin provider
func NewProvider() *Provider {
	return &Provider{}
}

// GetType returns the provider type
func (p *Provider) GetType() string {
	return "simplefin"
}

// ListAccounts fetches accounts from SimpleFin
func (p *Provider) ListAccounts(ctx context.Context, credentials map[string]interface{}) ([]provider.Account, error) {
	// TODO: Implement SimpleFin account listing
	// For now, return empty slice
	return []provider.Account{}, nil
}

// GetTransactions fetches transactions from SimpleFin
func (p *Provider) GetTransactions(ctx context.Context, credentials map[string]interface{}, accountID string, startDate, endDate time.Time) ([]provider.Transaction, error) {
	// TODO: Implement SimpleFin transaction fetching
	// For now, return empty slice
	return []provider.Transaction{}, nil
}