package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/romsar/gonertia/v2"
	"github.com/rs/zerolog/log"
	"github.com/supabase-community/gotrue-go/types"

	"finance_tracker/src/internal/config"
)

// AuthHandlers handles authentication endpoints with Inertia.js
type AuthHandlers struct {
	client  *config.Client
	inertia *config.InertiaConfig
}

// NewInertiaAuthHandlers creates a new instance of auth handlers with Inertia
func NewInertiaAuthHandlers(client *config.Client, inertia *config.InertiaConfig) *AuthHandlers {
	return &AuthHandlers{
		client:  client,
		inertia: inertia,
	}
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	OrganizationName string `json:"organizationName"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandleRegister handles user registration via Inertia
func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if h.client == nil || h.client.Anon == nil {
		log.Error().Msg("Supabase client not initialized")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Authentication service unavailable"},
		})
		h.inertia.Back(w, r, http.StatusServiceUnavailable)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid request format"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate request
	errors := make(gonertia.ValidationErrors)
	if req.Email == "" {
		errors["email"] = []string{"Email is required"}
	}
	if req.Password == "" {
		errors["password"] = []string{"Password is required"}
	}
	if len(req.Password) < 6 {
		errors["password"] = []string{"Password must be at least 6 characters"}
	}
	if req.OrganizationName == "" {
		errors["organizationName"] = []string{"Organization name is required"}
	}

	if len(errors) > 0 {
		h.inertia.ShareProp("errors", errors)
		h.inertia.Back(w, r, http.StatusUnprocessableEntity)
		return
	}

	// Sign up user
	user, err := h.client.Anon.Auth.Signup(types.SignupRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to sign up user")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"email": []string{"This email may already be registered"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Store auth token in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    user.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Flash success message and redirect
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Account created successfully!",
	})
	h.inertia.Location(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogin handles user login via Inertia
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if h.client == nil || h.client.Anon == nil {
		log.Error().Msg("Supabase client not initialized")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Authentication service unavailable"},
		})
		h.inertia.Back(w, r, http.StatusServiceUnavailable)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid request format"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate request
	errors := make(gonertia.ValidationErrors)
	if req.Email == "" {
		errors["email"] = []string{"Email is required"}
	}
	if req.Password == "" {
		errors["password"] = []string{"Password is required"}
	}

	if len(errors) > 0 {
		h.inertia.ShareProp("errors", errors)
		h.inertia.Back(w, r, http.StatusUnprocessableEntity)
		return
	}

	// Sign in user
	user, err := h.client.Anon.SignInWithEmailPassword(req.Email, req.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sign in user")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid email or password"},
		})
		h.inertia.Back(w, r, http.StatusUnauthorized)
		return
	}

	// Store auth token in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    user.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Flash success message and redirect
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Welcome back!",
	})
	h.inertia.Location(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogout handles user logout via Inertia
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// TODO: Implement proper Supabase sign out when API is available
	// For now, just clearing the cookie is sufficient

	// Flash success message and redirect
	h.inertia.ShareProp("flash", map[string]string{
		"success": "You have been logged out",
	})
	h.inertia.Location(w, r, "/", http.StatusSeeOther)
}

// extractTokenFromCookie extracts the auth token from cookie
func extractTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}