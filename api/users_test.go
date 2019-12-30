package api

import "github.com/cxuhua/xginx"

func (st *ApiTestSuite) TestGetUserInfo() {
	type result struct {
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"`
		Locks  xginx.Amount `json:"locks"`
	}
	res := &result{}
	err := st.Get("/v1/user/info", res)
	st.Require().NoError(err)
	st.Assert().Equal(res.Mobile, st.mobile, "mobile error")
	//101个区块，2个coinbase可用，99个锁定
	st.Assert().Equal(res.Coins, 100*xginx.COIN, "coins error")
	st.Assert().Equal(res.Locks, 4950*xginx.COIN, "locks error")
}
