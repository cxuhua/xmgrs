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
	if user, err := st.db.GetUserInfoWithMobile("17716858036"); err == nil {
		st.db.DeleteUser(user.ID)
	}
	user := NewUser("17716858036", "xh0714")
	err := st.db.InsertUser(user)
	st.Assert().NoError(err)
	st.user = user
}

func (st *AccountTestSuite) SetupTest() {
	st.Assert().NotNil(st.user, "default user miss")
	//创建 2-2证书
	acc, err := st.user.SaveAccount(st.db, 2, 2, false, "2-2账户描述", []string{})
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
	st.Assert().Equal(scs.All.Balance(), 5000*xginx.Coin, "list account error")
	st.Assert().Equal(scs.Coins.Balance(), 50*xginx.Coin, "list account error")
}

func (st *AccountTestSuite) TearDownTest() {
	//删除私钥
	for _, id := range st.acc.Kid {
		err := st.db.DeletePrivate(id)
		st.Assert().NoError(err)
	}
	//删除账户
	err := st.db.DeleteAccount(st.acc.ID, st.user.ID)
	st.Assert().NoError(err)
}

func (st *AccountTestSuite) TearDownSuite() {
	err := st.db.DeleteUser(st.user.ID)
	st.Assert().NoError(err)
}

func TestAccounts(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		st := new(AccountTestSuite)
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
