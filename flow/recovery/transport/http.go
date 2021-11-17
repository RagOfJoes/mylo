package transport

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/RagOfJoes/mylo/email"
	"github.com/RagOfJoes/mylo/flow/recovery"
	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	sessionHttp "github.com/RagOfJoes/mylo/session/transport"
	"github.com/RagOfJoes/mylo/transport"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/identity"
	"github.com/gin-gonic/gin"
)

type Http struct {
	e  email.Client
	sh sessionHttp.Http
	s  recovery.Service
	is identity.Service
}

func NewRecoveryHttp(e email.Client, sh sessionHttp.Http, s recovery.Service, is identity.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		e:  e,
		sh: sh,
		s:  s,
		is: is,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Recovery.URL))
	{
		group.GET("/", h.initFlow())
		group.GET("/:id", h.getFlow())
		group.POST("/:id", h.submitFlow())
	}
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if _, err := h.sh.Session(ctx, c.Request, c.Writer, true); err == nil {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", recovery.ErrAlreadyAuthenticated))
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

		id := c.Param("id")
		flow, err := h.s.Find(ctx, id)
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
		if _, err := h.sh.Session(ctx, c.Request, c.Writer, true); err == nil {
			c.Error(internal.NewErrorf(internal.ErrorCodeForbidden, "%v", recovery.ErrAlreadyAuthenticated))
			return
		}

		id := c.Param("id")
		flow, err := h.s.Find(ctx, id)
		if err != nil {
			c.Error(err)
			return
		}

		switch flow.Status {
		case recovery.IdentifierPending:
			var payload recovery.IdentifierPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", recovery.ErrInvalidIdentifierPaylod))
				return
			}
			submitted, err := h.s.SubmitIdentifier(ctx, *flow, payload)
			if err != nil && !errors.Is(recovery.ErrAccountDoesNotExist, err) {
				c.Error(err)
				return
			}

			// Send recovery email in the background
			// TODO: Look to add some dependency for callbacks on certain events
			go func(flow recovery.Flow) {
				if submitted.Status == recovery.LinkPending {
					var emails []string
					identity, err := h.is.Find(ctx, submitted.IdentityID.String())
					if err != nil {
						// TODO: Capture Error Here
						return
					}

					if len(identity.Contacts) == 1 {
						emails = append(emails, identity.Contacts[0].Value)
					} else {
						for _, c := range identity.Contacts {
							if c.Type == contact.Backup && c.Verified && c.State == contact.Completed {
								emails = append(emails, c.Value)
							}
						}
					}
					cfg := config.Get()
					recoveryURL := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Recovery.URL, flow.FlowID)
					if err := h.e.SendRecovery(emails, recoveryURL); err != nil {
						// TODO: Capture Error Here
						log.Print(err)
					}
				}
			}(*submitted)

			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: "Check your email for a link to reset your password. If it doesn’t appear within a few minutes, check your spam folder.",
			})
		case recovery.LinkPending:
			var payload recovery.SubmitPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", err))
				return
			}

			submitted, err := h.s.SubmitUpdatePassword(ctx, *flow, payload)
			if err != nil {
				c.Error(err)
				return
			}

			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: submitted,
			})
		default:
			c.Error(internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow))
			return
		}
	}
}
