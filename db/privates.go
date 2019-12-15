package db

import (
	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	TPrivatesName = "privates"
)

type CipherType int

const (
	CipherTypeNone   CipherType = 0
	CipherTypeAes128 CipherType = 1
)

func NewPrivate(pri *xginx.PrivateKey) *TPrivate {
	dp := &TPrivate{}
	dp.Id = pri.PublicKey().Hash().Bytes()
	dp.Cipher = CipherTypeNone
	dp.Body = pri.Encode()
	return dp
}

//私钥管理
type TPrivate struct {
	Id     []byte     `bson:"_id"`    //hash160作为id
	Cipher CipherType `bson:"cipher"` //加密方式
	Body   []byte     `bson:"body"`   //内容
}

func (p *TPrivate) ToPrivate() *xginx.PrivateKey {
	pri := &xginx.PrivateKey{}
	err := pri.Decode(p.Body)
	if err != nil {
		panic(err)
	}
	return pri
}

func (db *dbimp) DeletePrivate(id []byte) error {
	col := db.table(TPrivatesName)
	_, err := col.DeleteOne(db, bson.M{"_id": id})
	return err
}

//获取私钥信息
func (db *dbimp) GetPrivate(id []byte) (*TPrivate, error) {
	col := db.table(TPrivatesName)
	res := col.FindOne(db, bson.M{"_id": id})
	if res.Err() != nil {
		return nil, res.Err()
	}
	v := &TPrivate{}
	err := res.Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//添加一个私钥
func (db *dbimp) InsertPrivate(obj *TPrivate) error {
	col := db.table(TPrivatesName)
	_, err := col.InsertOne(db, obj)
	return err
}
