package db

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo/readpref"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/go-redis/redis/v7"
)

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
