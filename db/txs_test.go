package db

import (
	"context"
	"errors"
	"testing"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TxsTestSuite struct {
	suite.Suite
	app  *App
	db   IDbImp
	user *TUsers
	acc  *TAccount
}

func (st *TxsTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := &TUsers{}
	user.Id = primitive.NewObjectID()
	user.Mobile = "17716858036"
	user.Pass = xginx.Hash256([]byte("xh0714"))
	err := st.db.InsertUser(user)
	st.Assert().NoError(err)
	st.user = user
}

func (st *TxsTestSuite) SetupTest() {
	st.Assert().NotNil(st.user, "default user miss")
	p1 := st.user.NewPrivate()
	err := st.db.InsertPrivate(p1)
	st.Assert().NoError(err)
	//创建私钥2
	p2 := st.user.NewPrivate()
	err = st.db.InsertPrivate(p2)
	st.Assert().NoError(err)
	//创建 2-2证书
	acc, err := NewAccount(st.db, 2, 2, false, []string{p1.Id, p2.Id})
	st.Assert().NoError(err)
	err = st.db.InsertAccount(acc)
	st.Assert().NoError(err)
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
	//分析需要签名的输入存入数据库
	err = tx.Sign(bi, lis)
	st.Require().NoError(err)
	sigs := lis.GetSigs()
	if len(sigs) != 2 {
		st.Require().FailNow("sigs count error for 2-2")
	}
	//保存数据
	stx := NewTTx(st.user.Id, tx)
	err = st.db.InsertTx(stx)
	st.Require().NoError(err)
	err = lis.SaveSigs()
	st.Require().NoError(err)
	//执行签名
	for _, sig := range sigs {
		err = sig.Sign(st.db)
		st.Require().NoError(err)
	}
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
