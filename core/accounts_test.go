package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/cxuhua/xginx"
)

type AccountTestSuite struct {
	suite.Suite
	app  *App
	db   IDbImp
	user *TUser
	acc  *TAccount
}

func (st *AccountTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := NewUser("17716858036", []byte("xh0714"))
	err := st.db.InsertUser(user)
	st.Assert().NoError(err)
	st.user = user
}

func (st *AccountTestSuite) SetupTest() {
	st.Assert().NotNil(st.user, "default user miss")
	//创建 2-2证书
	acc, err := st.user.SaveAccount(st.db, 2, 2, false)
	st.Assert().NoError(err)
	st.acc = acc
}

func (st *AccountTestSuite) TestListCoins() {
	st.Assert().NotNil(st.acc, "default account miss")
	bi := xginx.NewTestBlockIndex(100, st.acc.GetAddress())
	defer xginx.CloseTestBlock(bi)
	//获取账户金额
	scs, err := st.acc.ListCoins(bi)
	st.Assert().NoError(err)
	st.Assert().Equal(scs.All.Balance(), 5000*xginx.COIN, "list account error")
	st.Assert().Equal(scs.Coins.Balance(), 50*xginx.COIN, "list account error")
}

func (st *AccountTestSuite) TearDownTest() {
	//删除私钥
	for _, id := range st.acc.Kid {
		err := st.db.DeletePrivate(id)
		st.Assert().NoError(err)
	}
	//删除账户
	err := st.db.DeleteAccount(st.acc.Id)
	st.Assert().NoError(err)
}

func (st *AccountTestSuite) TearDownSuite() {
	err := st.db.DeleteUser(st.user.Id)
	st.Assert().NoError(err)
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
