package db

import (
	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TTxName = "txs"
)

type TTxIn struct {
	OutHash  []byte       `bson:"oid"`
	OutIndex uint32       `bson:"idx"`
	Script   xginx.Script `bson:"script"`
	Sequence uint32       `bson:"seq"`
}

type TTxOut struct {
	Value  int64        `bson:"value"`
	Script xginx.Script `bson:"script"`
}

//db.txs.ensureIndex({uid:1})
type TTx struct {
	Id       []byte             `bson:"_id"`
	UserId   primitive.ObjectID `bson:"uid"`
	Ver      uint32             `bson:"ver"`
	Ins      []TTxIn            `bson:"ins"`
	Outs     []TTxOut           `bson:"outs"`
	LockTime uint32             `bson:"lt"`
}

//创建交易
func (u *TUsers) NewTTx(tx *xginx.TX) *TTx {
	return NewTTx(u.Id, tx)
}

//从区块交易创建
func NewTTx(uid primitive.ObjectID, tx *xginx.TX) *TTx {
	v := &TTx{}
	v.Id = tx.MustID().Bytes()
	v.Ver = tx.Ver.ToUInt32()
	v.LockTime = tx.LockTime
	for _, in := range tx.Ins {
		vi := TTxIn{}
		vi.OutHash = in.OutHash[:]
		vi.OutIndex = in.OutIndex.ToUInt32()
		vi.Script = in.Script
		vi.Sequence = in.Sequence
		v.Ins = append(v.Ins, vi)
	}
	for _, out := range tx.Outs {
		vo := TTxOut{}
		vo.Value = int64(out.Value)
		vo.Script = out.Script
		v.Outs = append(v.Outs, vo)
	}
	v.UserId = uid
	return v
}

//获取账户信息
func (ctx *dbimp) GetTx(id []byte) (*TAccount, error) {
	col := ctx.table(TTxName)
	a := &TAccount{}
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(a)
	return a, err
}

//删除账号
func (ctx *dbimp) DeleteTx(id []byte) error {
	col := ctx.table(TTxName)
	_, err := col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

//添加一个私钥
func (ctx *dbimp) InsertTx(tx *TTx) error {
	col := ctx.table(TTxName)
	_, err := col.InsertOne(ctx, tx)
	return err
}
