package core

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/assert"
)

func TestImportAccount(t *testing.T) {
	kpass := "11223344"
	//添加测试用户
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user, err := db.GetUserInfoWithMobile("17716858036")
		if err == nil {
			db.DeleteUser(user.ID)
		}
		user, err = NewUser("17716858036", "xh0714", kpass)
		if err != nil {
			return err
		}
		err = db.InsertUser(user)
		if err != nil {
			return err
		}
		acc, err := xginx.NewAccount(2, 2, false)
		if err != nil {
			return err
		}
		tacc, err := user.ImportAccount(db, acc, DefaultExpTime, "desc", nil, "135789")
		if err != nil {
			return err
		}
		tacc, err = db.GetAccount(tacc.ID)
		if err != nil {
			return err
		}
		for _, kid := range tacc.Kid {
			tpri, err := db.GetPrivate(kid)
			if err != nil {
				return err
			}
			pri, err := tpri.ToPrivate("222")
			if err == nil {
				return fmt.Errorf("password wroing")
			}
			pri, err = tpri.ToPrivate("135789")
			if err != nil {
				return fmt.Errorf("password ok")
			}
			if pri == nil {
				return fmt.Errorf("private nil")
			}
		}
		return err
	})
	assert.NoError(t, err)
}

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
		u, err := NewUser("17716858036", "xh0714", kpass)
		if err != nil {
			return err
		}
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
		u, err := NewUser("17716858036", "xh0714")
		if err != nil {
			return err
		}
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
