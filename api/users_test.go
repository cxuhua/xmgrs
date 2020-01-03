package api

import (
	"net/url"

	"github.com/cxuhua/xginx"
)

func (st *ApiTestSuite) TestAll() {
	st.GetUserInfo()

	st.ListUserAccounts()

	st.GetUserCoins()
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
	st.Assert().Equal(xginx.Amount(any.Get("coins").ToInt64()), 100*xginx.COIN, "coins error")
	st.Assert().Equal(xginx.Amount(any.Get("locks").ToInt64()), 4950*xginx.COIN, "locks error")
	//创建一个私钥
	v := url.Values{}
	v.Set("desc", "私钥信息")
	any, err = st.Post("/v1/new/private", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
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

func (st *ApiTestSuite) GetUserCoins() {
	any, err := st.Get("/v1/user/coins")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
	//获取用户的金额列表
	st.Assert().Equal(101, any.Get("items").Size(), "txs count error")

	txid := any.Get("items").Get(0).Get("tx").ToString()
	addr := any.Get("items").Get(0).Get("id").ToString()

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
