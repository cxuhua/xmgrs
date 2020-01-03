package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cxuhua/xmgrs/util"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	TokenFormat   = "[(%s)]"
	TokenPassword = "_Ufdf&^23232(j3434_"
	TokenHeader   = "X-Access-Token"
)

/*
//数据库索引
core.accounts.ensureIndex({uid:1})
core.accounts.ensureIndex({pkh:1})
core.privates.ensureIndex({uid:1})
core.txs.ensureIndex({uid:1})
core.users.ensureIndex({mobile:1})
core.sigs.ensureIndex({tid:1})
core.sigs.ensureIndex({uid:1})
*/

var (
	//连接字符串设置
	RedisURI = "redis://127.0.0.1:6379"
	//数据库连接字符串
	MongoURI = "mongodb://127.0.0.1:27017"
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
	cipher   = util.NewAESCipher([]byte(TokenPassword))
)

//两个id是否相等
func ObjectIDEqual(v1 primitive.ObjectID, v2 primitive.ObjectID) bool {
	return bytes.Equal(v1[:], v2[:])
}

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

type IAppDbImp interface {
	mongo.SessionContext
	//是否在事务环境下
	IsTx() bool
	//使用事务连接
	UseTx(fn func(ctx IDbImp) error) error
}

type App struct {
	context.Context
	redis *redis.Client
	mongo *mongo.Client
}

//生成一个token
func (app *App) GenToken() string {
	id := primitive.NewObjectID()
	return id.Hex()
}

//time=0不过期
func (app *App) SetUserId(k string, id string, time time.Duration) error {
	s := app.redis.Set(k, id, time)
	return s.Err()
}

//获取token
func (app *App) GetUserId(k string) (string, error) {
	s := app.redis.Get(k)
	return s.Result()
}

//加密token
func (app *App) EncryptToken(token string) string {
	tk := fmt.Sprintf(TokenFormat, token)
	ck, err := util.AesEncrypt(cipher, []byte(tk))
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(ck)
}

//解密token
func (app *App) DecryptToken(cks string) (string, error) {
	ck, err := base64.URLEncoding.DecodeString(cks)
	if err != nil {
		return "", err
	}
	tk, err := util.AesDecrypt(cipher, ck)
	if err != nil {
		return "", err
	}
	if len(tk) != 28 {
		return "", errors.New("token length error")
	}
	mk := tk[2 : len(tk)-2]
	if string(tk) != fmt.Sprintf(TokenFormat, string(mk)) {
		return "", errors.New("token error")
	}
	if _, err := primitive.ObjectIDFromHex(string(mk)); err != nil {
		return "", err
	}
	return string(mk), nil
}

func (app *App) Close() {
	//
}

//启用数据库和redis
//如果需要处理redis超时用 conn.ProcessContext 方法
func (app *App) UseDbWithTimeout(timeout time.Duration, fn func(db IDbImp) error) error {
	ctx, cancel := context.WithTimeout(app, timeout)
	defer cancel()
	return mongocli.UseSession(ctx, func(sctx mongo.SessionContext) error {
		//获取一个redis连接
		conn := rediscli.Conn()
		defer conn.Close()
		//创建数据对象
		return fn(NewDbImp(sctx, conn, false))
	})
}

//单独使用redis
func (app *App) UseRedis(fn func(conn *redis.Conn) error) error {
	conn := rediscli.Conn()
	defer conn.Close()
	return fn(conn)
}

//使用默认超时
func (app *App) UseDb(fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, fn)
}

//使用自定义超时事务
func (app *App) UseTxWithTimeout(timeout time.Duration, fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(timeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//使用默认超时事务
func (app *App) UseTx(fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//mongodb://user:pwd@localhost:27017
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

//获取实例对象
func GetApp(c *gin.Context) *App {
	return c.MustGet(appkey).(*App)
}

func AppHandler(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		app := InitApp(ctx)
		defer app.Close()
		c.Set(appkey, app)
		c.Next()
	}
}
