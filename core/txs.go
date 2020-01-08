package core

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

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
	user *TUser //当前用户
	db   IDbImp //db接口
	sigs []*TSigs
}

func NewSignListener(db IDbImp, user *TUser) *DbSignListener {
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
	return st.db.InsertSigs(st.sigs...)
}

//获取金额对应的账户
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
	return ds.Coins.Sort()
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

//获取签名信息
func (st *DbSignListener) SignTx(singer xginx.ISigner) error {
	addr := singer.GetAddress()
	//获取对应的账号
	acc, err := st.db.GetAccount(addr)
	if err != nil {
		return err
	}
	tx, _, _, idx := singer.GetObjs()
	tid, err := tx.ID()
	if err != nil {
		return err
	}
	hash, err := singer.GetSigHash()
	if err != nil {
		return err
	}
	//分析账户用到的密钥，并保存记录等候签名
	for _, pkh := range acc.Pkh {
		kid := GetPrivateId(pkh)
		pk, err := st.db.GetPrivate(kid)
		if err != nil {
			return err
		}
		sigs := NewSigs(tid, pk.UserId, pk.Id, hash, idx)
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

func NewTTxIn(in *xginx.TxIn) TTxIn {
	vi := TTxIn{}
	vi.OutHash = in.OutHash[:]
	vi.OutIndex = in.OutIndex.ToUInt32()
	vi.Script = in.Script
	vi.Sequence = in.Sequence
	return vi
}

func (in TTxIn) ToTxIn() *xginx.TxIn {
	iv := &xginx.TxIn{}
	iv.OutHash = xginx.NewHASH256(in.OutHash)
	iv.OutIndex = xginx.VarUInt(in.OutIndex)
	iv.Script = in.Script
	iv.Sequence = in.Sequence
	return iv
}

type TTxOut struct {
	Value  int64        `bson:"value"`
	Script xginx.Script `bson:"script"`
}

func NewTTXOut(out *xginx.TxOut) TTxOut {
	vo := TTxOut{}
	vo.Value = int64(out.Value)
	vo.Script = out.Script
	return vo
}

func (out TTxOut) ToTxOut() *xginx.TxOut {
	ov := &xginx.TxOut{}
	ov.Value = xginx.Amount(out.Value)
	ov.Script = out.Script
	return ov
}

//保存私钥id 需要签名的hash 并且标记是否已经签名
type TSigs struct {
	Id     primitive.ObjectID `bson:"_id"`  //id
	UserId primitive.ObjectID `bson:"uid"`  //私钥所属用户
	TxId   xginx.HASH256      `bson:"tid"`  //交易id
	KeyId  string             `bson:"kid"`  //私钥id
	Hash   []byte             `bson:"hash"` //签名hash
	Idx    int                `bson:"idx"`  //输入索引
	IsSign bool               `bson:"sigb"` //是否签名
	Sigs   xginx.SigBytes     `bson:"sigs"` //签名结果
}

//签名并保存
func (sig *TSigs) Sign(db IDbImp) error {
	pri, err := db.GetPrivate(sig.KeyId)
	if err != nil {
		return err
	}
	sb, err := pri.ToPrivate().Sign(sig.Hash)
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

//临时交易信息
type TTx struct {
	Id       []byte             `bson:"_id"` //交易id
	UserId   primitive.ObjectID `bson:"uid"` //谁创建的交易
	Ver      uint32             `bson:"ver"`
	Ins      []TTxIn            `bson:"ins"`
	Outs     []TTxOut           `bson:"outs"`
	LockTime uint32             `bson:"lt"`
	Time     int64              `bson:"time"` //创建时间
	Desc     string             `bson:"desc"`
}

//创建待签名对象
func NewSigs(tid xginx.HASH256, uid primitive.ObjectID, kid string, hash []byte, idx int) *TSigs {
	sigs := &TSigs{}
	sigs.Id = primitive.NewObjectID()
	sigs.UserId = uid
	sigs.TxId = tid
	sigs.KeyId = kid
	sigs.Hash = hash
	sigs.Idx = idx
	sigs.IsSign = false
	return sigs
}

//设置交易签名
type setsigner struct {
	db IDbImp //db接口
}

//查询签名数据并设置脚本
func (st *setsigner) SignTx(singer xginx.ISigner) error {
	tx, in, out, iidx := singer.GetObjs()
	txid, err := tx.ID()
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
	//创建脚本
	wits := acc.ToAccount().NewWitnessScript()
	hash, err := singer.GetSigHash()
	if err != nil {
		return err
	}
	//获取每个密钥的签名
	for idx, pkh := range acc.Pkh {
		keyid := GetPrivateId(pkh)
		//获取需要签名的记录
		sigs, err := st.db.GetSigs(txid, keyid, hash, iidx)
		if err != nil {
			continue
		}
		//如果还未签名
		if !sigs.IsSign {
			return fmt.Errorf("kid %s not sign at %d", keyid, idx)
		}
		wits.Sig = append(wits.Sig, sigs.Sigs)
	}
	//检测脚本
	err = wits.Check()
	if err != nil {
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
	for _, in := range stx.Ins {
		tx.Ins = append(tx.Ins, in.ToTxIn())
	}
	for _, out := range stx.Outs {
		tx.Outs = append(tx.Outs, out.ToTxOut())
	}
	tx.LockTime = stx.LockTime
	//使用数据库中的签名设置脚本
	err := tx.Sign(bi, &setsigner{db: db})
	if err != nil {
		return nil, err
	}
	tid, err := tx.ID()
	if err != nil {
		return nil, err
	}
	//检测交易id是否正确
	if !bytes.Equal(tid[:], stx.Id) {
		return nil, errors.New("tx ttx id error")
	}
	//校验交易
	err = tx.Check(bi, true)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

//创建交易
func (u *TUser) NewTTx(tx *xginx.TX) *TTx {
	return NewTTx(u.Id, tx)
}

func (u *TUser) SaveTx(db IDbImp, tx *xginx.TX, lis ISaveSigs, desc ...string) (*TTx, error) {
	if !db.IsTx() {
		return nil, errors.New("need use tx")
	}
	stx := u.NewTTx(tx)
	if len(desc) > 0 {
		stx.Desc = strings.Join(desc, "")
	}
	err := db.InsertTx(stx)
	if err != nil {
		return nil, err
	}
	//保存签名
	return stx, lis.SaveSigs()
}

//从区块交易创建
func NewTTx(uid primitive.ObjectID, tx *xginx.TX) *TTx {
	v := &TTx{}
	v.Id = tx.MustID().Bytes()
	v.Ver = tx.Ver.ToUInt32()
	v.LockTime = tx.LockTime
	for _, in := range tx.Ins {
		v.Ins = append(v.Ins, NewTTxIn(in))
	}
	for _, out := range tx.Outs {
		v.Outs = append(v.Outs, NewTTXOut(out))
	}
	v.UserId = uid
	v.Time = time.Now().Unix()
	return v
}

//获取用户需要处理的交易
func (ctx *dbimp) ListUserTxs(uid primitive.ObjectID, sign bool) ([]*TTx, error) {
	ids := map[xginx.HASH256]bool{}
	col := ctx.table(TSigName)
	iter, err := col.Find(ctx, bson.M{"uid": uid, "sigb": sign})
	if err != nil {
		return nil, err
	}
	for iter.Next(ctx) {
		v := &TSigs{}
		err := iter.Decode(v)
		if err != nil {
			return nil, err
		}
		ids[v.TxId] = true
	}
	err = iter.Close(ctx)
	if err != nil {
		return nil, err
	}
	txs := []*TTx{}
	for tid, _ := range ids {
		tx, err := ctx.GetTx(tid[:])
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

//设置签名内容
func (ctx *dbimp) SetSigs(id primitive.ObjectID, sigs xginx.SigBytes) error {
	col := ctx.table(TSigName)
	doc := bson.M{"$set": bson.M{"sigb": true, "sigs": sigs}}
	return col.FindOneAndUpdate(ctx, bson.M{"_id": id}, doc).Err()
}

//获取签名对象
func (ctx *dbimp) GetSigs(tid xginx.HASH256, kid string, hash []byte, idx int) (*TSigs, error) {
	col := ctx.table(TSigName)
	res := col.FindOne(ctx, bson.M{"tid": tid, "kid": kid, "hash": hash, "idx": idx})
	v := &TSigs{}
	err := res.Decode(v)
	return v, err
}

//获取需要签名的记录
func (ctx *dbimp) ListUserSigs(uid primitive.ObjectID, tid xginx.HASH256) (TxSigs, error) {
	col := ctx.table(TSigName)
	iter, err := col.Find(ctx, bson.M{"uid": uid, "tid": tid, "sigb": false})
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

//获取交易相关的签名对象
func (ctx *dbimp) ListSigs(tid xginx.HASH256) (TxSigs, error) {
	col := ctx.table(TSigName)
	iter, err := col.Find(ctx, bson.M{"tid": tid, "sigb": false})
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
func (ctx *dbimp) InsertSigs(sigs ...*TSigs) error {
	col := ctx.table(TSigName)
	ds := []interface{}{}
	for _, sig := range sigs {
		ds = append(ds, sig)
	}
	_, err := col.InsertMany(ctx, ds)
	return err
}

//获取账户信息
func (ctx *dbimp) GetTx(id []byte) (*TTx, error) {
	col := ctx.table(TTxName)
	a := &TTx{}
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
