package transport

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gin-gonic/gin"
)

var (
	errFailedInit = func(src error) error {
		return transport.GetHttpError(src, transport.NewHttpClientError(src, http.StatusInternalServerError, "Login_FailedInit", "Failed to crete new login flow. Please try again later.", nil), HttpCodeMap)
	}
	errInvalidFlowID = func(src error, fid string) error {
		return transport.GetHttpError(src, transport.NewHttpClientError(src, http.StatusNotFound, "Login_InvalidFlow", "Invalid or expired flow", map[string]interface{}{"FlowID": fid}), HttpCodeMap)
	}
	errInvalidPayload = func(src error, f login.Flow) error {
		return transport.GetHttpError(src, transport.NewHttpClientError(src, http.StatusBadRequest, "Login_InvalidPayload", "Invalid identifier and/or password provided", map[string]interface{}{
			"Flow": f,
		}), HttpCodeMap)
	}
)

type Http struct {
	s  login.Service
	sm *session.Manager
}

func NewLoginHttp(s login.Service, sm *session.Manager, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		s:  s,
		sm: sm,
	}

	group := r.Group(fmt.Sprintf("/%s", cfg.Login.URL))
	{
		group.GET("/", h.initFlow())
		group.GET("/:flow_id", h.getFlow())
		group.POST("/:flow_id", h.submitFlow())
	}
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}

		reqURL := c.Request.URL.Path
		reqQuery := c.Request.URL.Query().Encode()
		fullURL := reqURL
		if len(reqQuery) > 0 {
			fullURL = fmt.Sprintf("%s?%s", reqURL, reqQuery)
		}
		newFlow, err := h.s.New(fullURL)
		if err != nil {
			c.Error(errFailedInit(err))
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
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}

		fid := c.Param("flow_id")
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(errInvalidFlowID(err, fid))
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
		if sess := transport.IsAuthenticated(c); sess != nil {
			c.Error(transport.ErrAlreadyAuthenticated(nil, c.Request.URL.Path, *sess.Identity))
			return
		}

		// Validate flow id
		fid := c.Param("flow_id")
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(errInvalidFlowID(err, fid))
			return
		}
		// Validate that all the required
		// inputs are present
		var dest login.Payload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload(err, *f))
			return
		}

		user, err := h.s.Submit(*f, dest)
		if err != nil {
			c.Error(errInvalidPayload(err, *f))
			return
		}

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

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
