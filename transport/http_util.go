package transport

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RagOfJoes/idp/session"
	"github.com/gin-gonic/gin"
)

// IsAuthenticated checks context for identity
func IsAuthenticated(ctx *gin.Context) *session.Session {
	v, ok := ctx.Get("sess")
	if !ok || v == nil {
		return nil
	}
	session, ok := v.(*session.Session)
	if !ok || session == nil {
		return nil
	}
	if session.ExpiresAt.Before(time.Now()) || session.Identity == nil {
		return nil
	}
	return session
}

// RequestURL retrieves entry path of request
func RequestURL(req *http.Request) string {
	path := req.URL.Path
	query := req.URL.Query().Encode()
	url := path
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", path, query)
	}
	return url
}
