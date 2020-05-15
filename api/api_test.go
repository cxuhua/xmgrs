package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-playground/form/v4"

	jsoniter "github.com/json-iterator/go"

	"github.com/cxuhua/xmgrs/core"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/suite"

	"github.com/gin-gonic/gin"
)

func init() {
	xginx.DebugScript = true
}

//APITestSuite api测试集合
type APITestSuite struct {
	suite.Suite
	ctx    context.Context
	db     core.IDbImp
	token  string
	m      *gin.Engine
	A      string
	au     *core.TUser
	aa     *core.TAccount
	bi     *xginx.BlockIndex
	B      string
	bu     *core.TUser
	ab     *core.TAccount
	mobile string //测试手机号
}

func (st *APITestSuite) SetupSuite() {
	st.ctx = context.Background()

	st.mobile = "18018989"
	st.A = "17716858036"
	st.B = "18602851011"

	xginx.NewTestConfig()

	gin.SetMode(gin.TestMode)
	st.m = InitEngine(st.ctx)

	app := core.InitApp(st.ctx)
	err := app.UseTx(func(sdb core.IDbImp) error {
		//先删除测试用户
		if u, err := sdb.GetUserInfoWithMobile(st.A); err == nil {
			sdb.DeleteUser(u.ID)
		}
		if u, err := sdb.GetUserInfoWithMobile(st.B); err == nil {
			sdb.DeleteUser(u.ID)
		}
		if u, err := sdb.GetUserInfoWithMobile(st.mobile); err == nil {
			sdb.DeleteUser(u.ID)
		}
		//创建测试用户A
		a := core.NewUser(st.A, "xh0714")
		err := sdb.InsertUser(a)
		if err != nil {
			return err
		}
		st.au = a
		//创建测试账号A
		aa, err := a.SaveAccount(sdb, 1, 1, false, "A账户描述", []string{})
		if err != nil {
			return err
		}
		st.aa = aa
		xginx.LogInfo("Test Account = ", aa.GetAddress())
		//生成101个测试区块
		st.bi = xginx.NewTestBlockIndex(101, aa.GetAddress())
		//创建测试用户B
		b := core.NewUser(st.B, "xh0714")
		err = sdb.InsertUser(b)
		if err != nil {
			return err
		}
		st.bu = b
		//创建测试账号B
		ab, err := b.SaveAccount(sdb, 1, 1, false, "B账户描述", []string{})
		if err != nil {
			return err
		}
		st.ab = ab
		return err
	})
	st.Require().NoError(err)
	//登陆a账户
	err = st.LoginA()
	st.Require().NoError(err)
}

func (st *APITestSuite) Post(uri string, v url.Values) (jsoniter.Any, error) {
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
	return jsoniter.Get(body), nil
}

func (st *APITestSuite) Get(uri string) (jsoniter.Any, error) {
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
	return jsoniter.Get(body), nil
}

//登陆A
func (st *APITestSuite) LoginB() error {
	if st.token != "" {
		st.Get("/v1/quit/login")
		st.token = ""
	}
	args := struct {
		Mobile string `form:"mobile" binding:"required"`
		Pass   string `form:"pass" binding:"required"`
	}{
		Mobile: st.B,
		Pass:   "xh0714",
	}
	enc := form.NewEncoder()
	v, err := enc.Encode(&args)
	if err != nil {
		return err
	}
	any, err := st.Post("/v1/login", v)
	if err != nil {
		return err
	}
	if any.Get("code").ToInt() != 0 {
		return fmt.Errorf("meta error = %v", any.Get("error"))
	}
	st.token = any.Get("token").ToString()
	xginx.LogInfo("login B account Success token=", st.token)
	return nil
}

//登陆A
func (st *APITestSuite) LoginA() error {
	if st.token != "" {
		st.Get("/v1/quit/login")
		st.token = ""
	}
	args := struct {
		Mobile string `form:"mobile" binding:"required"`
		Pass   string `form:"pass" binding:"required"`
	}{
		Mobile: st.A,
		Pass:   "xh0714",
	}
	enc := form.NewEncoder()
	v, err := enc.Encode(&args)
	if err != nil {
		return err
	}
	any, err := st.Post("/v1/login", v)
	if err != nil {
		return err
	}
	if any.Get("code").ToInt() != 0 {
		return fmt.Errorf("meta error = %v", any.Get("error"))
	}
	st.token = any.Get("token").ToString()
	xginx.LogInfo("login A account Success token=", st.token)
	return nil
}

func (st *APITestSuite) Do(w http.ResponseWriter, req *http.Request) {
	st.m.ServeHTTP(w, req)
}

func (st *APITestSuite) SetupTest() {

}

func (st *APITestSuite) TearDownTest() {

}

func (st *APITestSuite) TearDownSuite() {
	xginx.CloseTestBlock(st.bi)
	any, err := st.Get("/v1/quit/login")
	st.Require().NoError(err)
	st.Require().NotNil(any)
	st.Require().Equal(any.Get("code").ToInt(), 0, any.Get("error").ToString())
}

func TestApi(t *testing.T) {
	app := core.InitApp(context.Background())
	defer app.Close()
	app.UseDb(func(db core.IDbImp) error {
		st := new(APITestSuite)
		st.db = db
		suite.Run(t, st)
		return nil
	})
}
