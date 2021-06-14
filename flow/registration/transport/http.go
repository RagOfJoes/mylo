package transport

import (
	"fmt"
	"net/http"

	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/transport"
	"github.com/gin-gonic/gin"
)

var (
	errInit           error = transport.NewHttpClientError(http.StatusInternalServerError, "registration_init_fail", "Failed to initialize registration flow", nil)
	errInvalidPayload error = transport.NewHttpClientError(http.StatusBadRequest, "registration_payload_invalid", "Invalid payload provided", nil)
)

type Http struct {
	s registration.Service
}

func NewRegistrationHttp(s registration.Service, r *gin.Engine) {
	h := &Http{
		s: s,
	}
	r.GET("/registration", h.initFlow())
	r.GET("/registration/:flow_id", h.getFlow())
	r.POST("/registration/:flow_id", h.submitFlow())
}

func (h *Http) initFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqURL := c.Request.URL.Path
		reqQuery := c.Request.URL.Query().Encode()
		fullURL := reqURL
		if len(reqQuery) > 0 {
			fullURL = fmt.Sprintf("%s?%s", reqURL, reqQuery)
		}
		newFlow, err := h.s.New(fullURL)
		if err != nil {
			c.Error(errInit)
			return
		}
		resp := transport.HttpResponse{
			Success: true,
			Payload: newFlow,
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (h *Http) getFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		fid := c.Param("flow_id")
		f, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, f)
	}
}

func (h *Http) submitFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Check if User is already
		// logged in
		fid := c.Param("flow_id")
		_, err := h.s.Find(fid)
		if err != nil {
			c.Error(err)
			return
		}
		var dest registration.RegistrationPayload
		if err := c.ShouldBind(&dest); err != nil {
			c.Error(errInvalidPayload)
			return
		}
		if err := h.s.Submit(fid, dest); err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
		})
	}
}
