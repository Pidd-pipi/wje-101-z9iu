package router

import (
	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/handler"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
)

func registerBeanRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.BeanHandler, limiter *middleware.RateLimiter) {
	beans := v1.Group("/beans")
	// Browsing stays public; an optional identity attaches personal list state.
	beans.GET("", middleware.AuthOptional(cfg), h.List)
	auth := beans.Group("", middleware.AuthRequired(cfg))
	auth.POST("/:id/list", limiter.Limit(), h.AddToList)
	auth.DELETE("/:id/list", h.RemoveFromList)
	admin := beans.Group("", middleware.AuthRequired(cfg), middleware.RequireRole("admin"))
	admin.POST("", limiter.Limit(), h.Create)
	admin.PUT("/:id", h.Update)
	admin.DELETE("/:id", h.Delete)
}
