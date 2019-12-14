package db

import (
	"errors"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
)

type IDbImp interface {
	//使用事务连接
	UseTx(fn func(sdb IDbImp) error) error
	//
}

type dbimp struct {
	ctx  mongo.SessionContext
	redv *redis.Conn
	isTx bool
}

func (db *dbimp) UseTx(fn func(db IDbImp) error) error {
	if db.isTx {
		return errors.New("tx db can't invoke Transaction")
	}
	_, err := db.ctx.WithTransaction(db.ctx, func(sdb mongo.SessionContext) (i interface{}, err error) {
		imp := newMongoRedisImp(sdb, db.redv, true)
		return nil, fn(imp)
	})
	return err
}

func newMongoRedisImp(ctx mongo.SessionContext, redv *redis.Conn, tx bool) IDbImp {
	return &dbimp{
		ctx:  ctx,
		redv: redv,
		isTx: tx,
	}
}
