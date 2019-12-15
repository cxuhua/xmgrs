package db

import (
	"context"
	"errors"
	"log"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xginx"
)

func TestAddUsers(t *testing.T) {
	//添加测试用户
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		u := &TUsers{}
		u.Id = primitive.NewObjectID()
		u.Mobile = "17716858036"
		u.Pass = xginx.Hash256([]byte("xh0714"))
		err := db.InsertUser(u)
		if err != nil {
			return err
		}

		u1, err := db.GetUserInfo(u.Id)
		if err != nil {
			return err
		}
		if !ObjectIDEqual(u.Id, u1.Id) {
			return errors.New("find user error")
		}
		return nil
	})
	log.Println(err)
}
