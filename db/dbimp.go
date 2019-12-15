package db

import (
	"bytes"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/cxuhua/xmgrs/config"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
)

//两个id是否相等
func ObjectIDEqual(v1 primitive.ObjectID, v2 primitive.ObjectID) bool {
	return bytes.Equal(v1[:], v2[:])
}

func ToObjectID(v interface{}) primitive.ObjectID {
	switch v.(type) {
	case primitive.ObjectID:
		return v.(primitive.ObjectID)
	case string:
		id, err := primitive.ObjectIDFromHex(v.(string))
		if err != nil {
			panic(err)
		}
		return id
	case []byte:
		bs := v.([]byte)
		id := primitive.ObjectID{}
		copy(id[:], bs)
		return id
	default:
		panic(errors.New("v to ObjectID error"))
	}
}

type IDbImp interface {
	context.Context
	//是否在事务环境下
	IsTx() bool
	//使用事务连接
	UseTx(fn func(ctx IDbImp) error) error
	//添加一个用户信息
	InsertUser(obj *TUsers) error
	//获取用户信息
	GetUserInfo(id interface{}) (*TUsers, error)
	//根据手机号获取用户信息
	GetUserInfoWithMobile(mobile string) (*TUsers, error)
	//添加一个私钥
	InsertPrivate(obj *TPrivate) error
	//删除一个私钥(危险)
	DeletePrivate(id []byte) error
	//获取私钥信息
	GetPrivate(id []byte) (*TPrivate, error)
	//添加一个账号
	InsertAccount(obj *TAccount) error
	//获取账户信息
	GetAccount(id string) (*TAccount, error)
	//删除私钥(危险)
	DeleteAccount(id string) error
}

type dbimp struct {
	mongo.SessionContext
	redv *redis.Conn
	isTx bool
}

func (db *dbimp) table(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	return db.Client().Database(config.DbName).Collection(name, opts...)
}

func (db *dbimp) UseTx(fn func(db IDbImp) error) error {
	if db.isTx {
		return errors.New("tx db can't invoke Transaction")
	}
	_, err := db.WithTransaction(db, func(sdb mongo.SessionContext) (i interface{}, err error) {
		imp := newMongoRedisImp(sdb, db.redv, true)
		return nil, fn(imp)
	})
	return err
}

func (db *dbimp) IsTx() bool {
	return db.isTx
}

func newMongoRedisImp(ctx mongo.SessionContext, redv *redis.Conn, tx bool) IDbImp {
	return &dbimp{
		SessionContext: ctx,
		redv:           redv,
		isTx:           tx,
	}
}
