package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddUsers(t *testing.T) {
	//添加测试用户
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		u := NewUser("17716858036", []byte("xh0714"))
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
		_, err = u.NewPrivate(db, "第一个私钥")
		if err != nil {
			return err
		}
		_, err = u.NewPrivate(db, "第二个私钥")
		if err != nil {
			return err
		}
		if u.Deter.Index != 2 {
			return errors.New("count error")
		}
		err = db.DeleteUser(u.Id)
		if err != nil {
			return err
		}
		return err
	})
	assert.NoError(t, err)
}
