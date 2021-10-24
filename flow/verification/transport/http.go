package transport

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/email"
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
	e  email.Client
	sm *session.Manager
	s  verification.Service
}

func NewVerificationHttp(e email.Client, sm *session.Manager, s verification.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		e:  e,
		sm: sm,
		s:  s,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Verification.URL))
	{
		group.GET("/:contact_id", h.initFlow())
		group.GET("/retrieve/:id", h.getFlow())
		group.POST("/:id", h.verifyFlow())
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
		// Check if contact provided actually belongs to the user
		contactID := c.Param("contact_id")
		var foundContact contact.Contact
		if got := getContact(*sess.Identity, contactID); got != nil {
			foundContact = *got
		} else {
			c.Error(errInvalidContactID(nil, *sess.Identity, contactID))
			return
		}

		requestURL := transport.RequestURL(c.Request)
		// Check if session has passed its half-life
		halfLife := sess.ExpiresAt.Sub(sess.AuthenticatedAt) / 2
		if time.Since(sess.AuthenticatedAt) >= halfLife {
			newFlow, err := h.s.NewSessionWarn(*sess.Identity, foundContact, requestURL)
			if err != nil {
				c.Error(transport.GetHttpError(err, errFailedCreate(err, *sess.Identity, foundContact), HttpCodeMap))
				return
			}
			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: newFlow,
			})
			return
		}

		// Create new flow
		newFlow, err := h.s.NewDefault(*sess.Identity, foundContact, requestURL)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedCreate(err, *sess.Identity, foundContact), HttpCodeMap))
			return
		}
		// Send email in the background
		go func(identity identity.Identity, flow verification.Flow, contact contact.Contact) {
			// TODO: Capture error here
			if err := h.sendEmail(identity, flow, contact); err != nil {
				log.Print("Failed to send verification email: ", err)
			}
		}(*sess.Identity, *newFlow, foundContact)

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
		// Retrieve FlowID or VerifyID
		id := c.Param("id")
		flow, err := h.s.Find(id, *sess.Identity)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, *sess.Identity, id), HttpCodeMap))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: flow,
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
		// Retrieve FlowID or VerifyID
		id := c.Param("id")
		// Retrieve flow
		flow, err := h.s.Find(id, *sess.Identity)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, *sess.Identity, id), HttpCodeMap))
			return
		}

		switch flow.Status {
		case verification.SessionWarn:
			var payload verification.SessionWarnPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(errInvalidPayload(err, *sess.Identity, *flow))
			}
			submittedFlow, err := h.s.SubmitSessionWarn(*flow, *sess.Identity, payload)
			if err != nil {
				c.Error(transport.GetHttpError(err, errFailedVerify(err, *sess.Identity, *flow, payload), HttpCodeMap))
				return
			}

			// If status was updated then send email in the background
			go func(i identity.Identity, f verification.Flow) {
				if submittedFlow.Status == verification.LinkPending {
					var foundContact contact.Contact
					if got := getContact(*sess.Identity, flow.ContactID.String()); got != nil {
						foundContact = *got
					} else {
						// TODO: Capture error here
						return
					}

					// TODO: Capture error here
					if err := h.sendEmail(i, f, foundContact); err != nil {
						log.Print("Failed to send verification email: ", err)
					}
				}
			}(*sess.Identity, *submittedFlow)

			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: submittedFlow,
			})
			return
		}

		verifiedFlow, err := h.s.Verify(*flow, *sess.Identity)
		if err != nil {
			c.Error(transport.GetHttpError(err, errFailedVerify(err, *sess.Identity, *flow, nil), HttpCodeMap))
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: verifiedFlow,
		})
	}
}

func getContact(identity identity.Identity, contactID string) *contact.Contact {
	var foundContact *contact.Contact
	for _, c := range identity.Contacts {
		if c.ID.String() == contactID {
			foundContact = &c
		}
	}
	if foundContact != nil && foundContact.Verified {
		return nil
	}
	return foundContact
}

func (h *Http) sendEmail(identity identity.Identity, flow verification.Flow, contact contact.Contact) error {
	cfg := config.Get()
	url := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, flow.VerifyID)
	return h.e.SendVerification(contact.Value, identity, url)
}
