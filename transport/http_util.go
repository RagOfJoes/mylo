package transport

import (
	"time"

	"github.com/RagOfJoes/idp/session"
	"github.com/gin-gonic/gin"
)

// IsAuthenticated checks context for identity
func IsAuthenticated(ctx *gin.Context) *session.Session {
	session, ok := ctx.Value("sess").(*session.Session)
	if !ok || session == nil {
		return nil
	}
	if session.ExpiresAt.Before(time.Now()) || session.Identity == nil {
		return nil
	}
	return session
}
