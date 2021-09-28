package transport

import (
	"fmt"
	"net/http"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/identity"
)

type HttpClientError struct {
	// Source of the error for better
	// insight when capturing errors
	Source error `json:"-"`
	// Http StatusCode code
	StatusCode int `json:"-"`
	// Human readable summary of error
	Summary string `json:"title"`
	// Message that will be sent back to the client
	Description string `json:"message"`
	// Object that can provide further insight
	// to the error. Only accessible internally
	Details map[string]interface{} `json:"-"`
}

type HttpInternalError struct {
	Original    error                  `json:"-"`
	File        string                 `json:"-"`
	Line        int                    `json:"-"`
	Summary     string                 `json:"-"`
	Description string                 `json:"-"`
	Details     map[string]interface{} `json:"-"`
}

func NewHttpClientError(src error, status int, summ string, desc string, details map[string]interface{}) error {
	err := &HttpClientError{
		Source:      src,
		Summary:     summ,
		Description: desc,
		StatusCode:  status,
	}
	if len(details) > 0 {
		err.Details = details
	}
	return err
}

func (h *HttpClientError) Error() string {
	return h.Description
}
func (h *HttpClientError) Headers() (int, map[string]string) {
	return h.StatusCode, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
}
func (h *HttpClientError) Title() string {
	return h.Summary
}
func (h *HttpClientError) Message() string {
	return h.Description
}

func NewHttpInternalError(orig error, file string, line int, summ string, desc string, details map[string]interface{}) error {
	return &HttpInternalError{
		Original:    orig,
		File:        file,
		Line:        line,
		Summary:     summ,
		Description: desc,
		Details:     details,
	}
}

func (h *HttpInternalError) Error() string {
	return fmt.Sprintf("%s\n%s", h.Source(), h.Description)
}
func (h *HttpInternalError) Source() string {
	return fmt.Sprintf("[%s:%d] Original Error: %s", h.File, h.Line, h.Original)
}
func (h *HttpInternalError) Title() string {
	return h.Summary
}
func (h *HttpInternalError) Message() string {
	return h.Description
}

// Common errors

func ErrNotAuthenticated(src error, requestURL string) error {
	return NewHttpClientError(src, http.StatusUnauthorized, "NotAuthenticated", "You must be logged in to access this resource", map[string]interface{}{
		"RequestURL": requestURL,
	})
}

func ErrAlreadyAuthenticated(src error, requestURL string, user identity.Identity) error {
	return NewHttpClientError(src, http.StatusForbidden, "AlreadyAuthenticated", "Cannot access this resource while logged in", map[string]interface{}{
		"RequestURL": requestURL,
		"Identity":   user,
	})
}

// GetHttpError is a utility function that takes an error, fallback error, and a map of Http status codes then generates a valid Http error response
func GetHttpError(src error, fallback error, errMap map[string]int) error {
	e, ok := src.(*internal.ServiceClientError)
	if ok {
		d := e.Details
		code, ok := errMap[e.Summary]
		if !ok {
			// Default to 400
			code = http.StatusBadRequest
		}
		return NewHttpClientError(src, code, e.Summary, e.Description, d)
	}
	return fallback
}
