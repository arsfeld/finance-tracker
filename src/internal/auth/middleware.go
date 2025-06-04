package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	supa "github.com/supabase-community/supabase-go"
	"github.com/supabase-community/gotrue-go/types"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
	OrgContextKey  contextKey = "organization"
)

// Middleware handles JWT validation for protected routes
type Middleware struct {
	supabase        *supa.Client
	serviceSupabase *supa.Client // For admin operations that bypass RLS
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(supabase *supa.Client, serviceSupabase *supa.Client) *Middleware {
	return &Middleware{
		supabase:        supabase,
		serviceSupabase: serviceSupabase,
	}
}

// RequireAuth validates JWT token and adds user to context
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("RequireAuth middleware started")

		// Extract token from Authorization header or cookie
		token := extractToken(r)
		if token == "" {
			log.Error().
				Str("path", r.URL.Path).
				Msg("No token found in request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().
			Str("path", r.URL.Path).
			Msg("Token extracted successfully")

		// Validate JWT token with Supabase by creating an authenticated client
		authClient := m.supabase.Auth.WithToken(token)
		user, err := authClient.GetUser()
		if err != nil {
			log.Error().
				Err(err).
				Str("path", r.URL.Path).
				Msg("Failed to validate JWT token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user == nil {
			log.Error().
				Str("path", r.URL.Path).
				Msg("No user found from token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		log.Debug().
			Str("user_id", user.ID.String()).
			Str("email", user.Email).
			Str("path", r.URL.Path).
			Msg("User authenticated via JWT")

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireOrganization validates organization access or provides default
func (m *Middleware) RequireOrganization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("RequireOrganization middleware started")

		// Get user from context
		user := GetUser(r.Context())
		if user == nil {
			log.Error().
				Str("path", r.URL.Path).
				Msg("No user found in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().
			Str("user_id", user.ID.String()).
			Str("path", r.URL.Path).
			Msg("User found in context")

		// Try to get organization ID from header first
		orgID := r.Header.Get("X-Organization-ID")
		var orgUUID uuid.UUID
		var err error

		if orgID != "" {
			log.Info().Str("org_id_header", orgID).Msg("Using organization ID from header")
			// Use provided organization ID
			orgUUID, err = uuid.Parse(orgID)
			if err != nil {
				log.Error().Err(err).Str("org_id", orgID).Msg("Invalid organization ID format")
				http.Error(w, "Invalid organization ID", http.StatusBadRequest)
				return
			}
		} else {
			log.Debug().
				Str("user_id", user.ID.String()).
				Msg("No org header, querying database for user organizations")

			// Get user's first organization from database
			var result []struct {
				OrganizationID uuid.UUID `json:"organization_id"`
				Role           string    `json:"role"`
			}
			
			// Use service client to bypass RLS for admin queries
			client := m.supabase
			if m.serviceSupabase != nil {
				client = m.serviceSupabase
			}
			
			_, err := client.From("organization_members").
				Select("organization_id,role", "", false).
				Eq("user_id", user.ID.String()).
				ExecuteTo(&result)

			if err != nil {
				log.Error().
					Err(err).
					Str("user_id", user.ID.String()).
					Msg("Failed to get user organizations")
				// Set empty UUID - user has no organizations yet
				orgUUID = uuid.Nil
			} else if len(result) > 0 {
				orgUUID = result[0].OrganizationID
				log.Info().
					Str("user_id", user.ID.String()).
					Str("org_id", orgUUID.String()).
					Str("role", result[0].Role).
					Msg("Using user's first organization")
			} else {
				// User has no organizations
				orgUUID = uuid.Nil
				log.Info().
					Str("user_id", user.ID.String()).
					Msg("User has no organizations")
			}
		}

		log.Debug().
			Str("user_id", user.ID.String()).
			Str("org_id", orgUUID.String()).
			Str("path", r.URL.Path).
			Msg("Organization context set")

		// Add organization ID to context (may be uuid.Nil if user has no orgs)
		ctx := context.WithValue(r.Context(), OrgContextKey, orgUUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole checks if user has required role in organization
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user and organization from context
			user := GetUser(r.Context())
			orgID := GetOrganization(r.Context())
			
			if user == nil || orgID == uuid.Nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Query user's role in organization using Supabase
			var result struct {
				Role string `json:"role"`
			}
			
			// Use service client to bypass RLS for admin queries
			client := m.supabase
			if m.serviceSupabase != nil {
				client = m.serviceSupabase
			}
			
			_, err := client.From("organization_members").
				Select("role", "", false).
				Eq("user_id", user.ID.String()).
				Eq("organization_id", orgID.String()).
				Single().
				ExecuteTo(&result)

			if err != nil {
				log.Error().Err(err).Msg("Failed to get user role")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Check if user has required role
			hasRole := false
			for _, role := range roles {
				if result.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken gets the JWT token from the Authorization header
func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:7]) == "BEARER " {
		return bearer[7:]
	}
	
	// Also check for token in cookie (for web UI)
	// Try auth_token first (set by our auth handlers)
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}
	
	// Fallback to sb-access-token (Supabase default)
	if cookie, err := r.Cookie("sb-access-token"); err == nil {
		return cookie.Value
	}
	
	return ""
}

// GetUser retrieves user from context
func GetUser(ctx context.Context) *types.User {
	// First try types.User
	if user, ok := ctx.Value(UserContextKey).(*types.User); ok {
		return user
	}
	// Then try types.UserResponse and convert
	if userResp, ok := ctx.Value(UserContextKey).(*types.UserResponse); ok {
		// Convert UserResponse to User
		return &types.User{
			ID:               userResp.ID,
			Email:            userResp.Email,
			Phone:            userResp.Phone,
			EmailConfirmedAt: userResp.EmailConfirmedAt,
			CreatedAt:        userResp.CreatedAt,
			UpdatedAt:        userResp.UpdatedAt,
			LastSignInAt:     userResp.LastSignInAt,
			Role:             userResp.Role,
			UserMetadata:     userResp.UserMetadata,
			AppMetadata:      userResp.AppMetadata,
		}
	}
	return nil
}

// GetOrganization retrieves organization ID from context
func GetOrganization(ctx context.Context) uuid.UUID {
	if orgID, ok := ctx.Value(OrgContextKey).(uuid.UUID); ok {
		return orgID
	}
	return uuid.Nil
}