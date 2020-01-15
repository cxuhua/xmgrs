package core

import (
	"errors"
	"time"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//私钥表名
const (
	TPrivatesName = "privates"
)

//CipherType 私钥加密类型
type CipherType int

//加密类型
const (
	CipherTypeNone  CipherType = 0
	CipherTypeAes   CipherType = 1    //aes加密方式
	PrivateIDPrefix            = "kp" //私钥前缀
)

//GetPrivateID 获取私钥ID
func GetPrivateID(pkh xginx.HASH160) string {
	id, err := xginx.EncodeAddressWithPrefix(PrivateIDPrefix, pkh)
	if err != nil {
		panic(err)
	}
	return id
}

//NewPrivate 创建一个私钥
func NewPrivate(uid primitive.ObjectID, idx uint32, dk *DeterKey, desc string, pass ...string) *TPrivate {
	dp := &TPrivate{}
	ndk := dk.New(idx)
	dp.Pks = ndk.GetPks()
	dp.Pkh = dp.Pks.Hash()
	dp.ID = GetPrivateID(dp.Pkh)
	dp.Parent = dk.GetID()
	dp.UserID = uid
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

//NewPrivate 新建并写入私钥
func (user *TUser) NewPrivate(db IDbImp, desc string, pass ...string) (*TPrivate, error) {
	if !db.IsTx() {
		return nil, errors.New("need use tx")
	}
	//如果是两个密码，第一个为主私钥密码, 第二个新私钥密码
	upass := []string{}
	kpass := []string{}
	if len(pass) == 2 {
		upass = []string{pass[0]}
		kpass = []string{pass[1]}
	} else if len(pass) == 1 {
		upass = []string{pass[0]}
		kpass = []string{pass[0]}
	}
	dk, err := user.GetDeterKey(upass...)
	if err != nil {
		return nil, err
	}
	ptr := NewPrivate(user.ID, user.Idx, dk, desc, kpass...)
	err = db.InsertPrivate(ptr)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TUsersName, user.ID)
	if err != nil {
		return nil, err
	}
	user.Idx++
	return ptr, nil
}

//TPrivate 私钥管理
type TPrivate struct {
	ID     string             `bson:"_id"`    //私钥id GetPrivateId(pkh)生成
	Parent string             `bson:"parent"` //父私钥id
	UserID primitive.ObjectID `bson:"uid"`    //所属用户
	Cipher CipherType         `bson:"cipher"` //加密方式
	Pks    xginx.PKBytes      `bson:"pks"`    //公钥
	Pkh    xginx.HASH160      `bson:"pkh"`    //公钥hash
	Keys   string             `bson:"keys"`   //私钥内容
	Idx    uint32             `bson:"idx"`    //索引
	Time   int64              `bson:"time"`   //创建时间
	Desc   string             `bson:"desc"`   //描述
}

//GetDeter 加载密钥
func (p *TPrivate) GetDeter(pass ...string) (*DeterKey, error) {
	return LoadDeterKey(p.Keys, pass...)
}

//New pass存在启用加密方式
func (p *TPrivate) New(db IDbImp, desc string, pass ...string) (*TPrivate, error) {
	dk, err := p.GetDeter(pass...)
	if err != nil {
		return nil, err
	}
	pri := NewPrivate(p.UserID, p.Idx, dk, desc, pass...)
	err = db.InsertPrivate(pri)
	if err != nil {
		return nil, err
	}
	err = db.IncDeterIdx(TPrivatesName, p.ID)
	if err != nil {
		return nil, err
	}
	p.Idx++
	return pri, nil
}

//ToPrivate  根据加密方式暂时解密生成私钥对象
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

func (ctx *dbimp) SetPrivateKeyPass(uid primitive.ObjectID, pid string, old string, new string) error {
	if !ctx.IsTx() {
		return errors.New("use tx")
	}
	pri, err := ctx.GetPrivate(pid)
	if err != nil {
		return err
	}
	if !ObjectIDEqual(pri.UserID, uid) {
		return errors.New("can't update key pass")
	}
	dk, err := pri.GetDeter(old)
	if err != nil {
		return err
	}
	keys, err := dk.Dump(new)
	if err != nil {
		return err
	}
	col := ctx.table(TPrivatesName)
	_, err = col.UpdateOne(ctx, bson.M{"_id": pri.ID, "uid": uid}, bson.M{"$set": bson.M{"keys": keys}})
	return err
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
	//没有引用账户才能删除
	num, err := ctx.GetPrivateRefs(id)
	if err != nil {
		return err
	}
	if num > 0 {
		return errors.New("has refs acc,can't delete")
	}
	col := ctx.table(TPrivatesName)
	_, err = col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (ctx *dbimp) GetUserPrivate(id string, uid primitive.ObjectID) (*TPrivate, error) {
	col := ctx.table(TPrivatesName)
	v := &TPrivate{}
	err := col.FindOne(ctx, bson.M{"_id": id, "uid": uid}).Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
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

func (ctx *dbimp) GetPrivateRefs(id string) (int, error) {
	col := ctx.table(TAccountName)
	num, err := col.CountDocuments(ctx, bson.M{"kid": id})
	return int(num), err
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
	_, err := ctx.GetPrivate(obj.ID)
	if err == nil {
		return errors.New("private exists")
	}
	col := ctx.table(TPrivatesName)
	_, err = col.InsertOne(ctx, obj)
	return err
}
