package api

import (
	"net/url"
	"testing"

	"github.com/cxuhua/xginx"
)

func TestParseValueScript(t *testing.T) {
	s := "aa->100,scr->,->,,"
	a1, v1 := parseValueAddress(s)
	if a1 != "aa" || v1 != "100,scr->,->,," {
		t.Fatal("t4 error")
	}

	s = "111"
	s1, s2 := parseValueScript(s)
	if s1 == "" && s2 == "" {
		t.Fatal("empty error")
	}
	if s1 != s || s2 != "" {
		t.Fatal("t1 error")
	}
	s = "222,"
	s1, s2 = parseValueScript(s)
	if s1 == "" && s2 == "" {
		t.Fatal("empty error")
	}
	if s1 != "222" || s2 != "" {
		t.Fatal("t2 error")
	}
	s = "333,444"
	s1, s2 = parseValueScript(s)
	if s1 == "" && s2 == "" {
		t.Fatal("empty error")
	}
	if s1 != "333" || s2 != "444" {
		t.Fatal("t3 error")
	}

}

func (st *APITestSuite) NewTx() {
	assert := st.Require()
	//创建一个交易
	v := url.Values{}
	//向B转账1000个积分,交易费 10
	dst := AddrValue{
		Addr:      st.ab.GetAddress(),
		Value:     1 * xginx.Coin,
		OutScript: string(xginx.DefaultLockedScript),
	}
	v.Add("dst", dst.Format())
	v.Set("fee", "0.01")
	v.Set("desc", "this is desc")
	v.Set("script", "return true")
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
