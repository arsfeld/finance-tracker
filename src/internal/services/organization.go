package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	supa "github.com/supabase-community/supabase-go"
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

// Organization represents an organization
type Organization struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Settings  map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt string                 `json:"created_at" db:"created_at"`
	UpdatedAt string                 `json:"updated_at" db:"updated_at"`
}

// OrganizationMember represents a member of an organization
type OrganizationMember struct {
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Role           string    `json:"role" db:"role"`
	JoinedAt       string    `json:"joined_at" db:"joined_at"`
}

// CreateOrganizationWithOwner creates a new organization and adds the user as owner
func (s *OrganizationService) CreateOrganizationWithOwner(ctx context.Context, userID string, orgName string) (*Organization, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create organization
	orgData := map[string]interface{}{
		"name": orgName,
	}

	var result []Organization
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

	return &result[0], nil
}

// createDefaultCategories creates default categories for a new organization
func (s *OrganizationService) createDefaultCategories(ctx context.Context, orgID uuid.UUID) error {
	type Category struct {
		OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
		Name           string    `json:"name" db:"name"`
		Color          string    `json:"color" db:"color"`
		Icon           string    `json:"icon" db:"icon"`
	}

	categories := []Category{
		{OrganizationID: orgID, Name: "Food & Dining", Color: "#FF6B6B", Icon: "utensils"},
		{OrganizationID: orgID, Name: "Transportation", Color: "#4ECDC4", Icon: "car"},
		{OrganizationID: orgID, Name: "Shopping", Color: "#45B7D1", Icon: "shopping-bag"},
		{OrganizationID: orgID, Name: "Entertainment", Color: "#96CEB4", Icon: "film"},
		{OrganizationID: orgID, Name: "Bills & Utilities", Color: "#FECA57", Icon: "file-text"},
		{OrganizationID: orgID, Name: "Healthcare", Color: "#FF9FF3", Icon: "heart"},
		{OrganizationID: orgID, Name: "Education", Color: "#54A0FF", Icon: "book"},
		{OrganizationID: orgID, Name: "Travel", Color: "#48DBFB", Icon: "plane"},
		{OrganizationID: orgID, Name: "Other", Color: "#A0A0A0", Icon: "dots-horizontal"},
	}

	_, err := s.client.From("categories").
		Insert(categories, false, "", "", "").
		ExecuteTo(nil)
	
	return err
}

// GetUserOrganizations gets all organizations for a user
func (s *OrganizationService) GetUserOrganizations(ctx context.Context, userID string) ([]Organization, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Query organizations through organization_members
	var result []struct {
		Organization Organization `json:"organizations"`
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
	orgs := make([]Organization, len(result))
	for i, r := range result {
		orgs[i] = r.Organization
	}

	return orgs, nil
}

// GetOrganization gets a single organization by ID
func (s *OrganizationService) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
	// Parse org ID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	var result []Organization
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
	member := OrganizationMember{
		OrganizationID: orgUUID,
		UserID:         userUUID,
		Role:           role,
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
func (s *OrganizationService) GetOrganizationMembers(ctx context.Context, orgID string) ([]OrganizationMember, error) {
	// Parse org ID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	var members []OrganizationMember
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