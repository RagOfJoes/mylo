package transport

import (
	"log"
	"net/http"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/validate"
	"github.com/gin-gonic/gin"
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

// ErrorMiddleware is a post middleware
// that handles errors for every requests
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		var actualError *idp.ClientError
		for _, err := range c.Errors {
			switch err.Err.(type) {
			case idp.ClientError:
				t := err.Err.(idp.ClientError)
				actualError = &t
			default:
				internalError, ok := err.Err.(idp.InternalError)
				if ok {
					// TODO: Capture error here with some
					// sort of error tracking service
					log.Print("Captured error: ", internalError.Error())
				}
			}
		}
		if actualError != nil {
			code, headers := (*actualError).Headers()
			for k, v := range headers {
				c.Header(k, v)
			}
			switch (*actualError).(type) {
			case *validate.FormError:
				wrap := (*actualError).(*validate.FormError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error: &HttpClientError{
						StatusCode:  code,
						Summary:     wrap.Title(),
						Description: wrap.Error(),
					},
				})
				return
			case *idp.ServiceClientError:
				wrap := (*actualError).(*idp.ServiceClientError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error: &HttpClientError{
						StatusCode:  code,
						Summary:     wrap.Title(),
						Description: wrap.Error(),
					},
				})
				return
			case *HttpClientError:
				wrap := (*actualError).(*HttpClientError)
				c.JSON(code, HttpResponse{
					Success: false,
					Error:   wrap,
				})
				return
			}
			// TODO: Capture error here
		}
		c.JSON(http.StatusInternalServerError, HttpResponse{
			Success: false,
			Error: &HttpClientError{
				Summary:     "internal_server_error",
				Description: "Oops! Something went wrong. Please try again later.",
			},
		})
	}
}
