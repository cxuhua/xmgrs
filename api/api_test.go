package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/cxuhua/xmgrs/core"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/suite"

	"github.com/gin-gonic/gin"
)

type ApiTestSuite struct {
	suite.Suite
	ctx   context.Context
	token string
	m     *gin.Engine
	A     string
	au    *core.TUser
	aa    *core.TAccount
	bi    *xginx.BlockIndex
	B     string
	bu    *core.TUser
	ab    *core.TAccount
}

func (st *ApiTestSuite) SetupSuite() {
	st.ctx = context.Background()

	st.A = "17716858036"
	st.B = "18602851011"

	xginx.NewTestConfig()

	gin.SetMode(gin.TestMode)
	st.m = InitEngine(st.ctx)

	app := core.InitApp(st.ctx)
	err := app.UseTx(func(sdb core.IDbImp) error {
		//创建测试用户A
		a := core.NewUser(st.A, []byte("xh0714"))
		err := sdb.InsertUser(a)
		if err != nil {
			return err
		}
		st.au = a
		//创建测试账号
		aa, err := a.SaveAccount(sdb, 1, 1, false)
		if err != nil {
			return err
		}
		st.aa = aa
		xginx.LogInfo("Test Account = ", aa.GetAddress())
		//生成101个区块
		st.bi = xginx.NewTestBlockIndex(101, aa.GetAddress())
		//创建测试用户B
		b := core.NewUser(st.B, []byte("xh0714"))
		err = sdb.InsertUser(b)
		if err != nil {
			return err
		}
		st.bu = b
		//创建测试账号
		ab, err := b.SaveAccount(sdb, 1, 1, false)
		if err != nil {
			return err
		}
		st.ab = ab
		return err
	})
	st.Require().NoError(err)
	//
	err = st.Login()
	st.Require().NoError(err)
}

func (st *ApiTestSuite) Post(uri string, v url.Values) (jsoniter.Any, error) {
	log.Println("POST:", v.Encode())
	req := httptest.NewRequest(http.MethodPost, uri, strings.NewReader(v.Encode()))
	if st.token != "" {
		req.Header.Set("X-Access-Token", st.token)
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	wr := httptest.NewRecorder()
	st.Do(wr, req)
	res := wr.Result()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("status error")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Println("POST RECV:", string(body))
	return jsoniter.Get(body), nil
}

func (st *ApiTestSuite) Get(uri string) (jsoniter.Any, error) {
	req := httptest.NewRequest(http.MethodGet, uri, nil)
	if st.token != "" {
		req.Header.Set(core.TokenHeader, st.token)
	}
	wr := httptest.NewRecorder()
	st.Do(wr, req)
	res := wr.Result()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("status error")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Println("GET RECV:", string(body))
	return jsoniter.Get(body), nil
}

//登陆
func (st *ApiTestSuite) Login() error {
	v := url.Values{}
	v.Set("mobile", st.A)
	v.Set("pass", "xh0714")

	any, err := st.Post("/v1/login", v)
	if err != nil {
		return err
	}
	if any.Get("code").ToInt() != 0 {
		return fmt.Errorf("meta error = %v", any.Get("error"))
	}
	st.token = any.Get("token").ToString()
	return nil
}

func (st *ApiTestSuite) Do(w http.ResponseWriter, req *http.Request) {
	st.m.ServeHTTP(w, req)
}

func (st *ApiTestSuite) SetupTest() {

}

func (st *ApiTestSuite) TearDownTest() {

}

func (st *ApiTestSuite) TearDownSuite() {
	xginx.CloseTestBlock(st.bi)
	app := core.InitApp(st.ctx)
	err := app.UseTx(func(sdb core.IDbImp) error {
		err := sdb.DeleteUser(st.au.Id)
		if err != nil {
			return err
		}
		err = sdb.DeleteUser(st.bu.Id)
		if err != nil {
			return err
		}
		return err
	})
	st.Require().NoError(err)
}

func TestApi(t *testing.T) {
	s := new(ApiTestSuite)
	suite.Run(t, s)
}
