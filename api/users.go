package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cxuhua/xmgrs/db"

	"github.com/gin-gonic/gin"
)

func loginApi(c *gin.Context) {
	args := struct {
		Mobile string `form:"mobile"`
		Pass   string `form:"pass"`
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.Error(NewError(100, err))
		return
	}
	if args.Mobile == "" || args.Pass == "" {
		c.Error(NewError(101, "mobile or pass args error"))
		return
	}
	type result struct {
		Meta  int    `json:"meta"`
		Token string `json:"token"`
	}
	rv := result{
		Meta: 0,
	}
	app := db.GetApp(c)
	err := app.UseDb(func(db db.IDbImp) error {
		user, err := db.GetUserInfoWithMobile(args.Mobile)
		if err != nil {
			return fmt.Errorf("get user info error %w", err)
		}
		if !user.CheckPass(args.Pass) {
			return errors.New("password error")
		}
		tk := app.GenToken()
		err = db.SetUserToken(user.Id, tk)
		if err != nil {
			return err
		}
		err = app.SetUserId(tk, user.Id.Hex(), time.Hour*24*7)
		if err != nil {
			return err
		}
		rv.Token = app.EncryptToken(tk)
		return nil
	})
	if err != nil {
		c.Error(NewError(200, err))
		return
	}
	c.JSON(http.StatusOK, rv)
}

func userInfoApi(c *gin.Context) {
	uid := GetAppUserId(c)
	app := db.GetApp(c)
	type result struct {
		Mobile string `json:"mobile"`
	}
	res := result{}
	err := app.UseDb(func(sdb db.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
			return err
		}
		res.Mobile = user.Mobile
		return nil
	})
	if err != nil {
		c.Error(NewError(200, err))
		return
	}
	c.JSON(http.StatusOK, res)
}
