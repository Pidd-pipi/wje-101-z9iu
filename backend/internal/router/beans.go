package router

import (
	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/handler"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
)

func registerBeanRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.BeanHandler, ubh *handler.UserBeanHandler, limiter *middleware.RateLimiter) {
	beans := v1.Group("/beans")
	// Anonymous browsing is allowed; logged-in viewers get their tracking state.
	beans.GET("", middleware.OptionalAuth(cfg), h.List)
	// Authenticated per-user to-drink list actions.
	auth := beans.Group("/:id", middleware.AuthRequired(cfg))
	auth.POST("/want", limiter.Limit(), ubh.Add)
	auth.DELETE("/want", ubh.Remove)
	admin := beans.Group("", middleware.AuthRequired(cfg), middleware.RequireRole("admin"))
	admin.POST("", limiter.Limit(), h.Create)
	admin.PUT("/:id", h.Update)
	admin.DELETE("/:id", h.Delete)
}
