package transport

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/RagOfJoes/idp/email"
	"github.com/RagOfJoes/idp/flow/recovery"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

var (
	errFailedCreate = func(src error) error {
		return transport.NewHttpClientError(src, http.StatusInternalServerError, "Recovery_FailedCreate", "Failed to create new recovery flow", nil)
	}
	errInvalidFlowID = func(src error, f string) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Recovery_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"FlowID": f,
		})
	}
	errInvalidFlow = func(src error, f recovery.Flow) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Recovery_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Flow": f,
		})
	}
	errInvalidPayload = func(src error, f recovery.Flow) error {
		return transport.NewHttpClientError(src, http.StatusNotFound, "Recovery_InvalidPayload", "Invalid payload provided", map[string]interface{}{
			"Flow": f,
		})
	}
)

type Http struct {
	e  email.Client
	s  recovery.Service
	is identity.Service
}

func NewRecoveryHttp(e email.Client, s recovery.Service, is identity.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		e:  e,
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
		// Check if user is already authenticated
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}

		requestURL := transport.RequestURL(c.Request)
		// Create new flow
		newFlow, err := h.s.New(requestURL)
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
		// Retrieve FlowID or RecoverID
		id := c.Param("id")
		// Retrieve Flow
		flow, err := h.s.Find(id)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, id), HttpCodeMap))
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
		// Check if user is already authenticated
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}
		// Retrieve FlowID or RecoverID
		id := c.Param("id")
		// Retrieve Flow
		flow, err := h.s.Find(id)
		if err != nil {
			c.Error(transport.GetHttpError(err, errInvalidFlowID(err, id), HttpCodeMap))
			return
		}

		switch flow.Status {
		case recovery.IdentifierPending:
			// Make sure user provided proper payload
			var payload recovery.IdentifierPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(errInvalidPayload(err, *flow))
				return
			}
			// Call service
			submittedFlow, err := h.s.SubmitIdentifier(*flow, payload)
			if err != nil {
				clientErr, ok := err.(internal.ClientError)
				if ok && clientErr.Title() == "Recovery_InvalidIdentifier" {
					// TODO: Capture Error Here
					// Return early so that we don't accidentally refer to a nil pointer
					c.JSON(http.StatusOK, transport.HttpResponse{
						Success: true,
						Payload: "Check your email for a link to reset your password. If it doesn’t appear within a few minutes, check your spam folder.",
					})
					return
				}
				c.Error(transport.GetHttpError(err, transport.NewHttpClientError(err, http.StatusInternalServerError, "Recovery_FailedSubmit", "Failed to submit flow. Please try again later", map[string]interface{}{
					"Flow":    flow,
					"Payload": payload,
				}), HttpCodeMap))
				return
			}
			// Send email in the background
			go func(flow recovery.Flow) {
				if submittedFlow.Status == recovery.LinkPending {
					var emails []string
					identity, err := h.is.Find(submittedFlow.IdentityID.String())
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
			}(*submittedFlow)

			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: "Check your email for a link to reset your password. If it doesn’t appear within a few minutes, check your spam folder.",
			})
		case recovery.LinkPending:
			// Make sure user provided proper payload
			var payload recovery.SubmitPayload
			if err := c.ShouldBind(&payload); err != nil {
				c.Error(errInvalidPayload(err, *flow))
				return
			}
			// Call service
			submittedFlow, err := h.s.SubmitUpdatePassword(*flow, payload)
			if err != nil {
				_, ok := err.(internal.ClientError)
				if ok {
					c.Error(err)
					return
				}
				_, file, line, _ := runtime.Caller(1)
				c.Error(transport.GetHttpError(err, transport.NewHttpInternalError(err, file, line, "Recovery_FailedSubmit", "Failed to submit flow. Please try again later", map[string]interface{}{
					"IdentityID": flow.IdentityID,
					"Flow":       flow,
					"Payload":    payload,
				}), HttpCodeMap))
				return
			}
			c.JSON(http.StatusOK, transport.HttpResponse{
				Success: true,
				Payload: submittedFlow,
			})
		default:
			c.Error(errInvalidFlow(err, *flow))
			return
		}
	}
}
