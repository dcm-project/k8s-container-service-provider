package store

import "fmt"

// NotFoundError indicates the requested container was not found.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("container %q not found", e.ID)
}

// ConflictError indicates a resource conflict (e.g., duplicate name or ID).
type ConflictError struct {
	InstanceRef string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("container %q already exists", e.InstanceRef)
}

// InvalidArgumentError indicates a validation failure in the request.
type InvalidArgumentError struct {
	Message string
}

func (e *InvalidArgumentError) Error() string {
	return e.Message
}
