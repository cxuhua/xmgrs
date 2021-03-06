package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/cxuhua/xmgrs/util"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/cxuhua/xginx"
)

//账号表
const (
	TAccountName = "accounts"
)

//SaveAccount 自动创建账号并保存
func SaveAccount(db IDbImp, user *TUser, num uint8, less uint8, arb bool, desc string, tags []string) (*TAccount, error) {
	if !db.IsTx() {
		return nil, errors.New("use tx core")
	}
	ids := []string{}
	for i := 0; i < int(num); i++ {
		pri, err := user.NewPrivate(db, "自动创建")
		if err != nil {
			return nil, err
		}
		ids = append(ids, pri.ID)
	}
	acc, err := NewAccount(db, num, less, arb, ids, desc, tags)
	if err != nil {
		return nil, err
	}
	err = db.InsertAccount(acc)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

//SaveAccount 创建账号并保存
func (user *TUser) SaveAccount(db IDbImp, num uint8, less uint8, arb bool, desc string, tags []string) (*TAccount, error) {
	return SaveAccount(db, user, num, less, arb, desc, tags)
}

//NewAccountFrom 创建账号从区块账号
func NewAccountFrom(uids []primitive.ObjectID, acc *xginx.Account, desc string, tags []string) (*TAccount, error) {
	tags = util.RemoveRepeat(tags)
	id, err := acc.GetAddress()
	if err != nil {
		return nil, err
	}
	a := &TAccount{ID: id, UserID: uids}
	a.Num = acc.Num
	a.Less = acc.Less
	a.Arb = acc.Arb
	a.Pks = acc.GetPks()
	for _, pkh := range acc.GetPkhs() {
		a.Kid = append(a.Kid, GetPrivateID(pkh))
	}
	a.Tags = tags
	a.Desc = desc
	a.Time = time.Now().Unix()
	return a, nil
}

//NewAccount 利用多个公钥id创建账号
func NewAccount(db IDbImp, num uint8, less uint8, arb bool, ids []string, desc string, tags []string) (*TAccount, error) {
	if num == 0 {
		return nil, errors.New("num error")
	}
	//移除重复的私钥id
	ids = util.RemoveRepeat(ids)
	if len(ids) != int(num) {
		return nil, errors.New("pkhs count != num")
	}
	//获取和这些私钥相关的用户
	imap := map[primitive.ObjectID]bool{}
	pks := []xginx.PKBytes{}
	for idx, id := range ids {
		pri, err := db.GetPrivate(id)
		if err != nil {
			return nil, fmt.Errorf("pkh idx = %d private key miss", idx)
		}
		imap[pri.UserID] = true
		pks = append(pks, pri.Pks)
	}
	//根据相关私钥创建账户地址
	acc, err := xginx.NewAccountWithPks(num, less, arb, pks)
	if err != nil {
		return nil, err
	}
	uids := []primitive.ObjectID{}
	for uid := range imap {
		uids = append(uids, uid)
	}
	return NewAccountFrom(uids, acc, desc, tags)
}

//TAccount 账户数据结构
//一个账号可能有多个私钥构成，签名时必须按照规则签名所需的私钥
type TAccount struct {
	ID     xginx.Address        `bson:"_id"`  //账号地址id
	UserID []primitive.ObjectID `bson:"uid"`  //所属的多个账户，当用多个私钥创建时，所属私钥的用户集合
	Tags   []string             `bson:"tags"` //标签，分组用
	Num    uint8                `bson:"num"`  //总的密钥数量
	Less   uint8                `bson:"less"` //至少通过的签名数量
	Arb    uint8                `bson:"arb"`  //是否仲裁
	Pks    []xginx.PKBytes      `bson:"pks"`  //包含的公钥
	Kid    []string             `bson:"kid"`  //包含的密钥id
	Time   int64                `bson:"time"` //创建时间
	Desc   string               `bson:"desc"` //描述
}

//HasUserID 是否包含用户
func (acc TAccount) HasUserID(uid primitive.ObjectID) bool {
	for _, v := range acc.UserID {
		if ObjectIDEqual(uid, v) {
			return true
		}
	}
	return false
}

//GetPrivate 获取第几个私钥
func (acc TAccount) GetPrivate(db IDbImp, idx int) (*TPrivate, error) {
	if idx < 0 || idx <= len(acc.Kid) {
		return nil, errors.New("idx out bound")
	}
	return db.GetPrivate(acc.Kid[idx])
}

//ToAccount pri是否加载私钥
func (acc *TAccount) ToAccount(db IDbImp, pri bool, pass ...string) (*xginx.Account, error) {
	aj := &xginx.Account{
		Num:  acc.Num,
		Less: acc.Less,
		Arb:  acc.Arb,
		Pubs: []*xginx.PublicKey{},
		Pris: xginx.PrivatesMap{},
	}
	for _, pks := range acc.Pks {
		pub, err := xginx.NewPublicKey(pks.Bytes())
		if err != nil {
			panic(err)
		}
		aj.Pubs = append(aj.Pubs, pub)
	}
	//如果不加载私钥
	if !pri {
		return aj, nil
	}
	for _, kid := range acc.Kid {
		pri, err := db.GetPrivate(kid)
		if err != nil {
			return nil, err
		}
		kp, err := pri.ToPrivate(pass...)
		if err != nil {
			return nil, err
		}
		aj.Pris[pri.Pks.Hash()] = kp
	}
	return aj, nil
}

//GetAddress 获取地址
func (acc TAccount) GetAddress() xginx.Address {
	return acc.ID
}

//GetPkh 获取公钥hash
func (acc TAccount) GetPkh() (xginx.HASH160, error) {
	return xginx.HashPks(acc.Num, acc.Less, acc.Arb, acc.Pks)
}

//ListCoins 获取账户金额
func (acc *TAccount) ListCoins(bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
	pkh, err := acc.ID.GetPkh()
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

//GetAccount 获取账户信息
func (ctx *dbimp) GetAccount(id xginx.Address) (*TAccount, error) {
	col := ctx.table(TAccountName)
	a := &TAccount{}
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(a)
	return a, err
}

//DeleteAccount 删除用户账户
func (ctx *dbimp) DeleteAccount(id xginx.Address, uid primitive.ObjectID) error {
	col := ctx.table(TAccountName)
	// 移除账户内相关的所属用户
	sr := col.FindOneAndUpdate(ctx, bson.M{"_id": id, "uid": uid}, bson.M{"$pull": bson.M{"uid": uid}})
	if sr.Err() != nil {
		return sr.Err()
	}
	// 如果没有所属用户删除账号信息
	sr = col.FindOneAndDelete(ctx, bson.M{"_id": id, "uid": bson.M{"$size": 0}})
	if sr.Err() != nil {
		return sr.Err()
	}
	return nil
}

//添加一个私钥
func (ctx *dbimp) InsertAccount(obj *TAccount) error {
	col := ctx.table(TAccountName)
	_, err := col.InsertOne(ctx, obj)
	return err
}
