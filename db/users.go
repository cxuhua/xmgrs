package db

import (
	"github.com/cxuhua/xginx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TUsersName = "users"
)

//db.users.ensureIndex({mobile:1})
//用户管理
type TUsers struct {
	Id     primitive.ObjectID `bson:"_id"`
	Mobile string             `bson:"mobile"`
	Pass   []byte             `bson:"pass"` //hash256密钥
	Token  string             `bson:"token"`
}

//获取用户余额
func (u *TUsers) ListCoins(db IDbImp, bi *xginx.BlockIndex) (*xginx.CoinsState, error) {
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

//获取用户相关的账户
func (db *dbimp) ListAccounts(uid primitive.ObjectID) ([]*TAccount, error) {
	keys, err := db.ListPrivates(uid)
	if err != nil {
		return nil, err
	}
	col := db.table(TAccountName)
	rmap := map[string]*TAccount{}
	for _, v := range keys {
		iter, err := col.Find(db, bson.M{"pkh": v.Pkh})
		if err != nil {
			return nil, err
		}
		for iter.Next(db) {
			a := &TAccount{}
			err := iter.Decode(a)
			if err != nil {
				return nil, err
			}
			rmap[a.Id] = a
		}
	}
	rets := []*TAccount{}
	for _, v := range rmap {
		rets = append(rets, v)
	}
	return rets, nil
}

//获取一个用户信息
func (db *dbimp) GetUserInfoWithMobile(mobile string) (*TUsers, error) {
	col := db.table(TUsersName)
	res := col.FindOne(db, bson.M{"mobile": mobile})
	v := &TUsers{}
	err := res.Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//获取一个用户信息
func (db *dbimp) GetUserInfo(id interface{}) (*TUsers, error) {
	col := db.table(TUsersName)
	objID := ToObjectID(id)
	v := &TUsers{}
	err := col.FindOne(db, bson.M{"_id": objID}).Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (db *dbimp) DeleteUser(id interface{}) error {
	col := db.table(TUsersName)
	objID := ToObjectID(id)
	_, err := col.DeleteOne(db, bson.M{"_id": objID})
	return err
}

//添加一个用户
func (db *dbimp) InsertUser(obj *TUsers) error {
	col := db.table(TUsersName)
	_, err := col.InsertOne(db, obj)
	return err
}
