package transport

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/common"
	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gin-gonic/gin"
)

const (
	loginFlowIDCookie string = "loginFid"
)

var (
	errInit                 error = transport.NewHttpClientError(http.StatusInternalServerError, "login_init_fail", "Failed to initialize login flow", nil)
	errInvalidCSRFToken     error = transport.NewHttpClientError(http.StatusBadRequest, "invalid_csrf", "Invalid CSRF Token provided", nil)
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

		fidCookie, err := c.Cookie(loginFlowIDCookie)
		if err != http.ErrNoCookie && fidCookie != "" {
			f, err := h.s.Find(fidCookie)
			if err == nil {
				resp := transport.HttpResponse{
					Success: true,
					Payload: f,
				}
				c.JSON(http.StatusOK, resp)
				return
			}
		}

		c.SetCookie(loginFlowIDCookie, newFlow.FlowID, int(time.Until(newFlow.ExpiresAt)/time.Second), "/login", "", false, false)
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

		// Validate flow id
		fid := c.Param("flow_id")
		flow, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		// Validate that all the required
		// inputs are present
		var dest login.LoginPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}
		// Check the csrf token is valid
		// TODO: Determine whether or not to invalidate flow when an invalid CSRF token is passed
		if dest.CSRFToken != flow.CSRFToken {
			c.Error(errInvalidCSRFToken)
			return
		}

		user, err := h.s.Submit(fid, dest)
		if err != nil {
			c.Error(err)
			return
		}

		// Clone gin's raw Context to allow session manager to manipulate it
		// Update request with updated context
		cpy := c.Request.Context()
		if err := h.sm.PutAuth(cpy, *user, []credential.CredentialType{credential.Password}); err != nil {
			_, file, line, _ := runtime.Caller(1)
			c.Error(transport.NewHttpInternalError(file, line, "session_auth_put", "Failed to create a new Auth Session"))
			return
		}
		c.Request = c.Request.WithContext(cpy)

		sess := h.sm.GetAuth(c.Request.Context(), true)
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
