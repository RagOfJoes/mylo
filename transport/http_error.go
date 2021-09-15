package transport

import (
	"fmt"
)

type HttpClientError struct {
	// Http StatusCode code
	StatusCode int `json:"-"`
	// Human readable summary of error
	Summary string `json:"title"`
	// Message that will be sent back to the client
	Description string `json:"message"`
	// Object that can provide further insight
	// to the error. Only accessible internally
	Details *map[string]interface{} `json:"-"`
}

type HttpInternalError struct {
	Addr        string `json:"-"`
	File        string `json:"-"`
	Line        int    `json:"-"`
	Summary     string `json:"-"`
	Description string `json:"-"`
}

func NewHttpClientError(status int, summ string, desc string, details *map[string]interface{}) error {
	err := &HttpClientError{
		Summary:     summ,
		Description: desc,
		StatusCode:  status,
	}
	if details != nil && len(*details) > 0 {
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

func NewHttpInternalError(file string, line int, summ string, desc string) error {
	return &HttpInternalError{
		File:        file,
		Line:        line,
		Summary:     summ,
		Description: desc,
	}
}

func (h *HttpInternalError) Error() string {
	return fmt.Sprintf("%s: %s", h.Source(), h.Description)
}
func (h *HttpInternalError) Source() string {
	return fmt.Sprintf("[%s:%d]", h.File, h.Line)
}
func (h *HttpInternalError) Title() string {
	return h.Summary
}
func (h *HttpInternalError) Message() string {
	return h.Description
}
