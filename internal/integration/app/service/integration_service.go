package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/integration/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

type IntegrationService struct {
	integrations map[string]*model.Integration
	httpClient   *http.Client
	logger       logger.Logger
	eventBus     EventPublisher
}

type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
}

func NewIntegrationService(logger logger.Logger, eventBus EventPublisher) *IntegrationService {
	return &IntegrationService{
		integrations: make(map[string]*model.Integration),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:   logger,
		eventBus: eventBus,
	}
}

type CreateIntegrationCommand struct {
	UserID         string
	OrganizationID string
	Name           string
	Description    string
	Type           string
	Config         map[string]interface{}
}

func (s *IntegrationService) CreateIntegration(ctx context.Context, cmd CreateIntegrationCommand) (*model.Integration, error) {
	integration, err := model.NewIntegration(
		cmd.UserID,
		cmd.OrganizationID,
		cmd.Name,
		model.IntegrationType(cmd.Type),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}
	
	// Set configuration
	for key, value := range cmd.Config {
		integration.SetConfig(key, value)
	}
	
	// Store integration
	s.integrations[string(integration.ID())] = integration
	
	// Publish event
	if s.eventBus != nil {
		event := events.IntegrationCreatedEvent{
			IntegrationID: string(integration.ID()),
			UserID:        cmd.UserID,
			Type:          cmd.Type,
			Timestamp:     time.Now(),
		}
		s.eventBus.Publish(ctx, event)
	}
	
	s.logger.Info("Integration created",
		"integration_id", integration.ID(),
		"type", cmd.Type,
		"name", cmd.Name,
	)
	
	return integration, nil
}

type AuthorizeIntegrationCommand struct {
	IntegrationID string
	UserID        string
	Credentials   map[string]interface{}
}

func (s *IntegrationService) AuthorizeIntegration(ctx context.Context, cmd AuthorizeIntegrationCommand) error {
	integration, exists := s.integrations[cmd.IntegrationID]
	if !exists {
		return fmt.Errorf("integration not found")
	}
	
	// Verify ownership
	if integration.UserID() != cmd.UserID {
		return fmt.Errorf("access denied")
	}
	
	// Validate credentials based on integration type
	if err := s.validateCredentials(integration.Type(), cmd.Credentials); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	
	// Store encrypted credentials (encryption would be done here)
	integration.SetCredentials(cmd.Credentials)
	
	// Activate integration
	if err := integration.Activate(); err != nil {
		return fmt.Errorf("failed to activate: %w", err)
	}
	
	// Test connection
	if err := s.testConnection(ctx, integration); err != nil {
		integration.SetError(err.Error())
		return fmt.Errorf("connection test failed: %w", err)
	}
	
	s.logger.Info("Integration authorized",
		"integration_id", cmd.IntegrationID,
		"type", integration.Type(),
	)
	
	return nil
}

func (s *IntegrationService) validateCredentials(integrationType model.IntegrationType, creds map[string]interface{}) error {
	switch integrationType {
	case model.IntegrationTypeSlack:
		if _, ok := creds["webhook_url"]; !ok {
			return fmt.Errorf("webhook_url required for Slack")
		}
	case model.IntegrationTypeGitHub:
		if _, ok := creds["access_token"]; !ok {
			return fmt.Errorf("access_token required for GitHub")
		}
	case model.IntegrationTypeGoogleDrive:
		if _, ok := creds["client_id"]; !ok {
			return fmt.Errorf("client_id required for Google Drive")
		}
		if _, ok := creds["client_secret"]; !ok {
			return fmt.Errorf("client_secret required for Google Drive")
		}
	}
	return nil
}

func (s *IntegrationService) testConnection(ctx context.Context, integration *model.Integration) error {
	switch integration.Type() {
	case model.IntegrationTypeSlack:
		return s.testSlackConnection(ctx, integration)
	case model.IntegrationTypeGitHub:
		return s.testGitHubConnection(ctx, integration)
	case model.IntegrationTypeWebhook:
		return s.testWebhookConnection(ctx, integration)
	default:
		return nil // Skip test for unknown types
	}
}

func (s *IntegrationService) testSlackConnection(ctx context.Context, integration *model.Integration) error {
	webhookURL, ok := integration.Config()["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("webhook URL not configured")
	}
	
	payload := map[string]string{
		"text": "Integration test from LinkFlow AI",
	}
	
	data, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(webhookURL, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack test failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (s *IntegrationService) testGitHubConnection(ctx context.Context, integration *model.Integration) error {
	token, ok := integration.Config()["access_token"].(string)
	if !ok {
		return fmt.Errorf("access token not configured")
	}
	
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github test failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (s *IntegrationService) testWebhookConnection(ctx context.Context, integration *model.Integration) error {
	url, ok := integration.Config()["url"].(string)
	if !ok {
		return fmt.Errorf("webhook URL not configured")
	}
	
	resp, err := s.httpClient.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

func (s *IntegrationService) SyncIntegration(ctx context.Context, integrationID string) error {
	integration, exists := s.integrations[integrationID]
	if !exists {
		return fmt.Errorf("integration not found")
	}
	
	// Record sync time
	integration.RecordSync()
	
	// Publish sync event
	if s.eventBus != nil {
		event := events.IntegrationSyncedEvent{
			IntegrationID: integrationID,
			Timestamp:     time.Now(),
		}
		s.eventBus.Publish(ctx, event)
	}
	
	s.logger.Info("Integration synced", "integration_id", integrationID)
	return nil
}

func (s *IntegrationService) ListIntegrations(ctx context.Context, userID string) ([]*model.Integration, error) {
	var result []*model.Integration
	
	for _, integration := range s.integrations {
		if integration.UserID() == userID {
			result = append(result, integration)
		}
	}
	
	return result, nil
}

func (s *IntegrationService) GetIntegration(ctx context.Context, integrationID, userID string) (*model.Integration, error) {
	integration, exists := s.integrations[integrationID]
	if !exists {
		return nil, fmt.Errorf("integration not found")
	}
	
	if integration.UserID() != userID {
		return nil, fmt.Errorf("access denied")
	}
	
	return integration, nil
}

func (s *IntegrationService) DeleteIntegration(ctx context.Context, integrationID, userID string) error {
	integration, exists := s.integrations[integrationID]
	if !exists {
		return fmt.Errorf("integration not found")
	}
	
	if integration.UserID() != userID {
		return fmt.Errorf("access denied")
	}
	
	// Deactivate first
	integration.Deactivate()
	
	// Remove from storage
	delete(s.integrations, integrationID)
	
	// Publish event
	if s.eventBus != nil {
		event := events.IntegrationDeletedEvent{
			IntegrationID: integrationID,
			UserID:        userID,
			Timestamp:     time.Now(),
		}
		s.eventBus.Publish(ctx, event)
	}
	
	s.logger.Info("Integration deleted", "integration_id", integrationID)
	return nil
}
