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
	assert.NoError(suite.T(), err)
	suite.user = user
}

func (suite *AccountTestSuite) SetupTest() {
	assert.NotNil(suite.T(), suite.user, "default user miss")
	p1 := suite.user.NewPrivate()
	err := suite.db.InsertPrivate(p1)
	assert.NoError(suite.T(), err)
	//创建私钥2
	p2 := suite.user.NewPrivate()
	err = suite.db.InsertPrivate(p2)
	assert.NoError(suite.T(), err)
	//创建 2-2证书
	acc, err := NewAccount(suite.db, 2, 2, false, []string{p1.Id, p2.Id})
	assert.NoError(suite.T(), err)
	err = suite.db.InsertAccount(acc)
	assert.NoError(suite.T(), err)
	suite.acc = acc
}

func (suite *AccountTestSuite) TestListCoins() {
	assert.NotNil(suite.T(), suite.acc, "default account miss")
	bi := xginx.NewTestBlockIndex(100, suite.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	scs, err := suite.acc.ListCoins(bi)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), scs.All.Balance(), 5000*xginx.COIN, "list account error")
	assert.Equal(suite.T(), scs.Coins.Balance(), 50*xginx.COIN, "list account error")
}

func (suite *AccountTestSuite) TearDownTest() {
	//删除私钥
	for _, v := range suite.acc.Pkh {
		id := GetPrivateId(v)
		err := suite.db.DeletePrivate(id)
		assert.NoError(suite.T(), err)
	}
	//删除账户
	err := suite.db.DeleteAccount(suite.acc.Id)
	assert.NoError(suite.T(), err)
}

func (suite *AccountTestSuite) TearDownSuite() {
	err := suite.db.DeleteUser(suite.user.Id)
	assert.NoError(suite.T(), err)
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
