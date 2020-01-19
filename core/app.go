package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cxuhua/xginx"
	"github.com/cxuhua/xmgrs/config"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//token定义
const (
	//token保存格式
	TokenFormat = "[(%s)]"
	//token密码
	TokenPassword = config.TokenKey
	//token在header中的名称
	TokenHeader = "X-Access-Token"
	//token超时时间设置
	TokenTime = time.Hour * 24 * 4
)

/*
//数据库索引
core.accounts.ensureIndex({uid:1,tags:1})
core.accounts.ensureIndex({pkh:1})
core.privates.ensureIndex({uid:1})
core.txs.ensureIndex({uid:1})
core.users.ensureIndex({mobile:1})
core.sigs.ensureIndex({tid:1})
core.sigs.ensureIndex({uid:1})
*/

//数据连接地址
var (
	//连接字符串设置
	RedisURI = config.Redis
	//数据库连接字符串
	MongoURI = config.Mongo
	//连接池设置
	MaxPoolSize = uint64(2000)
	MinPoolSize = uint64(10)
	//默认数据库操作超时时间
	DbTimeout = time.Second * 30
)

var (
	dbonce   = sync.Once{}
	rediscli *redis.Client
	mongocli *mongo.Client
	cipher   = xginx.NewAESCipher([]byte(TokenPassword))
)

//ObjectIDEqual 两个id是否相等
func ObjectIDEqual(v1 primitive.ObjectID, v2 primitive.ObjectID) bool {
	return bytes.Equal(v1[:], v2[:])
}

//ToObjectID 转换为objectID
func ToObjectID(v interface{}) primitive.ObjectID {
	switch v.(type) {
	case primitive.ObjectID:
		return v.(primitive.ObjectID)
	case string:
		id, err := primitive.ObjectIDFromHex(v.(string))
		if err != nil {
			panic(err)
		}
		return id
	case []byte:
		bs := v.([]byte)
		id := primitive.ObjectID{}
		copy(id[:], bs)
		return id
	default:
		panic(errors.New("v to ObjectID error"))
	}
}

//IAppDbImp app接口
type IAppDbImp interface {
	mongo.SessionContext
	//是否在事务环境下
	IsTx() bool
	//使用事务连接
	UseTx(fn func(ctx IDbImp) error) error
}

//App 定义
type App struct {
	context.Context
	redis *redis.Client
	mongo *mongo.Client
}

//GenToken 生成一个token
func (app *App) GenToken() string {
	return primitive.NewObjectID().Hex()
}

//EncryptToken 加密token
func (app *App) EncryptToken(token string) string {
	tk := fmt.Sprintf(TokenFormat, token)
	ck, err := xginx.AesEncrypt(cipher, []byte(tk))
	if err != nil {
		panic(err)
	}
	return xginx.B58Encode(ck, xginx.BitcoinAlphabet)
}

//DecryptToken 解密token
func (app *App) DecryptToken(cks string) (string, error) {
	ck, err := xginx.B58Decode(cks, xginx.BitcoinAlphabet)
	if err != nil {
		return "", err
	}
	tk, err := xginx.AesDecrypt(cipher, ck)
	if err != nil {
		return "", err
	}
	if len(tk) != len(primitive.NilObjectID)*2+4 {
		return "", errors.New("token length error")
	}
	mk := tk[2 : len(tk)-2]
	if string(tk) != fmt.Sprintf(TokenFormat, string(mk)) {
		return "", errors.New("token error")
	}
	oid, err := primitive.ObjectIDFromHex(string(mk))
	if err != nil {
		return "", err
	}
	//检测token是否过期
	tdv := time.Now().Sub(oid.Timestamp())
	if tdv < 0 || tdv > TokenTime {
		return "", fmt.Errorf("token time error %v", tdv)
	}
	return string(mk), nil
}

//Close 关闭app
//一次接口访问结束时调用
func (app *App) Close() {
	//
}

//UseRedisWithTimeout 单独使用redis带有超时
func (app *App) UseRedisWithTimeout(timeout time.Duration, fn func(redv IRedisImp) error) error {
	ctx, cancel := context.WithTimeout(app, timeout)
	defer cancel()
	return fn(NewRedisImp(ctx, rediscli))
}

//UseRedis 单独使用redis使用默认时间
func (app *App) UseRedis(fn func(redv IRedisImp) error) error {
	return app.UseRedisWithTimeout(DbTimeout, fn)
}

//UseDbWithTimeout 启用数据库和redis
//如果需要处理redis超时用 conn.ProcessContext 方法
func (app *App) UseDbWithTimeout(timeout time.Duration, fn func(db IDbImp) error) error {
	ctx, cancel := context.WithTimeout(app, timeout)
	defer cancel()
	return mongocli.UseSession(ctx, func(sctx mongo.SessionContext) error {
		//创建数据对象
		return fn(NewDbImp(sctx, rediscli, false))
	})
}

//UseDb 使用默认超时
func (app *App) UseDb(fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, fn)
}

//UseTxWithTimeout 使用自定义超时事务
func (app *App) UseTxWithTimeout(timeout time.Duration, fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(timeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//UseTx 使用默认超时事务
func (app *App) UseTx(fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//InitApp 初始化全局app
//mongodb://user:pwd@localhost:27017/?
//初始化一个实例对象
func InitApp(ctx context.Context) *App {
	dbonce.Do(func() {
		//redis init
		ropts, err := redis.ParseURL(RedisURI)
		if err != nil {
			panic(err)
		}
		ropts.PoolSize = int(MaxPoolSize)
		ropts.MinIdleConns = int(MinPoolSize)
		rediscli = redis.NewClient(ropts).WithContext(ctx)
		//mongodb init
		mopts := options.Client().
			ApplyURI(MongoURI).
			SetMaxPoolSize(MaxPoolSize).
			SetMinPoolSize(MinPoolSize)
		mcli, err := mongo.NewClient(mopts)
		if err != nil {
			panic(err)
		}
		err = mcli.Connect(ctx)
		if err != nil {
			panic(err)
		}
		mongocli = mcli
	})
	return &App{
		Context: ctx,
		redis:   rediscli,
		mongo:   mongocli,
	}
}

const (
	appkey = "appkey"
)

//GetApp 获取实例对象
func GetApp(c *gin.Context) *App {
	return c.MustGet(appkey).(*App)
}

//AppHandler app gin中间件
func AppHandler(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		app := InitApp(ctx)
		defer app.Close()
		c.Set(appkey, app)
		c.Next()
	}
}
