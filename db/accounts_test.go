package db

import (
	"context"
	"log"
	"testing"

	"github.com/cxuhua/xginx"
)

func init() {
	xginx.InitConfig("../v10000.json")
}

func TestDeleteAccount(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		return db.DeleteAccount("st1qmwdvr706cqux5rrnltvqgh0xhmjscmzn2afune")
	})
	if err != nil {
		panic(err)
	}
}

func TestGetAccount(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseDb(func(db IDbImp) error {
		acc, err := db.GetAccount("st1qmwdvr706cqux5rrnltvqgh0xhmjscmzn2afune")
		if err != nil {
			return err
		}
		act := acc.ToAccount(db)
		log.Println(act)
		return act.Check()
	})
	if err != nil {
		panic(err)
	}
}

func TestNewAccount(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user, err := db.GetUserInfoWithMobile("17716858036")
		if err != nil {
			return err
		}
		obj, err := user.NewAccount(3, 2, true)
		if err != nil {
			return err
		}
		err = db.InsertAccount(obj)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
