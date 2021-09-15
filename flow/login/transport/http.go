package transport

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal/config"
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
	errAlreadyAuthenticated error = transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Cannot login since you're already authenticated", nil)
	errInvalidPayload       error = transport.NewHttpClientError(http.StatusBadRequest, "login_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	s  login.Service
	sm *session.Manager
}

func NewLoginHttp(s login.Service, sm *session.Manager, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		s:  s,
		sm: sm,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Login.URL))
	{
		group.GET("/", h.initFlow())
		group.GET("/:flow_id", h.getFlow())
		group.POST("/:flow_id", h.submitFlow())
	}
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if transport.IsAuthenticated(c) != nil {
			c.Error(errAlreadyAuthenticated)
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
		if transport.IsAuthenticated(c) != nil {
			c.Error(errAlreadyAuthenticated)
			return
		}

		// Validate flow id
		fid := c.Param("flow_id")
		_, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		// Validate that all the required
		// inputs are present
		var dest login.Payload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}

		user, err := h.s.Submit(fid, dest)
		if err != nil {
			c.Error(err)
			return
		}

		// Clone gin's raw Context to allow session manager to manipulate it
		// Then update request with updated context
		cpy := c.Request.Context()
		sess, err := h.sm.Insert(cpy, user, []credential.CredentialType{credential.Password})
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			c.Error(transport.NewHttpInternalError(file, line, "session_insert", "Failed to create a new Auth Session"))
			return
		}
		c.Request = c.Request.WithContext(cpy)

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
