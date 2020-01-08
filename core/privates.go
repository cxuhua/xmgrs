package core

import (
	"errors"
	"time"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TPrivatesName = "privates"
)

type CipherType int

const (
	CipherTypeNone  CipherType = 0
	CipherTypeAes   CipherType = 1    //aes加密方式
	PrivateIDPrefix            = "kp" //私钥前缀
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
	dp.Pks = dp.Deter.GetPks()
	dp.Pkh = dp.Pks.Hash()
	dp.Id = GetPrivateId(dp.Pkh)
	dp.UserId = uid
	dp.Cipher = CipherTypeNone
	dp.Desc = desc
	dp.Time = time.Now().Unix()
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
	Time   int64              `json:"time"`   //创建时间
	Desc   string             `bson:"desc"`   //描述
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
