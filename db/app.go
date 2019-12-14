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
	RedisURI = "127.0.0.1:6379"
	MongoURI = "mongodb://127.0.0.1:27017"
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
func (app *App) UseSession(timeout time.Duration, fn func(ctx mongo.SessionContext, redv *redis.Conn) error) error {
	ctx, cancel := context.WithTimeout(app, timeout)
	defer cancel()
	return mongocli.UseSession(ctx, func(sctx mongo.SessionContext) error {
		conn := rediscli.Conn()
		defer conn.Close()
		return fn(sctx, conn)
	})
}

//mongodb://user:pwd@localhost:27017
//初始化一个实例对象
func InitApp(ctx context.Context) *App {
	dbonce.Do(func() {
		basectx = ctx
		//redis初始化
		rcli := redis.NewClient(&redis.Options{
			Addr:         RedisURI,
			PoolSize:     1000,
			MinIdleConns: 5,
		})
		rediscli = rcli.WithContext(basectx)
		//数据库初始化
		opts := options.Client().ApplyURI(MongoURI)
		mcli, err := mongo.NewClient(opts)
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
