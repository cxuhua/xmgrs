package api

import (
	"context"

	"github.com/cxuhua/xmgrs/db"
	"github.com/gin-gonic/gin"
)

//获取默认gin引擎

func InitHandler(ctx context.Context, islogin gin.HandlerFunc) *gin.Engine {
	m := gin.New()
	m.Use(gin.Logger(), gin.Recovery())

	v1 := m.Group("/v1")

	v1.Use(db.AppHandler(ctx))

	ApiEntry(v1, islogin)

	return m
}

const (
	AppUserKey = "AppUserKey"
)

func GetUserInfo(c *gin.Context) *db.TUsers {
	return c.MustGet(AppUserKey).(*db.TUsers)
}

func IsLogin(c *gin.Context) {

}

func ApiEntry(g *gin.RouterGroup, islogin gin.HandlerFunc) {

	g.POST("/login", loginApi)

	a := g.Group("/", islogin)

	a.GET("/user/info", userInfoApi)
}
