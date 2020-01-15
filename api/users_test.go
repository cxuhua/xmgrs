package api

import (
	"net/url"

	"github.com/cxuhua/xginx"
)

func (st *ApiTestSuite) TestAll() {
	st.GetUserInfo()

	st.ListUserAccounts()

	st.GetUserCoins()

	st.NewTx()
}

func (st *ApiTestSuite) GetUserInfo() {
	any, err := st.Get("/v1/user/info")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	msg := any.Get("error").ToString()
	st.Require().Equal(code, 0, msg)
	st.Assert().Equal(any.Get("mobile").ToString(), st.A, "mobile error")
	//101个区块，2个coinbase可用，99个锁定
	st.Assert().Equal(xginx.Amount(any.Get("coins").ToInt64()), 100*xginx.Coin, "coins error")
	st.Assert().Equal(xginx.Amount(any.Get("locks").ToInt64()), 4950*xginx.Coin, "locks error")
	//创建一个私钥
	v := url.Values{}
	v.Set("desc", "私钥信息")
	any, err = st.Post("/v1/new/private", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//使用指定私钥创建账号
	v = url.Values{}
	v.Set("num", "1")
	v.Set("less", "1")
	v.Set("arb", "false")
	v.Add("id", any.Get("item").Get("id").ToString())
	v.Set("desc", "账号描述")
	any, err = st.Post("/v1/new/account", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//获取私钥列表
	any, err = st.Get("/v1/list/privates")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code = any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
}

func (st *ApiTestSuite) ListUserAccounts() {
	any, err := st.Get("/v1/list/accounts")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
}

//检测A的金额
func (st *ApiTestSuite) GetUserCoins() {
	any, err := st.Get("/v1/user/coins")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
	//获取用户的金额列表
	st.Assert().Equal(101, any.Get("items").Size(), "txs count error")

	//获取第一个交易id和地址
	txid := any.Get("items", 0, "tx").ToString()
	addr := any.Get("items", 0, "id").ToString()

	any, err = st.Get("/v1/tx/info/" + txid)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code = any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))

	any, err = st.Get("/v1/list/txs/" + addr)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code = any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
}
