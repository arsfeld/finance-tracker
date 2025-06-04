package services

import (
	"context"

	"github.com/google/uuid"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
)

// ConnectionService handles connection data operations
type ConnectionService struct {
	client *config.Client
}

// NewConnectionService creates a new connection service
func NewConnectionService(client *config.Client) *ConnectionService {
	return &ConnectionService{
		client: client,
	}
}

// ListConnections returns all connections for an organization
func (s *ConnectionService) ListConnections(ctx context.Context, orgID uuid.UUID) ([]models.Connection, error) {
	if s.client.Service == nil {
		return []models.Connection{}, nil
	}

	var connections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Select("id, name, provider_type, last_sync, sync_status, error_message, settings, created_at", "", false).
		Eq("organization_id", orgID.String()).
		ExecuteTo(&connections)

	return connections, err
}

// GetConnection returns a specific connection
func (s *ConnectionService) GetConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) (*models.Connection, error) {
	if s.client.Service == nil {
		return nil, nil
	}

	var connections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Select("*", "", false).
		Eq("id", connectionID.String()).
		Eq("organization_id", orgID.String()).
		ExecuteTo(&connections)

	if err != nil {
		return nil, err
	}

	if len(connections) == 0 {
		return nil, nil
	}

	return &connections[0], nil
}

// CreateConnection creates a new connection
func (s *ConnectionService) CreateConnection(ctx context.Context, conn models.Connection) (*models.Connection, error) {
	if s.client.Service == nil {
		return &conn, nil
	}

	newConnection := map[string]interface{}{
		"id":                     conn.ID,
		"organization_id":        conn.OrganizationID,
		"provider_type":          conn.ProviderType,
		"name":                   conn.Name,
		"credentials_encrypted":  conn.CredentialsEncrypted,
		"sync_status":            conn.SyncStatus,
	}

	var createdConnections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Insert(newConnection, false, "", "*", "").
		ExecuteTo(&createdConnections)

	if err != nil {
		return nil, err
	}

	if len(createdConnections) > 0 {
		return &createdConnections[0], nil
	}

	return &conn, nil
}

// DeleteConnection deletes a connection
func (s *ConnectionService) DeleteConnection(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) error {
	if s.client.Service == nil {
		return nil
	}

	var deletedConnections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Delete("*", "").
		Eq("id", connectionID.String()).
		Eq("organization_id", orgID.String()).
		ExecuteTo(&deletedConnections)

	return err
}

// UpdateConnectionStatus updates the status of a connection
func (s *ConnectionService) UpdateConnectionStatus(ctx context.Context, connectionID uuid.UUID, status string, errorMsg *string) error {
	if s.client.Service == nil {
		return nil
	}

	updateData := map[string]interface{}{
		"sync_status":   status,
		"error_message": errorMsg,
	}

	var updatedConnections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Update(updateData, "*", "").
		Eq("id", connectionID.String()).
		ExecuteTo(&updatedConnections)

	return err
}

// CheckDuplicateName checks if a connection name already exists
func (s *ConnectionService) CheckDuplicateName(ctx context.Context, orgID uuid.UUID, name string) (bool, error) {
	if s.client.Service == nil {
		return false, nil
	}

	var existingConnections []models.Connection
	_, err := s.client.Service.
		From("provider_connections").
		Select("id", "", false).
		Eq("organization_id", orgID.String()).
		Eq("name", name).
		ExecuteTo(&existingConnections)

	if err != nil {
		return false, err
	}

	return len(existingConnections) > 0, nil
}