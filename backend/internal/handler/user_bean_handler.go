package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/service"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// UserBeanHandler exposes the per-user to-drink list endpoints.
type UserBeanHandler struct {
	svc    *service.UserBeanService
	logger *slog.Logger
}

// NewUserBeanHandler creates a UserBeanHandler.
func NewUserBeanHandler(svc *service.UserBeanService, logger *slog.Logger) *UserBeanHandler {
	return &UserBeanHandler{svc: svc, logger: logger}
}

// Add handles POST /beans/:id/want — put the bean into the viewer's list.
func (h *UserBeanHandler) Add(c *gin.Context) {
	beanID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid bean id"))
		return
	}
	ub, err := h.svc.Add(middleware.GetUserID(c), uint(beanID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(gin.H{"id": ub.ID, "bean_id": ub.BeanID, "track_status": constants.BeanStatusWant}))
}

// Remove handles DELETE /beans/:id/want — move the bean out of the 待喝 list.
func (h *UserBeanHandler) Remove(c *gin.Context) {
	beanID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid bean id"))
		return
	}
	if err := h.svc.Remove(middleware.GetUserID(c), uint(beanID)); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"removed": true}))
}
