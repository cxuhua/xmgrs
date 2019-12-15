package db

import (
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

//获取一个用户信息
func (db *dbimp) GetUserInfoWithMobile(mobile string) (*TUsers, error) {
	col := db.table(TUsersName)
	res := col.FindOne(db, bson.M{"mobile": mobile})
	if res.Err() != nil {
		return nil, res.Err()
	}
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
	res := col.FindOne(db, bson.M{"_id": objID})
	if res.Err() != nil {
		return nil, res.Err()
	}
	v := &TUsers{}
	err := res.Decode(v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

//添加一个用户
func (db *dbimp) InsertUser(obj *TUsers) error {
	col := db.table(TUsersName)
	_, err := col.InsertOne(db, obj)
	return err
}
