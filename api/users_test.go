package api

import "github.com/cxuhua/xginx"

func (st *ApiTestSuite) TestGetUserInfo() {
	any, err := st.Get("/v1/user/info")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	msg := any.Get("error").ToString()
	st.Require().Equal(code, 0, msg)
	st.Assert().Equal(any.Get("mobile").ToString(), st.mobile, "mobile error")
	//101个区块，2个coinbase可用，99个锁定
	st.Assert().Equal(xginx.Amount(any.Get("coins").ToInt64()), 100*xginx.COIN, "coins error")
	st.Assert().Equal(xginx.Amount(any.Get("locks").ToInt64()), 4950*xginx.COIN, "locks error")
	//获取用户的金额列表
}

func (st *ApiTestSuite) TestGetUserCoins() {
	any, err := st.Get("/v1/user/coins")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	code := any.Get("code").ToInt()
	st.Require().Equal(code, 0, any.Get("error"))
	//获取用户的金额列表
}
