// Package validation provides input validation utilities
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validator provides validation methods
type Validator struct {
	errors []string
}

// New creates a new Validator
func New() *Validator {
	return &Validator{
		errors: []string{},
	}
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() []string {
	return v.errors
}

// Error returns a combined error message
func (v *Validator) Error() string {
	return strings.Join(v.errors, "; ")
}

// AddError adds a custom error
func (v *Validator) AddError(message string) {
	v.errors = append(v.errors, message)
}

// Required validates that a value is not empty
func (v *Validator) Required(value, field string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
	}
	return v
}

// RequiredInt validates that an int is not zero
func (v *Validator) RequiredInt(value int, field string) *Validator {
	if value == 0 {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
	}
	return v
}

// MinLength validates minimum string length
func (v *Validator) MinLength(value string, min int, field string) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
	return v
}

// MaxLength validates maximum string length
func (v *Validator) MaxLength(value string, max int, field string) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return v
}

// Email validates email format
func (v *Validator) Email(value, field string) *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid email address", field))
	}
	return v
}

// URL validates URL format
func (v *Validator) URL(value, field string) *Validator {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid URL", field))
	}
	return v
}

// UUID validates UUID format
func (v *Validator) UUID(value, field string) *Validator {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid UUID", field))
	}
	return v
}

// Alphanumeric validates alphanumeric characters only
func (v *Validator) Alphanumeric(value, field string) *Validator {
	alphaRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s must contain only alphanumeric characters", field))
	}
	return v
}

// Slug validates slug format (lowercase, alphanumeric, hyphens)
func (v *Validator) Slug(value, field string) *Validator {
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	if !slugRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid slug (lowercase letters, numbers, hyphens)", field))
	}
	return v
}

// Min validates minimum int value
func (v *Validator) Min(value, min int, field string) *Validator {
	if value < min {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at least %d", field, min))
	}
	return v
}

// Max validates maximum int value
func (v *Validator) Max(value, max int, field string) *Validator {
	if value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at most %d", field, max))
	}
	return v
}

// Range validates int is within range
func (v *Validator) Range(value, min, max int, field string) *Validator {
	if value < min || value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
	return v
}

// OneOf validates value is one of allowed values
func (v *Validator) OneOf(value string, allowed []string, field string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors = append(v.errors, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
	return v
}

// Pattern validates value matches a regex pattern
func (v *Validator) Pattern(value, pattern, field, message string) *Validator {
	re := regexp.MustCompile(pattern)
	if !re.MatchString(value) {
		v.errors = append(v.errors, message)
	}
	return v
}

// CronExpression validates cron expression format
func (v *Validator) CronExpression(value, field string) *Validator {
	// Simple cron validation (5 or 6 fields)
	parts := strings.Fields(value)
	if len(parts) < 5 || len(parts) > 6 {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid cron expression", field))
	}
	return v
}

// Password validates password strength
func (v *Validator) Password(value, field string) *Validator {
	if len(value) < 8 {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at least 8 characters", field))
		return v
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(value)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(value)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(value)

	if !hasUpper || !hasLower || !hasNumber {
		v.errors = append(v.errors, fmt.Sprintf("%s must contain uppercase, lowercase, and number", field))
	}
	return v
}

// JSON validates JSON string
func (v *Validator) JSON(value, field string) *Validator {
	if value == "" {
		return v
	}
	// Check for basic JSON structure
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		v.errors = append(v.errors, fmt.Sprintf("%s must be valid JSON", field))
	}
	return v
}

// ValidateStruct is a helper for common struct validation
type FieldRule struct {
	Field string
	Value string
	Rules []string // "required", "email", "min:3", "max:100", etc.
}

// ValidateFields validates multiple fields with rules
func ValidateFields(rules []FieldRule) *Validator {
	v := New()
	for _, r := range rules {
		for _, rule := range r.Rules {
			switch {
			case rule == "required":
				v.Required(r.Value, r.Field)
			case rule == "email":
				if r.Value != "" {
					v.Email(r.Value, r.Field)
				}
			case rule == "url":
				if r.Value != "" {
					v.URL(r.Value, r.Field)
				}
			case rule == "uuid":
				if r.Value != "" {
					v.UUID(r.Value, r.Field)
				}
			case rule == "slug":
				if r.Value != "" {
					v.Slug(r.Value, r.Field)
				}
			case strings.HasPrefix(rule, "min:"):
				var min int
				fmt.Sscanf(rule, "min:%d", &min)
				v.MinLength(r.Value, min, r.Field)
			case strings.HasPrefix(rule, "max:"):
				var max int
				fmt.Sscanf(rule, "max:%d", &max)
				v.MaxLength(r.Value, max, r.Field)
			}
		}
	}
	return v
}
