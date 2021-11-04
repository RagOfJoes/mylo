package transport

import (
	"net/http"

	sessionHttp "github.com/RagOfJoes/idp/session/transport"
	"github.com/RagOfJoes/idp/transport"
	"github.com/gin-gonic/gin"
)

type Http struct {
	sh sessionHttp.Http
}

func NewIdentityHttp(sh sessionHttp.Http, r *gin.Engine) {
	h := &Http{
		sh: sh,
	}
	r.GET("/me", h.me())
}

func (h *Http) me() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess, err := h.sh.Session(c.Request, c.Writer)
		if err != nil {
			c.Error(transport.ErrNotAuthenticated(err, c.Request.URL.Path))
			return
		}

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
