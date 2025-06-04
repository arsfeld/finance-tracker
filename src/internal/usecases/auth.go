package usecases

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/supabase-community/gotrue-go/types"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/services"
)

// AuthUseCase handles authentication-related business logic
type AuthUseCase struct {
	client     *config.Client
	orgService *services.OrganizationService
}

// NewAuthUseCase creates a new authentication use case
func NewAuthUseCase(client *config.Client) *AuthUseCase {
	return &AuthUseCase{
		client:     client,
		orgService: services.NewOrganizationService(client.Service),
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email            string
	Password         string
	OrganizationName string
}

// AuthResult represents the result of authentication operations
type AuthResult struct {
	AccessToken    string
	RefreshToken   string
	UserID         string
	Email          string
	OrganizationID string
	Organizations  []OrganizationSummary
}

// OrganizationSummary represents a summary of an organization
type OrganizationSummary struct {
	ID   string
	Name string
	Role string
}

// Register creates a new user account
func (uc *AuthUseCase) Register(ctx context.Context, req RegisterRequest) (*AuthResult, error) {
	if uc.client == nil || uc.client.Anon == nil {
		return nil, fmt.Errorf("supabase client not available")
	}

	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	// Set default organization name if not provided
	if req.OrganizationName == "" {
		req.OrganizationName = req.Email + "'s Organization"
	}

	// Create user with Supabase Auth
	signupReq := types.SignupRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	log.Info().
		Str("email", req.Email).
		Str("org_name", req.OrganizationName).
		Msg("Creating user account")

	signupResp, err := uc.client.Anon.Auth.Signup(signupReq)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to create user")
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	log.Info().
		Str("user_id", signupResp.User.ID.String()).
		Str("email", signupResp.User.Email).
		Bool("email_confirmed", signupResp.User.EmailConfirmedAt != nil).
		Msg("User created successfully")

	// Organization will be created automatically by database trigger
	result := &AuthResult{
		AccessToken:  signupResp.Session.AccessToken,
		RefreshToken: signupResp.Session.RefreshToken,
		UserID:       signupResp.User.ID.String(),
		Email:        signupResp.User.Email,
	}

	// Try to get organizations (might be empty if trigger hasn't run yet)
	orgs, err := uc.orgService.GetUserOrganizations(ctx, signupResp.User.ID.String())
	if err == nil && len(orgs) > 0 {
		result.Organizations = make([]OrganizationSummary, len(orgs))
		for i, org := range orgs {
			result.Organizations[i] = OrganizationSummary{
				ID:   org.ID.String(),
				Name: org.Name,
				Role: "owner", // First org is typically owned by creator
			}
		}
		if len(orgs) > 0 {
			result.OrganizationID = orgs[0].ID.String()
		}
	}

	log.Info().
		Str("user_id", result.UserID).
		Int("org_count", len(result.Organizations)).
		Msg("Registration completed")

	return result, nil
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string
	Password string
}

// Login authenticates a user
func (uc *AuthUseCase) Login(ctx context.Context, req LoginRequest) (*AuthResult, error) {
	if uc.client == nil || uc.client.Anon == nil {
		return nil, fmt.Errorf("supabase client not available")
	}

	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	log.Info().Str("email", req.Email).Msg("Attempting user authentication")

	// Authenticate with Supabase
	session, err := uc.client.Anon.SignInWithEmailPassword(req.Email, req.Password)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Authentication failed")
		return nil, fmt.Errorf("invalid credentials")
	}

	log.Info().
		Str("user_id", session.User.ID.String()).
		Str("email", session.User.Email).
		Bool("email_confirmed", session.User.EmailConfirmedAt != nil).
		Msg("User authenticated successfully")

	// Get user's organizations
	orgs, err := uc.orgService.GetUserOrganizations(ctx, session.User.ID.String())
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", session.User.ID.String()).
			Msg("Failed to get user organizations")
		// Continue without organizations
	}

	result := &AuthResult{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		UserID:       session.User.ID.String(),
		Email:        session.User.Email,
	}

	// Add organizations to response
	if len(orgs) > 0 {
		result.Organizations = make([]OrganizationSummary, len(orgs))
		for i, org := range orgs {
			result.Organizations[i] = OrganizationSummary{
				ID:   org.ID.String(),
				Name: org.Name,
				Role: "member", // TODO: Get actual role from organization_members
			}
		}
		// Set the first organization as default
		result.OrganizationID = orgs[0].ID.String()
	}

	log.Info().
		Str("user_id", result.UserID).
		Int("org_count", len(result.Organizations)).
		Msg("Login completed successfully")

	return result, nil
}

// Logout signs out a user
func (uc *AuthUseCase) Logout(ctx context.Context, token string) error {
	// TODO: Implement proper Supabase sign out
	// For now, just log the logout
	log.Info().Msg("User logged out")
	return nil
}