package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	supa "github.com/supabase-community/supabase-go"
	
	"finance_tracker/src/internal/models"
)

// OrganizationService handles organization-related operations
type OrganizationService struct {
	client *supa.Client
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(client *supa.Client) *OrganizationService {
	return &OrganizationService{
		client: client,
	}
}


// CreateOrganizationWithOwner creates a new organization and adds the user as owner
func (s *OrganizationService) CreateOrganizationWithOwner(ctx context.Context, userID string, orgName string) (*models.Organization, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create organization
	orgData := map[string]interface{}{
		"name": orgName,
	}

	var result []models.Organization
	_, err = s.client.From("organizations").
		Insert(orgData, false, "", "*", "").
		ExecuteTo(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("organization created but not returned")
	}

	// Add user as owner
	memberData := map[string]interface{}{
		"organization_id": result[0].ID.String(),
		"user_id":         userUUID.String(),
		"role":            "owner",
	}

	_, err = s.client.From("organization_members").
		Insert(memberData, false, "", "", "").
		ExecuteTo(nil)
	
	if err != nil {
		// Rollback by deleting the organization
		s.client.From("organizations").
			Delete("", "").
			Eq("id", result[0].ID.String()).
			ExecuteTo(nil)
		return nil, fmt.Errorf("failed to add user as owner: %w", err)
	}

	// Create default categories for the organization
	if err := s.createDefaultCategories(ctx, result[0].ID); err != nil {
		// Log error but don't fail organization creation
		fmt.Printf("Failed to create default categories for organization %s: %v\n", result[0].ID, err)
	}

	return &result[0], nil
}

// createDefaultCategories creates default categories for a new organization
func (s *OrganizationService) createDefaultCategories(ctx context.Context, orgID uuid.UUID) error {
	// Helper function to create string pointers
	stringPtr := func(s string) *string { return &s }

	categories := []models.Category{
		{OrganizationID: orgID, Name: "Food & Dining", Color: stringPtr("#FF6B6B"), Icon: stringPtr("utensils")},
		{OrganizationID: orgID, Name: "Transportation", Color: stringPtr("#4ECDC4"), Icon: stringPtr("car")},
		{OrganizationID: orgID, Name: "Shopping", Color: stringPtr("#45B7D1"), Icon: stringPtr("shopping-bag")},
		{OrganizationID: orgID, Name: "Entertainment", Color: stringPtr("#96CEB4"), Icon: stringPtr("film")},
		{OrganizationID: orgID, Name: "Bills & Utilities", Color: stringPtr("#FECA57"), Icon: stringPtr("file-text")},
		{OrganizationID: orgID, Name: "Healthcare", Color: stringPtr("#FF9FF3"), Icon: stringPtr("heart")},
		{OrganizationID: orgID, Name: "Education", Color: stringPtr("#54A0FF"), Icon: stringPtr("book")},
		{OrganizationID: orgID, Name: "Travel", Color: stringPtr("#48DBFB"), Icon: stringPtr("plane")},
		{OrganizationID: orgID, Name: "Other", Color: stringPtr("#A0A0A0"), Icon: stringPtr("dots-horizontal")},
	}

	_, err := s.client.From("categories").
		Insert(categories, false, "", "", "").
		ExecuteTo(nil)
	
	return err
}

// GetUserOrganizations gets all organizations for a user
func (s *OrganizationService) GetUserOrganizations(ctx context.Context, userID string) ([]models.Organization, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Query organizations through organization_members
	var result []struct {
		Organization models.Organization `json:"organizations"`
		Role         string       `json:"role"`
	}

	_, err = s.client.From("organization_members").
		Select("role,organizations(*)", "", false).
		Eq("user_id", userUUID.String()).
		ExecuteTo(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	// Extract organizations
	orgs := make([]models.Organization, len(result))
	for i, r := range result {
		orgs[i] = r.Organization
	}

	return orgs, nil
}

// GetOrganization gets a single organization by ID
func (s *OrganizationService) GetOrganization(ctx context.Context, orgID string) (*models.Organization, error) {
	// Parse org ID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	var result []models.Organization
	_, err = s.client.From("organizations").
		Select("*", "", false).
		Eq("id", orgUUID.String()).
		Single().
		ExecuteTo(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("organization not found")
	}

	return &result[0], nil
}

// InviteMember adds a new member to an organization
func (s *OrganizationService) InviteMember(ctx context.Context, orgID string, userEmail string, role string) error {
	// Validate role
	validRoles := map[string]bool{
		"owner":  true,
		"admin":  true,
		"member": true,
		"viewer": true,
	}
	if !validRoles[role] {
		return fmt.Errorf("invalid role: %s", role)
	}

	// Parse org ID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	// Get user by email
	var users []struct {
		ID string `json:"id"`
	}
	_, err = s.client.From("auth.users").
		Select("id", "", false).
		Eq("email", userEmail).
		ExecuteTo(&users)
	if err != nil || len(users) == 0 {
		return fmt.Errorf("user not found with email: %s", userEmail)
	}

	userUUID, err := uuid.Parse(users[0].ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Add member to organization
	member := models.OrganizationMember{
		OrganizationID: orgUUID,
		UserID:         userUUID,
		Role:           models.Role(role),
	}

	_, err = s.client.From("organization_members").
		Insert(member, false, "", "", "").
		ExecuteTo(nil)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// GetOrganizationMembers gets all members of an organization
func (s *OrganizationService) GetOrganizationMembers(ctx context.Context, orgID string) ([]models.OrganizationMember, error) {
	// Parse org ID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	var members []models.OrganizationMember
	_, err = s.client.From("organization_members").
		Select("*", "", false).
		Eq("organization_id", orgUUID.String()).
		ExecuteTo(&members)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization members: %w", err)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role in an organization
func (s *OrganizationService) UpdateMemberRole(ctx context.Context, orgID string, userID string, newRole string) error {
	// Validate role
	validRoles := map[string]bool{
		"owner":  true,
		"admin":  true,
		"member": true,
		"viewer": true,
	}
	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Parse IDs
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Update member role
	updates := map[string]interface{}{
		"role": newRole,
	}

	_, err = s.client.From("organization_members").
		Update(updates, "", "").
		Eq("organization_id", orgUUID.String()).
		Eq("user_id", userUUID.String()).
		ExecuteTo(nil)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

// RemoveMember removes a member from an organization
func (s *OrganizationService) RemoveMember(ctx context.Context, orgID string, userID string) error {
	// Parse IDs
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Remove member
	_, err = s.client.From("organization_members").
		Delete("", "").
		Eq("organization_id", orgUUID.String()).
		Eq("user_id", userUUID.String()).
		ExecuteTo(nil)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}