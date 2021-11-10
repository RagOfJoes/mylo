package transport

import (
	"fmt"
	"net/http"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	sessionHttp "github.com/RagOfJoes/idp/session/transport"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gin-gonic/gin"
)

type Http struct {
	sh sessionHttp.Http
	s  login.Service
}

func NewLoginHttp(sh sessionHttp.Http, s login.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		sh: sh,
		s:  s,
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
		sess, _ := h.sh.SessionOrNewAndSetCookie(c.Request, c.Writer, false)
		if sess != nil && sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrAlreadyAuthenticated))
			return
		}
		fullURL := transport.RequestURL(c.Request)
		newFlow, err := h.s.New(fullURL)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusCreated, transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		})
	}
}

func (h *Http) getFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := h.sh.Session(c.Request, c.Writer, true); err == nil {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrAlreadyAuthenticated))
			return
		}
		flowID := c.Param("flow_id")
		flow, err := h.s.Find(flowID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: flow,
		})
	}
}

func (h *Http) submitFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess, _ := h.sh.SessionOrNewAndSetCookie(c.Request, c.Writer, false)
		if sess != nil && sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrAlreadyAuthenticated))
			return
		}
		// Validate flow id
		flowID := c.Param("flow_id")
		flow, err := h.s.Find(flowID)
		if err != nil {
			c.Error(err)
			return
		}
		// Check to see if required payload was provided
		var payload login.Payload
		if err := c.ShouldBind(&payload); err != nil {
			c.Error(internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod))
			return
		}
		user, err := h.s.Submit(*flow, payload)
		if err != nil {
			c.Error(err)
			return
		}
		// Authenticate session with password credential method
		if err := sess.Authenticate(*user, credential.Password); err != nil {
			c.Error(err)
			return
		}
		// Save session
		if sess, err = h.sh.Upsert(*sess); err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
