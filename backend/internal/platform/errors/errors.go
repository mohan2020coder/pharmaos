package apperrors

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Details interface{}
}

func (e *AppError) Error() string { return e.Message }

func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

var (
	ErrNotFound                = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrEmployeeProfileNotFound = New("NO_EMPLOYEE_PROFILE", "No employee profile is linked to this account", http.StatusNotFound)
	ErrNoEmployeeProfile       = New("NO_EMPLOYEE_PROFILE", "No employee profile is linked to this account", http.StatusForbidden)
	ErrConflict                = New("CONFLICT", "Resource already exists", http.StatusConflict)
	ErrForbidden               = New("FORBIDDEN", "You do not have access to this resource", http.StatusForbidden)
	ErrUnauthorized            = New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	ErrBadRequest              = New("BAD_REQUEST", "Invalid request", http.StatusBadRequest)
	ErrValidation              = New("VALIDATION_FAILED", "Validation failed", http.StatusUnprocessableEntity)
	ErrTenantMismatch          = New("TENANT_MISMATCH", "Resource belongs to another tenant", http.StatusNotFound)
)

func Status(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return http.StatusInternalServerError
}

func Public(err error) (string, interface{}) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message, appErr.Details
	}
	return "Internal server error", nil
}
