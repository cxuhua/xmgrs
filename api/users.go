package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func loginApi(c *gin.Context) {

}

func userInfoApi(c *gin.Context) {
	c.JSON(http.StatusOK, GetUserInfo(c))
}
