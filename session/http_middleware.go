package session

import (
	"github.com/gin-gonic/gin"
)

// AuthMiddleware retrieves identity from session
// and passes it to context
func AuthMiddleware(sm *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := sm.GetIdentity(c.Request.Context(), true)
		c.Set("session_identity", id)

		c.Next()
	}
}
