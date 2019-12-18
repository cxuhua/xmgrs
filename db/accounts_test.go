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

func (st *AccountTestSuite) SetupSuite() {
	xginx.NewTestConfig()
	user := &TUsers{}
	user.Id = primitive.NewObjectID()
	user.Mobile = "17716858036"
	user.Pass = xginx.Hash256([]byte("xh0714"))
	err := st.db.InsertUser(user)
	st.Assert().NoError(err)
	st.user = user
}

func (st *AccountTestSuite) SetupTest() {
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
	for _, v := range st.acc.Pkh {
		id := GetPrivateId(v)
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
