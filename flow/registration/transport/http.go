package transport

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RagOfJoes/idp/email"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	sessionHttp "github.com/RagOfJoes/idp/session/transport"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

var (
	errFailedCreate = func(src error) error {
		return transport.NewHttpClientError(src, http.StatusInternalServerError, "Registration_FailedCreate", "Failed to create new registration flow", nil)
	}
	errInvalidFlowID = func(src error, fid string) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Registration_InvalidFlow", src.Error(), map[string]interface{}{
			"FlowID": fid,
		})
	}
	errInvalidPayload = func(src error, f registration.Flow) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Registration_InvalidPayload", "Invalid payload provided", map[string]interface{}{
			"Flow": f,
		})
	}
	errFailedSubmit = func(src error, f registration.Flow, p registration.Payload) error {
		return transport.NewHttpClientError(src, http.StatusInternalServerError, "Registration_FailedSubmit", "Failed to submit registration flow. Please try again later.", map[string]interface{}{
			"Flow":    f,
			"Payload": p,
		})
	}
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}
		// Retrieve request URL
		reqURL := c.Request.URL.Path
		reqQuery := c.Request.URL.Query().Encode()
		fullURL := reqURL
		if len(reqQuery) > 0 {
			fullURL = fmt.Sprintf("%s?%s", reqURL, reqQuery)
		}
		// Create new flow
		newFlow, err := h.s.New(fullURL)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedCreate(err), HttpCodeMap))
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}
		// Retrieve FlowID
		fid := c.Param("flow_id")
		//
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, fid), HttpCodeMap))
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}
		// Retrieve flow id
		fid := c.Param("flow_id")
		// Check if flow id provided is valid
		existing, err := h.s.Find(fid)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, fid), HttpCodeMap))
			return
		}
		// Check to see if required payload was provided
		var payload registration.Payload
		if err := c.ShouldBind(&payload); err != nil {
			c.Error(errInvalidPayload(err, *existing))
			return
		}
		// Submit flow
		user, err := h.s.Submit(*existing, payload)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedSubmit(err, *existing, payload), HttpCodeMap))
			return
		}
		// Create new authenticated session
		sess, err := session.NewAuthenticated(*user, credential.Password)
		if err != nil {
			c.Error(err)
			return
		}
		if sess, err = h.sh.UpsertAndSetCookie(c.Request, c.Writer, *sess); err != nil {
			c.Error(err)
			return
		}
		// Create a new verification flow in the background
		go func(user identity.Identity) {
			vf, err := h.vs.NewDefault(user, user.Contacts[0], fmt.Sprintf("/registration/%s", fid))
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
