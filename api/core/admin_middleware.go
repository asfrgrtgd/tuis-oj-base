package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// AdminOnly ensures the session role is admin.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionAny, _ := c.Get("session")
		sess, _ := sessionAny.(*sessions.Session)
		role, _ := sess.Values["role"].(string)
		if role != "admin" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "管理者権限が必要です")
			c.Abort()
			return
		}
		c.Next()
	}
}
