package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/cxuhua/xginx"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xmgrs/core"
	"github.com/gin-gonic/gin"
)

//Model 通用返回
type Model struct {
	Code  int    `json:"code"`
	Error string `json:"error,omitempty"`
}

//NewModel 创建错误信息
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

//IsAddress 检测是否是地址
func IsAddress(fl validator.FieldLevel) bool {
	v := fl.Field().String()
	_, err := xginx.DecodeAddress(xginx.Address(v))
	return err == nil
}

//HexHash160 字段是否是hash160 hex字符串
func HexHash160(fl validator.FieldLevel) bool {
	v, err := hex.DecodeString(fl.Field().String())
	return err == nil && len(v) == len(xginx.HASH160{})
}

//HexHash256 字段是否是hash256 hex字符串
func HexHash256(fl validator.FieldLevel) bool {
	v, err := hex.DecodeString(fl.Field().String())
	return err == nil && len(v) == len(xginx.HASH256{})
}

//IsScript 字段是否是合法的脚本
func IsScript(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	err := xginx.CheckScript([]byte(str))
	return err == nil
}

//InitEngine 获取默认gin引擎
func InitEngine(ctx context.Context) *gin.Engine {
	//注册自定义校验器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("HexHash256", HexHash256)
		v.RegisterValidation("HexHash160", HexHash160)
		v.RegisterValidation("IsAddress", IsAddress)
		v.RegisterValidation("IsScript", IsScript)
	}
	//
	m := gin.New()
	m.Use(gin.Logger(), gin.Recovery())
	v1 := m.Group("/v1")
	v1.Use(core.AppHandler(ctx))
	V1Entry(v1)
	return m
}

//app key 定义
const (
	AppUserIDKey = "AppUserIDKey"
)

//GetAppUserID 获取用户id
func GetAppUserID(c *gin.Context) primitive.ObjectID {
	return c.MustGet(AppUserIDKey).(primitive.ObjectID)
}

//IsLogin 是否登陆
func IsLogin(c *gin.Context) {
	app := core.GetApp(c)
	args := struct {
		Token string `header:"X-Access-Token" binding:"required"`
	}{}
	if err := c.ShouldBindHeader(&args); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	tk, err := app.DecryptToken(args.Token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	err = app.UseRedis(func(redv core.IRedisImp) error {
		oid, err := redv.GetUserID(tk)
		if err != nil {
			return err
		}
		c.Set(AppUserIDKey, oid)
		return nil
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	c.Next()
}

//V1Entry v1接口初始化
func V1Entry(rg *gin.RouterGroup) {
	rg.POST("/register", registerAPI)
	rg.POST("/login", loginAPI)

	auth := rg.Group("/", IsLogin)
	auth.POST("/set/pushid", setUserPushIDAPI)
	auth.GET("/quit/login", quitLoginAPI)
	auth.GET("/user/info", userInfoAPI)
	auth.GET("/user/coins", listCoinsAPI)
	auth.GET("/tx/info/:id", getTxInfoAPI)
	auth.GET("/list/txs/:addr", listTxsAPI)
	auth.GET("/list/accounts", listUserAccountsAPI)
	auth.GET("/list/sign/txs", listUserSignTxsAPI)
	auth.GET("/list/privates", listPrivatesAPI)
	auth.POST("/new/private", createUserPrivateAPI)
	auth.POST("/new/account", createAccountAPI)
	auth.POST("/new/tx", createTxAPI)
	auth.POST("/sign/tx", signTxAPI)
	auth.POST("/submit/tx", submitTxAPI)
	auth.POST("/import/account", importAccountAPI)
}
