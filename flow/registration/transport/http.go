package transport

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
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
		// Check if user is already authenticated
		if sess := transport.IsAuthenticated(c); sess != nil {
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
		if sess := transport.IsAuthenticated(c); sess != nil {
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
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}
		// Retrieve flow id
		fid := c.Param("flow_id")
		// Check if flow id provided is valid
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, fid), HttpCodeMap))
			return
		}
		// Check to see if required payload was provided
		var dest registration.Payload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload(err, *f))
			return
		}
		// Submit flow
		user, err := h.s.Submit(*f, dest)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedSubmit(err, *f, dest), HttpCodeMap))
			return
		}
		// Create a new verification flow
		go func(user identity.Identity) {
			_, err := h.vs.New(user, user.Contacts[0], fmt.Sprintf("/registration/%s", fid), verification.LinkPending, true)
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
			c.Error(transport.NewHttpInternalError(err, file, line, "Session_FailedInsert", "Failed to insert new session into session store", map[string]interface{}{
				"Flow":     f,
				"Identity": user,
			}))
			return
		}
		c.Request = c.Request.WithContext(cpy)

		c.JSON(http.StatusCreated, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
