package provider

import (
	"context"
	"time"
)

// Provider defines the interface for financial data providers used by River workers
type Provider interface {
	// Provider identification
	GetType() string
	
	// Account operations
	ListAccounts(ctx context.Context, credentials map[string]interface{}) ([]Account, error)
	
	// Transaction operations  
	GetTransactions(ctx context.Context, credentials map[string]interface{}, accountID string, startDate, endDate time.Time) ([]Transaction, error)
}

// Account represents an account from a provider
type Account struct {
	ID          string  `json:"id"`
	ProviderID  string  `json:"provider_id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Balance     float64 `json:"balance"`
	Currency    string  `json:"currency"`
}

// Transaction represents a transaction from a provider
type Transaction struct {
	ID          string    `json:"id"`
	ProviderID  string    `json:"provider_id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Category    string    `json:"category"`
	Memo        string    `json:"memo"`
}