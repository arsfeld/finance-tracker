package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Balance represents a monetary value that can be unmarshaled from either string or float64
type Balance float64

// UnmarshalJSON implements the json.Unmarshaler interface for Balance
func (b *Balance) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first (since API specifies numeric string)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		// If successful, try to convert to float64
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("error parsing balance string '%s': %w", s, err)
		}
		*b = Balance(f)
		return nil
	}

	// If string unmarshal fails, try float64 (for backward compatibility)
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("error unmarshaling balance: %w", err)
	}
	*b = Balance(f)
	return nil
}

// String returns a string representation of the balance
func (b Balance) String() string {
	return fmt.Sprintf("%.2f", float64(b))
}

// NotificationType defines the type of notification
type NotificationType string

// Available notification types
const (
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeNtfy  NotificationType = "ntfy"
)

// DateRangeType defines the type of date range for analysis
type DateRangeType string

// Available date range types
const (
	DateRangeTypeCurrentMonth DateRangeType = "current_month"
	DateRangeTypeLastMonth    DateRangeType = "last_month"
	DateRangeTypeLast3Months  DateRangeType = "last_3_months"
	DateRangeTypeCurrentYear  DateRangeType = "current_year"
	DateRangeTypeLastYear     DateRangeType = "last_year"
	DateRangeTypeCustom       DateRangeType = "custom"
)

// Organization represents a financial institution or organization
type Organization struct {
	SfinURL string  `json:"sfin-url"`
	Domain  *string `json:"domain,omitempty"`
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	ID      *string `json:"id,omitempty"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID           string                  `json:"id"`
	Description  string                  `json:"description"`
	Amount       Balance                 `json:"amount"`
	Posted       int64                   `json:"posted"`
	TransactedAt *int64                  `json:"transacted_at,omitempty"`
	Pending      *bool                   `json:"pending,omitempty"`
	Extra        *map[string]interface{} `json:"extra,omitempty"`
}

// Account represents a financial account
type Account struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Balance          Balance       `json:"balance"`
	BalanceDate      int64         `json:"balance-date"`
	Org              Organization  `json:"org"`
	Transactions     []Transaction `json:"transactions,omitempty"`
	Currency         *string       `json:"currency,omitempty"`
	AvailableBalance *Balance      `json:"available-balance,omitempty"`
	Holdings         []interface{} `json:"holdings,omitempty"`
}

// AccountsResponse represents the response from the SimpleFin API
type AccountsResponse struct {
	Accounts    []Account `json:"accounts"`
	Errors      []string  `json:"errors,omitempty"`
	XAPIMessage []string  `json:"x-api-message,omitempty"`
}
