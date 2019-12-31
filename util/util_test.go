package util

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"

	"github.com/vmihailenco/taskq/v2"

	"github.com/vmihailenco/taskq/v2/redisq"
)

func TestAes(t *testing.T) {
	key := "jzxc972198hasdhsad^^027302173102"
	block := NewAESCipher([]byte(key))
	log.Println(block.BlockSize())
	s := "skdfjslnxvc97934734"
	db, err := AesEncrypt(block, []byte(s))
	if err != nil {
		t.Fatal(err)
	}
	d, err := AesDecrypt(block, db)
	if err != nil {
		t.Fatal(err)
	}
	if s != string(d) {
		t.Fatal("dec enc failed")
	}
}

var Redis = redis.NewClient(&redis.Options{
	Addr: ":6379",
})

var (
	QueueFactory = redisq.NewFactory()
	MainQueue    = QueueFactory.RegisterQueue(&taskq.QueueOptions{
		Name:  "api-worker",
		Redis: Redis,
	})
	CountTask = taskq.RegisterTask(&taskq.TaskOptions{
		Name: "counter",
		Handler: func() error {
			log.Println("Handler")
			return nil
		},
	})
)

func TestTaskQ(t *testing.T) {
	go func() {
		err := QueueFactory.StartConsumers(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(time.Second * 5)
	go func() {
		MainQueue.Add(CountTask.WithArgs(context.Background()))
	}()
	time.Sleep(time.Hour)
}
