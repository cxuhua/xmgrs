package api

import (
	"net/url"

	"github.com/cxuhua/xginx"
)

func (st *ApiTestSuite) NewTx() {
	//创建一个交易
	av := xginx.AddrValue{
		Addr:  st.ab.GetAddress(),
		Value: 1 * xginx.COIN,
	}
	v := url.Values{}
	v.Add("dst", av.String())
	v.Set("fee", "10")
	v.Set("desc", "this is desc")
	v.Set("lt", "0")
	any, err := st.Post("/v1/new/tx", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	v = url.Values{}
	id := any.Get("item").Get("id").ToString()
	v.Set("id", id)
	//获取a待签名交易
	any, err = st.Get("/v1/list/sign/txs")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	st.Require().Equal(any.Get("items").Size(), 1, "ts get error")
	//签名交易
	any, err = st.Post("/v1/sign/tx", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//发布易交
	any, err = st.Post("/v1/submit/tx", v)
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	bi := xginx.GetBlockIndex()
	txp := bi.GetTxPool()
	st.Require().Equal(txp.Len(), 1, "push txpool error")
	//获取b的交易
	any, err = st.Get("/v1/list/txs/" + string(st.ab.GetAddress()))
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	st.Require().Equal(any.Get("items").Size(), 1, "ts get error")
	//获取a的交易
	any, err = st.Get("/v1/list/txs/" + string(st.aa.GetAddress()))
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//其中有101 coinbase tx
	st.Require().Equal(any.Get("items").Size(), 102, "ts get error")
	//登陆到B 获取b的金额
	err = st.LoginB()
	st.Require().NoError(err)
	any, err = st.Get("/v1/user/coins")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//创建一个新区块打包交易
	err = xginx.NewTestOneBlock()
	st.Require().NoError(err)
	//再次获取B的金额
	any, err = st.Get("/v1/user/coins")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//登陆回A
	err = st.LoginA()
	st.Require().NoError(err)
}
