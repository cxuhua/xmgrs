package api

import "github.com/cxuhua/xginx"

func (st *ApiTestSuite) TestGetUserInfo() {
	type result struct {
		Code   int          `json:"code"`
		Error  string       `json:"error"`
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"`
		Locks  xginx.Amount `json:"locks"`
	}
	res := &result{}
	err := st.Get("/v1/user/info", res)
	st.Require().Equal(res.Code, 0, res.Error)
	st.Require().NoError(err)
	st.Assert().Equal(res.Mobile, st.mobile, "mobile error")
	//101个区块，2个coinbase可用，99个锁定
	st.Assert().Equal(res.Coins, 100*xginx.COIN, "coins error")
	st.Assert().Equal(res.Locks, 4950*xginx.COIN, "locks error")
	//获取用户的金额列表
}

func (st *ApiTestSuite) TestGetUserCoins() {
	type item struct {
		Id      xginx.Address `json:"id"`      //所属账号地址
		Matured bool          `json:"matured"` //是否成熟
		Pool    bool          `json:"pool"`    //是否是内存池中的
		Value   xginx.Amount  `json:"value"`   //数量
		TxId    string        `json:"tx"`      //交易id
		Index   uint32        `json:"index"`   //输出索引
		Height  uint32        `json:"height"`  //所在区块高度
	}
	type result struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
		Items []item `json:"items"`
	}
	res := &result{}
	err := st.Get("/v1/user/coins", res)
	st.Require().Equal(res.Code, 0, res.Error)
	st.Require().NoError(err)
	//获取用户的金额列表
}
