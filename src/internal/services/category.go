package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
)

// CategoryService handles category data operations
type CategoryService struct {
	client *config.Client
}

// NewCategoryService creates a new category service
func NewCategoryService(client *config.Client) *CategoryService {
	return &CategoryService{
		client: client,
	}
}

// GetCategoriesByOrganization returns all categories for an organization
func (s *CategoryService) GetCategoriesByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*models.Category, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var dbResults []struct {
		ID             int       `json:"id"`
		OrganizationID uuid.UUID `json:"organization_id"`
		Name           string    `json:"name"`
		ParentID       *int      `json:"parent_id"`
		Color          *string   `json:"color"`
		Icon           *string   `json:"icon"`
		CreatedAt      string    `json:"created_at"`
	}

	_, err := s.client.Service.
		From("categories").
		Select("*", "", false).
		Eq("organization_id", organizationID.String()).
		Order("name", nil).
		ExecuteTo(&dbResults)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	// Convert to model format
	categories := make([]*models.Category, len(dbResults))
	for i, result := range dbResults {
		categories[i] = &models.Category{
			ID:             result.ID,
			OrganizationID: result.OrganizationID,
			Name:           result.Name,
			ParentID:       result.ParentID,
			Color:          result.Color,
			Icon:           result.Icon,
		}
	}

	return categories, nil
}

// GetOrCreateCategory gets a category by name or creates it if it doesn't exist
func (s *CategoryService) GetOrCreateCategory(ctx context.Context, organizationID uuid.UUID, name string) (*models.Category, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	// First try to get the category
	var dbResult struct {
		ID             int       `json:"id"`
		OrganizationID uuid.UUID `json:"organization_id"`
		Name           string    `json:"name"`
		ParentID       *int      `json:"parent_id"`
		Color          *string   `json:"color"`
		Icon           *string   `json:"icon"`
	}

	_, err := s.client.Service.
		From("categories").
		Select("*", "", false).
		Eq("organization_id", organizationID.String()).
		Eq("name", name).
		Single().
		ExecuteTo(&dbResult)

	if err == nil {
		// Category exists
		return &models.Category{
			ID:             dbResult.ID,
			OrganizationID: dbResult.OrganizationID,
			Name:           dbResult.Name,
			ParentID:       dbResult.ParentID,
			Color:          dbResult.Color,
			Icon:           dbResult.Icon,
		}, nil
	}

	// Category doesn't exist, create it
	var newCategory struct {
		ID             int       `json:"id"`
		OrganizationID uuid.UUID `json:"organization_id"`
		Name           string    `json:"name"`
		ParentID       *int      `json:"parent_id"`
		Color          *string   `json:"color"`
		Icon           *string   `json:"icon"`
	}

	_, err = s.client.Service.
		From("categories").
		Insert(map[string]interface{}{
			"organization_id": organizationID.String(),
			"name":           name,
		}, false, "", "", "").
		Single().
		ExecuteTo(&newCategory)

	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &models.Category{
		ID:             newCategory.ID,
		OrganizationID: newCategory.OrganizationID,
		Name:           newCategory.Name,
		ParentID:       newCategory.ParentID,
		Color:          newCategory.Color,
		Icon:           newCategory.Icon,
	}, nil
}

// CreateCategory creates a new category
func (s *CategoryService) CreateCategory(ctx context.Context, category *models.Category) (*models.Category, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var newCategory struct {
		ID             int       `json:"id"`
		OrganizationID uuid.UUID `json:"organization_id"`
		Name           string    `json:"name"`
		ParentID       *int      `json:"parent_id"`
		Color          *string   `json:"color"`
		Icon           *string   `json:"icon"`
	}

	insertData := map[string]interface{}{
		"organization_id": category.OrganizationID.String(),
		"name":           category.Name,
	}

	if category.ParentID != nil {
		insertData["parent_id"] = *category.ParentID
	}
	if category.Color != nil {
		insertData["color"] = *category.Color
	}
	if category.Icon != nil {
		insertData["icon"] = *category.Icon
	}

	_, err := s.client.Service.
		From("categories").
		Insert(insertData, false, "", "", "").
		Single().
		ExecuteTo(&newCategory)

	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	category.ID = newCategory.ID
	return category, nil
}

// UpdateCategory updates an existing category
func (s *CategoryService) UpdateCategory(ctx context.Context, category *models.Category) error {
	if s.client == nil || s.client.Service == nil {
		return fmt.Errorf("database client not available")
	}

	updateData := map[string]interface{}{
		"name": category.Name,
	}

	if category.ParentID != nil {
		updateData["parent_id"] = *category.ParentID
	}
	if category.Color != nil {
		updateData["color"] = *category.Color
	}
	if category.Icon != nil {
		updateData["icon"] = *category.Icon
	}

	_, _, err := s.client.Service.
		From("categories").
		Update(updateData, "", "").
		Eq("id", fmt.Sprintf("%d", category.ID)).
		Eq("organization_id", category.OrganizationID.String()).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}

// DeleteCategory deletes a category
func (s *CategoryService) DeleteCategory(ctx context.Context, organizationID uuid.UUID, categoryID int) error {
	if s.client == nil || s.client.Service == nil {
		return fmt.Errorf("database client not available")
	}

	_, _, err := s.client.Service.
		From("categories").
		Delete("", "").
		Eq("id", fmt.Sprintf("%d", categoryID)).
		Eq("organization_id", organizationID.String()).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}