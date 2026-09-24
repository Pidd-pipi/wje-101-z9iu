package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// OptionalAuth injects user claims when a valid Bearer token is present,
// but lets the request continue anonymously otherwise. Used by public
// endpoints that enrich their payload for logged-in viewers.
func OptionalAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			if claims, err := util.ParseToken(strings.TrimPrefix(header, "Bearer "), cfg.JWTSecret); err == nil {
				c.Set(UserKey, claims)
			}
		}
		c.Next()
	}
}
