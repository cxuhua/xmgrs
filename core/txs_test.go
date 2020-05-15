package core

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
	//删除测试账号
	if uv, err := st.db.GetUserInfoWithMobile("17716858036"); err == nil {
		st.db.DeleteUser(uv.ID)
	}
	//创建测试账号
	user := NewUser("17716858036", "xh0714")
	err := st.db.InsertUser(user)
	st.Require().NoError(err)
	st.user = user
}

func (st *TxsTestSuite) SetupTest() {
	st.Require().NotNil(st.user, "default user miss")
	p1, err := st.user.NewPrivate(st.db, "第一个私钥")
	st.Require().NoError(err)
	//创建私钥2
	p2, err := st.user.NewPrivate(st.db, "第二个私钥")
	st.Require().NoError(err)
	//创建 2-2证书
	acc, err := NewAccount(st.db, 2, 2, false, []string{p1.ID, p2.ID}, "账户描述", []string{})
	st.Require().NoError(err)
	err = st.db.InsertAccount(acc)
	st.Require().NoError(err)
	st.acc = acc
	num, err := st.db.GetPrivateRefs(p1.ID)
	st.Require().NoError(err)
	st.Require().Equal(num, 1, "ref num error")
	num, err = st.db.GetPrivateRefs(p2.ID)
	st.Require().NoError(err)
	st.Require().Equal(num, 1, "ref num error")
}

//创建新交易测试
func (st *TxsTestSuite) TestNewTx() {
	st.Require().NotNil(st.acc, "default account miss")
	//作为矿工账户
	bi := xginx.NewTestBlockIndex(100, st.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	ds, err := st.acc.ListCoins(bi)
	st.Require().NoError(err)
	//应该有一个金额可用
	st.Require().Equal(len(ds.Coins), 1, "coins miss")
	//获取测试账户
	accs := xginx.GetTestAccount(bi)
	st.Require().NotNil(accs, "get test accounts error")
	//1作为目标转账用户
	dst, err := accs[1].GetAddress()
	st.Require().NoError(err)
	//创建签名处理lis
	lis := NewSignListener(st.db, st.user)
	//生成交易
	mi := bi.NewTrans(lis)
	//向dst转账1COIN
	mi.Add(dst, 1*xginx.Coin)
	//1000作为交易费
	mi.Fee = 1000
	tx, err := mi.NewTx(0)
	st.Require().NoError(err)
	//2-2账户会生成两个签名
	sigs := lis.GetSigs()
	if len(sigs) != 2 {
		st.Require().FailNow("sigs count error for 2-2")
	}
	//保存交易
	stx, err := st.user.SaveTx(st.db, tx, lis, "这个2-2签名交易")
	st.Require().NoError(err)
	//获取用户需要签名的交易 false表示获取未签名的交易
	txs, err := st.user.ListTxs(st.db, false)
	st.Require().NoError(err)
	st.Require().Equal(len(txs), 1, "txs error")
	//执行签名 未设置密钥密码
	for _, sig := range sigs {
		err = sig.Sign(st.db)
		st.Require().NoError(err)
	}
	//获取用户不需要签名的交易 true表示获取已签名的交易
	txs, err = st.user.ListTxs(st.db, true)
	st.Require().NoError(err)
	st.Require().Equal(len(txs), 1, "txs error")
	//转换合并签名
	ntx, err := stx.ToTx(st.db, bi)
	st.Require().NoError(err)
	//转换后ID应该一致
	st.Require().Equal(ntx.MustID().Bytes(), stx.ID, "id error")
	//删除交易
	err = st.db.DeleteTx(stx.ID)
	st.Require().NoError(err)
}

func (st *TxsTestSuite) TearDownTest() {
	//删除账户
	err := st.db.DeleteAccount(st.acc.ID, st.user.ID)
	st.Require().NoError(err)
	//删除私钥
	for _, id := range st.acc.Kid {
		err := st.db.DeletePrivate(id)
		st.Require().NoError(err)
	}
}

func (st *TxsTestSuite) TearDownSuite() {
	err := st.db.DeleteUser(st.user.ID)
	st.Require().NoError(err)
}

func TestTxsSuite(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		st := new(TxsTestSuite)
		st.app = app
		st.db = db
		suite.Run(t, st)
		if t.Failed() {
			return errors.New("TestAccounts test failed")
		}
		return nil
	})
	assert.NoError(t, err)
}
