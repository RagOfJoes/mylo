package transport

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

var (
	errInvalidContactID = func(src error, i identity.Identity, c string) error {
		return transport.NewHttpClientError(src, http.StatusBadRequest, "Verification_InvalidContact", "Contact is either already verified or does not exist", map[string]interface{}{
			"Identity":  i,
			"ContactID": c,
		})
	}
	errInvalidContact = func(src error, i identity.Identity, c contact.Contact) error {
		return transport.NewHttpClientError(src, http.StatusForbidden, "Verification_InvalidContact", "Contact is either already verified or does not exist", map[string]interface{}{
			"Identity": i,
			"Contact":  c,
		})
	}
	errFailedCreate = func(src error, i identity.Identity, c contact.Contact) error {
		_, file, line, _ := runtime.Caller(1)
		return transport.NewHttpInternalError(src, file, line, "Verification_FailedCreate", "Failed to create verification flow. Please try again later", map[string]interface{}{
			"Identity": i,
			"Contact":  c,
		})
	}
	errInvalidFlowID = func(src error, i identity.Identity, fid string) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Verification_InvalidFlow", src.Error(), map[string]interface{}{
			"Identity": i,
			"FlowID":   fid,
		})
	}
	errInvalidPayload = func(src error, i identity.Identity, f verification.Flow) error {
		return transport.NewHttpClientError(src, http.StatusBadRequest, "Verification_InvalidPayload", "Invalid payload provided", map[string]interface{}{
			"Identity": i,
			"Flow":     f,
		})
	}
	errFailedVerify = func(src error, i identity.Identity, f verification.Flow, p interface{}) error {
		return transport.NewHttpClientError(src, http.StatusInternalServerError, "Verification_FailedVerify", "Failed to process verification flow. Please try again later.", map[string]interface{}{
			"Identity": i,
			"Flow":     f,
			"Payload":  p,
		})
	}
)

type Http struct {
	sm *session.Manager
	s  verification.Service
}

func NewVerificationHttp(sm *session.Manager, s verification.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		sm: sm,
		s:  s,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Verification.URL))
	{
		group.POST("/", h.initFlow())
		group.GET("/:flow_id", h.getFlow())
		group.POST("/:flow_id", h.verifyFlow())
	}
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := transport.IsAuthenticated(c)
		// Check if user is not authenticated
		if sess == nil {
			c.Error(transport.ErrNotAuthenticated(nil, c.Request.URL.Path))
			return
		}
		// Validate that payload required is provided
		var dest verification.NewPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidContactID(err, *sess.Identity, ""))
			return
		}
		// Check if payload provided is actually a user's contact id
		var foundContact contact.Contact
		for _, c := range sess.Contacts {
			if c.ID.String() == dest.Contact {
				foundContact = c
			}
		}
		if foundContact.Verified {
			c.Error(errInvalidContact(nil, *sess.Identity, foundContact))
			return
		}
		// Retrieve request URL
		reqURL := c.Request.URL.Path
		reqQuery := c.Request.URL.Query().Encode()
		fullURL := reqURL
		if len(reqQuery) > 0 {
			fullURL = fmt.Sprintf("%s?%s", reqURL, reqQuery)
		}
		// Get proper status for new flow
		stat := verification.LinkPending
		halfLife := sess.ExpiresAt.Sub(sess.AuthenticatedAt) / 2
		if time.Since(sess.AuthenticatedAt) >= halfLife {
			stat = verification.SessionWarn
		}
		// Create new flow
		newFlow, err := h.s.New(*sess.Identity, foundContact, fullURL, stat, false)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedCreate(err, *sess.Identity, foundContact), HttpCodeMap))
			return
		}
		// Respond
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		})
	}
}

func (h *Http) getFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := transport.IsAuthenticated(c)
		// Check if user is not authenticated
		if sess == nil {
			c.Error(transport.ErrNotAuthenticated(nil, c.Request.URL.Path))
			return
		}
		// Validate flow id
		fid := c.Param("flow_id")
		f, err := h.s.Find(fid, *sess.Identity)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, *sess.Identity, fid), HttpCodeMap))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: f,
		})
	}
}

func (h *Http) verifyFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := transport.IsAuthenticated(c)
		// Check if user is authenticated
		if sess == nil {
			c.Error(transport.ErrNotAuthenticated(nil, c.Request.URL.Path))
			return
		}
		// Retrieve flow id
		fid := c.Param("flow_id")
		// Check if flow id provided is valid
		f, err := h.s.Find(fid, *sess.Identity)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, *sess.Identity, fid), HttpCodeMap))
			return
		}
		// Require proper payload depending on status of flow
		var res *verification.Flow
		switch f.Status {
		case verification.SessionWarn:
			var psw verification.SessionWarnPayload
			if err := c.ShouldBind(&psw); err != nil {
				c.Error(errInvalidPayload(err, *sess.Identity, *f))
			}
			v, err := h.s.Verify(*f, *sess.Identity, psw)
			if err != nil {
				c.Error(transport.GetHttpError(err, errFailedVerify(err, *sess.Identity, *f, psw), HttpCodeMap))
				return
			}
			res = v
		default:
			v, err := h.s.Verify(*f, *sess.Identity, nil)
			if err != nil {
				c.Error(transport.GetHttpError(err, errFailedVerify(err, *sess.Identity, *f, nil), HttpCodeMap))
				return
			}
			res = v
		}
		// If for some reason we either haven't error'd out or flow has been left nil then just error out
		if res == nil {
			c.Error(errFailedVerify(nil, *sess.Identity, *f, nil))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: res,
		})
	}
}
