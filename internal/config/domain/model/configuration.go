package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type ConfigID string

func NewConfigID() ConfigID {
	return ConfigID(uuid.New().String())
}

type ConfigScope string

const (
	ScopeGlobal       ConfigScope = "global"
	ScopeOrganization ConfigScope = "organization"
	ScopeUser         ConfigScope = "user"
	ScopeService      ConfigScope = "service"
)

type Configuration struct {
	id             ConfigID
	key            string
	value          interface{}
	scope          ConfigScope
	scopeID        string // ID of the scoped entity (org ID, user ID, service name)
	description    string
	dataType       string
	defaultValue   interface{}
	validationRule string
	isSecret       bool
	isReadOnly     bool
	version        int
	createdAt      time.Time
	updatedAt      time.Time
	updatedBy      string
}

func NewConfiguration(key string, value interface{}, scope ConfigScope, scopeID string) (*Configuration, error) {
	if key == "" {
		return nil, errors.New("configuration key is required")
	}

	now := time.Now()
	return &Configuration{
		id:        NewConfigID(),
		key:       key,
		value:     value,
		scope:     scope,
		scopeID:   scopeID,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (c *Configuration) ID() ConfigID               { return c.id }
func (c *Configuration) Key() string                { return c.key }
func (c *Configuration) Value() interface{}         { return c.value }
func (c *Configuration) Scope() ConfigScope         { return c.scope }
func (c *Configuration) ScopeID() string            { return c.scopeID }
func (c *Configuration) Version() int               { return c.version }
func (c *Configuration) IsSecret() bool             { return c.isSecret }

func (c *Configuration) Update(value interface{}, updatedBy string) error {
	if c.isReadOnly {
		return errors.New("configuration is read-only")
	}

	c.value = value
	c.version++
	c.updatedAt = time.Now()
	c.updatedBy = updatedBy

	return nil
}

func (c *Configuration) SetDescription(desc string) {
	c.description = desc
	c.updatedAt = time.Now()
}

func (c *Configuration) SetSecret(secret bool) {
	c.isSecret = secret
	c.updatedAt = time.Now()
}

func (c *Configuration) SetReadOnly(readOnly bool) {
	c.isReadOnly = readOnly
	c.updatedAt = time.Now()
}

type ConfigHistory struct {
	id        string
	configID  ConfigID
	key       string
	oldValue  interface{}
	newValue  interface{}
	version   int
	changedBy string
	changedAt time.Time
	reason    string
}

func NewConfigHistory(configID ConfigID, key string, oldValue, newValue interface{}, version int, changedBy string) *ConfigHistory {
	return &ConfigHistory{
		id:        uuid.New().String(),
		configID:  configID,
		key:       key,
		oldValue:  oldValue,
		newValue:  newValue,
		version:   version,
		changedBy: changedBy,
		changedAt: time.Now(),
	}
}
