package transport

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RagOfJoes/mylo/email"
	"github.com/RagOfJoes/mylo/flow/recovery"
	"github.com/RagOfJoes/mylo/flow/registration"
	"github.com/RagOfJoes/mylo/flow/verification"
	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	sessionHttp "github.com/RagOfJoes/mylo/session/transport"
	"github.com/RagOfJoes/mylo/transport"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/RagOfJoes/mylo/user/identity"
	"github.com/gin-gonic/gin"
)

type Http struct {
	e  email.Client
	sh sessionHttp.Http
	s  registration.Service
	vs verification.Service
}

func NewRegistrationHttp(e email.Client, sh sessionHttp.Http, s registration.Service, vs verification.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		e:  e,
		sh: sh,
		vs: vs,
		s:  s,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Registration.URL))
	{
		group.GET("/", h.initFlow())
		group.GET("/:flow_id", h.getFlow())
		group.POST("/:flow_id", h.submitFlow())
	}
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		// Check if user is already authenticated
		if _, err := h.sh.Session(ctx, c.Request, c.Writer, true); err == nil {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", registration.ErrAlreadyAuthenticated))
			return
		}
		newFlow, err := h.s.New(ctx, transport.RequestURL(c.Request))
		if err != nil {
			c.Error(err)
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
		ctx := c.Request.Context()
		if _, err := h.sh.Session(ctx, c.Request, c.Writer, true); err == nil {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", recovery.ErrAlreadyAuthenticated))
			return
		}
		flowID := c.Param("flow_id")
		flow, err := h.s.Find(ctx, flowID)
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
		ctx := c.Request.Context()
		sess, _ := h.sh.SessionOrNewAndSetCookie(ctx, c.Request, c.Writer, false)
		if sess != nil && sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrAlreadyAuthenticated))
			return
		}
		// Retrieve flow id
		flowID := c.Param("flow_id")
		// Check if flow id provided is valid
		flow, err := h.s.Find(ctx, flowID)
		if err != nil {
			c.Error(err)
			return
		}
		// Check to see if required payload was provided
		var payload registration.Payload
		if err := c.ShouldBind(&payload); err != nil {
			c.Error(internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", registration.ErrInvalidPaylod))
			return
		}
		user, err := h.s.Submit(ctx, *flow, payload)
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
		if sess, err = h.sh.Upsert(ctx, *sess); err != nil {
			c.Error(err)
			return
		}

		// Create a new verification flow in the background
		// TODO: Look to add some dependency for callbacks on certain events
		go func(user identity.Identity) {
			vf, err := h.vs.NewDefault(ctx, user, user.Contacts[0], fmt.Sprintf("/registration/%s", flowID))
			if err != nil {
				// TODO: Capture error
				log.Print(err)
				return
			}
			cfg := config.Get()
			url := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, vf.FlowID)
			if err := h.e.SendWelcome(user.Contacts[0].Value, user, url); err != nil {
				// TODO: Capture error
				log.Print(err)
				return
			}
		}(*user)

		c.JSON(http.StatusCreated, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
