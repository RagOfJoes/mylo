package transport

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

const (
	verificationFlowIDCookie string = "verificationFid"
)

var (
	errNotAuthenticated error = transport.NewHttpClientError(http.StatusUnauthorized, "not_authenticated", "Not authenticated", nil)
	errInvalidFlow            = func(src error, fid string, i identity.Identity) error {
		return transport.NewHttpClientError(http.StatusNotFound, "verification_flowid_invalid", src.Error(), &map[string]interface{}{
			"FlowID":   fid,
			"Identity": i,
		})
	}
	errInvalidContactID = func(i identity.Identity, p string) error {
		return transport.NewHttpClientError(http.StatusBadRequest, "verification_payload_invalid", "Contact is either already verified or does not exist", &map[string]interface{}{
			"Identity":  i,
			"ContactID": p,
		})
	}
	errInvalidPayload = func(i identity.Identity, fid string) error {
		return transport.NewHttpClientError(http.StatusBadRequest, "verification_payload_invalid", "Invalid payload provided", &map[string]interface{}{
			"Identity": i,
			"FlowID":   fid,
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
		// Check if user is authenticated
		if sess == nil {
			c.Error(errNotAuthenticated)
			return
		}
		// Validate that payload required is provided
		var dest verification.NewPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidContactID(*sess.Identity, ""))
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
			c.Error(errInvalidContactID(*sess.Identity, dest.Contact))
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
		// TODO: Create configuration value for flow's duration
		stat := verification.LinkPending
		if time.Until(sess.ExpiresAt).Minutes()+10 < (10 / 2) {
			stat = verification.SessionWarn
		}
		// Create new flow
		newFlow, err := h.s.New(*sess.Identity, foundContact, fullURL, stat)
		if err != nil {
			c.Error(err)
			return
		}
		// Respond
		resp := transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (h *Http) getFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		if transport.IsAuthenticated(c) == nil {
			c.Error(errNotAuthenticated)
			return
		}
		// Validate flow id
		fid := c.Param("flow_id")
		// Get session
		sess, _ := c.Value("sess").(*session.Session)
		f, err := h.s.Find(fid, sess.IdentityID)
		if err != nil {
			c.Error(errInvalidFlow(err, fid, *sess.Identity))
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
			c.Error(errNotAuthenticated)
			return
		}
		// Validate flow id
		fid := c.Param("flow_id")
		var dest verification.SessionWarnPayload
		pastHalf := time.Until(sess.ExpiresAt).Minutes()+10 < (10 / 2)
		if pastHalf {
			if err := c.ShouldBind(&dest); err != nil {
				c.Error(errInvalidPayload(*sess.Identity, fid))
				return
			}
		}
		v, err := h.s.Verify(fid, *sess.Identity, dest)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: v,
		})
	}
}
