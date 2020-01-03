package core

import (
	"errors"

	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TUsersName = "users"
)

//用户管理
type TUser struct {
	Id     primitive.ObjectID `bson:"_id"`
	Mobile string             `bson:"mobile"`
	Pass   xginx.HASH256      `bson:"pass"`  //hash256密钥
	Deter  *DeterKey          `bson:"deter"` //确定性key
	Token  string             `bson:"token"`
}

func NewUser(mobile string, pass []byte) *TUser {
	u := &TUser{}
	u.Id = primitive.NewObjectID()
	u.Mobile = mobile
	u.Deter = NewDeterKey()
	u.Pass = xginx.Hash256From(pass)
	return u
}

func (u *TUser) CheckPass(pass string) bool {
	hv := xginx.Hash256From([]byte(pass))
	return hv.Equal(u.Pass)
}

func (u *TUser) ListTxs(db IDbImp, sign bool) ([]*TTx, error) {
	return db.ListUserTxs(u.Id, sign)
}

//获取用户相关的账号
func (u *TUser) ListAccounts(db IDbImp) ([]*TAccount, error) {
	return db.ListAccounts(u.Id)
}

//获取用户余额
func (u *TUser) ListCoins(db IDbImp, bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
	accs, err := db.ListAccounts(u.Id)
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
	_, err := col.UpdateOne(ctx, bson.M{"_id": uid}, bson.M{"$set": bson.M{"token": tk}})
	return err
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
