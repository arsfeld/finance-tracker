package models

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated user
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Organization represents a tenant in the system
type Organization struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Settings  map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Role           Role      `json:"role" db:"role"`
	JoinedAt       time.Time `json:"joined_at" db:"joined_at"`
}

// Role represents a user's role in an organization
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// Scan implements sql.Scanner interface
func (r *Role) Scan(value interface{}) error {
	*r = Role(value.(string))
	return nil
}

// Value implements driver.Valuer interface
func (r Role) Value() (driver.Value, error) {
	return string(r), nil
}

// ProviderConnection represents a connection to a financial data provider
type ProviderConnection struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	OrganizationID       uuid.UUID              `json:"organization_id" db:"organization_id"`
	ProviderType         string                 `json:"provider_type" db:"provider_type"`
	Name                 string                 `json:"name" db:"name"`
	CredentialsEncrypted string                 `json:"-" db:"credentials_encrypted"`
	Settings             map[string]interface{} `json:"settings" db:"settings"`
	LastSync             *time.Time             `json:"last_sync" db:"last_sync"`
	SyncStatus           string                 `json:"sync_status" db:"sync_status"`
	ErrorMessage         *string                `json:"error_message" db:"error_message"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
}

// BankAccount represents a bank account from a provider
type BankAccount struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	OrganizationID     uuid.UUID              `json:"organization_id" db:"organization_id"`
	ConnectionID       uuid.UUID              `json:"connection_id" db:"connection_id"`
	ProviderAccountID  string                 `json:"provider_account_id" db:"provider_account_id"`
	Name               string                 `json:"name" db:"name"`
	Institution        *string                `json:"institution" db:"institution"`
	AccountType        *string                `json:"account_type" db:"account_type"`
	Balance            *float64               `json:"balance" db:"balance"`
	Currency           string                 `json:"currency" db:"currency"`
	IsActive           bool                   `json:"is_active" db:"is_active"`
	Metadata           map[string]interface{} `json:"metadata" db:"metadata"`
	LastSync           *time.Time             `json:"last_sync" db:"last_sync"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID                     uuid.UUID              `json:"id" db:"id"`
	OrganizationID         uuid.UUID              `json:"organization_id" db:"organization_id"`
	BankAccountID          uuid.UUID              `json:"bank_account_id" db:"bank_account_id"`
	ProviderTransactionID  *string                `json:"provider_transaction_id" db:"provider_transaction_id"`
	Amount                 float64                `json:"amount" db:"amount"`
	Description            *string                `json:"description" db:"description"`
	MerchantName           *string                `json:"merchant_name" db:"merchant_name"`
	CategoryID             *int                   `json:"category_id" db:"category_id"`
	Date                   time.Time              `json:"date" db:"date"`
	PostedDate             *time.Time             `json:"posted_date" db:"posted_date"`
	Pending                bool                   `json:"pending" db:"pending"`
	TransactionType        *string                `json:"transaction_type" db:"transaction_type"`
	Metadata               map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt              time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at" db:"updated_at"`
	
	// Joined fields
	Category               *Category              `json:"category,omitempty"`
	BankAccount            *BankAccount           `json:"bank_account,omitempty"`
}

// Category represents a transaction category
type Category struct {
	ID             int                    `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Name           string                 `json:"name" db:"name"`
	ParentID       *int                   `json:"parent_id" db:"parent_id"`
	Color          *string                `json:"color" db:"color"`
	Icon           *string                `json:"icon" db:"icon"`
	Rules          []CategoryRule         `json:"rules" db:"rules"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	
	// Nested fields
	Children       []Category             `json:"children,omitempty"`
}

// CategoryRule represents a rule for auto-categorization
type CategoryRule struct {
	Field         string `json:"field"`
	Operator      string `json:"operator"`
	Value         string `json:"value"`
	CaseSensitive bool   `json:"case_sensitive"`
}

// AIAnalysis represents an AI-generated analysis
type AIAnalysis struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	DateStart      time.Time              `json:"date_start" db:"date_start"`
	DateEnd        time.Time              `json:"date_end" db:"date_end"`
	AnalysisType   string                 `json:"analysis_type" db:"analysis_type"`
	Content        string                 `json:"content" db:"content"`
	Model          *string                `json:"model" db:"model"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedBy      *uuid.UUID             `json:"created_by" db:"created_by"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// ChatSession represents a chat session
type ChatSession struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	Title          *string                `json:"title" db:"title"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	SessionID uuid.UUID              `json:"session_id" db:"session_id"`
	Role      string                 `json:"role" db:"role"`
	Content   string                 `json:"content" db:"content"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}