package internal

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrInvalidExpiredFlow   = errors.New("Invalid or expired flow")
	ErrFailedNanoID         = errors.New("Failed to generate nano id")
	ErrAlreadyAuthenticated = errors.New("Cannot access this resource while logged in")
	ErrUnauthorized         = errors.New("You must be logged in to access this resource")
)

type ErrorCode string

const (
	ErrorCodeInternal        ErrorCode = "Internal"
	ErrorCodeNotFound        ErrorCode = "NotFound"
	ErrorCodeForbidden       ErrorCode = "Forbidden"
	ErrorCodeUnauthorized    ErrorCode = "Unauthorized"
	ErrorCodeInvalidArgument ErrorCode = "InvalidArgument"
)

type Error struct {
	src  error
	msg  string
	code ErrorCode
}

// WrapErrorf returns a wrapped error
func WrapErrorf(src error, code ErrorCode, format string, a ...interface{}) error {
	return &Error{
		src:  src,
		code: code,
		msg:  fmt.Sprintf(format, a...),
	}
}

// NewErrorf instantiates a new error
func NewErrorf(code ErrorCode, format string, a ...interface{}) error {
	return WrapErrorf(nil, code, format, a...)
}

// Error returns the message, when wrapping errors the wrapped error is returned
func (e *Error) Error() string {
	if e.src != nil {
		return fmt.Sprintf("%s: %v", e.msg, e.src)
	}
	return e.msg
}

// Unwrap returns the wrapped error, if any
func (e *Error) Unwrap() error {
	return e.src
}

// Code returns the code representing this error
func (e *Error) Code() ErrorCode {
	return e.code
}

// Message returns the message of the error. Unlike Error(), this will only return the last error's message as opposed to the entire chain error
func (e *Error) Message() string {
	return e.msg
}
