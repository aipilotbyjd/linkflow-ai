// Package response provides standardized HTTP response helpers
package response

import (
	"encoding/json"
	"net/http"
)

// Response is the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta contains pagination and other metadata
type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"totalPages,omitempty"`
}

// APIError represents an API error with HTTP status
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]string
}

func (e *APIError) Error() string {
	return e.Message
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(key, value string) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// Common errors
var (
	ErrBadRequest = &APIError{
		StatusCode: http.StatusBadRequest,
		Code:       "BAD_REQUEST",
		Message:    "Invalid request",
	}

	ErrUnauthorized = &APIError{
		StatusCode: http.StatusUnauthorized,
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
	}

	ErrForbidden = &APIError{
		StatusCode: http.StatusForbidden,
		Code:       "FORBIDDEN",
		Message:    "Access denied",
	}

	ErrNotFound = &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
	}

	ErrConflict = &APIError{
		StatusCode: http.StatusConflict,
		Code:       "CONFLICT",
		Message:    "Resource already exists",
	}

	ErrValidation = &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "VALIDATION_ERROR",
		Message:    "Validation failed",
	}

	ErrRateLimit = &APIError{
		StatusCode: http.StatusTooManyRequests,
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    "Too many requests",
	}

	ErrInternal = &APIError{
		StatusCode: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
	}

	ErrServiceUnavailable = &APIError{
		StatusCode: http.StatusServiceUnavailable,
		Code:       "SERVICE_UNAVAILABLE",
		Message:    "Service temporarily unavailable",
	}
)

// JSON sends a JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}

// JSONWithMeta sends a JSON response with metadata
func JSONWithMeta(w http.ResponseWriter, statusCode int, data interface{}, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
		Meta:    meta,
	}

	json.NewEncoder(w).Encode(resp)
}

// Error sends an error response
func Error(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// ErrorWithMessage sends an error response with a custom message
func ErrorWithMessage(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Created sends a 201 Created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// OK sends a 200 OK response
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// Paginated sends a paginated response
func Paginated(w http.ResponseWriter, data interface{}, page, limit int, total int64) {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	JSONWithMeta(w, http.StatusOK, data, &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
