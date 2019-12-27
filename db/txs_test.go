package db

import (
	"context"
	"errors"
	"testing"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TxsTestSuite struct {
	suite.Suite
	app  *App
	db   IDbImp
	user *TUser
	acc  *TAccount
}

func (st *TxsTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := NewUser("17716858036", []byte("xh0714"))
	err := st.db.InsertUser(user)
	st.Require().NoError(err)
	st.user = user
}

func (st *TxsTestSuite) SetupTest() {
	st.Require().NotNil(st.user, "default user miss")
	p1, err := st.user.NewPrivate(st.db)
	st.Require().NoError(err)
	//创建私钥2
	p2, err := st.user.NewPrivate(st.db)
	st.Require().NoError(err)
	//创建 2-2证书
	acc, err := NewAccount(st.db, st.user.Id, 2, 2, false, []string{p1.Id, p2.Id})
	st.Require().NoError(err)
	err = st.db.InsertAccount(acc)
	st.Require().NoError(err)
	st.acc = acc
}

func (st *TxsTestSuite) TestNewTx() {
	st.Require().NotNil(st.acc, "default account miss")
	bi := xginx.NewTestBlockIndex(100, st.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	ds, err := st.acc.ListCoins(bi)
	st.Require().NoError(err)
	st.Require().Equal(len(ds.Coins), 1, "coins miss")
	accs := xginx.GetTestAccount(bi)
	st.Require().NotNil(accs, "get test accounts error")
	dst, err := accs[1].GetAddress()
	st.Require().NoError(err)
	//创建签名处理lis
	lis := NewSignListener(st.db, st.user)
	//生成交易
	mi := bi.NewTrans(lis)
	mi.Add(dst, 1*xginx.COIN)
	mi.Fee = 1000
	tx, err := mi.NewTx()
	st.Require().NoError(err)
	sigs := lis.GetSigs()
	if len(sigs) != 2 {
		st.Require().FailNow("sigs count error for 2-2")
	}
	stx, err := st.user.SaveTx(st.db, tx, lis, "这个2-2签名交易")
	st.Require().NoError(err)
	//获取用户需要签名的交易
	txs, err := st.user.ListTxs(st.db, false)
	st.Require().NoError(err)
	st.Require().Equal(len(txs), 1, "txs error")
	//执行签名
	for _, sig := range sigs {
		err = sig.Sign(st.db)
		st.Require().NoError(err)
	}
	//获取用户不需要签名的交易
	txs, err = st.user.ListTxs(st.db, true)
	st.Require().NoError(err)
	st.Require().Equal(len(txs), 1, "txs error")
	//转换合并签名
	ntx, err := stx.ToTx(st.db, bi)
	st.Require().NoError(err)

	st.Require().Equal(ntx.MustID().Bytes(), stx.Id, "id error")
	//从数据库获取签名
	//
	err = st.db.DeleteTx(stx.Id)
	st.Require().NoError(err)
}

func (st *TxsTestSuite) TearDownTest() {
	//删除私钥
	for _, v := range st.acc.Pkh {
		id := GetPrivateId(v)
		err := st.db.DeletePrivate(id)
		st.Require().NoError(err)
	}
	//删除账户
	err := st.db.DeleteAccount(st.acc.Id)
	st.Require().NoError(err)
}

func (st *TxsTestSuite) TearDownSuite() {
	err := st.db.DeleteUser(st.user.Id)
	st.Require().NoError(err)
}

func TestTxsSuite(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		s := new(TxsTestSuite)
		s.app = app
		s.db = db
		suite.Run(t, s)
		if t.Failed() {
			return errors.New("TestAccounts test failed")
		} else {
			return nil
		}
	})
	assert.NoError(t, err)
}
