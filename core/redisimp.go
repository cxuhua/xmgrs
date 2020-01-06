package core

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/go-redis/redis/v7"
)

//redis接口
type IRedisImp interface {
	//保存用户id到redis
	SetUserId(k string, id primitive.ObjectID, time time.Duration) error
	//从redis获取用户id
	GetUserId(k string) (primitive.ObjectID, error)
	//清楚token
	DelUserId(k string) error
}

type redisImp struct {
	redv *redis.Conn
}

func (conn *redisImp) DelUserId(k string) error {
	return conn.redv.Del(k).Err()
}

//保存用户id
func (conn *redisImp) SetUserId(k string, id primitive.ObjectID, time time.Duration) error {
	return conn.redv.Set(k, id.Hex(), time).Err()
}

//获取token
func (conn *redisImp) GetUserId(k string) (primitive.ObjectID, error) {
	s := conn.redv.Get(k)
	hs, err := s.Result()
	if err != nil {
		return primitive.NilObjectID, err
	}
	return primitive.ObjectIDFromHex(hs)
}

func NewRedisImp(redv *redis.Conn) IRedisImp {
	return &redisImp{redv: redv}
}
