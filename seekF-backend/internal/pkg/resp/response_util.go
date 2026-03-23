package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, message string, data interface{}) {
	if data != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": message,
			"data":    data,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": message,
		})
	}
}

// Error 错误响应，code 默认值为 400
func Error(c *gin.Context, message string, code int) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}
