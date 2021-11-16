package transport

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RagOfJoes/idp/email"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	sessionHttp "github.com/RagOfJoes/idp/session/transport"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

type Http struct {
	e  email.Client
	sh sessionHttp.Http
	s  verification.Service
}

func NewVerificationHttp(e email.Client, sh sessionHttp.Http, s verification.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		e:  e,
		sh: sh,
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
		ctx := c.Request.Context()
		sess, err := h.sh.SessionOrNewAndSetCookie(ctx, c.Request, c.Writer, false)
		if err != nil {
			c.Error(err)
			return
		} else if !sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrUnauthorized))
			return
		}
		// - Check if contact provided actually belongs to the user
		// - Check if session has passed its half life
		contactID := c.Param("contact_id")
		var foundContact contact.Contact
		if got := getContact(*sess.Identity, contactID); got != nil {
			foundContact = *got
		} else {
			c.Error(internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", verification.ErrInvalidContact))
			return
		}
		requestURL := transport.RequestURL(c.Request)
		halfLife := sess.ExpiresAt.Sub(*sess.AuthenticatedAt) / 2
		if time.Since(*sess.AuthenticatedAt) >= halfLife {
			newFlow, err := h.s.NewSessionWarn(ctx, *sess.Identity, foundContact, requestURL)
			if err != nil {
				c.Error(err)
				return
			}
			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: newFlow,
			})
			return
		}

		newFlow, err := h.s.NewDefault(ctx, *sess.Identity, foundContact, requestURL)
		if err != nil {
			c.Error(err)
			return
		}

		// Send verification email in the background
		// TODO: Look to add some dependency for callbacks on certain events
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
		ctx := c.Request.Context()
		sess, err := h.sh.SessionOrNewAndSetCookie(ctx, c.Request, c.Writer, false)
		if err != nil {
			c.Error(err)
			return
		} else if !sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrUnauthorized))
			return
		}

		id := c.Param("id")
		flow, err := h.s.Find(ctx, id, *sess.Identity)
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

func (h *Http) verifyFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sess, err := h.sh.SessionOrNewAndSetCookie(ctx, c.Request, c.Writer, false)
		if err != nil {
			c.Error(err)
			return
		} else if !sess.Authenticated() {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", internal.ErrUnauthorized))
			return
		}

		id := c.Param("id")
		flow, err := h.s.Find(ctx, id, *sess.Identity)
		if err != nil {
			c.Error(err)
			return
		}

		switch flow.Status {
		case verification.SessionWarn:
			var payload verification.SessionWarnPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "Must provide password"))
			}
			submittedFlow, err := h.s.SubmitSessionWarn(ctx, *flow, *sess.Identity, payload)
			if err != nil {
				c.Error(err)
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

		verified, err := h.s.Verify(ctx, *flow, *sess.Identity)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: verified,
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
