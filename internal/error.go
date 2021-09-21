package internal

import (
	"fmt"
	"net/http"
)

// ClientError are errors that can be shared
// publicly
type ClientError interface {
	// Ensures that this interface
	// also implements std error
	// library
	Error() string

	// Will be appended to response
	Headers() (int, map[string]string)

	Title() string
	Message() string
}

// InternalError are errors that could/should
// be used internally for logging, metrics, etc.
type InternalError interface {
	// Ensures that this interface
	// also implements std error
	// library
	Error() string

	// Could be an internal error, stack trace, etc.
	// Anything that could give further insight for
	// future debugging and for internal logging
	Source() string
	// Human readable summary of error
	Title() string
	// Human readable explanation of
	// error
	Message() string
}

// Base Implementations
//
//

// ServiceClientError
type ServiceClientError struct {
	// Source of the error for better
	// insight when capturing errors
	Source error `json:"-"`
	// Human readable summary of error
	Summary string `json:"title"`
	// Message that will be sent back to the client
	Description string `json:"message"`
	// Object that can provide further insight
	// to the client
	Details map[string]interface{} `json:"details,omitempty"`
}

func NewServiceClientError(src error, summ string, desc string, details map[string]interface{}) error {
	err := &ServiceClientError{
		Source:      src,
		Summary:     summ,
		Description: desc,
		Details:     details,
	}
	return err
}

func (h *ServiceClientError) Error() string {
	return h.Description
}
func (h *ServiceClientError) Headers() (int, map[string]string) {
	return http.StatusBadRequest, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
}
func (h *ServiceClientError) Title() string {
	return h.Summary
}
func (h *ServiceClientError) Message() string {
	return h.Description
}

// ServiceInternalError
type ServiceInternalError struct {
	// Original error
	Original error `json:"-"`
	// File defines the file that threw the error
	File string `json:"-"`
	// Line defines the line that threw the error
	Line int `json:"-"`
	// Summary defines a human readable summary of error
	Summary string `json:"-"`
	// Description define a human readable description of error
	Description string `json:"-"`
	// Detail describes an object that can provide further insight of error
	Details map[string]interface{}
}

func NewServiceInternalError(orig error, file string, line int, summ string, desc string, details map[string]interface{}) error {
	return &ServiceInternalError{
		Original:    orig,
		File:        file,
		Line:        line,
		Summary:     summ,
		Description: desc,
		Details:     details,
	}
}

func (h *ServiceInternalError) Error() string {
	return fmt.Sprintf("%s\n%s", h.Source(), h.Description)
}
func (h *ServiceInternalError) Source() string {
	return fmt.Sprintf("[%s:%d] Original Error: %s", h.File, h.Line, h.Original)
}
func (h *ServiceInternalError) Title() string {
	return h.Summary
}
func (h *ServiceInternalError) Message() string {
	return h.Description
}
