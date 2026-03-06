package user

import (
	"net/http"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// Register 用户注册
func Register(c *gin.Context) {
	var req userservice.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zlog.Info("Register err: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "参数绑定失败",
		})
		return
	}

	err := userservice.Register(&req)
	if err != nil {
		zlog.Info("Register service err: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "注册成功",
	})
}
