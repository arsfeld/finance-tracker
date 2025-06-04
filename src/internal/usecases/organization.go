package usecases

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/services"
)

// OrganizationUseCase handles organization-related business logic
type OrganizationUseCase struct {
	client     *config.Client
	orgService *services.OrganizationService
}

// NewOrganizationUseCase creates a new organization use case
func NewOrganizationUseCase(client *config.Client) *OrganizationUseCase {
	return &OrganizationUseCase{
		client:     client,
		orgService: services.NewOrganizationService(client.Service),
	}
}

// CreateOrganizationRequest represents an organization creation request
type CreateOrganizationRequest struct {
	Name   string
	UserID string
}

// ListUserOrganizations returns all organizations for a user
func (uc *OrganizationUseCase) ListUserOrganizations(ctx context.Context, userID string) ([]OrganizationSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	orgs, err := uc.orgService.GetUserOrganizations(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user organizations")
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	result := make([]OrganizationSummary, len(orgs))
	for i, org := range orgs {
		result[i] = OrganizationSummary{
			ID:   org.ID.String(),
			Name: org.Name,
			Role: "member", // TODO: Get actual role from organization_members
		}
	}

	return result, nil
}

// CreateOrganization creates a new organization with the user as owner
func (uc *OrganizationUseCase) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*services.Organization, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("organization name is required")
	}

	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	log.Info().
		Str("user_id", req.UserID).
		Str("org_name", req.Name).
		Msg("Creating organization")

	org, err := uc.orgService.CreateOrganizationWithOwner(ctx, req.UserID, req.Name)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", req.UserID).
			Str("org_name", req.Name).
			Msg("Failed to create organization")
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	log.Info().
		Str("org_id", org.ID.String()).
		Str("org_name", org.Name).
		Str("user_id", req.UserID).
		Msg("Organization created successfully")

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (uc *OrganizationUseCase) GetOrganization(ctx context.Context, orgID string) (*services.Organization, error) {
	if orgID == "" {
		return nil, fmt.Errorf("organization ID is required")
	}

	org, err := uc.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("Failed to get organization")
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	return org, nil
}

// SwitchOrganization validates and switches user's current organization
func (uc *OrganizationUseCase) SwitchOrganization(ctx context.Context, userID, orgID string) error {
	if userID == "" || orgID == "" {
		return fmt.Errorf("user ID and organization ID are required")
	}

	// Verify user has access to this organization
	orgs, err := uc.orgService.GetUserOrganizations(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to verify organization access")
		return fmt.Errorf("failed to verify organization access: %w", err)
	}

	// Check if user has access to the requested organization
	hasAccess := false
	for _, org := range orgs {
		if org.ID.String() == orgID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		log.Warn().
			Str("user_id", userID).
			Str("org_id", orgID).
			Msg("User attempted to access unauthorized organization")
		return fmt.Errorf("access denied to organization")
	}

	// In a real implementation, you might:
	// 1. Update user's session to set current organization
	// 2. Return a new token with organization context
	// 3. Set a cookie with the current organization
	log.Info().
		Str("user_id", userID).
		Str("org_id", orgID).
		Msg("Organization switch validated")

	return nil
}

// MemberRequest represents a member management request
type MemberRequest struct {
	Email string
	Role  string
}

// GetOrganizationMembers retrieves all members of an organization
func (uc *OrganizationUseCase) GetOrganizationMembers(ctx context.Context, orgID string) ([]services.OrganizationMember, error) {
	if orgID == "" {
		return nil, fmt.Errorf("organization ID is required")
	}

	members, err := uc.orgService.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("Failed to get organization members")
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	return members, nil
}

// InviteMember invites a user to join an organization
func (uc *OrganizationUseCase) InviteMember(ctx context.Context, orgID string, req MemberRequest) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}

	if req.Email == "" || req.Role == "" {
		return fmt.Errorf("email and role are required")
	}

	log.Info().
		Str("org_id", orgID).
		Str("email", req.Email).
		Str("role", req.Role).
		Msg("Inviting member to organization")

	err := uc.orgService.InviteMember(ctx, orgID, req.Email, req.Role)
	if err != nil {
		log.Error().
			Err(err).
			Str("org_id", orgID).
			Str("email", req.Email).
			Str("role", req.Role).
			Msg("Failed to invite member")
		return fmt.Errorf("failed to invite member: %w", err)
	}

	log.Info().
		Str("org_id", orgID).
		Str("email", req.Email).
		Str("role", req.Role).
		Msg("Member invited successfully")

	return nil
}

// UpdateMemberRole updates a member's role in an organization
func (uc *OrganizationUseCase) UpdateMemberRole(ctx context.Context, orgID, userID, role string) error {
	if orgID == "" || userID == "" || role == "" {
		return fmt.Errorf("organization ID, user ID, and role are required")
	}

	log.Info().
		Str("org_id", orgID).
		Str("user_id", userID).
		Str("role", role).
		Msg("Updating member role")

	err := uc.orgService.UpdateMemberRole(ctx, orgID, userID, role)
	if err != nil {
		log.Error().
			Err(err).
			Str("org_id", orgID).
			Str("user_id", userID).
			Str("role", role).
			Msg("Failed to update member role")
		return fmt.Errorf("failed to update member role: %w", err)
	}

	log.Info().
		Str("org_id", orgID).
		Str("user_id", userID).
		Str("role", role).
		Msg("Member role updated successfully")

	return nil
}

// RemoveMember removes a member from an organization
func (uc *OrganizationUseCase) RemoveMember(ctx context.Context, orgID, userID string) error {
	if orgID == "" || userID == "" {
		return fmt.Errorf("organization ID and user ID are required")
	}

	log.Info().
		Str("org_id", orgID).
		Str("user_id", userID).
		Msg("Removing member from organization")

	err := uc.orgService.RemoveMember(ctx, orgID, userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("org_id", orgID).
			Str("user_id", userID).
			Msg("Failed to remove member")
		return fmt.Errorf("failed to remove member: %w", err)
	}

	log.Info().
		Str("org_id", orgID).
		Str("user_id", userID).
		Msg("Member removed successfully")

	return nil
}