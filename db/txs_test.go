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

func (suite *TxsTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := &TUsers{}
	user.Id = primitive.NewObjectID()
	user.Mobile = "17716858036"
	user.Pass = xginx.Hash256([]byte("xh0714"))
	err := suite.db.InsertUser(user)
	suite.Assert().NoError(err)
	suite.user = user
}

func (suite *TxsTestSuite) SetupTest() {
	suite.Assert().NotNil(suite.user, "default user miss")
	p1 := suite.user.NewPrivate()
	err := suite.db.InsertPrivate(p1)
	suite.Assert().NoError(err)
	//创建私钥2
	p2 := suite.user.NewPrivate()
	err = suite.db.InsertPrivate(p2)
	suite.Assert().NoError(err)
	//创建 2-2证书
	acc, err := NewAccount(suite.db, 2, 2, false, []string{p1.Id, p2.Id})
	suite.Assert().NoError(err)
	err = suite.db.InsertAccount(acc)
	suite.Assert().NoError(err)
	suite.acc = acc
}

//获取金额对应的账户方法
func (suite *TxsTestSuite) GetAcc(ckv *xginx.CoinKeyValue) *xginx.Account {
	return suite.acc.ToAccount()
}

//获取输出地址的扩展
func (suite *TxsTestSuite) GetExt(addr xginx.Address) []byte {
	return nil
}

//获取使用的金额
func (suite *TxsTestSuite) GetCoins() xginx.Coins {
	bi := xginx.GetBlockIndex()
	ds, err := suite.acc.ListCoins(bi)
	if err != nil {
		return nil
	}
	return ds.Coins
}

//获取找零地址
func (suite *TxsTestSuite) GetKeep() xginx.Address {
	return suite.acc.GetAddress()
}

//签名交易
func (suite *TxsTestSuite) SignTx(singer xginx.ISigner) error {
	_, in, out := singer.GetObjs()
	addr, err := out.Script.GetAddress()
	if err != nil {
		return err
	}
	acc, err := suite.db.GetAccount(addr)
	if err != nil {
		return err
	}
	wits, err := acc.Sign(suite.db, singer)
	if err != nil {
		return err
	}
	script, err := wits.ToScript()
	if err != nil {
		return err
	}
	in.Script = script
	return nil
}

func (suite *TxsTestSuite) TestNewTx() {
	suite.Assert().NotNil(suite.acc, "default account miss")
	bi := xginx.NewTestBlockIndex(100, suite.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	ds, err := suite.acc.ListCoins(bi)
	suite.Assert().NoError(err)
	suite.Assert().Equal(len(ds.Coins), 1, "coins miss")
	accs := xginx.GetTestAccount(bi)
	suite.Assert().NotNil(accs, "get test accounts error")
	dst, err := accs[1].GetAddress()
	suite.Assert().NoError(err)
	//生成交易
	mi := bi.NewTrans(suite)
	mi.Dst = []xginx.Address{dst}
	mi.Amts = []xginx.Amount{1 * xginx.COIN}
	mi.Fee = 1000
	tx, err := mi.NewTx()
	suite.Assert().NoError(err)
	err = tx.Sign(bi, suite)
	suite.Assert().NoError(err)
	err = tx.Check(bi, true)
	suite.Assert().NoError(err)
}

func (suite *TxsTestSuite) TearDownTest() {
	//删除私钥
	for _, v := range suite.acc.Pkh {
		id := GetPrivateId(v)
		err := suite.db.DeletePrivate(id)
		suite.Assert().NoError(err)
	}
	//删除账户
	err := suite.db.DeleteAccount(suite.acc.Id)
	suite.Assert().NoError(err)
}

func (suite *TxsTestSuite) TearDownSuite() {
	err := suite.db.DeleteUser(suite.user.Id)
	suite.Assert().NoError(err)
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
