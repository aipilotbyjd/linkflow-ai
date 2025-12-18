package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type IntegrationID string

func NewIntegrationID() IntegrationID {
	return IntegrationID(uuid.New().String())
}

type IntegrationType string

const (
	IntegrationTypeSlack      IntegrationType = "slack"
	IntegrationTypeGitHub     IntegrationType = "github"
	IntegrationTypeGoogleDrive IntegrationType = "google_drive"
	IntegrationTypeDropbox    IntegrationType = "dropbox"
	IntegrationTypeJira       IntegrationType = "jira"
	IntegrationTypeZapier     IntegrationType = "zapier"
	IntegrationTypeWebhook    IntegrationType = "webhook"
	IntegrationTypeCustom     IntegrationType = "custom"
)

type IntegrationStatus string

const (
	StatusActive       IntegrationStatus = "active"
	StatusInactive     IntegrationStatus = "inactive"
	StatusAuthRequired IntegrationStatus = "auth_required"
	StatusError        IntegrationStatus = "error"
)

type Integration struct {
	id               IntegrationID
	userID           string
	organizationID   string
	name             string
	description      string
	integrationType  IntegrationType
	config           map[string]interface{}
	credentials      map[string]interface{} // Encrypted
	status           IntegrationStatus
	lastSync         *time.Time
	syncFrequency    time.Duration
	errorMessage     string
	metadata         map[string]interface{}
	createdAt        time.Time
	updatedAt        time.Time
}

func NewIntegration(userID, organizationID, name string, integrationType IntegrationType) (*Integration, error) {
	if userID == "" || name == "" {
		return nil, errors.New("userID and name are required")
	}

	now := time.Now()
	return &Integration{
		id:              NewIntegrationID(),
		userID:          userID,
		organizationID:  organizationID,
		name:            name,
		integrationType: integrationType,
		config:          make(map[string]interface{}),
		credentials:     make(map[string]interface{}),
		metadata:        make(map[string]interface{}),
		status:          StatusInactive,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func (i *Integration) ID() IntegrationID               { return i.id }
func (i *Integration) UserID() string                  { return i.userID }
func (i *Integration) Name() string                    { return i.name }
func (i *Integration) Type() IntegrationType           { return i.integrationType }
func (i *Integration) Status() IntegrationStatus       { return i.status }
func (i *Integration) Config() map[string]interface{}  { return i.config }

func (i *Integration) SetConfig(key string, value interface{}) {
	i.config[key] = value
	i.updatedAt = time.Now()
}

func (i *Integration) SetCredentials(creds map[string]interface{}) {
	i.credentials = creds
	i.updatedAt = time.Now()
}

func (i *Integration) Activate() error {
	if len(i.credentials) == 0 {
		return errors.New("credentials required to activate integration")
	}
	i.status = StatusActive
	i.updatedAt = time.Now()
	return nil
}

func (i *Integration) Deactivate() {
	i.status = StatusInactive
	i.updatedAt = time.Now()
}

func (i *Integration) SetError(message string) {
	i.status = StatusError
	i.errorMessage = message
	i.updatedAt = time.Now()
}

func (i *Integration) RecordSync() {
	now := time.Now()
	i.lastSync = &now
	i.updatedAt = now
}
