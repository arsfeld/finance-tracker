package handlers

import (
	"net/http"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/usecases"
)

// DashboardHandlers handles dashboard-related HTTP requests
type DashboardHandlers struct {
	dashboardUseCase *usecases.DashboardUseCase
}

// NewDashboardHandlers creates new dashboard handlers
func NewDashboardHandlers(dashboardUseCase *usecases.DashboardUseCase) *DashboardHandlers {
	return &DashboardHandlers{
		dashboardUseCase: dashboardUseCase,
	}
}

// HandleGetDashboardStats returns dashboard statistics
func (h *DashboardHandlers) HandleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	stats, err := h.dashboardUseCase.GetDashboardStats(r.Context(), orgID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get dashboard stats", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, stats)
}
