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

func NewPrivate(uid primitive.ObjectID, idx uint32, dk *DeterKey, desc string, pass ...string) *TPrivate {
	dp := &TPrivate{}
	ndk := dk.New(idx)
	dp.Pks = ndk.GetPks()
	dp.Pkh = dp.Pks.Hash()
	dp.Id = GetPrivateId(dp.Pkh)
	dp.Parent = dk.GetId()
	dp.UserId = uid
	if len(pass) > 0 && pass[0] != "" {
		dp.Cipher = CipherTypeAes
	} else {
		dp.Cipher = CipherTypeNone
	}
	dp.Desc = desc
	dp.Time = time.Now().Unix()
	keys, err := ndk.Dump(pass...)
	if err != nil {
		panic(err)
	}
	dp.Keys = keys
	return dp
}

//新建并写入私钥
func (user *TUser) NewPrivate(db IDbImp, desc string, pass ...string) (*TPrivate, error) {
	if !db.IsTx() {
		return nil, errors.New("need use tx")
	}
	dk, err := user.GetDeterKey(pass...)
	if err != nil {
		return nil, err
	}
	ptr := NewPrivate(user.Id, user.Idx, dk, desc, pass...)
	err = db.InsertPrivate(ptr)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TUsersName, user.Id)
	if err != nil {
		return nil, err
	}
	user.Idx++
	return ptr, nil
}

//私钥管理
type TPrivate struct {
	Id     string             `bson:"_id"`    //私钥id GetPrivateId(pkh)生成
	Parent string             `bson:"parent"` //父私钥id
	UserId primitive.ObjectID `bson:"uid"`    //所属用户
	Cipher CipherType         `bson:"cipher"` //加密方式
	Pks    xginx.PKBytes      `bson:"pks"`    //公钥
	Pkh    xginx.HASH160      `bson:"pkh"`    //公钥hash
	Keys   string             `bson:"keys"`   //私钥内容
	Idx    uint32             `bson:"idx"`    //索引
	Time   int64              `bson:"time"`   //创建时间
	Desc   string             `bson:"desc"`   //描述
}

//加载密钥
func (p *TPrivate) GetDeter(pass ...string) (*DeterKey, error) {
	return LoadDeterKey(p.Keys, pass...)
}

//pass存在启用加密方式
func (p *TPrivate) New(db IDbImp, desc string, pass ...string) (*TPrivate, error) {
	dk, err := p.GetDeter(pass...)
	if err != nil {
		return nil, err
	}
	pri := NewPrivate(p.UserId, p.Idx, dk, desc, pass...)
	err = db.InsertPrivate(pri)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TPrivatesName, p.Id)
	if err != nil {
		return nil, err
	}
	p.Idx++
	return pri, nil
}

//pw 根据加密方式暂时解密生成私钥对象
func (p *TPrivate) ToPrivate(pass ...string) (*xginx.PrivateKey, error) {
	//如果有加密，密码不能为空
	if p.Cipher == CipherTypeAes && (len(pass) == 0 || pass[0] == "") {
		return nil, errors.New("miss keys pass")
	}
	dk, err := p.GetDeter(pass...)
	if err != nil {
		return nil, err
	}
	return dk.GetPrivateKey()
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

func (ctx *dbimp) IncDeterIdx(tbl string, id interface{}) error {
	col := ctx.table(tbl)
	doc := bson.M{"$inc": bson.M{"idx": 1}}
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
