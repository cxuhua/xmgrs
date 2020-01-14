package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/cxuhua/xginx"

	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v9"

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

//检测是否是地址
func IsAddress(fl validator.FieldLevel) bool {
	v := fl.Field().String()
	_, err := xginx.DecodeAddress(xginx.Address(v))
	if err != nil {
		return false
	}
	return true
}

//字段是否是hash160 hex字符串
func HexHash160(fl validator.FieldLevel) bool {
	v, err := hex.DecodeString(fl.Field().String())
	return err == nil && len(v) == 20
}

//字段是否是hash256 hex字符串
func HexHash256(fl validator.FieldLevel) bool {
	v, err := hex.DecodeString(fl.Field().String())
	return err == nil && len(v) == 32
}

//获取默认gin引擎
func InitEngine(ctx context.Context) *gin.Engine {
	//注册自定义校验器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("HexHash256", HexHash256)
		v.RegisterValidation("HexHash160", HexHash160)
		v.RegisterValidation("IsAddress", IsAddress)
	}

	m := gin.New()
	m.Use(gin.Logger(), gin.Recovery())

	v1 := m.Group("/v1")
	v1.Use(core.AppHandler(ctx))
	ApiV1Entry(v1)
	return m
}

const (
	AppUserIdKey = "AppUserIdKey"
)

//获取用户id
func GetAppUserId(c *gin.Context) primitive.ObjectID {
	return c.MustGet(AppUserIdKey).(primitive.ObjectID)
}

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
		oid, err := redv.GetUserId(tk)
		if err != nil {
			return err
		}
		c.Set(AppUserIdKey, oid)
		return nil
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewModel(1000, err))
		return
	}
	c.Next()
}

func ApiV1Entry(rg *gin.RouterGroup) {
	rg.POST("/register", registerApi)
	rg.POST("/login", loginApi)
	rg.POST("/account/prove", accountProveApi)
	auth := rg.Group("/", IsLogin)
	auth.GET("/quit/login", quitLoginApi)
	auth.GET("/user/info", userInfoApi)
	auth.GET("/user/coins", listCoinsApi)
	auth.GET("/tx/info/:id", getTxInfoApi)
	auth.GET("/list/txs/:addr", listTxsApi)
	auth.GET("/list/accounts", listUserAccountsApi)
	auth.GET("/list/sign/txs", listUserSignTxsApi)
	auth.GET("/list/privates", listPrivatesApi)
	auth.POST("/new/private", createUserPrivateApi)
	auth.POST("/new/account", createAccountApi)
	auth.POST("/new/tx", createTxApi)
	auth.POST("/sign/tx", signTxApi)
	auth.POST("/submit/tx", submitTxApi)
}
