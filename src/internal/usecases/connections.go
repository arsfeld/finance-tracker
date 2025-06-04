package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"finance_tracker/src/internal/models"
	"finance_tracker/src/providers"
)

// ConnectionService defines the interface for connection data operations
type ConnectionService interface {
	ListConnections(ctx context.Context, orgID uuid.UUID) ([]models.Connection, error)
	GetConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) (*models.Connection, error)
	CreateConnection(ctx context.Context, conn models.Connection) (*models.Connection, error)
	DeleteConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) error
	UpdateConnectionStatus(ctx context.Context, connectionID uuid.UUID, status string, errorMsg *string) error
	CheckDuplicateName(ctx context.Context, orgID uuid.UUID, name string) (bool, error)
}

// AccountService defines the interface for account operations
type AccountService interface {
	CreateAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID, accounts []providers.ProviderAccount) error
	ListConnectionAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) ([]models.ConnectionAccount, error)
	UpdateAccountStatus(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID, isActive bool) error
}

// ProviderService defines the interface for provider operations
type ProviderService interface {
	ValidateCredentials(providerType string, credentials map[string]string) error
	ExchangeSetupToken(ctx context.Context, providerType string, setupToken string) (string, error)
	ListAccounts(ctx context.Context, providerType string, credentials map[string]string) ([]providers.ProviderAccount, error)
	TestConnection(ctx context.Context, providerType string, credentials map[string]string) error
	EncryptCredentials(credentials map[string]string) (string, error)
	DecryptCredentials(encrypted string) (map[string]string, error)
}

// ConnectionUseCase handles connection business logic
type ConnectionUseCase struct {
	connectionService ConnectionService
	accountService    AccountService
	providerService   ProviderService
}

// NewConnectionUseCase creates a new connection use case
func NewConnectionUseCase(connectionService ConnectionService, accountService AccountService, providerService ProviderService) *ConnectionUseCase {
	return &ConnectionUseCase{
		connectionService: connectionService,
		accountService:    accountService,
		providerService:   providerService,
	}
}

// ListConnections returns all provider connections for the organization
func (uc *ConnectionUseCase) ListConnections(ctx context.Context, orgID uuid.UUID) ([]models.Connection, error) {
	return uc.connectionService.ListConnections(ctx, orgID)
}

// CreateConnectionRequest represents the request to create a connection
type CreateConnectionRequest struct {
	Name         string            `json:"name"`
	ProviderType string            `json:"provider_type"`
	Credentials  map[string]string `json:"credentials"`
	SetupToken   string            `json:"setup_token,omitempty"`
}

// CreateConnection creates a new provider connection
func (uc *ConnectionUseCase) CreateConnection(ctx context.Context, orgID uuid.UUID, req CreateConnectionRequest) (*models.Connection, error) {
	// Validate input
	if req.Name == "" {
		return nil, &ValidationError{Field: "name", Message: "Name is required"}
	}
	if req.ProviderType == "" {
		return nil, &ValidationError{Field: "provider_type", Message: "Provider type is required"}
	}
	if len(req.Name) > 100 {
		return nil, &ValidationError{Field: "name", Message: "Name must be 100 characters or less"}
	}

	// Check for duplicate name
	exists, err := uc.connectionService.CheckDuplicateName(ctx, orgID, req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ValidationError{Field: "name", Message: "A connection with this name already exists"}
	}

	// Handle setup token
	credentials := req.Credentials
	if req.SetupToken != "" {
		// Exchange setup token for access URL
		accessURL, err := uc.providerService.ExchangeSetupToken(ctx, req.ProviderType, req.SetupToken)
		if err != nil {
			return nil, err
		}
		credentials = map[string]string{"access_url": accessURL}
	}

	// Validate credentials with provider
	if err := uc.providerService.ValidateCredentials(req.ProviderType, credentials); err != nil {
		return nil, err
	}

	// Test connection by listing accounts
	accounts, err := uc.providerService.ListAccounts(ctx, req.ProviderType, credentials)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, &ValidationError{Field: "connection", Message: "No accounts found. Please verify your setup and try again."}
	}

	// Encrypt credentials
	encryptedCredentials, err := uc.providerService.EncryptCredentials(credentials)
	if err != nil {
		return nil, err
	}

	// Create connection
	connection := models.Connection{
		ID:                   uuid.New(),
		OrganizationID:       orgID,
		Name:                 req.Name,
		ProviderType:         req.ProviderType,
		CredentialsEncrypted: encryptedCredentials,
		SyncStatus:           "success",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	createdConnection, err := uc.connectionService.CreateConnection(ctx, connection)
	if err != nil {
		return nil, err
	}

	// Create bank accounts
	if err := uc.accountService.CreateAccounts(ctx, orgID, createdConnection.ID, accounts); err != nil {
		// Log error but don't fail the entire operation
		// TODO: Add proper logging
	}

	return createdConnection, nil
}

// DeleteConnection deletes a provider connection
func (uc *ConnectionUseCase) DeleteConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) error {
	return uc.connectionService.DeleteConnection(ctx, orgID, connectionID)
}

// TestConnection tests a provider connection
func (uc *ConnectionUseCase) TestConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) (bool, *string, error) {
	// Get connection
	connection, err := uc.connectionService.GetConnection(ctx, orgID, connectionID)
	if err != nil {
		return false, nil, err
	}

	// Decrypt credentials
	credentials, err := uc.providerService.DecryptCredentials(connection.CredentialsEncrypted)
	if err != nil {
		return false, nil, err
	}

	// Test connection
	err = uc.providerService.TestConnection(ctx, connection.ProviderType, credentials)
	
	var status string
	var errorMessage *string
	if err != nil {
		status = "error"
		errMsg := err.Error()
		errorMessage = &errMsg
	} else {
		status = "success"
		errorMessage = nil
	}

	// Update connection status
	if updateErr := uc.connectionService.UpdateConnectionStatus(ctx, connectionID, status, errorMessage); updateErr != nil {
		// Log error but continue
	}

	return err == nil, errorMessage, nil
}

// ListConnectionAccounts returns all bank accounts for a connection
func (uc *ConnectionUseCase) ListConnectionAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) ([]models.ConnectionAccount, error) {
	return uc.accountService.ListConnectionAccounts(ctx, orgID, connectionID)
}

// UpdateAccountStatus updates the active status of a bank account
func (uc *ConnectionUseCase) UpdateAccountStatus(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID, isActive bool) error {
	return uc.accountService.UpdateAccountStatus(ctx, orgID, accountID, isActive)
}