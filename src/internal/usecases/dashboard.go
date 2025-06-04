package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"finance_tracker/src/internal/models"
)

// AnalyticsService defines the interface for analytics operations
type AnalyticsService interface {
	GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error)
	GetAccountBalances(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]float64, error)
	GetMonthlySpending(ctx context.Context, orgID uuid.UUID, month time.Time) (float64, error)
	GetTransactionCount(ctx context.Context, orgID uuid.UUID) (int, error)
	GetActiveAccountCount(ctx context.Context, orgID uuid.UUID) (int, error)
}

// DashboardUseCase handles dashboard business logic
type DashboardUseCase struct {
	analyticsService AnalyticsService
}

// NewDashboardUseCase creates a new dashboard use case
func NewDashboardUseCase(analyticsService AnalyticsService) *DashboardUseCase {
	return &DashboardUseCase{
		analyticsService: analyticsService,
	}
}

// GetDashboardStats returns dashboard statistics for an organization
func (uc *DashboardUseCase) GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error) {
	return uc.analyticsService.GetDashboardStats(ctx, orgID)
}