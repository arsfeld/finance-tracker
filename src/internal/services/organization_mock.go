package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	supa "github.com/supabase-community/supabase-go"
)

// MockOrganizationService is a development mock
type MockOrganizationService struct {
	client *supa.Client
}

// NewMockOrganizationService creates a mock service for development
func NewMockOrganizationService(client *supa.Client) *MockOrganizationService {
	return &MockOrganizationService{
		client: client,
	}
}

// CreateOrganizationWithOwner creates a mock organization
func (s *MockOrganizationService) CreateOrganizationWithOwner(ctx context.Context, userID string, orgName string) (*Organization, error) {
	orgID := uuid.New()
	
	org := &Organization{
		ID:       orgID,
		Name:     orgName,
		Settings: make(map[string]interface{}),
	}

	return org, nil
}

// GetUserOrganizations returns mock organizations
func (s *MockOrganizationService) GetUserOrganizations(ctx context.Context, userID string) ([]Organization, error) {
	return []Organization{
		{
			ID:       uuid.New(),
			Name:     "Mock Organization",
			Settings: make(map[string]interface{}),
		},
	}, nil
}

// GetOrganization returns a mock organization
func (s *MockOrganizationService) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	return &Organization{
		ID:       orgUUID,
		Name:     "Mock Organization",
		Settings: make(map[string]interface{}),
	}, nil
}

// InviteMember is a mock implementation
func (s *MockOrganizationService) InviteMember(ctx context.Context, orgID string, userEmail string, role string) error {
	return nil
}

// GetOrganizationMembers returns mock members
func (s *MockOrganizationService) GetOrganizationMembers(ctx context.Context, orgID string) ([]OrganizationMember, error) {
	orgUUID, _ := uuid.Parse(orgID)
	userUUID := uuid.New()
	
	return []OrganizationMember{
		{
			OrganizationID: orgUUID,
			UserID:         userUUID,
			Role:           "owner",
		},
	}, nil
}

// UpdateMemberRole is a mock implementation
func (s *MockOrganizationService) UpdateMemberRole(ctx context.Context, orgID string, userID string, newRole string) error {
	return nil
}

// RemoveMember is a mock implementation
func (s *MockOrganizationService) RemoveMember(ctx context.Context, orgID string, userID string) error {
	return nil
}