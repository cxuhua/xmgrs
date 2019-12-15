package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cxuhua/xmgrs/db"
	"github.com/gin-gonic/gin"
)

func IsTestLogin(c *gin.Context) {
	app := db.GetApp(c)
	err := app.UseDb(func(db db.IDbImp) error {
		info, err := db.GetUserInfoWithMobile(TestUserMobile)
		if err != nil {
			return err
		}
		c.Set(AppUserKey, info)
		c.Next()
		return nil
	})
	if err != nil {
		_ = c.AbortWithError(http.StatusUnauthorized, err)
	}
}

func TestGetUserInfo(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/v1/user/info", nil)
	if err != nil {
		panic(err)
	}
	data, err := RequestJson(req)
	if err != nil {
		t.Fatal(err)
	}
	user := &db.TUsers{}
	if err := json.Unmarshal(data, user); err != nil {
		t.Fatal(err)
	}
	if user.Mobile != TestUserMobile {
		t.Fatal("get user test error")
	}
}
