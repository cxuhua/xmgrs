package db

import (
	"errors"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TPrivatesName = "privates"
)

type CipherType int

const (
	CipherTypeNone CipherType = 0
	CipherTypeAes  CipherType = 1

	PrivateIDPrefix = "kp"
)

func GetPrivateId(pkh xginx.HASH160) string {
	id, err := xginx.EncodeAddressWithPrefix(PrivateIDPrefix, pkh)
	if err != nil {
		panic(err)
	}
	return id
}

func NewPrivate(uid primitive.ObjectID, pri *xginx.PrivateKey) *TPrivate {
	dp := &TPrivate{}
	dp.Pks = pri.PublicKey().GetPks()
	dp.Pkh = dp.Pks.Hash()
	dp.Id = GetPrivateId(dp.Pkh)
	dp.UserId = uid
	dp.Cipher = CipherTypeNone
	dp.Pri = pri.Encode()
	return dp
}

func (user *TUsers) NewPrivate() *TPrivate {
	pri, err := xginx.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	return NewPrivate(user.Id, pri)
}

//db.privates.ensureIndex({uid:1})
//私钥管理
type TPrivate struct {
	Id     string             `bson:"_id"`    //hash160作为id
	UserId primitive.ObjectID `bson:"uid"`    //所属用户
	Cipher CipherType         `bson:"cipher"` //加密方式
	Pks    xginx.PKBytes      `bson:"pks"`    //公钥
	Pkh    xginx.HASH160      `bson:"pkh"`    //公钥hash
	Pri    []byte             `bson:"pri"`    //私钥
}

func (p *TPrivate) GetPkh() xginx.HASH160 {
	id := xginx.HASH160{}
	copy(id[:], p.Id)
	return id
}

func (p *TPrivate) ToPrivate() *xginx.PrivateKey {
	pri := &xginx.PrivateKey{}
	err := pri.Decode(p.Pri)
	if err != nil {
		panic(err)
	}
	return pri
}

//获取用户的私钥
func (db *dbimp) ListPrivates(uid primitive.ObjectID) ([]*TPrivate, error) {
	col := db.table(TPrivatesName)
	iter, err := col.Find(db, bson.M{"uid": uid})
	if err != nil {
		return nil, err
	}
	rets := []*TPrivate{}
	for iter.Next(db) {
		v := TPrivate{}
		err := iter.Decode(&v)
		if err != nil {
			return nil, err
		}
		rets = append(rets, &v)
	}
	return rets, nil
}

func (db *dbimp) DeletePrivate(id string) error {
	col := db.table(TPrivatesName)
	_, err := col.DeleteOne(db, bson.M{"_id": id})
	return err
}

//获取私钥信息
func (db *dbimp) GetPrivate(id string) (*TPrivate, error) {
	col := db.table(TPrivatesName)
	v := &TPrivate{}
	err := col.FindOne(db, bson.M{"_id": id}).Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//添加一个私钥
func (db *dbimp) InsertPrivate(obj *TPrivate) error {
	_, err := db.GetPrivate(obj.Id)
	if err == nil {
		return errors.New("private exists")
	}
	col := db.table(TPrivatesName)
	_, err = col.InsertOne(db, obj)
	return err
}
