// Package validators provides common validation utilities
package validators

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	messages := make([]string, len(e))
	for i, err := range e {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// ToJSON returns errors as JSON
func (e ValidationErrors) ToJSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// Common regex patterns
var (
	EmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	URLRegex      = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	UUIDRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	SlugRegex     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	AlphaRegex    = regexp.MustCompile(`^[a-zA-Z]+$`)
	AlphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	PhoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	CronRegex     = regexp.MustCompile(`^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$`)
)

// IsEmail validates email format
func IsEmail(email string) bool {
	return EmailRegex.MatchString(email)
}

// IsURL validates URL format
func IsURL(url string) bool {
	return URLRegex.MatchString(url)
}

// IsUUID validates UUID format
func IsUUID(uuid string) bool {
	return UUIDRegex.MatchString(uuid)
}

// IsSlug validates slug format
func IsSlug(slug string) bool {
	return SlugRegex.MatchString(slug)
}

// IsAlpha validates alphabetic characters only
func IsAlpha(s string) bool {
	return AlphaRegex.MatchString(s)
}

// IsAlphaNumeric validates alphanumeric characters only
func IsAlphaNumeric(s string) bool {
	return AlphaNumRegex.MatchString(s)
}

// IsPhone validates phone number format
func IsPhone(phone string) bool {
	return PhoneRegex.MatchString(phone)
}

// IsCronExpression validates cron expression
func IsCronExpression(expr string) bool {
	parts := strings.Fields(expr)
	return len(parts) >= 5 && len(parts) <= 6
}

// IsJSON validates JSON string
func IsJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

// IsEmpty checks if a string is empty or whitespace only
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsInRange checks if a value is within a range
func IsInRange(value, min, max int) bool {
	return value >= min && value <= max
}

// IsInList checks if a value is in a list
func IsInList(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

// IsFutureDate checks if a time is in the future
func IsFutureDate(t time.Time) bool {
	return t.After(time.Now())
}

// IsPastDate checks if a time is in the past
func IsPastDate(t time.Time) bool {
	return t.Before(time.Now())
}

// WorkflowValidator validates workflow data
type WorkflowValidator struct {
	errors ValidationErrors
}

// NewWorkflowValidator creates a new workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{
		errors: make(ValidationErrors, 0),
	}
}

// ValidateName validates workflow name
func (v *WorkflowValidator) ValidateName(name string) *WorkflowValidator {
	if IsEmpty(name) {
		v.errors = append(v.errors, ValidationError{
			Field:   "name",
			Message: "name is required",
			Code:    "REQUIRED",
		})
	} else if len(name) < 3 {
		v.errors = append(v.errors, ValidationError{
			Field:   "name",
			Message: "name must be at least 3 characters",
			Code:    "MIN_LENGTH",
		})
	} else if len(name) > 100 {
		v.errors = append(v.errors, ValidationError{
			Field:   "name",
			Message: "name must be at most 100 characters",
			Code:    "MAX_LENGTH",
		})
	}
	return v
}

// ValidateNodes validates workflow nodes
func (v *WorkflowValidator) ValidateNodes(nodes []map[string]interface{}) *WorkflowValidator {
	if len(nodes) == 0 {
		v.errors = append(v.errors, ValidationError{
			Field:   "nodes",
			Message: "workflow must have at least one node",
			Code:    "MIN_ITEMS",
		})
		return v
	}

	if len(nodes) > 100 {
		v.errors = append(v.errors, ValidationError{
			Field:   "nodes",
			Message: "workflow cannot have more than 100 nodes",
			Code:    "MAX_ITEMS",
		})
	}

	// Validate each node
	nodeIDs := make(map[string]bool)
	for i, node := range nodes {
		id, ok := node["id"].(string)
		if !ok || IsEmpty(id) {
			v.errors = append(v.errors, ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", i),
				Message: "node ID is required",
				Code:    "REQUIRED",
			})
		} else if nodeIDs[id] {
			v.errors = append(v.errors, ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", i),
				Message: "duplicate node ID",
				Code:    "DUPLICATE",
			})
		} else {
			nodeIDs[id] = true
		}

		nodeType, ok := node["type"].(string)
		if !ok || IsEmpty(nodeType) {
			v.errors = append(v.errors, ValidationError{
				Field:   fmt.Sprintf("nodes[%d].type", i),
				Message: "node type is required",
				Code:    "REQUIRED",
			})
		}
	}

	return v
}

// Errors returns validation errors
func (v *WorkflowValidator) Errors() ValidationErrors {
	return v.errors
}

// HasErrors returns true if there are validation errors
func (v *WorkflowValidator) HasErrors() bool {
	return len(v.errors) > 0
}

// CredentialValidator validates credential data
type CredentialValidator struct {
	errors ValidationErrors
}

// NewCredentialValidator creates a new credential validator
func NewCredentialValidator() *CredentialValidator {
	return &CredentialValidator{
		errors: make(ValidationErrors, 0),
	}
}

// ValidateName validates credential name
func (v *CredentialValidator) ValidateName(name string) *CredentialValidator {
	if IsEmpty(name) {
		v.errors = append(v.errors, ValidationError{
			Field:   "name",
			Message: "name is required",
			Code:    "REQUIRED",
		})
	}
	return v
}

// ValidateType validates credential type
func (v *CredentialValidator) ValidateType(credType string) *CredentialValidator {
	validTypes := []string{"api_key", "oauth2", "basic", "bearer", "custom"}
	if !IsInList(credType, validTypes) {
		v.errors = append(v.errors, ValidationError{
			Field:   "type",
			Message: "invalid credential type",
			Code:    "INVALID",
		})
	}
	return v
}

// Errors returns validation errors
func (v *CredentialValidator) Errors() ValidationErrors {
	return v.errors
}

// HasErrors returns true if there are validation errors
func (v *CredentialValidator) HasErrors() bool {
	return len(v.errors) > 0
}

// ScheduleValidator validates schedule data
type ScheduleValidator struct {
	errors ValidationErrors
}

// NewScheduleValidator creates a new schedule validator
func NewScheduleValidator() *ScheduleValidator {
	return &ScheduleValidator{
		errors: make(ValidationErrors, 0),
	}
}

// ValidateCron validates cron expression
func (v *ScheduleValidator) ValidateCron(expr string) *ScheduleValidator {
	if IsEmpty(expr) {
		v.errors = append(v.errors, ValidationError{
			Field:   "cronExpression",
			Message: "cron expression is required",
			Code:    "REQUIRED",
		})
	} else if !IsCronExpression(expr) {
		v.errors = append(v.errors, ValidationError{
			Field:   "cronExpression",
			Message: "invalid cron expression",
			Code:    "INVALID",
		})
	}
	return v
}

// ValidateTimezone validates timezone
func (v *ScheduleValidator) ValidateTimezone(tz string) *ScheduleValidator {
	if !IsEmpty(tz) {
		_, err := time.LoadLocation(tz)
		if err != nil {
			v.errors = append(v.errors, ValidationError{
				Field:   "timezone",
				Message: "invalid timezone",
				Code:    "INVALID",
			})
		}
	}
	return v
}

// Errors returns validation errors
func (v *ScheduleValidator) Errors() ValidationErrors {
	return v.errors
}

// HasErrors returns true if there are validation errors
func (v *ScheduleValidator) HasErrors() bool {
	return len(v.errors) > 0
}
