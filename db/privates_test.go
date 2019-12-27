package db

import (
	"context"
	"errors"
	"testing"
)

func TestNewPrivate(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user := NewUser("17716858036", []byte("xh0714"))
		err := db.InsertUser(user)
		if err != nil {
			return err
		}
		dp, err := user.NewPrivate(db, "测试私钥1")
		if err != nil {
			return err
		}
		pri, err := db.GetPrivate(dp.Id)
		if err != nil {
			return err
		}
		if !pri.Pkh.Equal(dp.Pkh) {
			return errors.New("pkh error")
		}
		err = db.DeleteUser(user.Id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
