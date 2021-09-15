package transport

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

const (
	registrationFlowIDCookie string = "registrationFid"
)

var (
	errInit                 error = transport.NewHttpClientError(http.StatusInternalServerError, "registration_init_fail", "Failed to initialize registration flow", nil)
	errAlreadyAuthenticated error = transport.NewHttpClientError(http.StatusForbidden, "already_authenticated", "Cannot register since you're already authenticated", nil)
	errInvalidPayload       error = transport.NewHttpClientError(http.StatusBadRequest, "registration_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	sm *session.Manager
	s  registration.Service
	vs verification.Service
}

func NewRegistrationHttp(s registration.Service, vs verification.Service, sm *session.Manager, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		s:  s,
		vs: vs,
		sm: sm,
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
		if transport.IsAuthenticated(c) != nil {
			c.Error(errAlreadyAuthenticated)
			return
		}

		fidCookie, err := c.Cookie(registrationFlowIDCookie)
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

		c.SetCookie(registrationFlowIDCookie, newFlow.FlowID, int(time.Until(newFlow.ExpiresAt)/time.Second), "/registration", "", false, false)
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

func (h *Http) submitFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if transport.IsAuthenticated(c) != nil {
			c.Error(errAlreadyAuthenticated)
			return
		}

		// Validate flow id
		fid := c.Param("flow_id")
		// Validate that all the required
		// inputs are present
		var dest registration.Payload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}
		// Submit flow
		user, err := h.s.Submit(fid, dest)
		if err != nil {
			c.Error(err)
			return
		}
		// Create a new verification flow
		go func(user identity.Identity) {
			_, err := h.vs.NewWelcome(user, user.Contacts[0], fmt.Sprintf("/registration/%s", fid))
			if err != nil {
				// TODO: Capture error
				log.Print(err)
			}
		}(*user)

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

		c.JSON(http.StatusCreated, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
