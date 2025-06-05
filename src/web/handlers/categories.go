package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/models"
	"finance_tracker/src/internal/services"
)

// CategoryHandler handles category-related API endpoints
type CategoryHandler struct {
	categoryService *services.CategoryService
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(categoryService *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// CategoryRequest represents a request to create or update a category
type CategoryRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Color       string  `json:"color"`
	ParentID    *int    `json:"parent_id,omitempty"`
}

// CategoryResponse represents a category in API responses
type CategoryResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Color       *string `json:"color"`
	ParentID    *int    `json:"parent_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	IsDefault   bool    `json:"is_default"`
}

// RegisterRoutes registers category routes
func (h *CategoryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/categories", func(r chi.Router) {
		r.Get("/", h.ListCategories)
		r.Post("/", h.CreateCategory)
		r.Get("/{categoryID}", h.GetCategory)
		r.Put("/{categoryID}", h.UpdateCategory)
		r.Delete("/{categoryID}", h.DeleteCategory)
	})
}

// ListCategories returns all categories for the organization
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	categories, err := h.categoryService.GetCategoriesByOrganization(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get categories: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]CategoryResponse, len(categories))
	for i, cat := range categories {
		responses[i] = CategoryResponse{
			ID:          fmt.Sprintf("%d", cat.ID),
			Name:        cat.Name,
			Description: cat.Name, // Using name as description for now
			Color:       cat.Color,
			ParentID:    cat.ParentID,
			CreatedAt:   "2025-01-01T00:00:00Z", // Default for now
			IsDefault:   false,                  // We'll implement this logic later
		}
	}

	respondWithJSON(w, r, http.StatusOK, responses)
}

// CreateCategory creates a new category
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	// Create category model
	category := &models.Category{
		OrganizationID: orgID,
		Name:           req.Name,
		ParentID:       req.ParentID,
	}

	if req.Color != "" {
		category.Color = &req.Color
	}

	// Create in database
	createdCategory, err := h.categoryService.CreateCategory(ctx, category)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create category: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := CategoryResponse{
		ID:          fmt.Sprintf("%d", createdCategory.ID),
		Name:        createdCategory.Name,
		Description: createdCategory.Name,
		Color:       createdCategory.Color,
		ParentID:    createdCategory.ParentID,
		CreatedAt:   "2025-01-01T00:00:00Z", // Default for now
		IsDefault:   false,
	}

	respondWithJSON(w, r, http.StatusCreated, response)
}

// GetCategory returns a specific category
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	categoryIDStr := chi.URLParam(r, "categoryID")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// For now, we'll fetch all categories and filter
	categories, err := h.categoryService.GetCategoriesByOrganization(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get categories: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the specific category
	var foundCategory *models.Category
	for _, cat := range categories {
		if cat.ID == categoryID {
			foundCategory = cat
			break
		}
	}

	if foundCategory == nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	// Convert to response format
	response := CategoryResponse{
		ID:          fmt.Sprintf("%d", foundCategory.ID),
		Name:        foundCategory.Name,
		Description: foundCategory.Name,
		Color:       foundCategory.Color,
		ParentID:    foundCategory.ParentID,
		CreatedAt:   "2025-01-01T00:00:00Z",
		IsDefault:   false,
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// UpdateCategory updates an existing category
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	categoryIDStr := chi.URLParam(r, "categoryID")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	// Create category model for update
	category := &models.Category{
		ID:             categoryID,
		OrganizationID: orgID,
		Name:           req.Name,
		ParentID:       req.ParentID,
	}

	if req.Color != "" {
		category.Color = &req.Color
	}

	// Update in database
	err = h.categoryService.UpdateCategory(ctx, category)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update category: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := CategoryResponse{
		ID:          fmt.Sprintf("%d", category.ID),
		Name:        category.Name,
		Description: category.Name,
		Color:       category.Color,
		ParentID:    category.ParentID,
		CreatedAt:   "2025-01-01T00:00:00Z",
		IsDefault:   false,
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// DeleteCategory deletes a category
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	categoryIDStr := chi.URLParam(r, "categoryID")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Delete from database
	err = h.categoryService.DeleteCategory(ctx, orgID, categoryID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete category: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

