package transport

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
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
		// Traverse errors and retrieve last ClientError
		// that was generated
		var actualError *internal.ClientError
		for _, err := range c.Errors {
			switch err.Err.(type) {
			case internal.ClientError:
				t := err.Err.(internal.ClientError)
				actualError = &t
			default:
				internalError, ok := err.Err.(internal.InternalError)
				if ok {
					// TODO: Capture error here with some
					// sort of error tracking service
					log.Print("Captured error: ", internalError.Error())
				}
			}
		}
		if actualError != nil {
			// Pass any special Headers on to response
			code, headers := (*actualError).Headers()
			for k, v := range headers {
				c.Header(k, v)
			}
			// Cast specific error type to map proper information
			// to response
			switch (*actualError).(type) {
			case *validate.FormatError:
				wrap := (*actualError).(*validate.FormatError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error: &HttpClientError{
						StatusCode:  code,
						Summary:     wrap.Title(),
						Description: wrap.Error(),
					},
				})
			case *internal.ServiceClientError:
				wrap := (*actualError).(*internal.ServiceClientError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error: &HttpClientError{
						StatusCode:  code,
						Summary:     wrap.Title(),
						Description: wrap.Error(),
					},
				})
			case *HttpClientError:
				wrap := (*actualError).(*HttpClientError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error:   wrap,
				})
			}
			// TODO: Capture error
			return
		}

		// If nothing was hit then respond with a 500 and capture
		// relevant info
		//
		// TODO: Capture error
		c.JSON(http.StatusInternalServerError, HttpResponse{
			Success: false,
			Error: &HttpClientError{
				Summary:     "internal_server_error",
				Description: "Oops! Something went wrong. Please try again later.",
			},
		})
	}
}
