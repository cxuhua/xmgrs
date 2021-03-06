package core

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo/readpref"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/go-redis/redis/v7"
)

func TestToken(t *testing.T) {
	app := InitApp(context.Background())
	tid := primitive.NewObjectID()
	err := app.UseRedis(func(redv IRedisImp) error {
		token := app.GenToken()
		err := redv.SetUserID(token, tid, time.Second*1)
		if err != nil {
			panic(err)
		}
		ct := app.EncryptToken(token)
		//ct 给客户端
		dt, err := app.DecryptToken(ct)
		if err != nil {
			panic(err)
		}
		if token != dt {
			t.Error("enc dec token error")
		}
		v1, err := redv.GetUserID(dt)
		if err != nil {
			panic(err)
		}
		if ObjectIDEqual(v1, tid) {
			t.Error("value error")
		}
		time.Sleep(time.Second * 2)
		v2, err := redv.GetUserID(dt)
		if err == nil {
			t.Error("expire set error", v2)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUseApp(t *testing.T) {
	app := InitApp(context.Background())
	err := app.UseDb(func(db IDbImp) error {
		return db.UseTx(func(sdb IDbImp) error {
			return nil
		})
	})
	err = app.UseTx(func(sdb IDbImp) error {
		return nil
	})
	log.Println(err)
}

func TestRedis(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		PoolSize:     3,
		MinIdleConns: 0,
	})

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{":7000", ":7001", ":7002", ":7003", ":7004", ":7005"},
	})
	rdb.Ping()

	for {
		conn := cli.Conn()
		pong, err := conn.Ping().Result()
		fmt.Println("err=", pong, err)
		time.Sleep(time.Second * 1)
		s := cli.PoolStats()
		log.Println(conn, s.TotalConns, s.IdleConns, s.StaleConns)
		//conn.Close()
	}
}

//mongodb://user:pwd@localhost:27017

func TestMongo(t *testing.T) {
	ctx := context.Background()
	opts := options.Client().ApplyURI("mongodb://127.0.0.1:27017/")
	mcli, err := mongo.NewClient(opts)
	if err != nil {
		panic(err)
	}
	err = mcli.Connect(ctx)
	if err != nil {
		panic(err)
	}
	for {
		go func() {
			sess, err := mcli.StartSession()
			if err != nil {
				panic(err)
			}
			log.Println(sess.Client().Ping(ctx, readpref.Nearest()))
			time.Sleep(time.Second * 10)
		}()

		time.Sleep(time.Second)

	}
}
