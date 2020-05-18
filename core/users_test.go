package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddUsersWithKeyPassword(t *testing.T) {
	kpass := "11223344"
	//添加测试用户
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user, err := db.GetUserInfoWithMobile("17716858036")
		if err == nil {
			db.DeleteUser(user.ID)
		}
		u := NewUser("17716858036", "xh0714", kpass)
		err = db.InsertUser(u)
		if err != nil {
			return err
		}
		u1, err := db.GetUserInfo(u.ID)
		if err != nil {
			return err
		}
		if !ObjectIDEqual(u.ID, u1.ID) {
			return errors.New("find user error")
		}
		_, err = u.NewPrivate(db, DefaultExpTime, "第一个加密的私钥", kpass)
		if err != nil {
			return err
		}
		_, err = u.NewPrivate(db, DefaultExpTime, "第二个加密的私钥", kpass)
		if err != nil {
			return err
		}
		if u.Idx != 2 {
			return errors.New("count error")
		}
		return err
	})
	assert.NoError(t, err)
}

func TestAddUsers(t *testing.T) {
	//添加测试用户
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user, err := db.GetUserInfoWithMobile("17716858036")
		if err == nil {
			db.DeleteUser(user.ID)
		}
		u := NewUser("17716858036", "xh0714")
		err = db.InsertUser(u)
		if err != nil {
			return err
		}
		u1, err := db.GetUserInfo(u.ID)
		if err != nil {
			return err
		}
		if !ObjectIDEqual(u.ID, u1.ID) {
			return errors.New("find user error")
		}
		_, err = u.NewPrivate(db, DefaultExpTime, "第一个私钥")
		if err != nil {
			return err
		}
		_, err = u.NewPrivate(db, DefaultExpTime, "第二个私钥")
		if err != nil {
			return err
		}
		if u.Idx != 2 {
			return errors.New("count error")
		}
		err = db.DeleteUser(u.ID)
		if err != nil {
			return err
		}
		return err
	})
	assert.NoError(t, err)
}
