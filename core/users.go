package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//用户表
const (
	TUsersName = "users"
)

//TUser 用户管理
type TUser struct {
	ID     primitive.ObjectID `bson:"_id"`    //id
	Mobile string             `bson:"mobile"` //手机号
	Pass   xginx.HASH256      `bson:"pass"`   //hash256登陆密码
	Keys   string             `bson:"keys"`   //b58编码存储的DeterKey内容,如果创建用户时设置了密码，这里会被加密
	Cipher CipherType         `bson:"cipher"` //key加密方式
	Idx    uint32             `bson:"idx"`    //keys idx
	Token  string             `bson:"token"`  //登陆token
	PushID string             `bson:"pid"`    //推送id
}

//NewUser 创建用户
//mobile 手机号
//upass 登陆密码
//kpass 存在设置密钥加密密码
func NewUser(mobile string, upass string, kpass ...string) (*TUser, error) {
	ndk := NewDeterKey()
	u := &TUser{}
	u.ID = primitive.NewObjectID()
	u.Mobile = mobile
	if len(kpass) > 0 && kpass[0] != "" {
		u.Cipher = CipherTypeAes
	} else {
		u.Cipher = CipherTypeNone
	}
	keys, err := ndk.Dump(kpass...)
	if err != nil {
		return nil, err
	}
	u.Keys = keys
	u.Idx = 0
	u.Pass = xginx.Hash256From([]byte(upass))
	return u, nil
}

//ImportAccount 导入账户
func (u *TUser) ImportAccount(db IDbImp, acc *xginx.Account, exp time.Duration, desc string, tags []string, pass ...string) (*TAccount, error) {
	if !db.IsTx() {
		return nil, fmt.Errorf("must use tx")
	}
	nacc, err := NewAccountFrom([]primitive.ObjectID{u.ID}, acc, desc, tags)
	if err != nil {
		return nil, err
	}
	//保存私钥
	for pkh, pri := range acc.Pris {
		id := GetPrivateID(pkh)
		//如果存在
		_, err := db.GetPrivate(id)
		if err == nil {
			continue
		}
		pri, err := NewPrivateFrom(u.ID, pri, exp, "import", pass...)
		if err != nil {
			return nil, err
		}
		//保存导入账户的ID
		pri.ParentID = string(nacc.ID)
		err = db.InsertPrivate(pri)
		if err != nil {
			return nil, err
		}
	}
	err = db.InsertAccount(nacc)
	return nacc, err
}

//GetDeterKey 获取密钥
func (u *TUser) GetDeterKey(pass ...string) (*DeterKey, error) {
	if u.Cipher == CipherTypeAes && (len(pass) == 0 || pass[0] == "") {
		return nil, errors.New("encrypt keys miss pass")
	}
	return LoadDeterKey(u.Keys, pass...)
}

//CheckPass 检测登陆密码
func (u *TUser) CheckPass(pass string) bool {
	hv := xginx.Hash256From([]byte(pass))
	return len(pass) > 0 && hv.Equal(u.Pass)
}

//ListTxs 获取用户相关的交易
func (u *TUser) ListTxs(db IDbImp, sign bool) ([]*TTx, error) {
	return db.ListUserTxs(u.ID, sign)
}

//ListAccounts 获取用户相关的账号
func (u *TUser) ListAccounts(db IDbImp) ([]*TAccount, error) {
	return db.ListAccounts(u.ID)
}

//ListCoins 获取用户余额
func (u *TUser) ListCoins(db IDbImp, bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
	accs, err := db.ListAccounts(u.ID)
	if err != nil {
		return nil, err
	}
	s := &xginx.CoinsState{}
	for _, acc := range accs {
		cs, err := acc.ListCoins(bi)
		if err != nil {
			return nil, err
		}
		s.Merge(cs)
	}
	return s, nil
}

func (ctx *dbimp) SetUserToken(uid primitive.ObjectID, tk string) error {
	col := ctx.table(TUsersName)
	sr := col.FindOneAndUpdate(ctx, bson.M{"_id": uid}, bson.M{"$set": bson.M{"token": tk}})
	return sr.Err()
}

//获取用户相关的账户
func (ctx *dbimp) ListAccounts(uid primitive.ObjectID) ([]*TAccount, error) {
	col := ctx.table(TAccountName)
	rets := []*TAccount{}
	iter, err := col.Find(ctx, bson.M{"uid": uid})
	if err != nil {
		return nil, err
	}
	for iter.Next(ctx) {
		a := &TAccount{}
		err := iter.Decode(a)
		if err != nil {
			return nil, err
		}
		rets = append(rets, a)
	}
	err = iter.Close(ctx)
	if err != nil {
		return nil, err
	}
	return rets, nil
}

//获取一个用户信息
func (ctx *dbimp) GetUserInfoWithMobile(mobile string) (*TUser, error) {
	col := ctx.table(TUsersName)
	res := col.FindOne(ctx, bson.M{"mobile": mobile})
	v := &TUser{}
	err := res.Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//获取一个用户信息
func (ctx *dbimp) GetUserInfo(id interface{}) (*TUser, error) {
	col := ctx.table(TUsersName)
	objID := ToObjectID(id)
	v := &TUser{}
	err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//修改主私钥密码，修改派生密码
func (ctx *dbimp) SetUserKeyPass(uid primitive.ObjectID, old string, new string) error {
	if !ctx.IsTx() {
		return errors.New("use tx")
	}
	user, err := ctx.GetUserInfo(uid)
	if err != nil {
		return err
	}
	dk, err := user.GetDeterKey(old)
	if err != nil {
		return err
	}
	keys, err := dk.Dump(new)
	if err != nil {
		return err
	}
	col := ctx.table(TUsersName)
	_, err = col.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"keys": keys}})
	return err
}

// 设置用户推送id
func (ctx *dbimp) SetPushID(uid primitive.ObjectID, pid string) error {
	col := ctx.table(TUsersName)
	sr := col.FindOneAndUpdate(ctx, bson.M{"_id": uid}, bson.M{"$set": bson.M{"pid": pid}})
	return sr.Err()
}

//删除用户
func (ctx *dbimp) DeleteUser(id interface{}) error {
	if !ctx.IsTx() {
		return errors.New("use tx")
	}
	uid := ToObjectID(id)
	//删除用户的私钥
	col := ctx.table(TPrivatesName)
	_, err := col.DeleteMany(ctx, bson.M{"uid": uid})
	if err != nil {
		return err
	}
	//删除用户创建的账号
	col = ctx.table(TAccountName)
	_, err = col.UpdateMany(ctx, bson.M{"uid": uid}, bson.M{"$pull": bson.M{"uid": uid}})
	if err != nil {
		return err
	}
	_, err = col.DeleteMany(ctx, bson.M{"uid": bson.M{"$size": 0}})
	if err != nil {
		return err
	}
	//删除需要用户签名的数据
	col = ctx.table(TSigName)
	_, err = col.DeleteMany(ctx, bson.M{"uid": uid})
	if err != nil {
		return err
	}
	//删除用户交易
	col = ctx.table(TTxName)
	_, err = col.DeleteMany(ctx, bson.M{"uid": uid})
	if err != nil {
		return err
	}
	//删除用户信息
	col = ctx.table(TUsersName)
	_, err = col.DeleteOne(ctx, bson.M{"_id": uid})
	return err
}

//添加一个用户
func (ctx *dbimp) InsertUser(obj *TUser) error {
	_, err := ctx.GetUserInfoWithMobile(obj.Mobile)
	if err == nil {
		return errors.New("user exists")
	}
	col := ctx.table(TUsersName)
	_, err = col.InsertOne(ctx, obj)
	return err
}
