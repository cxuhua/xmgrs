package db

import (
	"context"
	"testing"

	"github.com/cxuhua/xginx"
)

func TestNewPrivate(t *testing.T) {
	pri, err := xginx.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	dp := &TPrivate{}
	dp.Id = pri.PublicKey().Hash().Bytes()
	dp.Cipher = CipherTypeNone
	dp.Body = pri.Encode()

	app := InitApp(context.Background())
	defer app.Close()
	err = app.UseDb(func(db IDbImp) error {
		return db.InsertPrivate(dp)
	})
	if err != nil {
		panic(err)
	}
}
