package session

import (
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware retrieves identity from session
// and passes it to context
func AuthMiddleware(sm *Manager, is identity.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sm.Retrieve(c.Request.Context(), true)
		if sess != nil {
			u, err := is.Find(sess.IdentityID.String())
			if u != nil && err == nil {
				sess.Identity = u
				sess.Contacts = u.Contacts
				c.Set("sess", sess)
			}
		}

		c.Next()
	}
}
