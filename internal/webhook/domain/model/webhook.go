package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

type WebhookID string

func NewWebhookID() WebhookID {
	return WebhookID(uuid.New().String())
}

func (id WebhookID) String() string {
	return string(id)
}

type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusInactive WebhookStatus = "inactive"
	WebhookStatusFailed   WebhookStatus = "failed"
	WebhookStatusDisabled WebhookStatus = "disabled"
)

type WebhookMethod string

const (
	WebhookMethodGET    WebhookMethod = "GET"
	WebhookMethodPOST   WebhookMethod = "POST"
	WebhookMethodPUT    WebhookMethod = "PUT"
	WebhookMethodPATCH  WebhookMethod = "PATCH"
	WebhookMethodDELETE WebhookMethod = "DELETE"
)

type AuthenticationType string

const (
	AuthTypeNone   AuthenticationType = "none"
	AuthTypeBasic  AuthenticationType = "basic"
	AuthTypeBearer AuthenticationType = "bearer"
	AuthTypeAPIKey AuthenticationType = "api_key"
	AuthTypeHMAC   AuthenticationType = "hmac"
)

type RetryConfig struct {
	MaxRetries    int           `json:"maxRetries"`
	RetryDelay    time.Duration `json:"retryDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
}

type Webhook struct {
	id                 WebhookID
	userID             string
	organizationID     string
	workflowID         string
	name               string
	description        string
	endpointURL        string
	secret             string
	method             WebhookMethod
	headers            map[string]string
	queryParams        map[string]string
	authenticationType AuthenticationType
	authConfig         map[string]interface{}
	retryConfig        RetryConfig
	timeoutMs          int
	status             WebhookStatus
	lastTriggeredAt    *time.Time
	triggerCount       int64
	successCount       int64
	failureCount       int64
	lastError          string
	metadata           map[string]interface{}
	createdAt          time.Time
	updatedAt          time.Time
	version            int
}

func NewWebhook(userID, workflowID, name, endpointURL string, method WebhookMethod) (*Webhook, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if name == "" {
		return nil, errors.New("webhook name is required")
	}
	if endpointURL == "" {
		return nil, errors.New("endpoint URL is required")
	}

	now := time.Now()
	webhook := &Webhook{
		id:                 NewWebhookID(),
		userID:             userID,
		workflowID:         workflowID,
		name:               name,
		endpointURL:        endpointURL,
		secret:             generateSecret(),
		method:             method,
		headers:            make(map[string]string),
		queryParams:        make(map[string]string),
		authenticationType: AuthTypeNone,
		authConfig:         make(map[string]interface{}),
		retryConfig: RetryConfig{
			MaxRetries:    3,
			RetryDelay:    1 * time.Second,
			BackoffFactor: 2.0,
		},
		timeoutMs:    30000,
		status:       WebhookStatusActive,
		triggerCount: 0,
		successCount: 0,
		failureCount: 0,
		metadata:     make(map[string]interface{}),
		createdAt:    now,
		updatedAt:    now,
		version:      0,
	}

	return webhook, nil
}

// Getters
func (w *Webhook) ID() WebhookID                        { return w.id }
func (w *Webhook) UserID() string                       { return w.userID }
func (w *Webhook) WorkflowID() string                   { return w.workflowID }
func (w *Webhook) Name() string                         { return w.name }
func (w *Webhook) EndpointURL() string                  { return w.endpointURL }
func (w *Webhook) Secret() string                       { return w.secret }
func (w *Webhook) Method() WebhookMethod                { return w.method }
func (w *Webhook) Status() WebhookStatus                { return w.status }
func (w *Webhook) Headers() map[string]string           { return w.headers }
func (w *Webhook) AuthenticationType() AuthenticationType { return w.authenticationType }
func (w *Webhook) TriggerCount() int64                  { return w.triggerCount }
func (w *Webhook) SuccessCount() int64                  { return w.successCount }
func (w *Webhook) FailureCount() int64                  { return w.failureCount }
func (w *Webhook) CreatedAt() time.Time                 { return w.createdAt }
func (w *Webhook) UpdatedAt() time.Time                 { return w.updatedAt }
func (w *Webhook) Version() int                         { return w.version }

func (w *Webhook) SetEndpointURL(url string) error {
	if url == "" {
		return errors.New("endpoint URL cannot be empty")
	}
	w.endpointURL = url
	w.updatedAt = time.Now()
	w.version++
	return nil
}

func (w *Webhook) SetAuthentication(authType AuthenticationType, config map[string]interface{}) {
	w.authenticationType = authType
	w.authConfig = config
	w.updatedAt = time.Now()
	w.version++
}

func (w *Webhook) AddHeader(key, value string) {
	w.headers[key] = value
	w.updatedAt = time.Now()
	w.version++
}

func (w *Webhook) RecordTrigger(success bool, errorMessage string) {
	now := time.Now()
	w.lastTriggeredAt = &now
	w.triggerCount++
	
	if success {
		w.successCount++
		w.lastError = ""
	} else {
		w.failureCount++
		w.lastError = errorMessage
		
		// Mark as failed if too many failures
		failureRate := float64(w.failureCount) / float64(w.triggerCount)
		if w.triggerCount > 10 && failureRate > 0.5 {
			w.status = WebhookStatusFailed
		}
	}
	
	w.updatedAt = time.Now()
	w.version++
}

func (w *Webhook) GenerateSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(w.secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func (w *Webhook) VerifySignature(payload []byte, signature string) bool {
	expectedSignature := w.GenerateSignature(payload)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

func (w *Webhook) Activate() {
	w.status = WebhookStatusActive
	w.updatedAt = time.Now()
	w.version++
}

func (w *Webhook) Deactivate() {
	w.status = WebhookStatusInactive
	w.updatedAt = time.Now()
	w.version++
}

func (w *Webhook) Disable() {
	w.status = WebhookStatusDisabled
	w.updatedAt = time.Now()
	w.version++
}

func generateSecret() string {
	return uuid.New().String()
}
