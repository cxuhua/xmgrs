package core

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/go-redis/redis/v7"
)

//IRedisImp redis接口
type IRedisImp interface {
	//保存用户id到redis
	SetUserID(k string, id primitive.ObjectID, time time.Duration) error
	//从redis获取用户id
	GetUserID(k string) (primitive.ObjectID, error)
	//删除token
	DelUserID(k string) error
	//订阅消息
	Subscribe(channels ...string) *redis.PubSub
	//发布消息
	Publish(channel string, message interface{}) error
}

type redisImp struct {
	context.Context
	rcli *redis.Client
}

func (conn *redisImp) DelUserID(k string) error {
	return conn.rcli.Del(k).Err()
}

// Subscribe 开始利用redis订阅，成功后返回订阅
// 发布消息使用 IRidisImp Publish
func (conn *redisImp) Subscribe(channels ...string) *redis.PubSub {
	return conn.rcli.Subscribe(channels...)
}

func (conn *redisImp) Publish(channel string, message interface{}) error {
	return conn.rcli.Publish(channel, message).Err()
}

//SetUserId 保存用户id
func (conn *redisImp) SetUserID(k string, id primitive.ObjectID, time time.Duration) error {
	return conn.rcli.Set(k, id.Hex(), time).Err()
}

//获取token
func (conn *redisImp) GetUserID(k string) (primitive.ObjectID, error) {
	s := conn.rcli.Get(k)
	hs, err := s.Result()
	if err != nil {
		return primitive.NilObjectID, err
	}
	return primitive.ObjectIDFromHex(hs)
}

//NewRedisImp 创建缓存接口
func NewRedisImp(ctx context.Context, rcli *redis.Client) IRedisImp {
	return &redisImp{
		Context: ctx,
		rcli:    rcli,
	}
}
