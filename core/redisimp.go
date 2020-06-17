package core

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v7"
)

//ILocker 缓存锁，需要基于分布式锁实现
type ILocker interface {
	//Release 释放锁
	Release()
	//TTL 锁超时时间 返回0表示锁已经释放
	TTL() (time.Duration, error)
	//Refresh 更新锁超时时间
	Refresh(ttl time.Duration) error
	//获取meta数据
	Metadata() string
}

type redislocker struct {
	l *redislock.Lock
}

func (d *redislocker) Release() {
	d.l.Release()
}

func (d *redislocker) Metadata() string {
	return d.l.Metadata()
}

func (d *redislocker) TTL() (time.Duration, error) {
	return d.l.TTL()
}

func (d *redislocker) Refresh(ttl time.Duration) error {
	return d.l.Refresh(ttl, nil)
}

//NewRedisLocker 创建基于redis得分布式锁
func NewRedisLocker(c redislock.RedisClient, key string, ttl time.Duration, meta ...string) (ILocker, error) {
	opts := &redislock.Options{}
	if len(meta) > 0 {
		opts.Metadata = meta[0]
	}
	l := redislock.New(c)
	lp, err := l.Obtain(key, ttl, opts)
	if err != nil {
		return nil, err
	}
	return &redislocker{l: lp}, nil
}

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
	//分布式锁实现
	Locker(key string, ttl time.Duration, meta ...string) (ILocker, error)
}

type redisImp struct {
	context.Context
	rcli *redis.Client
	conn *redis.Conn
}

func (rimp *redisImp) Locker(key string, ttl time.Duration, meta ...string) (ILocker, error) {
	return NewRedisLocker(rimp.conn, key, ttl, meta...)
}

func (rimp *redisImp) DelUserID(k string) error {
	return rimp.conn.Del(k).Err()
}

// Subscribe 开始利用redis订阅，成功后返回订阅连接
// 发布消息使用 IRidisImp Publish
func (rimp *redisImp) Subscribe(channels ...string) *redis.PubSub {
	return rimp.rcli.Subscribe(channels...)
}

//Publish 发布消息
func (rimp *redisImp) Publish(channel string, message interface{}) error {
	return rimp.conn.Publish(channel, message).Err()
}

//SetUserId 保存用户id
func (rimp *redisImp) SetUserID(k string, id primitive.ObjectID, time time.Duration) error {
	return rimp.conn.Set(k, id.Hex(), time).Err()
}

//获取token
func (rimp *redisImp) GetUserID(k string) (primitive.ObjectID, error) {
	s := rimp.conn.Get(k)
	hs, err := s.Result()
	if err != nil {
		return primitive.NilObjectID, err
	}
	return primitive.ObjectIDFromHex(hs)
}

//NewRedisImp 创建缓存接口
func NewRedisImp(ctx context.Context, rcli *redis.Client, conn *redis.Conn) IRedisImp {
	return &redisImp{
		Context: ctx,
		rcli:    rcli,
		conn:    conn,
	}
}
