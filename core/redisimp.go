package core

import (
	"time"

	"github.com/go-redis/redis/v7"
)

//redis接口
type IRedisImp interface {
	//保存用户id到redis
	SetUserId(k string, id string, time time.Duration) error
	//从redis获取用户id
	GetUserId(k string) (string, error)
}

type redisImp struct {
	redv *redis.Conn
}

//保存用户id
func (conn *redisImp) SetUserId(k string, id string, time time.Duration) error {
	return conn.redv.Set(k, id, time).Err()
}

//获取token
func (conn *redisImp) GetUserId(k string) (string, error) {
	s := conn.redv.Get(k)
	return s.Result()
}

func NewRedisImp(redv *redis.Conn) IRedisImp {
	return &redisImp{redv: redv}
}
