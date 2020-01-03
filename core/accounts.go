package core

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/cxuhua/xginx"
)

const (
	TAccountName = "accounts"
)

//自动创建账号并保存
func SaveAccount(db IDbImp, user *TUser, num uint8, less uint8, arb bool) (*TAccount, error) {
	if !db.IsTx() {
		return nil, errors.New("use tx core")
	}
	ids := []string{}
	for i := 0; i < int(num); i++ {
		pri, err := user.NewPrivate(db, "自动创建")
		if err != nil {
			return nil, err
		}
		ids = append(ids, pri.Id)
	}
	acc, err := NewAccount(db, user.Id, num, less, arb, ids)
	if err != nil {
		return nil, err
	}
	err = db.InsertAccount(acc)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

//创建账号从区块账号
func NewAccountFrom(uid primitive.ObjectID, acc *xginx.Account) (*TAccount, error) {
	id, err := acc.GetAddress()
	if err != nil {
		return nil, err
	}
	a := &TAccount{Id: id, UserId: uid}
	a.Num = acc.Num
	a.Less = acc.Less
	a.Arb = acc.Arb
	a.Pks = acc.GetPks()
	a.Pkh = acc.GetPkhs()
	a.Tags = []string{}
	a.Time = time.Now().Unix()
	return a, nil
}

//利用多个公钥id创建账号
func NewAccount(db IDbImp, uid primitive.ObjectID, num uint8, less uint8, arb bool, ids []string) (*TAccount, error) {
	if len(ids) != int(num) {
		return nil, errors.New("pkhs count != num")
	}
	pks := []xginx.PKBytes{}
	for idx, id := range ids {
		pri, err := db.GetPrivate(id)
		if err != nil {
			return nil, fmt.Errorf("pkh idx = %d private key miss", idx)
		}
		pks = append(pks, pri.Pks)
	}
	acc, err := xginx.NewAccountWithPks(num, less, arb, pks)
	if err != nil {
		return nil, err
	}
	return NewAccountFrom(uid, acc)
}

//账户管理
type TAccount struct {
	Id     xginx.Address      `bson:"_id"`  //账号地址id
	UserId primitive.ObjectID `bson:"uid"`  //谁创建的
	Tags   []string           `bson:"tags"` //标签，分组用
	Num    uint8              `bson:"num"`  //总的密钥数量
	Less   uint8              `bson:"less"` //至少通过的签名数量
	Arb    uint8              `bson:"arb"`  //是否仲裁
	Pks    []xginx.PKBytes    `bson:"pks"`  //公钥
	Pkh    []xginx.HASH160    `bson:"pkh"`  //相关的私钥
	Time   int64              `json:"time"` //创建时间
	Desc   string             `bson:"desc"` //描述
}

//获取第几个私钥
func (a TAccount) GetPrivate(db IDbImp, idx int) (*TPrivate, error) {
	if idx < 0 || idx <= len(a.Pkh) {
		return nil, errors.New("idx out bound")
	}
	id := GetPrivateId(a.Pkh[idx])
	return db.GetPrivate(id[:])
}

func (a *TAccount) ToAccount() *xginx.Account {
	aj := &xginx.Account{
		Num:  a.Num,
		Less: a.Less,
		Arb:  a.Arb,
		Pubs: []*xginx.PublicKey{},
		Pris: xginx.PrivatesMap{},
	}
	for _, pks := range a.Pks {
		pub, err := xginx.NewPublicKey(pks.Bytes())
		if err != nil {
			panic(err)
		}
		aj.Pubs = append(aj.Pubs, pub)
	}
	return aj
}

func (a TAccount) GetAddress() xginx.Address {
	return a.Id
}

func (a TAccount) GetPkh() (xginx.HASH160, error) {
	return xginx.HashPks(a.Num, a.Less, a.Arb, a.Pks)
}

//获取账户金额
func (a *TAccount) ListCoins(bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
	pkh, err := a.Id.GetPkh()
	if err != nil {
		return nil, err
	}
	spent := bi.NextHeight()
	coins, err := bi.ListCoinsWithID(pkh)
	if err != nil {
		return nil, err
	}
	return coins.State(spent), nil
}

//获取账户信息
func (ctx *dbimp) GetAccount(id xginx.Address) (*TAccount, error) {
	col := ctx.table(TAccountName)
	a := &TAccount{}
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(a)
	return a, err
}

//删除账号
func (ctx *dbimp) DeleteAccount(id xginx.Address) error {
	col := ctx.table(TAccountName)
	_, err := col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

//添加一个私钥
func (ctx *dbimp) InsertAccount(obj *TAccount) error {
	col := ctx.table(TAccountName)
	_, err := col.InsertOne(ctx, obj)
	return err
}
