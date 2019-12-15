package db

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TAccountName = "accounts"
)

//从签名格式创建账户
func (user *TUsers) NewAccount(num uint8, less uint8, arb bool) (*TAccount, error) {
	act, err := xginx.NewAccount(num, less, arb)
	if err != nil {
		return nil, err
	}
	a := &TAccount{}
	id, err := act.GetAddress()
	if err != nil {
		return nil, err
	}
	a.Id = string(id)
	a.UserId = user.Id
	a.Num = act.Num
	a.Less = act.Less
	a.Arb = act.Arb
	a.Pks = act.GetPks()
	a.pris = act.Pris
	return a, nil
}

//db.accounts.ensureIndex({uid:1})
//账户管理
type TAccount struct {
	Id     string             `bson:"_id"`
	UserId primitive.ObjectID `bson:"uid"`
	Num    uint8              `bson:"num"`
	Less   uint8              `bson:"less"`
	Arb    uint8              `bson:"arb"`
	Pks    []xginx.PKBytes    `bson:"pks"`
	pris   xginx.PrivatesMap  `bson:"-"`
}

//转换未xginx账户类型
func (a *TAccount) ToAccount(db IDbImp) *xginx.Account {
	aj := &xginx.Account{
		Num:  a.Num,
		Less: a.Less,
		Arb:  a.Arb,
		Pubs: []*xginx.PublicKey{},
		Pris: xginx.PrivatesMap{},
	}
	for _, v := range a.Pks {
		pub, err := xginx.NewPublicKey(v.Bytes())
		if err != nil {
			panic(err)
		}
		aj.Pubs = append(aj.Pubs, pub)
		//如果获取到私钥
		pri, err := a.GetPrivate(db, v)
		if err == nil {
			aj.Pris[v.Hash()] = pri.ToPrivate()
		}
	}
	return aj
}

//获取私钥信息
func (a *TAccount) GetPrivate(db IDbImp, pks xginx.PKBytes) (*TPrivate, error) {
	id := pks.Hash().Bytes()
	return db.GetPrivate(id)
}

//获取账户信息
func (db *dbimp) GetAccount(id string) (*TAccount, error) {
	col := db.table(TAccountName)
	res := col.FindOne(db, bson.M{"_id": id})
	if res.Err() != nil {
		return nil, res.Err()
	}
	a := &TAccount{}
	err := res.Decode(a)
	if err != nil {
		return nil, err
	}
	//加载对应私钥
	for _, v := range a.Pks {
		pri, err := a.GetPrivate(db, v)
		if err == nil {
			a.pris[v.Hash()] = pri.ToPrivate()
		}
	}
	return a, nil
}

//删除账号
func (db *dbimp) DeleteAccount(id string) error {
	if !db.IsTx() {
		return errors.New("please use tx run")
	}
	acc, err := db.GetAccount(id)
	if err != nil {
		return err
	}
	col := db.table(TAccountName)
	_, err = col.DeleteOne(db, bson.M{"_id": id})
	if err != nil {
		return err
	}
	for _, v := range acc.Pks {
		err = db.DeletePrivate(v.Hash().Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

//添加一个私钥
func (db *dbimp) InsertAccount(obj *TAccount) error {
	if !db.IsTx() {
		return errors.New("please use tx run")
	}
	col := db.table(TAccountName)
	_, err := col.InsertOne(db, obj)
	if err != nil {
		return err
	}
	for _, pri := range obj.pris {
		dp := NewPrivate(pri)
		err = db.InsertPrivate(dp)
		if err != nil {
			return err
		}
	}
	return nil
}
