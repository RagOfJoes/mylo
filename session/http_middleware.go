package session

import (
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware retrieves identity from session
// and passes it to context
func AuthMiddleware(sm *Manager, is identity.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sess := sm.Retrieve(ctx, true)
		if sess != nil {
			u, err := is.Find(sess.IdentityID.String())
			// If user does not exist anymore
			// - Update session retrieved to nil
			// - Remove session from manager
			// - Update context of request
			if err != nil || u == nil {
				sess = nil
				sm.Remove(ctx, "sess")
				c.Request = c.Request.WithContext(ctx)
			} else if u != nil && err == nil {
				sess.Identity = u
				sess.Contacts = u.Contacts
			}
		}
		c.Set("sess", sess)
		c.Next()
	}
}
