package api

import (
	"context"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xmgrs/core"
	"github.com/gin-gonic/gin"
)

type Model struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// 创建错误信息
func NewModel(code int, msg interface{}) Model {
	err := Model{}
	err.Code = code
	switch msg.(type) {
	case error:
		err.Error = msg.(error).Error()
	case string:
		err.Error = msg.(string)
	default:
		err.Error = fmt.Sprintf("%v", msg)
	}
	return err
}

//获取默认gin引擎
func InitEngine(ctx context.Context) *gin.Engine {
	m := gin.New()
	m.Use(gin.Logger(), gin.Recovery())
	v1 := m.Group("/v1")
	v1.Use(core.AppHandler(ctx))
	ApiEntry(v1)
	return m
}

const (
	AppUserKey = "AppUserKey"
)

//获取用户id
func GetAppUserInfo(c *gin.Context) *core.TUser {
	return c.MustGet(AppUserKey).(*core.TUser)
}

func IsLogin(c *gin.Context) {
	args := struct {
		Token string `header:"X-Access-Token"`
	}{}
	if err := c.ShouldBindHeader(&args); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	app := core.GetApp(c)
	tk, err := app.DecryptToken(args.Token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	err = app.UseDb(func(db core.IDbImp) error {
		uid, err := db.GetUserId(tk)
		if err != nil {
			return err
		}
		oid, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			return err
		}
		user, err := db.GetUserInfo(oid)
		if err != nil {
			return err
		}
		c.Set(AppUserKey, user)
		return nil
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	c.Next()
}

func ApiEntry(g *gin.RouterGroup) {
	g.POST("/login", loginApi)
	a := g.Group("/", IsLogin)
	a.GET("/user/info", userInfoApi)
	a.GET("/user/coins", listCoinsApi)
}
