package transport

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
	"go.uber.org/ratelimit"
)

// RateLimiterMiddleware limits the number of operation
// per second
func RateLimiterMiddleware(rps int) gin.HandlerFunc {
	limit := ratelimit.New(rps)
	return func(c *gin.Context) {
		limit.Take()
	}
}

func SecurityMiddleware() gin.HandlerFunc {
	cfg := config.Get()
	secureMiddleware := secure.New(cfg.Server.Security)
	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}

		// Set some extra settings CORS
		c.Writer.Header().Set("Access-Control-Allow-Origin", cfg.Server.AccessControl.AllowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%v", cfg.Server.AccessControl.AllowCredentials))
		c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.Server.AccessControl.AllowHeaders, ", "))
		c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.Server.AccessControl.AllowMethods, ", "))
		// For redirection avoid Header rewrite
		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	}
}

// ErrorMiddleware is a post middleware
// that handles errors for every requests
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute whatever endpoint is hit
		c.Next()
		// If no errors occurred then return early
		if len(c.Errors) == 0 {
			return
		}
		var err *internal.Error
		status := http.StatusInternalServerError
		actualErr := HttpErrorResponse{
			Title:       "InternalServerError",
			Description: "Oops! Something went wrong. Please try again later.",
		}
		// If error is custom error then customize response
		if errors.As(c.Errors[len(c.Errors)-1], &err) {
			actualErr.Title = string(err.Code())
			actualErr.Description = err.Message()
			switch err.Code() {
			case internal.ErrorCodeNotFound:
				status = http.StatusNotFound
			case internal.ErrorCodeForbidden:
				status = http.StatusForbidden
			case internal.ErrorCodeUnauthorized:
				status = http.StatusUnauthorized
			case internal.ErrorCodeInvalidArgument:
				status = http.StatusBadRequest
			default:
				actualErr.Title = "InternalServerError"
				actualErr.Description = "Oops! Something went wrong. Please try again later."
			}
		}
		// If nothing was hit then respond with a 500 and capture relevant info
		c.JSON(status, HttpResponse{
			Success: false,
			Error:   &actualErr,
		})
	}
}
