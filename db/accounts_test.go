package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cxuhua/xginx"
)

type AccountTestSuite struct {
	suite.Suite
	app  *App
	db   IDbImp
	user *TUsers
	acc  *TAccount
}

func (suite *AccountTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := &TUsers{}
	user.Id = primitive.NewObjectID()
	user.Mobile = "17716858036"
	user.Pass = xginx.Hash256([]byte("xh0714"))
	err := suite.db.InsertUser(user)
	suite.Assert().NoError(err)
	suite.user = user
}

func (suite *AccountTestSuite) SetupTest() {
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

func (suite *AccountTestSuite) TestListCoins() {
	suite.Assert().NotNil(suite.acc, "default account miss")
	bi := xginx.NewTestBlockIndex(100, suite.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	scs, err := suite.acc.ListCoins(bi)
	suite.Assert().NoError(err)
	suite.Assert().Equal(scs.All.Balance(), 5000*xginx.COIN, "list account error")
	suite.Assert().Equal(scs.Coins.Balance(), 50*xginx.COIN, "list account error")
}

func (suite *AccountTestSuite) TearDownTest() {
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

func (suite *AccountTestSuite) TearDownSuite() {
	err := suite.db.DeleteUser(suite.user.Id)
	suite.Assert().NoError(err)
}

func TestAccounts(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		s := new(AccountTestSuite)
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
