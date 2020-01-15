package core

import (
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
}

type redisImp struct {
	redv *redis.Conn
}

func (conn *redisImp) DelUserID(k string) error {
	return conn.redv.Del(k).Err()
}

//SetUserId 保存用户id
func (conn *redisImp) SetUserID(k string, id primitive.ObjectID, time time.Duration) error {
	return conn.redv.Set(k, id.Hex(), time).Err()
}

//获取token
func (conn *redisImp) GetUserID(k string) (primitive.ObjectID, error) {
	s := conn.redv.Get(k)
	hs, err := s.Result()
	if err != nil {
		return primitive.NilObjectID, err
	}
	return primitive.ObjectIDFromHex(hs)
}

//NewRedisImp 创建缓存接口
func NewRedisImp(redv *redis.Conn) IRedisImp {
	return &redisImp{redv: redv}
}
