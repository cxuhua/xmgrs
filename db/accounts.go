package db

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/cxuhua/xginx"
)

const (
	TAccountName = "accounts"
)

//利用多个公钥id创建账号
func NewAccount(db IDbImp, num uint8, less uint8, arb bool, ids []string) (*TAccount, error) {
	if len(ids) != int(num) {
		return nil, errors.New("pkhs count != num")
	}
	pkss := []xginx.PKBytes{}
	for idx, id := range ids {
		pri, err := db.GetPrivate(id)
		if err != nil {
			return nil, fmt.Errorf("pkh idx = %d private key miss", idx)
		}
		pkss = append(pkss, pri.Pks)
	}
	acc, err := xginx.NewAccountWithPks(num, less, arb, pkss)
	if err != nil {
		return nil, err
	}
	id, err := acc.GetAddress()
	if err != nil {
		return nil, err
	}
	a := &TAccount{}
	a.Id = id
	a.Num = acc.Num
	a.Less = acc.Less
	a.Arb = acc.Arb
	a.Pks = acc.GetPks()
	a.Pkh = acc.GetPkhs()
	return a, nil
}

//db.accounts.ensureIndex({uid:1})
//db.accounts.ensureIndex({pkh:1})
//账户管理
type TAccount struct {
	Id   xginx.Address   `bson:"_id"`
	Num  uint8           `bson:"num"`
	Less uint8           `bson:"less"`
	Arb  uint8           `bson:"arb"`
	Pks  []xginx.PKBytes `bson:"pks"`
	Pkh  []xginx.HASH160 `bson:"pkh"`
}

//生成脚本签名
func (a TAccount) Sign(db IDbImp, signer xginx.ISigner) (*xginx.WitnessScript, error) {
	hv, err := signer.GetSigHash()
	if err != nil {
		return nil, err
	}
	wits := a.ToAccount().NewWitnessScript()
	for _, pkh := range a.Pkh {
		id := GetPrivateId(pkh)
		pri, err := db.GetPrivate(id)
		if err != nil {
			continue
		}
		sig, err := pri.ToPrivate().Sign(hv)
		if err != nil {
			return nil, err
		}
		wits.Sig = append(wits.Sig, sig.GetSigs())
	}
	return wits, nil
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
	return xginx.Address(a.Id)
}

func (a TAccount) GetPkh() (xginx.HASH160, error) {
	return xginx.HashPks(a.Num, a.Less, a.Arb, a.Pks)
}

//获取账户金额
func (a *TAccount) ListCoins(bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
	pkh, err := a.GetPkh()
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
