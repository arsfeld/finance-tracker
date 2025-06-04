package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/romsar/gonertia/v2"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/services"
	"finance_tracker/src/internal/usecases"
)

// OrganizationHandlers handles organization endpoints with Inertia.js
type OrganizationHandlers struct {
	client     *config.Client
	inertia    *config.InertiaConfig
	orgUseCase *usecases.OrganizationUseCase
	orgService *services.OrganizationService
}

// NewInertiaOrganizationHandlers creates a new instance of organization handlers with Inertia
func NewInertiaOrganizationHandlers(client *config.Client, inertia *config.InertiaConfig) *OrganizationHandlers {
	var orgService *services.OrganizationService
	if client != nil && client.Service != nil {
		orgService = services.NewOrganizationService(client.Service)
	} else {
		log.Warn().Msg("Using mock organization service - Supabase client not available")
		// For now, we'll skip the mock service since it requires refactoring
		return &OrganizationHandlers{
			client:  client,
			inertia: inertia,
		}
	}

	orgUseCase := usecases.NewOrganizationUseCase(client)

	return &OrganizationHandlers{
		client:     client,
		inertia:    inertia,
		orgUseCase: orgUseCase,
		orgService: orgService,
	}
}

// HandleListOrganizations returns user's organizations via Inertia
func (h *OrganizationHandlers) HandleListOrganizations(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(string)
	if !ok {
		h.inertia.ShareProp("error", "User not found in context")
		h.inertia.Back(w, r, http.StatusUnauthorized)
		return
	}

	organizations, err := h.orgService.GetUserOrganizations(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user organizations")
		h.inertia.ShareProp("error", "Failed to load organizations")
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Return as Inertia props (this would typically be rendered in a page component)
	h.inertia.Render(w, r, "Organizations/List", gonertia.Props{
		"organizations": organizations,
	})
}

// HandleCreateOrganization creates a new organization via Inertia
func (h *OrganizationHandlers) HandleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(string)
	if !ok {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"User not found in context"},
		})
		h.inertia.Back(w, r, http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid request format"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate
	errors := make(gonertia.ValidationErrors)
	if req.Name == "" {
		errors["name"] = []string{"Organization name is required"}
	}

	if len(errors) > 0 {
		h.inertia.ShareProp("errors", errors)
		h.inertia.Back(w, r, http.StatusUnprocessableEntity)
		return
	}

	org, err := h.orgUseCase.CreateOrganization(r.Context(), usecases.CreateOrganizationRequest{
		UserID: userID,
		Name:   req.Name,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create organization")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Failed to create organization"},
		})
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Flash success and redirect
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Organization created successfully",
	})
	h.inertia.Location(w, r, "/organizations/"+org.ID.String(), http.StatusSeeOther)
}

// HandleGetOrganization returns organization details via Inertia
func (h *OrganizationHandlers) HandleGetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		h.inertia.ShareProp("error", "Organization ID is required")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(orgID); err != nil {
		h.inertia.ShareProp("error", "Invalid organization ID")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	org, err := h.orgService.GetOrganization(r.Context(), orgID)
	if err != nil {
		log.Error().Err(err).Str("orgID", orgID).Msg("Failed to get organization")
		h.inertia.ShareProp("error", "Organization not found")
		h.inertia.Back(w, r, http.StatusNotFound)
		return
	}

	// Render organization detail page
	h.inertia.Render(w, r, "Organizations/Detail", gonertia.Props{
		"organization": org,
	})
}

// HandleSwitchOrganization switches the current organization via Inertia
func (h *OrganizationHandlers) HandleSwitchOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		h.inertia.ShareProp("error", "Organization ID is required")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(orgID); err != nil {
		h.inertia.ShareProp("error", "Invalid organization ID")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Store in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "current_org",
		Value:    orgID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Flash success and redirect
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Organization switched successfully",
	})
	h.inertia.Location(w, r, "/dashboard", http.StatusSeeOther)
}

// Member management handlers
func (h *OrganizationHandlers) HandleGetMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		h.inertia.ShareProp("error", "Organization ID is required")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	members, err := h.orgService.GetOrganizationMembers(r.Context(), orgID)
	if err != nil {
		log.Error().Err(err).Str("orgID", orgID).Msg("Failed to get organization members")
		h.inertia.ShareProp("error", "Failed to load members")
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Render members list
	h.inertia.Render(w, r, "Organizations/Members", gonertia.Props{
		"organizationId": orgID,
		"members":        members,
	})
}

// HandleInviteMember invites a new member via Inertia
func (h *OrganizationHandlers) HandleInviteMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Organization ID is required"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid request format"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	// Validate
	errors := make(gonertia.ValidationErrors)
	if req.Email == "" {
		errors["email"] = []string{"Email is required"}
	}
	if req.Role == "" {
		req.Role = "member"
	}

	if len(errors) > 0 {
		h.inertia.ShareProp("errors", errors)
		h.inertia.Back(w, r, http.StatusUnprocessableEntity)
		return
	}

	err := h.orgUseCase.InviteMember(r.Context(), orgID, usecases.MemberRequest{
		Email: req.Email,
		Role:  req.Role,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to invite member")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Failed to invite member"},
		})
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Flash success
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Member invited successfully",
	})
	h.inertia.Back(w, r, http.StatusOK)
}

// HandleUpdateMemberRole updates a member's role via Inertia
func (h *OrganizationHandlers) HandleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	userID := chi.URLParam(r, "userID")

	if orgID == "" || userID == "" {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Organization ID and User ID are required"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Invalid request format"},
		})
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"role": []string{"Role is required"},
		})
		h.inertia.Back(w, r, http.StatusUnprocessableEntity)
		return
	}

	err := h.orgService.UpdateMemberRole(r.Context(), orgID, userID, req.Role)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update member role")
		h.inertia.ShareProp("errors", gonertia.ValidationErrors{
			"_": []string{"Failed to update member role"},
		})
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Flash success
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Member role updated successfully",
	})
	h.inertia.Back(w, r, http.StatusOK)
}

// HandleRemoveMember removes a member via Inertia
func (h *OrganizationHandlers) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	userID := chi.URLParam(r, "userID")

	if orgID == "" || userID == "" {
		h.inertia.ShareProp("error", "Organization ID and User ID are required")
		h.inertia.Back(w, r, http.StatusBadRequest)
		return
	}

	err := h.orgService.RemoveMember(r.Context(), orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove member")
		h.inertia.ShareProp("error", "Failed to remove member")
		h.inertia.Back(w, r, http.StatusInternalServerError)
		return
	}

	// Flash success
	h.inertia.ShareProp("flash", map[string]string{
		"success": "Member removed successfully",
	})
	h.inertia.Back(w, r, http.StatusOK)
}