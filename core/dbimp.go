package core

import (
	"errors"

	"github.com/cxuhua/xginx"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/cxuhua/xmgrs/config"

	"github.com/go-redis/redis/v7"
	"go.mongodb.org/mongo-driver/mongo"
)

//IDbImp 数据库接口定义
type IDbImp interface {
	IAppDbImp
	IRedisImp
	//设置用户推送id
	SetPushID(uid primitive.ObjectID, pid string) error
	//保存用户token
	SetUserToken(uid primitive.ObjectID, tk string) error
	//添加一个用户信息
	InsertUser(obj *TUser) error
	//获取用户信息
	GetUserInfo(id interface{}) (*TUser, error)
	//删除用户(危险)
	DeleteUser(id interface{}) error
	//根据手机号获取用户信息
	GetUserInfoWithMobile(mobile string) (*TUser, error)
	//修改用户主私钥密码
	SetUserKeyPass(uid primitive.ObjectID, old string, new string) error
	//修改用户私钥密码
	SetPrivateKeyPass(uid primitive.ObjectID, pid string, old string, new string) error
	//添加一个私钥
	InsertPrivate(obj *TPrivate) error
	//只检测是否有引用
	HasPrivateRef(id string) (bool, error)
	//获取私钥id引用到的账户数量
	GetPrivateRefs(id string) (int, error)
	//删除一个私钥(危险)
	DeletePrivate(id string) error
	//获取私钥信息
	GetPrivate(id string) (*TPrivate, error)
	//获取用户的私钥
	GetUserPrivate(id string, uid primitive.ObjectID) (*TPrivate, error)
	//添加一个账号
	InsertAccount(obj *TAccount) error
	//获取账户信息
	GetAccount(id xginx.Address) (*TAccount, error)
	//删除用户私钥
	DeleteAccount(id xginx.Address, uid primitive.ObjectID) error
	//获取用户的私钥
	ListPrivates(uid primitive.ObjectID) ([]*TPrivate, error)
	//获取用户相关的账号
	ListAccounts(uid primitive.ObjectID) ([]*TAccount, error)
	//获取交易信息
	GetTx(id []byte) (*TTx, error)
	//更新交易状态
	SetTxState(id []byte, state TTxState) error
	//删除交易信息
	DeleteTx(id []byte) error
	//插入交易信息
	InsertTx(tx *TTx) error
	//保存签名对象
	InsertSigs(sigs ...*TSigs) error
	//设置签名
	SetSigs(id primitive.ObjectID, sigs xginx.SigBytes) error
	//获取交易相关的签名对象
	ListSigs(tid xginx.HASH256) (TxSigs, error)
	//获取需要用户签名的数据
	ListUserSigs(uid primitive.ObjectID, tid xginx.HASH256) (TxSigs, error)
	//获取签名对象
	GetSigs(tid xginx.HASH256, kid string, hash []byte, idx int) (*TSigs, error)
	//获取用户需要签名的交易
	ListUserTxs(uid primitive.ObjectID, sign bool) ([]*TTx, error)
	//自增密钥索引
	IncDeterIdx(name string, id interface{}) error
}

type dbimp struct {
	mongo.SessionContext
	*redisImp
	isTx bool
}

func (db *dbimp) database(opts ...*options.DatabaseOptions) *mongo.Database {
	return db.Client().Database(config.DbName, opts...)
}

func (db *dbimp) table(tbl string, opts ...*options.CollectionOptions) *mongo.Collection {
	return db.database().Collection(tbl, opts...)
}

func (db *dbimp) UseTx(fn func(db IDbImp) error) error {
	if db.IsTx() {
		return errors.New("tx core can't invoke Transaction")
	}
	_, err := db.WithTransaction(db, func(sdb mongo.SessionContext) (i interface{}, err error) {
		return nil, fn(NewDbImp(sdb, db.rcli, true))
	})
	return err
}

func (db *dbimp) IsTx() bool {
	return db.isTx
}

//NewDbImp 新建一个数据库接口
func NewDbImp(ctx mongo.SessionContext, rcli *redis.Client, tx bool) IDbImp {
	return &dbimp{
		SessionContext: ctx,
		redisImp:       &redisImp{Context: ctx, rcli: rcli},
		isTx:           tx,
	}
}
