package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cxuhua/xmgrs/db"

	"github.com/cxuhua/xginx"
	"github.com/stretchr/testify/suite"

	"gopkg.in/go-playground/validator.v9"

	"github.com/gin-gonic/gin"
)

func TestValid(t *testing.T) {
	fn := func(ctx context.Context, fl validator.FieldLevel) bool {
		log.Println(ctx.Value("aaa"))
		fl.Field().SetInt(100)
		return true
	}
	validate := validator.New()
	err := validate.RegisterValidationCtx("new", fn)
	if err != nil {
		panic(err)
	}
	type x struct {
		A int `validate:"new"`
	}
	ctx := context.WithValue(context.Background(), "aaa", 1000)
	a := &x{}
	err = validate.StructCtx(ctx, a)
	if err != nil {
		panic(err)
	}
}

func RequestJson(req *http.Request) ([]byte, error) {
	gin.SetMode(gin.TestMode)
	if req.Method != http.MethodGet {
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	}
	m := InitHandler(context.Background())
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status code = %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type ApiTestSuite struct {
	suite.Suite
	ctx    context.Context
	m      *gin.Engine
	mobile string
	token  string
}

func (st *ApiTestSuite) SetupSuite() {
	st.ctx = context.Background()
	st.mobile = "17716858036"

	xginx.NewTestConfig()

	st.m = InitHandler(st.ctx)

	app := db.InitApp(st.ctx)
	err := app.UseTx(func(sdb db.IDbImp) error {
		//创建测试用户
		user := db.NewUser(st.mobile, []byte("xh0714"))
		err := sdb.InsertUser(user)
		if err != nil {
			return err
		}
		//创建测试账号
		_, err = user.GenAccount(sdb, 1, 1, false)
		if err != nil {
			return err
		}
		return err
	})
	st.Require().NoError(err)
}

func (st *ApiTestSuite) Post(uri string, v url.Values, jv interface{}) error {
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
		return errors.New("status error")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if jv == nil {
		return nil
	}
	log.Println("POST RECV:", string(body))
	return json.Unmarshal(body, jv)
}

func (st *ApiTestSuite) Get(uri string, jv interface{}) error {
	req := httptest.NewRequest(http.MethodGet, uri, nil)
	if st.token != "" {
		req.Header.Set("X-Access-Token", st.token)
	}
	wr := httptest.NewRecorder()
	st.Do(wr, req)
	res := wr.Result()
	if res.StatusCode != http.StatusOK {
		return errors.New("status error")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	log.Println("GET RECV:", string(body))
	if jv == nil {
		return nil
	}
	return json.Unmarshal(body, jv)
}

//登陆
func (st *ApiTestSuite) Login() error {
	v := url.Values{}
	v.Set("mobile", st.mobile)
	v.Set("pass", "xh0714")
	r := struct {
		Meta  int    `json:"meta"`
		Token string `json:"token"`
	}{}
	err := st.Post("/v1/login", v, &r)
	if err != nil {
		return err
	}
	if r.Meta != 0 {
		return fmt.Errorf("meta error = %d", r.Meta)
	}
	st.token = r.Token
	return nil
}

func (st *ApiTestSuite) Do(w http.ResponseWriter, req *http.Request) {
	st.m.ServeHTTP(w, req)
}

func (st *ApiTestSuite) SetupTest() {
	err := st.Login()
	st.Require().NoError(err)
}

func (st *ApiTestSuite) TestGetUserInfo() {
	type result struct {
		Mobile string `json:"mobile"`
	}
	res := &result{}
	err := st.Get("/v1/user/info", res)
	st.Require().NoError(err)
	st.Require().Equal(res.Mobile, st.mobile, "mobile error")
}

func (st *ApiTestSuite) TearDownTest() {

}

func (st *ApiTestSuite) TearDownSuite() {
	app := db.InitApp(st.ctx)
	err := app.UseTx(func(sdb db.IDbImp) error {
		user, err := sdb.GetUserInfoWithMobile(st.mobile)
		if err != nil {
			return err
		}
		err = sdb.DeleteUser(user.Id)
		return err
	})
	st.Require().NoError(err)
}

func TestApi(t *testing.T) {
	s := new(ApiTestSuite)
	suite.Run(t, s)
}
