package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"finance_tracker/src/providers"
	"finance_tracker/src/providers/simplefin"
)

// ProviderService handles provider operations
type ProviderService struct{}

// NewProviderService creates a new provider service
func NewProviderService() *ProviderService {
	return &ProviderService{}
}

// ValidateCredentials validates provider credentials
func (s *ProviderService) ValidateCredentials(providerType string, credentials map[string]string) error {
	provider, err := s.getProvider(providerType)
	if err != nil {
		return err
	}
	return provider.ValidateCredentials(credentials)
}

// ExchangeSetupToken exchanges a setup token for access credentials
func (s *ProviderService) ExchangeSetupToken(ctx context.Context, providerType string, setupToken string) (string, error) {
	if providerType != "simplefin" {
		return "", fmt.Errorf("setup token only supported for SimpleFin provider")
	}

	provider := simplefin.NewSimpleFin()
	return provider.SetupTokenExchange(ctx, setupToken)
}

// ListAccounts lists accounts from a provider
func (s *ProviderService) ListAccounts(ctx context.Context, providerType string, credentials map[string]string) ([]providers.ProviderAccount, error) {
	provider, err := s.getProvider(providerType)
	if err != nil {
		return nil, err
	}
	return provider.ListAccounts(ctx, credentials)
}

// TestConnection tests a provider connection
func (s *ProviderService) TestConnection(ctx context.Context, providerType string, credentials map[string]string) error {
	provider, err := s.getProvider(providerType)
	if err != nil {
		return err
	}
	return provider.HealthCheck(ctx, credentials)
}

// EncryptCredentials encrypts provider credentials
func (s *ProviderService) EncryptCredentials(credentials map[string]string) (string, error) {
	// TODO: Use proper encryption in production
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(credentialsJSON), nil
}

// DecryptCredentials decrypts provider credentials
func (s *ProviderService) DecryptCredentials(encrypted string) (map[string]string, error) {
	// TODO: Use proper decryption in production
	credentialsJSON, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}
	
	var credentials map[string]string
	if err := json.Unmarshal(credentialsJSON, &credentials); err != nil {
		return nil, err
	}
	
	return credentials, nil
}

// getProvider returns a provider instance based on type
func (s *ProviderService) getProvider(providerType string) (providers.FinancialProvider, error) {
	switch providerType {
	case "simplefin":
		return simplefin.NewSimpleFin(), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}