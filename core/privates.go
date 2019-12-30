package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"math/big"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//确定性私钥地址
type DeterKey struct {
	Root  []byte `bson:"root"` //私钥内容
	Key   []byte `bson:"key"`  //私钥编码
	Index uint32 `bson:"idx"`  //自增索引
}

//加载key
func LoadDeterKey(s string) (*DeterKey, error) {
	data, err := xginx.B58Decode(s, xginx.BitcoinAlphabet)
	if err != nil {
		return nil, err
	}
	if len(data) != 68 {
		return nil, errors.New("data length error")
	}
	dl := len(data)
	hbytes := xginx.Hash256(data[:dl-4])
	if !bytes.Equal(hbytes[:4], data[dl-4:]) {
		return nil, errors.New("checksum error")
	}
	dk := &DeterKey{
		Root: data[:32],
		Key:  data[32 : dl-4],
	}
	return dk, nil
}

func (k DeterKey) GetPrivateKey() *xginx.PrivateKey {
	pri := &xginx.PrivateKey{}
	pri.D = new(big.Int).SetBytes(k.Root)
	return pri
}

func (k DeterKey) Dump() string {
	data := append([]byte{}, k.Root...)
	data = append(data, k.Key...)
	hbytes := xginx.Hash256(data)
	data = append(data, hbytes[:4]...)
	return xginx.B58Encode(data, xginx.BitcoinAlphabet)
}

func (k DeterKey) String() string {
	return fmt.Sprintf("%s %s", hex.EncodeToString(k.Root), hex.EncodeToString(k.Key))
}

//派生一个密钥
func (k *DeterKey) New(idx uint32) *DeterKey {
	h := hmac.New(func() hash.Hash {
		return sha512.New()
	}, k.Key)
	_, err := h.Write(k.Root)
	if err != nil {
		panic(err)
	}
	err = binary.Write(h, binary.BigEndian, idx)
	if err != nil {
		panic(err)
	}
	b := h.Sum(nil)
	if len(b) != 64 {
		panic(errors.New("hmac sha512 sum error"))
	}
	return &DeterKey{
		Root: b[:32],
		Key:  b[32:],
	}
}

func NewDeterKey() *DeterKey {
	pri, err := xginx.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	k := &DeterKey{}
	k.Root = pri.Bytes()
	k.Key = make([]byte, 32)
	_, err = rand.Read(k.Key)
	if err != nil {
		panic(err)
	}
	return k
}

const (
	TPrivatesName = "privates"
)

type CipherType int

const (
	CipherTypeNone  CipherType = 0
	CipherTypeAes   CipherType = 1
	PrivateIDPrefix            = "kp"
)

func GetPrivateId(pkh xginx.HASH160) string {
	id, err := xginx.EncodeAddressWithPrefix(PrivateIDPrefix, pkh)
	if err != nil {
		panic(err)
	}
	return id
}

func NewPrivate(uid primitive.ObjectID, dk *DeterKey, desc string) *TPrivate {
	dp := &TPrivate{}
	dp.Deter = dk.New(dk.Index)
	dp.Pks = dp.Deter.GetPrivateKey().PublicKey().GetPks()
	dp.Pkh = dp.Pks.Hash()
	dp.Id = GetPrivateId(dp.Pkh)
	dp.UserId = uid
	dp.Cipher = CipherTypeNone
	dp.Desc = desc
	return dp
}

//创建账号并保存
func (user *TUser) SaveAccount(db IDbImp, num uint8, less uint8, arb bool) (*TAccount, error) {
	return SaveAccount(db, user, num, less, arb)
}

//新建并写入私钥
func (user *TUser) NewPrivate(db IDbImp, desc string) (*TPrivate, error) {
	if !db.IsTx() {
		return nil, errors.New("need use tx")
	}
	ptr := NewPrivate(user.Id, user.Deter, desc)
	err := db.InsertPrivate(ptr)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TUsersName, user.Id)
	if err != nil {
		return nil, err
	}
	user.Deter.Index++
	return ptr, nil
}

//私钥管理
type TPrivate struct {
	Id     string             `bson:"_id"`    //hash160作为id
	UserId primitive.ObjectID `bson:"uid"`    //所属用户
	Cipher CipherType         `bson:"cipher"` //加密方式
	Pks    xginx.PKBytes      `bson:"pks"`    //公钥
	Pkh    xginx.HASH160      `bson:"pkh"`    //公钥hash
	Deter  *DeterKey          `bson:"deter"`  //私钥内容
	Desc   string             `bson:"desc"`
}

func (p *TPrivate) GetPkh() xginx.HASH160 {
	id := xginx.HASH160{}
	copy(id[:], p.Id)
	return id
}

func (p *TPrivate) New(db IDbImp, desc string) (*TPrivate, error) {
	pri := NewPrivate(p.UserId, p.Deter, desc)
	err := db.InsertPrivate(pri)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TPrivatesName, p.Id)
	if err != nil {
		return nil, err
	}
	p.Deter.Index++
	return pri, nil
}

//pw 根据加密方式暂时解密生成私钥对象
func (p *TPrivate) ToPrivate() *xginx.PrivateKey {
	return p.Deter.GetPrivateKey()
}

//获取用户的私钥
func (ctx *dbimp) ListPrivates(uid primitive.ObjectID) ([]*TPrivate, error) {
	col := ctx.table(TPrivatesName)
	iter, err := col.Find(ctx, bson.M{"uid": uid})
	if err != nil {
		return nil, err
	}
	defer iter.Close(ctx)
	rets := []*TPrivate{}
	for iter.Next(ctx) {
		v := &TPrivate{}
		err := iter.Decode(v)
		if err != nil {
			return nil, err
		}
		rets = append(rets, v)
	}
	return rets, nil
}

func (ctx *dbimp) DeletePrivate(id string) error {
	col := ctx.table(TPrivatesName)
	_, err := col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

//获取私钥信息
func (ctx *dbimp) GetPrivate(id string) (*TPrivate, error) {
	col := ctx.table(TPrivatesName)
	v := &TPrivate{}
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (ctx *dbimp) IncDeterIdx(name string, id interface{}) error {
	col := ctx.table(name)
	doc := bson.M{"$inc": bson.M{"deter.idx": 1}}
	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, doc)
	return err
}

//添加一个私钥
func (ctx *dbimp) InsertPrivate(obj *TPrivate) error {
	if !ctx.IsTx() {
		return errors.New("need tx")
	}
	_, err := ctx.GetPrivate(obj.Id)
	if err == nil {
		return errors.New("private exists")
	}
	col := ctx.table(TPrivatesName)
	_, err = col.InsertOne(ctx, obj)
	return err
}
