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

//签名表和交易表
const (
	TTxName  = "txs"
	TSigName = "sigs"
)

//ISaveSigs 保存签名接口
type ISaveSigs interface {
	SaveSigs() error
}

//DbSignListener 用来分析并保存签名对象
type DbSignListener struct {
	user *TUser //当前用户
	db   IDbImp //db接口
	sigs []*TSigs
}

//NewSignListener 创建签名列表
func NewSignListener(db IDbImp, user *TUser) *DbSignListener {
	return &DbSignListener{
		user: user,
		db:   db,
		sigs: []*TSigs{},
	}
}

//GetSigs 获取需要保存的交易签名列表
func (st *DbSignListener) GetSigs() []*TSigs {
	return st.sigs
}

//SaveSigs 保存
func (st *DbSignListener) SaveSigs() error {
	return st.db.InsertSigs(st.sigs...)
}

//GetAcc 获取金额对应的账户
func (st *DbSignListener) GetAcc(ckv *xginx.CoinKeyValue) (*xginx.Account, error) {
	acc, err := st.db.GetAccount(ckv.GetAddress())
	if err != nil {
		return nil, err
	}
	//只需要账号信息，不需要私钥
	return acc.ToAccount(st.db, false)
}

//GetTxOutExec 获取输出执行脚本 addr 输出的地址
func (st *DbSignListener) GetTxOutExec(addr xginx.Address) []byte {
	return xginx.DefaultLockedScript
}

//GetTxInExec 获取输入执行脚本 ckv消费的金额对象
func (st *DbSignListener) GetTxInExec(ckv *xginx.CoinKeyValue) []byte {
	return xginx.DefaultInputScript
}

//GetCoins 获取使用的金额
func (st *DbSignListener) GetCoins() xginx.Coins {
	bi := xginx.GetBlockIndex()
	ds, err := st.user.ListCoins(st.db, bi)
	if err != nil {
		return nil
	}
	return ds.Coins.Sort()
}

//GetKeep 获取找零地址
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

//SignTx 获取签名信息,保存需要签名的信息
func (st *DbSignListener) SignTx(singer xginx.ISigner, pass ...string) error {
	addr := singer.GetOutAddress()
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
	for _, kid := range acc.Kid {
		pk, err := st.db.GetPrivate(kid)
		if err != nil {
			return err
		}
		sigs := NewSigs(tid, pk.UserID, pk.ID, hash, idx)
		st.sigs = append(st.sigs, sigs)
	}
	return nil
}

//TTxIn 输入
type TTxIn struct {
	OutHash  []byte       `bson:"oid"`
	OutIndex uint32       `bson:"idx"`
	Script   xginx.Script `bson:"script"`
	Sequence uint32       `bson:"seq"`
}

//NewTTxIn 创建输入
func NewTTxIn(in *xginx.TxIn) TTxIn {
	vi := TTxIn{}
	vi.OutHash = in.OutHash[:]
	vi.OutIndex = in.OutIndex.ToUInt32()
	vi.Script = in.Script.Clone()
	vi.Sequence = in.Sequence.ToUInt32()
	return vi
}

//ToTxIn 转换
func (in TTxIn) ToTxIn() *xginx.TxIn {
	iv := &xginx.TxIn{}
	iv.OutHash = xginx.NewHASH256(in.OutHash)
	iv.OutIndex = xginx.VarUInt(in.OutIndex)
	iv.Script = in.Script.Clone()
	iv.Sequence = xginx.VarUInt(in.Sequence)
	return iv
}

//TTxOut 输出
type TTxOut struct {
	Value  int64        `bson:"value"`
	Script xginx.Script `bson:"script"`
}

//NewTTXOut 创建输出
func NewTTXOut(out *xginx.TxOut) TTxOut {
	vo := TTxOut{}
	vo.Value = int64(out.Value)
	vo.Script = out.Script.Clone()
	return vo
}

//ToTxOut 转换
func (out TTxOut) ToTxOut() *xginx.TxOut {
	ov := &xginx.TxOut{}
	ov.Value = xginx.Amount(out.Value)
	ov.Script = out.Script.Clone()
	return ov
}

//TSigs 保存私钥id 需要签名的hash 并且标记是否已经签名
type TSigs struct {
	ID     primitive.ObjectID `bson:"_id"`  //ID
	UserID primitive.ObjectID `bson:"uid"`  //私钥所属用户
	TxID   xginx.HASH256      `bson:"tid"`  //交易id
	KeyID  string             `bson:"kid"`  //私钥id
	Hash   []byte             `bson:"hash"` //需要签名的HASH数据
	Idx    int                `bson:"idx"`  //输入索引
	IsSign bool               `bson:"sigb"` //是否签名
	Sigs   xginx.SigBytes     `bson:"sigs"` //签名结果
}

//Sign 签名并保存
func (sig *TSigs) Sign(db IDbImp, pass ...string) error {
	//如果已经签名直接返回成功
	if sig.IsSign {
		return nil
	}
	pri, err := db.GetPrivate(sig.KeyID)
	if err != nil {
		return err
	}
	xpri, err := pri.ToPrivate(pass...)
	if err != nil {
		return err
	}
	sb, err := xpri.Sign(sig.Hash)
	if err != nil {
		return err
	}
	err = db.SetSigs(sig.ID, sb.GetSigs())
	if err == nil {
		sig.Sigs = sb.GetSigs()
	}
	return err
}

//TxSigs 签名集合
type TxSigs []*TSigs

//IsSign 是否都签名了
func (ss TxSigs) IsSign() bool {
	for _, v := range ss {
		if !v.IsSign {
			return false
		}
	}
	return true
}

//TTxState 交易状态
type TTxState int

//交易状态定义
const (
	TTxStateNew    TTxState = 0 //新交易
	TTxStateSign   TTxState = 1 //已签名
	TTxStatePool   TTxState = 2 //进入交易池
	TTxStateBlock  TTxState = 3 //进入区块
	TTxStateCancel TTxState = 4 //作废
)

//TTx 临时交易信息
type TTx struct {
	ID     []byte             `bson:"_id"`    //交易id
	UserID primitive.ObjectID `bson:"uid"`    //谁创建的交易
	Ver    uint32             `bson:"ver"`    //TxVer
	Ins    []TTxIn            `bson:"ins"`    //TxInputs
	Outs   []TTxOut           `bson:"outs"`   //TxOuts
	Script xginx.Script       `bson:"script"` //交易脚本
	Time   int64              `bson:"time"`   //创建时间
	Desc   string             `bson:"desc"`   //TxDesc
	State  TTxState           `bson:"state"`  //TTxState*
}

//NewSigs 创建待签名对象
func NewSigs(tid xginx.HASH256, uid primitive.ObjectID, kid string, hash []byte, idx int) *TSigs {
	sigs := &TSigs{}
	sigs.ID = primitive.NewObjectID()
	sigs.UserID = uid
	sigs.TxID = tid
	sigs.KeyID = kid
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
func (st *setsigner) SignTx(singer xginx.ISigner, pass ...string) error {
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
	//转换为xginx acc结构，不需要私钥
	xacc, err := acc.ToAccount(st.db, false, pass...)
	if err != nil {
		return err
	}
	//创建脚本
	wits := xacc.NewWitnessScript()
	hash, err := singer.GetSigHash()
	if err != nil {
		return err
	}
	//获取每个密钥的签名
	for idx, kid := range acc.Kid {
		//获取需要签名的记录
		sigs, err := st.db.GetSigs(txid, kid, hash, iidx)
		if err != nil {
			continue
		}
		//如果还未签名
		if !sigs.IsSign {
			return fmt.Errorf("kid %s not sign at %d", kid, idx)
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

//SetTxState 设置交易状态
func (stx *TTx) SetTxState(db IDbImp, state TTxState) error {
	return db.SetTxState(stx.ID, state)
}

//Verify 验证签名是否成功
func (stx *TTx) Verify(db IDbImp, bi *xginx.BlockIndex) bool {
	//转换成功校验就成功
	_, err := stx.ToTx(db, bi)
	return err == nil
}

//ToTx 转换为tx并将签名合并进去
func (stx *TTx) ToTx(db IDbImp, bi *xginx.BlockIndex, pass ...string) (*xginx.TX, error) {
	tx := xginx.NewTx(0)
	tx.Ver = xginx.VarUInt(stx.Ver)
	for _, in := range stx.Ins {
		tx.Ins = append(tx.Ins, in.ToTxIn())
	}
	for _, out := range stx.Outs {
		tx.Outs = append(tx.Outs, out.ToTxOut())
	}
	tx.Script = stx.Script.Clone()
	//使用数据库中的签名设置脚本
	err := tx.Sign(bi, &setsigner{db: db}, pass...)
	if err != nil {
		return nil, err
	}
	tid, err := tx.ID()
	if err != nil {
		return nil, err
	}
	//检测交易id是否正确
	if !bytes.Equal(tid[:], stx.ID) {
		return nil, errors.New("tx ttx id error")
	}
	//校验交易
	err = tx.Check(bi, true)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

//NewTTx 创建交易
func (u *TUser) NewTTx(tx *xginx.TX) *TTx {
	return NewTTx(u.ID, tx)
}

//SaveTx 保存用户的交易
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

//NewTTx 从区块交易创建
func NewTTx(uid primitive.ObjectID, tx *xginx.TX) *TTx {
	v := &TTx{}
	v.State = TTxStateNew
	v.ID = tx.MustID().Bytes()
	v.Ver = tx.Ver.ToUInt32()
	for _, in := range tx.Ins {
		v.Ins = append(v.Ins, NewTTxIn(in))
	}
	for _, out := range tx.Outs {
		v.Outs = append(v.Outs, NewTTXOut(out))
	}
	v.Script = tx.Script.Clone()
	v.UserID = uid
	v.Time = time.Now().Unix()
	return v
}

//ListUserTxs 获取用户需要处理的交易
//sign 是否签名
func (ctx *dbimp) ListUserTxs(uid primitive.ObjectID, sign bool) ([]*TTx, error) {
	ids := map[xginx.HASH256]bool{}
	//获取需要uid签名的记录
	col := ctx.table(TSigName)
	iter, err := col.Find(ctx, bson.M{"uid": uid, "sigb": sign})
	if err != nil {
		return nil, err
	}
	//获取交易id列表
	for iter.Next(ctx) {
		v := &TSigs{}
		err := iter.Decode(v)
		if err != nil {
			return nil, err
		}
		ids[v.TxID] = true
	}
	err = iter.Close(ctx)
	if err != nil {
		return nil, err
	}
	//获取对应的交易信息
	txs := []*TTx{}
	for tid := range ids {
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

//获取交易签名记录
func (ctx *dbimp) GetSigs(tid xginx.HASH256, kid string, hash []byte, idx int) (*TSigs, error) {
	col := ctx.table(TSigName)
	res := col.FindOne(ctx, bson.M{"tid": tid, "kid": kid, "hash": hash, "idx": idx})
	v := &TSigs{}
	err := res.Decode(v)
	return v, err
}

//获取交易需要签名的记录
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

//设置交易状态
func (ctx *dbimp) SetTxState(id []byte, state TTxState) error {
	col := ctx.table(TTxName)
	sr := col.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"state": state}},
	)
	return sr.Err()
}

//获取账户信息
func (ctx *dbimp) GetTx(id []byte) (*TTx, error) {
	col := ctx.table(TTxName)
	a := &TTx{}
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(a)
	return a, err
}

//删除交易信息
func (ctx *dbimp) DeleteTx(id []byte) error {
	if !ctx.IsTx() {
		return errors.New("need use tx")
	}
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
