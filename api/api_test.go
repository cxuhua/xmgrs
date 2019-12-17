package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/go-playground/validator.v9"

	"github.com/gin-gonic/gin"
)

const (
	TestUserMobile = "17716858036"
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
	m := InitHandler(context.Background(), IsTestLogin)
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
