package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Balance represents a monetary value that can be unmarshaled from either string or float64.
type Balance float64

func (b *Balance) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("error parsing balance string '%s': %w", s, err)
		}
		*b = Balance(f)
		return nil
	}

	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("error unmarshaling balance: %w", err)
	}
	*b = Balance(f)
	return nil
}

func (b Balance) String() string {
	return fmt.Sprintf("%.2f", float64(b))
}

func (b Balance) Float64() float64 {
	return float64(b)
}

// NotificationType defines the type of notification.
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeNtfy  NotificationType = "ntfy"
)

// DateRangeType defines the type of date range for analysis.
type DateRangeType string

const (
	DateRangeTypeCurrentMonth        DateRangeType = "current_month"
	DateRangeTypeLastMonth           DateRangeType = "last_month"
	DateRangeTypeCurrentAndLastMonth DateRangeType = "current_and_last_month"
	DateRangeTypeLast3Months         DateRangeType = "last_3_months"
	DateRangeTypeCurrentYear         DateRangeType = "current_year"
	DateRangeTypeLastYear            DateRangeType = "last_year"
	DateRangeTypeCustom              DateRangeType = "custom"
)

// Organization represents a financial institution.
type Organization struct {
	SfinURL string  `json:"sfin-url"`
	Domain  *string `json:"domain,omitempty"`
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	ID      *string `json:"id,omitempty"`
}

// Transaction represents a financial transaction from SimpleFin.
type Transaction struct {
	ID           string                  `json:"id"`
	Description  string                  `json:"description"`
	Amount       Balance                 `json:"amount"`
	Posted       int64                   `json:"posted"`
	TransactedAt *int64                  `json:"transacted_at,omitempty"`
	Pending      *bool                   `json:"pending,omitempty"`
	Extra        *map[string]interface{} `json:"extra,omitempty"`
}

// Account represents a financial account from SimpleFin.
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

// AccountsResponse represents the response from the SimpleFin API.
type AccountsResponse struct {
	Accounts    []Account `json:"accounts"`
	Errors      []string  `json:"errors,omitempty"`
	XAPIMessage []string  `json:"x-api-message,omitempty"`
}

// BillingPeriod represents a single billing cycle.
type BillingPeriod struct {
	Label      string
	Start      time.Time
	End        time.Time
	IsComplete bool
	IsFocus    bool
}

// Analysis represents a stored LLM analysis result.
type Analysis struct {
	ID            int64  `json:"id"`
	CreatedAt     string `json:"created_at"`
	PeriodStart   int64  `json:"period_start"`
	PeriodEnd     int64  `json:"period_end"`
	BillingDay    int    `json:"billing_day"`
	DateRangeType string `json:"date_range_type"`
	ResponseText  string `json:"response_text"`
	ModelUsed     string `json:"model_used"`
	IsMultiPeriod bool   `json:"is_multi_period"`
	Status        string `json:"status"`
}

// DBAccount represents an account row in the database.
type DBAccount struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Balance     float64 `json:"balance"`
	BalanceDate int64   `json:"balance_date"`
	Currency    string  `json:"currency"`
	OrgName     string  `json:"org_name"`
	OrgDomain   string  `json:"org_domain"`
	IsIncluded  bool    `json:"is_included"`
	FirstSeenAt string  `json:"first_seen_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// DBTransaction represents a transaction row in the database.
type DBTransaction struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	Posted       int64   `json:"posted"`
	TransactedAt *int64  `json:"transacted_at,omitempty"`
	Pending      bool    `json:"pending"`
	Category     string  `json:"category"`
	CachedAt     string  `json:"cached_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// SyncLogEntry represents a sync operation log.
type SyncLogEntry struct {
	ID                  int64  `json:"id"`
	StartedAt           string `json:"started_at"`
	CompletedAt         string `json:"completed_at,omitempty"`
	Status              string `json:"status"`
	TransactionsAdded   int    `json:"transactions_added"`
	TransactionsUpdated int    `json:"transactions_updated"`
	ErrorMessage        string `json:"error_message,omitempty"`
	APIErrors           string `json:"api_errors,omitempty"`
}

// Budget represents a per-category spending limit.
type Budget struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

// CategoryEntry represents a merchant-to-category mapping.
type CategoryEntry struct {
	MerchantDescription string `json:"merchant_description"`
	Category            string `json:"category"`
	Source              string `json:"source"`
	Excluded            bool   `json:"excluded"`
	UpdatedAt           string `json:"updated_at"`
}

// TransactionCategory represents a single categorization result from the LLM.
type TransactionCategory struct {
	Description string `json:"description"`
	Category    string `json:"category"`
}

// CategorizationResponse represents the LLM response for transaction categorization.
type CategorizationResponse struct {
	Categories []TransactionCategory `json:"categories"`
}

// StaleConnection describes an included account whose bank connection has
// stopped refreshing at the source. Both the sync alert and the analysis prompt
// are built from it, so the rule for "stale" lives in exactly one place.
type StaleConnection struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	OrgName         string `json:"org_name"`
	BalanceDate     int64  `json:"balance_date"`
	LastTransaction int64  `json:"last_transaction"` // 0 when the account has no transactions
}
