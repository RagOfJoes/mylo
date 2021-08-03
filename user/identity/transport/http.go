package transport

import (
	"net/http"

	"github.com/RagOfJoes/idp/common"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/gin-gonic/gin"
)

var (
	errInit                 error = transport.NewHttpClientError(http.StatusInternalServerError, "registration_init_fail", "Failed to initialize registration flow", nil)
	errNotAuthenticated     error = transport.NewHttpClientError(http.StatusUnauthorized, "not_authenticated", "Not authenticated", nil)
	errAlreadyAuthenticated error = transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Cannot register since you're already authenticated", nil)
	errInvalidPayload       error = transport.NewHttpClientError(http.StatusBadRequest, "registration_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	sm *session.Manager
}

func NewIdentityHttp(sm *session.Manager, r *gin.Engine) {
	h := &Http{
		sm: sm,
	}
	r.GET("/me", h.me())
}

func (h *Http) me() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.IsAuthenticated(c) {
			c.Error(errNotAuthenticated)
			return
		}

		sess := h.sm.GetAuth(c.Request.Context(), true)
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
