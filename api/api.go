package api

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xmgrs/core"
	"github.com/gin-gonic/gin"
)

// 创建错误信息
func NewError(code int, msg interface{}) *gin.Error {
	err := &gin.Error{}
	err.Meta = code
	switch msg.(type) {
	case error:
		err.Err = msg.(error)
	case string:
		err.Err = errors.New(msg.(string))
	default:
		err.Err = errors.New(fmt.Sprintf("%v", msg))
	}
	return err
}

//获取默认gin引擎
func InitEngine(ctx context.Context) *gin.Engine {
	m := gin.New()
	m.Use(gin.Logger(), gin.Recovery())
	m.Use(gin.ErrorLogger())
	v1 := m.Group("/v1")
	v1.Use(core.AppHandler(ctx))
	ApiEntry(v1)
	return m
}

const (
	AppUserIdKey = "AppUserIdKey"
)

//获取用户id
func GetAppUserId(c *gin.Context) primitive.ObjectID {
	return c.MustGet(AppUserIdKey).(primitive.ObjectID)
}

//获取用户信息
func GetAppUserInfo(db core.IDbImp, c *gin.Context) (*core.TUser, error) {
	uid := GetAppUserId(c)
	return db.GetUserInfo(uid)
}

func IsLogin(c *gin.Context) {
	args := struct {
		Token string `header:"X-Access-Token"`
	}{}
	if err := c.ShouldBindHeader(&args); err != nil {
		c.Error(NewError(1000, err))
		c.Abort()
		return
	}
	app := core.GetApp(c)
	tk, err := app.DecryptToken(args.Token)
	if err != nil {
		c.Error(NewError(1000, err))
		c.Abort()
		return
	}
	uid, err := app.GetUserId(tk)
	if err != nil {
		c.Error(NewError(1000, err))
		c.Abort()
		return
	}
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		c.Error(NewError(1000, err))
		c.Abort()
		return
	}
	c.Set(AppUserIdKey, oid)
	c.Next()
}

func ApiEntry(g *gin.RouterGroup) {
	g.POST("/login", loginApi)
	a := g.Group("/", IsLogin)
	a.GET("/user/info", userInfoApi)
	a.GET("/user/coins", listCoinsApi)
}
