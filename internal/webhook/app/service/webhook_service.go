package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/repository"
)

var (
	ErrWebhookNotFound = errors.New("webhook not found")
	ErrUnauthorized    = errors.New("unauthorized")
)

type WebhookService struct {
	repository     repository.WebhookRepository
	eventPublisher *kafka.EventPublisher
	cache          *cache.RedisCache
	logger         logger.Logger
	httpClient     *http.Client
}

func NewWebhookService(
	repository repository.WebhookRepository,
	eventPublisher *kafka.EventPublisher,
	cache *cache.RedisCache,
	logger logger.Logger,
) *WebhookService {
	return &WebhookService{
		repository:     repository,
		eventPublisher: eventPublisher,
		cache:          cache,
		logger:         logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type CreateWebhookCommand struct {
	UserID           string
	OrganizationID   string
	WorkflowID       string
	Name             string
	Description      string
	EndpointURL      string
	Method           string
	Headers          map[string]string
	AuthType         string
	AuthConfig       map[string]interface{}
}

func (s *WebhookService) CreateWebhook(ctx context.Context, cmd CreateWebhookCommand) (*model.Webhook, error) {
	method := model.WebhookMethod(cmd.Method)
	if method == "" {
		method = model.WebhookMethodPOST
	}

	webhook, err := model.NewWebhook(
		cmd.UserID,
		cmd.WorkflowID,
		cmd.Name,
		cmd.EndpointURL,
		method,
	)
	if err != nil {
		return nil, err
	}

	if cmd.AuthType != "" {
		webhook.SetAuthentication(model.AuthenticationType(cmd.AuthType), cmd.AuthConfig)
	}

	for key, value := range cmd.Headers {
		webhook.AddHeader(key, value)
	}

	if err := s.repository.Save(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to save webhook: %w", err)
	}

	// Publish event
	if s.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"webhookId":  webhook.ID().String(),
			"workflowId": webhook.WorkflowID(),
			"url":        webhook.EndpointURL(),
		})
		event := &events.Event{
			AggregateID:   webhook.ID().String(),
			AggregateType: "Webhook",
			EventType:     "webhook.created",
			UserID:        cmd.UserID,
			Timestamp:     time.Now(),
			Payload:       json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Webhook created", "webhook_id", webhook.ID())
	return webhook, nil
}

func (s *WebhookService) TriggerWebhook(ctx context.Context, webhookID model.WebhookID, payload interface{}) error {
	webhook, err := s.repository.FindByID(ctx, webhookID)
	if err != nil {
		return ErrWebhookNotFound
	}

	// Prepare request body
	bodyData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(
		ctx,
		string(webhook.Method()),
		webhook.EndpointURL(),
		bytes.NewReader(bodyData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-ID", webhook.ID().String())
	req.Header.Set("X-Webhook-Signature", webhook.GenerateSignature(bodyData))
	
	for key, value := range webhook.Headers() {
		req.Header.Set(key, value)
	}

	// Add authentication
	s.addAuthentication(req, webhook)

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		webhook.RecordTrigger(false, err.Error())
		_ = s.repository.Update(ctx, webhook)
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(resp.Body)

	// Check status
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		webhook.RecordTrigger(false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
	} else {
		webhook.RecordTrigger(true, "")
	}

	_ = s.repository.Update(ctx, webhook)

	// Publish event
	if s.eventPublisher != nil {
		eventType := "webhook.triggered"
		if !success {
			eventType = "webhook.failed"
		}
		
		payload, _ := json.Marshal(map[string]interface{}{
			"webhookId": webhook.ID().String(),
			"status":    resp.StatusCode,
			"success":   success,
		})
		event := &events.Event{
			AggregateID:   webhook.ID().String(),
			AggregateType: "Webhook",
			EventType:     eventType,
			Timestamp:     time.Now(),
			Payload:       json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	if !success {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *WebhookService) GetWebhook(ctx context.Context, webhookID model.WebhookID) (*model.Webhook, error) {
	// Try cache first
	if s.cache != nil {
		var webhook model.Webhook
		cacheKey := fmt.Sprintf("webhook:%s", webhookID)
		if err := s.cache.Get(ctx, cacheKey, &webhook); err == nil {
			return &webhook, nil
		}
	}

	webhook, err := s.repository.FindByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Cache the result
	if s.cache != nil {
		cacheKey := fmt.Sprintf("webhook:%s", webhookID)
		_ = s.cache.Set(ctx, cacheKey, webhook, 5*time.Minute)
	}

	return webhook, nil
}

func (s *WebhookService) addAuthentication(req *http.Request, webhook *model.Webhook) {
	// Authentication would be stored in authConfig, not headers
	// For now, using headers as a simple implementation
	switch webhook.AuthenticationType() {
	case model.AuthTypeBasic:
		// Add basic auth
		if username, ok := webhook.Headers()["username"]; ok {
			if password, ok := webhook.Headers()["password"]; ok {
				req.SetBasicAuth(username, password)
			}
		}
	case model.AuthTypeBearer:
		// Add bearer token
		if token, ok := webhook.Headers()["token"]; ok {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case model.AuthTypeAPIKey:
		// Add API key
		if key, ok := webhook.Headers()["api_key"]; ok {
			if header, ok := webhook.Headers()["api_key_header"]; ok {
				req.Header.Set(header, key)
			} else {
				req.Header.Set("X-API-Key", key)
			}
		}
	}
}
