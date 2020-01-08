package api

import (
	"net/url"

	"github.com/cxuhua/xginx"
)

func (st *ApiTestSuite) NewTx() {
	assert := st.Require()
	//创建一个交易
	v := url.Values{}
	//向B转账1个积分
	v.Add("dst", AddrValue{Addr: st.ab.GetAddress(), Value: 1 * xginx.COIN}.String())
	v.Set("fee", "10")
	v.Set("desc", "this is desc")
	v.Set("lt", "0")
	any, err := st.Post("/v1/new/tx", v)
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//获取交易id
	v = url.Values{}
	id := any.Get("item").Get("id").ToString()
	v.Set("id", id)
	//获取a待签名交易
	any, err = st.Get("/v1/list/sign/txs")
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	assert.Equal(any.Get("items").Size(), 1, "ts get error")
	//签名交易
	any, err = st.Post("/v1/sign/tx", v)
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//发布交易
	any, err = st.Post("/v1/submit/tx", v)
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	bi := xginx.GetBlockIndex()
	txp := bi.GetTxPool()
	assert.Equal(txp.Len(), 1, "push txpool error")
	//获取b的交易
	any, err = st.Get("/v1/list/txs/" + string(st.ab.GetAddress()))
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	assert.Equal(any.Get("items").Size(), 1, "ts get error")
	//获取a的交易
	any, err = st.Get("/v1/list/txs/" + string(st.aa.GetAddress()))
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//其中有101 coinbase tx
	assert.Equal(any.Get("items").Size(), 102, "ts get error")
	//登陆到B 获取b的金额
	err = st.LoginB()
	assert.NoError(err)
	any, err = st.Get("/v1/user/coins")
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//创建一个新区块打包交易
	err = xginx.NewTestOneBlock()
	assert.NoError(err)
	//再次获取B的金额
	any, err = st.Get("/v1/user/coins")
	assert.NoError(err)
	assert.NotNil(any)
	assert.Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
	//登陆回A
	err = st.LoginA()
	assert.NoError(err)
}
