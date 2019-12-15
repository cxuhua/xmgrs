package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

const (
	TestUserMobile = "17716858036"
)

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
