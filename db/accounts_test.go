package db

import (
	"context"
	"errors"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xginx"
)

type testlis struct {
	xginx.Listener
}

func (lis *testlis) OnStart() {

}

func (lis *testlis) OnStop(sig os.Signal) {

}

func init() {
	//创建测试配置
	xginx.NewTestConfig()
}

func TestListCoins(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		//创建账号
		user := &TUsers{}
		user.Id = primitive.NewObjectID()
		user.Mobile = "17716858036"
		user.Pass = xginx.Hash256([]byte("xh0714"))
		err := db.InsertUser(user)
		if err != nil {
			return err
		}
		//创建私钥1
		p1 := user.NewPrivate()
		err = db.InsertPrivate(p1)
		if err != nil {
			return err
		}
		//创建私钥2
		p2 := user.NewPrivate()
		err = db.InsertPrivate(p2)
		if err != nil {
			return err
		}
		//创建 2-2证书
		acc, err := NewAccount(db, 2, 2, false, []string{p1.Id, p2.Id})
		if err != nil {
			return err
		}
		err = db.InsertAccount(acc)
		if err != nil {
			return err
		}
		accs, err := db.ListAccounts(user.Id)
		if err != nil {
			return err
		}
		if len(accs) != 1 {
			return errors.New("list account error")
		}
		//获取用户相关的账号
		//创建区块
		bi := xginx.NewTestBlockIndex(100, acc.GetAddress())
		defer xginx.CloseTestBlock(bi)

		scs, err := user.ListCoins(db, bi)
		if err != nil {
			return err
		}
		if scs.All.Balance() != 5000*xginx.COIN {
			return errors.New("all balance error")
		}
		if scs.Coins.Balance() != 50*xginx.COIN {
			return errors.New("coins balance error")
		}
		//删除私钥
		err = db.DeletePrivate(p1.Id)
		if err != nil {
			return err
		}
		err = db.DeletePrivate(p2.Id)
		if err != nil {
			return err
		}
		//删除账户
		err = db.DeleteAccount(acc.Id)
		if err != nil {
			return err
		}
		//删除用户
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
