package db

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	//连接字符串设置
	RedisURI = "redis://127.0.0.1:6379"
	MongoURI = "mongodb://127.0.0.1:27017,127.0.0.1:27018,127.0.0.1:27019"
	//连接池设置
	MaxPoolSize = uint64(2000)
	MinPoolSize = uint64(10)
	//默认超时世界
	DbTimeout = time.Second * 30
)

var (
	dbonce   = sync.Once{}
	rediscli *redis.Client
	mongocli *mongo.Client
	basectx  context.Context
)

type App struct {
	context.Context
	redis *redis.Client
	mongo *mongo.Client
}

func (app *App) Close() {

}

func (app *App) Clone() *App {
	return &App{
		redis:   app.redis,
		mongo:   app.mongo,
		Context: app.Context,
	}
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
		return fn(newMongoRedisImp(sctx, conn, false))
	})
}

//使用默认超时
func (app *App) UseDb(fn func(db IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, fn)
}

//使用自定义超时事务
func (app *App) UseTxWithTimeout(timeout time.Duration, fn func(sdb IDbImp) error) error {
	return app.UseDbWithTimeout(timeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//使用默认超时事务
func (app *App) UseTx(fn func(sdb IDbImp) error) error {
	return app.UseDbWithTimeout(DbTimeout, func(db IDbImp) error {
		return db.UseTx(fn)
	})
}

//mongodb://user:pwd@localhost:27017
//初始化一个实例对象
func InitApp(ctx context.Context) *App {
	dbonce.Do(func() {
		basectx = ctx
		//redis init
		ropts, err := redis.ParseURL(RedisURI)
		if err != nil {
			panic(err)
		}
		ropts.PoolSize = int(MaxPoolSize)
		ropts.MinIdleConns = int(MinPoolSize)
		rediscli = redis.NewClient(ropts).WithContext(basectx)
		//mongodb init
		mopts := options.
			Client().
			ApplyURI(MongoURI).
			SetMaxPoolSize(MaxPoolSize).
			SetMinPoolSize(MinPoolSize)
		mcli, err := mongo.NewClient(mopts)
		if err != nil {
			panic(err)
		}
		err = mcli.Connect(basectx)
		if err != nil {
			panic(err)
		}
		mongocli = mcli
	})
	return &App{
		Context: basectx,
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

func AppHandler(app *App) gin.HandlerFunc {
	return func(c *gin.Context) {
		app = app.Clone()
		defer app.Close()
		c.Set(appkey, app)
		c.Next()
	}
}
