package core

import "github.com/gin-gonic/gin"

// respondError sends unified error payload {"error": {"code", "message"}}.
func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
