package transport

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/RagOfJoes/idp/common"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gin-gonic/gin"
)

var (
	errInit           error = transport.NewHttpClientError(http.StatusInternalServerError, "registration_init_fail", "Failed to initialize registration flow", nil)
	errInvalidPayload error = transport.NewHttpClientError(http.StatusBadRequest, "registration_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	sm *session.Manager
	s  registration.Service
}

func NewRegistrationHttp(s registration.Service, sm *session.Manager, r *gin.Engine) {
	h := &Http{
		s:  s,
		sm: sm,
	}
	r.GET("/registration", h.initFlow())
	r.GET("/registration/:flow_id", h.getFlow())
	r.POST("/registration/:flow_id", h.submitFlow(sm))
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsAuthenticated(c) {
			c.Error(transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Already authenticated", nil))
			return
		}

		reqURL := c.Request.URL.Path
		reqQuery := c.Request.URL.Query().Encode()
		fullURL := reqURL
		if len(reqQuery) > 0 {
			fullURL = fmt.Sprintf("%s?%s", reqURL, reqQuery)
		}
		newFlow, err := h.s.New(fullURL)
		if err != nil {
			c.Error(errInit)
			return
		}
		resp := transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (h *Http) getFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		fid := c.Param("flow_id")
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		resp := transport.HttpResponse{
			Success: true,
			Payload: f,
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (h *Http) submitFlow(sm *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsAuthenticated(c) {
			c.Error(transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Already authenticated", nil))
			return
		}

		fid := c.Param("flow_id")
		_, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		var dest registration.RegistrationPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}
		user, err := h.s.Submit(fid, dest)
		if err != nil {
			c.Error(err)
			return
		}
		if err := sm.PutIdentity(c.Request.Context(), *user, []credential.CredentialType{credential.Password}); err != nil {
			_, file, line, _ := runtime.Caller(1)
			c.Error(transport.NewHttpInternalError(file, line, "session_identity_put", "Failed to add new Identity to Session"))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
		})
	}
}
