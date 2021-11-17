package transport

import (
	"net/http"

	"github.com/RagOfJoes/mylo/internal"
	sessionHttp "github.com/RagOfJoes/mylo/session/transport"
	"github.com/RagOfJoes/mylo/transport"
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
		sess, err := h.sh.Session(c.Request.Context(), c.Request, c.Writer, true)
		if err != nil || sess == nil {
			c.Error(internal.WrapErrorf(err, internal.ErrorCodeUnauthorized, "%v", internal.ErrUnauthorized))
			return
		}

		c.JSON(http.StatusOK, transport.HttpResponse{
			Success: true,
			Payload: sess,
		})
	}
}
