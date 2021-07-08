package transport

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/RagOfJoes/idp/common"
	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gin-gonic/gin"
)

var (
	errInit                 error = transport.NewHttpClientError(http.StatusInternalServerError, "login_init_fail", "Failed to initialize login flow", nil)
	errAlreadyAuthenticated error = transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Cannot login since you're already authenticated", nil)
	errInvalidPayload       error = transport.NewHttpClientError(http.StatusBadRequest, "login_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	s  login.Service
	sm *session.Manager
}

func NewLoginHttp(s login.Service, sm *session.Manager, r *gin.Engine) {
	h := &Http{
		s:  s,
		sm: sm,
	}
	r.GET("/login", h.initFlow())
	r.GET("/login/:flow_id", h.getFlow())
	r.POST("/login/:flow_id", h.submitFlow())
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		})
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
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: f,
		})
	}
}

func (h *Http) submitFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsAuthenticated(c) {
			c.Error(errAlreadyAuthenticated)
			return
		}

		fid := c.Param("flow_id")
		_, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		var dest login.LoginPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}
		user, err := h.s.Submit(fid, dest)
		if err != nil {
			c.Error(err)
			return
		}
		if err := h.sm.PutAuth(c.Request.Context(), *user, []credential.CredentialType{credential.Password}); err != nil {
			_, file, line, _ := runtime.Caller(1)
			c.Error(transport.NewHttpInternalError(file, line, "session_auth_put", "Failed to create a new Auth Session"))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
		})
	}
}
