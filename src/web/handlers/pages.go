package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romsar/gonertia/v2"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/config"
)

// PageHandlers handles page rendering using Inertia.js
type PageHandlers struct {
	inertia *config.InertiaConfig
}

// NewInertiaPageHandlers creates a new instance of page handlers with Inertia
func NewInertiaPageHandlers(inertia *config.InertiaConfig) *PageHandlers {
	return &PageHandlers{
		inertia: inertia,
	}
}

// HomePage renders the home page
func (h *PageHandlers) HomePage(w http.ResponseWriter, r *http.Request) {
	err := h.inertia.Render(w, r, "Home/Index", gonertia.Props{
		"title": "Welcome to WalletMind",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to render home page")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// LoginPage renders the login page
func (h *PageHandlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.inertia.Render(w, r, "Auth/Login", gonertia.Props{
		"title": "Login",
	})
}

// RegisterPage renders the register page
func (h *PageHandlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.inertia.Render(w, r, "Auth/Register", gonertia.Props{
		"title": "Create Account",
	})
}

// DashboardPage renders the dashboard
func (h *PageHandlers) DashboardPage(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Dashboard/Index", gonertia.Props{
		"title": "Dashboard",
		"user":  user,
		"organization": org,
	})
}

// TransactionsPage renders the transactions page
func (h *PageHandlers) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Transactions/Index", gonertia.Props{
		"title": "Transactions",
		"user":  user,
		"organization": org,
	})
}

// AccountsPage renders the accounts page
func (h *PageHandlers) AccountsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Accounts/Index", gonertia.Props{
		"title": "Accounts",
		"user":  user,
		"organization": org,
	})
}

// AccountDetailPage renders the account detail page
func (h *PageHandlers) AccountDetailPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})
	accountID := chi.URLParam(r, "accountID")

	h.inertia.Render(w, r, "Accounts/Detail", gonertia.Props{
		"title":     "Account Details",
		"user":      user,
		"organization": org,
		"accountId": accountID,
	})
}

// AnalyticsPage renders the analytics page
func (h *PageHandlers) AnalyticsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Analytics/Index", gonertia.Props{
		"title": "Analytics",
		"user":  user,
		"organization": org,
	})
}

// SettingsPage renders the settings page
func (h *PageHandlers) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Settings/Index", gonertia.Props{
		"title": "Settings",
		"user":  user,
		"organization": org,
	})
}

// ConnectionsPage renders the connections management page
func (h *PageHandlers) ConnectionsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(auth.UserContextKey).(map[string]interface{})
	org, _ := r.Context().Value(auth.OrgContextKey).(map[string]interface{})

	h.inertia.Render(w, r, "Settings/Connections", gonertia.Props{
		"title": "Account Connections",
		"user":  user,
		"organization": org,
	})
}