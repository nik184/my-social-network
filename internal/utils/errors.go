package utils

import (
	"fmt"
)

// Common error types for better error handling
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
}

type DatabaseError struct {
	Operation string
	Err       error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database operation '%s' failed: %v", e.Operation, e.Err)
}

type NetworkError struct {
	Operation string
	PeerID    string
	Err       error
}

func (e NetworkError) Error() string {
	return fmt.Sprintf("network operation '%s' failed for peer '%s': %v", e.Operation, e.PeerID, e.Err)
}

// Helper functions for common error patterns
func WrapDatabaseError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return DatabaseError{Operation: operation, Err: err}
}

func WrapNetworkError(operation, peerID string, err error) error {
	if err == nil {
		return nil
	}
	return NetworkError{Operation: operation, PeerID: peerID, Err: err}
}

func NewValidationError(field, message string) error {
	return ValidationError{Field: field, Message: message}
}

func NewNotFoundError(resource, id string) error {
	return NotFoundError{Resource: resource, ID: id}
}
