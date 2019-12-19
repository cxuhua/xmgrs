package db

import (
	"errors"
	"fmt"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TTxName  = "txs"
	TSigName = "sigs"
)

type ISaveSigs interface {
	SaveSigs() error
}

//用来分析并保存签名对象
type DbSignListener struct {
	user *TUsers //当前用户
	db   IDbImp  //db接口
	sigs []*TSigs
}

func NewSignListener(db IDbImp, user *TUsers) *DbSignListener {
	return &DbSignListener{
		user: user,
		db:   db,
		sigs: []*TSigs{},
	}
}

//获取需要保存的交易签名列表
func (st *DbSignListener) GetSigs() []*TSigs {
	return st.sigs
}

//保存
func (st *DbSignListener) SaveSigs() error {
	for _, sig := range st.sigs {
		err := st.db.InsertSigs(sig)
		if err != nil {
			return err
		}
	}
	return nil
}

//获取金额对应的账户方法
func (st *DbSignListener) GetAcc(ckv *xginx.CoinKeyValue) *xginx.Account {
	acc, err := st.db.GetAccount(ckv.GetAddress())
	if err != nil {
		return nil
	}
	return acc.ToAccount()
}

//获取输出地址的扩展
func (st *DbSignListener) GetExt(addr xginx.Address) []byte {
	return nil
}

//获取使用的金额
func (st *DbSignListener) GetCoins() xginx.Coins {
	bi := xginx.GetBlockIndex()
	ds, err := st.user.ListCoins(st.db, bi)
	if err != nil {
		return nil
	}
	return ds.Coins
}

//获取找零地址
func (st *DbSignListener) GetKeep() xginx.Address {
	accs, err := st.user.ListAccounts(st.db)
	if err != nil {
		panic(err)
	}
	if len(accs) == 0 {
		panic(errors.New("user no accounts"))
	}
	//默认使用第一个地址作为找零地址
	return accs[0].GetAddress()
}

//签名交易
func (st *DbSignListener) SignTx(singer xginx.ISigner) error {
	addr := singer.GetAddress()
	//获取对应的账号
	acc, err := st.db.GetAccount(addr)
	if err != nil {
		return err
	}
	tid := singer.GetTxId()
	hash, err := singer.GetSigHash()
	if err != nil {
		return err
	}
	for _, pkh := range acc.Pkh {
		kid := GetPrivateId(pkh)
		sigs := NewSigs(tid, kid, hash)
		st.sigs = append(st.sigs, sigs)
	}
	return nil
}

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

//保存私钥id 需要签名的hash 并且标记是否已经签名
//db.sigs.ensureIndex({tid:1})
type TSigs struct {
	Id     primitive.ObjectID `bson:"_id"`  //id
	TxId   xginx.HASH256      `bson:"tid"`  //交易id
	KeyId  string             `bson:"kid"`  //私钥id
	Hash   []byte             `bson:"hash"` //签名hash
	IsSign bool               `bson:"sigb"` //是否签名
	Sigs   xginx.SigBytes     `bson:"sigs"` //签名结果
}

//签名并保存
func (sig *TSigs) Sign(db IDbImp, pw ...string) error {
	pri, err := db.GetPrivate(sig.KeyId)
	if err != nil {
		return err
	}
	pkv, err := pri.ToPrivate(pw...)
	if err != nil {
		return err
	}
	sb, err := pkv.Sign(sig.Hash)
	if err != nil {
		return err
	}
	return db.SetSigs(sig.Id, sb.GetSigs())
}

type TxSigs []*TSigs

//是否都签名了
func (ss TxSigs) IsSign() bool {
	for _, v := range ss {
		if !v.IsSign {
			return false
		}
	}
	return true
}

//db.txs.ensureIndex({uid:1})
type TTx struct {
	Id       []byte             `bson:"_id"`
	UserId   primitive.ObjectID `bson:"uid"` //谁创建的交易
	Ver      uint32             `bson:"ver"`
	Ins      []TTxIn            `bson:"ins"`
	Outs     []TTxOut           `bson:"outs"`
	LockTime uint32             `bson:"lt"`
}

//创建待签名对象
func NewSigs(tid xginx.HASH256, kid string, hash []byte) *TSigs {
	sigs := &TSigs{}
	sigs.Id = primitive.NewObjectID()
	sigs.TxId = tid
	sigs.KeyId = kid
	sigs.Hash = hash
	sigs.IsSign = false
	return sigs
}

//设置交易签名
type setsigner struct {
	db IDbImp //db接口
}

//查询签名数据并设置
func (st *setsigner) SignTx(singer xginx.ISigner) error {
	tx, in, out := singer.GetObjs()
	tid, err := tx.ID()
	if err != nil {
		return err
	}
	addr, err := out.Script.GetAddress()
	if err != nil {
		return err
	}
	acc, err := st.db.GetAccount(addr)
	if err != nil {
		return err
	}
	wits := acc.ToAccount().NewWitnessScript()
	hash, err := singer.GetSigHash()
	if err != nil {
		return err
	}
	for idx, pkh := range acc.Pkh {
		kid := GetPrivateId(pkh)
		sigs, err := st.db.GetSigs(tid, kid, hash)
		if err != nil {
			continue
		}
		//如果还未签名
		if !sigs.IsSign {
			return fmt.Errorf("kid %s not sign at %d", kid, idx)
		}
		wits.Sig = append(wits.Sig, sigs.Sigs)
	}
	//检测签名脚本
	if err := wits.Check(); err != nil {
		return err
	}
	script, err := wits.ToScript()
	if err != nil {
		return err
	}
	in.Script = script
	return nil
}

//转换为tx并将签名合并进去
func (stx *TTx) ToTx(db IDbImp, bi *xginx.BlockIndex) (*xginx.TX, error) {
	tx := xginx.NewTx()
	tx.Ver = xginx.VarUInt(stx.Ver)
	tx.Ins = []*xginx.TxIn{}
	tx.Outs = []*xginx.TxOut{}
	for _, in := range stx.Ins {
		iv := &xginx.TxIn{}
		iv.OutHash = xginx.NewHASH256(in.OutHash)
		iv.OutIndex = xginx.VarUInt(in.OutIndex)
		iv.Script = in.Script
		iv.Sequence = in.Sequence
		tx.Ins = append(tx.Ins, iv)
	}
	for _, out := range stx.Outs {
		ov := &xginx.TxOut{}
		ov.Value = xginx.Amount(out.Value)
		ov.Script = out.Script
		tx.Outs = append(tx.Outs, ov)
	}
	tx.LockTime = stx.LockTime
	//使用数据库中的签名设置脚本
	err := tx.Sign(bi, &setsigner{db: db})
	if err != nil {
		return nil, err
	}
	err = tx.Check(bi, true)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

//创建交易
func (u *TUsers) NewTTx(tx *xginx.TX) *TTx {
	return NewTTx(u.Id, tx)
}

func (u *TUsers) SaveTx(db IDbImp, tx *xginx.TX, lis ISaveSigs) (*TTx, error) {
	if !db.IsTx() {
		return nil, errors.New("need use tx")
	}
	stx := u.NewTTx(tx)
	err := db.InsertTx(stx)
	if err != nil {
		return nil, err
	}
	return stx, lis.SaveSigs()
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

//设置签名内容
func (ctx *dbimp) SetSigs(id primitive.ObjectID, sigs xginx.SigBytes) error {
	col := ctx.table(TSigName)
	doc := bson.M{"$set": bson.M{"sigb": true, "sigs": sigs}}
	return col.FindOneAndUpdate(ctx, bson.M{"_id": id}, doc).Err()
}

//获取签名对象
func (ctx *dbimp) GetSigs(tid xginx.HASH256, kid string, hash []byte) (*TSigs, error) {
	col := ctx.table(TSigName)
	res := col.FindOne(ctx, bson.M{"tid": tid, "kid": kid, "hash": hash})
	v := &TSigs{}
	err := res.Decode(v)
	return v, err
}

//获取交易相关的签名对象
func (ctx *dbimp) ListSigs(tid xginx.HASH256) (TxSigs, error) {
	col := ctx.table(TSigName)
	iter, err := col.Find(ctx, bson.M{"tid": tid})
	if err != nil {
		return nil, err
	}
	defer iter.Close(ctx)
	rets := TxSigs{}
	for iter.Next(ctx) {
		v := &TSigs{}
		err := iter.Decode(v)
		if err != nil {
			return nil, err
		}
		rets = append(rets, v)
	}
	return rets, nil
}

//添加一个私钥
func (ctx *dbimp) InsertSigs(sigs *TSigs) error {
	col := ctx.table(TSigName)
	_, err := col.InsertOne(ctx, sigs)
	return err
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
	//删除交易对应的签名列表
	col := ctx.table(TSigName)
	_, err := col.DeleteMany(ctx, bson.M{"tid": id})
	if err != nil {
		return err
	}
	//删除交易
	col = ctx.table(TTxName)
	_, err = col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

//添加一个私钥
func (ctx *dbimp) InsertTx(tx *TTx) error {
	col := ctx.table(TTxName)
	_, err := col.InsertOne(ctx, tx)
	return err
}
