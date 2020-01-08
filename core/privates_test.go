package core

import (
	"context"
	"errors"
	"testing"

	"github.com/cxuhua/xginx"

	"github.com/stretchr/testify/assert"
)

func TestLoadDumpKey(t *testing.T) {
	as := assert.New(t)
	k1 := NewDeterKey()
	s := k1.Dump()
	k2, err := LoadDeterKey(s)
	as.NoError(err)
	as.Equal(k1.Root, k2.Root)
	as.Equal(k1.Key, k2.Key)
	msg := xginx.Hash256([]byte("dkfsdnf(9343"))
	pri := k2.GetPrivateKey()
	sig, err := pri.Sign(msg)
	as.NoError(err)
	pub := pri.PublicKey()
	vb := pub.Verify(msg, sig)
	as.True(vb, "sign verify error")
}

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
