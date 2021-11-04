package transport

import (
	"fmt"
	"net/http"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/session"
	sessionHttp "github.com/RagOfJoes/idp/session/transport"
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
	sh sessionHttp.Http
}

func NewLoginHttp(sh sessionHttp.Http, s login.Service, r *gin.Engine) {
	cfg := config.Get()
	h := &Http{
		s:  s,
		sh: sh,
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
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
		// Check if user is already authenticated
		if sess, err := h.sh.Session(c.Request, c.Writer); err == nil {
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
		var payload login.Payload
		if err := c.ShouldBind(&payload); err != nil {
			c.Error(errInvalidPayload(err, *f))
			return
		}
		user, err := h.s.Submit(*f, payload)
		if err != nil {
			c.Error(errInvalidPayload(err, *f))
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

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
