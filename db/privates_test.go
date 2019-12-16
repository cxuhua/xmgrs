package db

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xginx"
)

func TestNewPrivate(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user := &TUsers{}
		user.Id = primitive.NewObjectID()
		user.Mobile = "17716858036"
		user.Pass = xginx.Hash256([]byte("xh0714"))
		err := db.InsertUser(user)
		if err != nil {
			return err
		}
		dp := user.NewPrivate()
		err = db.InsertPrivate(dp)
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
