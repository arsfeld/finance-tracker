package providers

import (
	"context"
	"time"
)

// FinancialProvider defines the interface for all financial data providers
type FinancialProvider interface {
	// Provider identification
	GetProviderType() string
	ValidateCredentials(credentials map[string]string) error

	// Account operations
	ListAccounts(ctx context.Context, credentials map[string]string) ([]ProviderAccount, error)
	GetAccount(ctx context.Context, credentials map[string]string, accountID string) (*ProviderAccount, error)

	// Transaction operations
	GetTransactions(ctx context.Context, credentials map[string]string, accountID string, startDate, endDate time.Time) ([]ProviderTransaction, error)

	// Health check
	HealthCheck(ctx context.Context, credentials map[string]string) error
}

// ProviderAccount represents an account from any provider
type ProviderAccount struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Institution string                 `json:"institution"`
	Type        string                 `json:"type"` // checking, savings, credit, etc.
	Balance     float64                `json:"balance"`
	Currency    string                 `json:"currency"`
	LastUpdated time.Time              `json:"last_updated"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProviderTransaction represents a transaction from any provider
type ProviderTransaction struct {
	ID          string                 `json:"id"`
	AccountID   string                 `json:"account_id"`
	Amount      float64                `json:"amount"`
	Description string                 `json:"description"`
	Merchant    string                 `json:"merchant"`
	Date        time.Time              `json:"date"`
	PostedDate  *time.Time             `json:"posted_date"`
	Pending     bool                   `json:"pending"`
	Type        string                 `json:"type"` // debit, credit, transfer
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProviderError represents a provider-specific error
type ProviderError struct {
	Code    string
	Message string
	Retry   bool
}

func (e *ProviderError) Error() string {
	return e.Message
}

// Registry holds all available providers
type Registry struct {
	providers map[string]FinancialProvider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]FinancialProvider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider FinancialProvider) {
	r.providers[provider.GetProviderType()] = provider
}

// Get retrieves a provider by type
func (r *Registry) Get(providerType string) (FinancialProvider, bool) {
	provider, ok := r.providers[providerType]
	return provider, ok
}

// List returns all available provider types
func (r *Registry) List() []string {
	types := make([]string, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}
	return types
}