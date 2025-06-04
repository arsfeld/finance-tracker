package models

import (
	"time"

	"github.com/google/uuid"
)

// Aliases and additional types for API compatibility

// Connection is an alias for ProviderConnection
type Connection = ProviderConnection

// APITransaction represents a transaction with additional API fields
type APITransaction struct {
	ID                 uuid.UUID  `json:"id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	AccountID          uuid.UUID  `json:"account_id"`
	AccountName        string     `json:"account_name"`
	Date               time.Time  `json:"date"`
	Amount             float64    `json:"amount"`
	Description        string     `json:"description"`
	ProviderTransID    string     `json:"provider_transaction_id"`
	Category           *string    `json:"category,omitempty"`
	Subcategory        *string    `json:"subcategory,omitempty"`
	Merchant           *string    `json:"merchant,omitempty"`
	Location           *string    `json:"location,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	Tags               []string   `json:"tags,omitempty"`
	Pending            bool       `json:"pending"`
	CategoryConfidence *float64   `json:"category_confidence,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Account is an alias for BankAccount with additional fields for API compatibility
type Account struct {
	ID            uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ConnectionID  uuid.UUID  `json:"connection_id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Balance       float64    `json:"balance"`
	Currency      string     `json:"currency"`
	LastSyncDate  time.Time  `json:"last_sync_date"`
	IsActive      bool       `json:"is_active"`
	ProviderID    string     `json:"provider_id"`
	ProviderName  string     `json:"provider_name"`
	Institution   *string    `json:"institution"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ConnectionAccount is an alias for BankAccount with connection-specific fields
type ConnectionAccount struct {
	ID                uuid.UUID  `json:"id"`
	ConnectionID      uuid.UUID  `json:"connection_id"`
	ProviderAccountID string     `json:"provider_account_id"`
	Name              string     `json:"name"`
	Institution       *string    `json:"institution"`
	AccountType       *string    `json:"account_type"`
	Balance           *float64   `json:"balance"`
	Currency          string     `json:"currency"`
	IsActive          bool       `json:"is_active"`
	LastSync          *time.Time `json:"last_sync"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalBalance     float64   `json:"total_balance"`
	MonthlySpending  float64   `json:"monthly_spending"`
	TransactionCount int       `json:"transaction_count"`
	AccountCount     int       `json:"account_count"`
	LastSync         time.Time `json:"last_sync"`
}

// TransactionFilter represents filters for querying transactions
type TransactionFilter struct {
	AccountID  *uuid.UUID
	Category   *string
	StartDate  *time.Time
	EndDate    *time.Time
	Search     *string
	Limit      int
	Offset     int
}