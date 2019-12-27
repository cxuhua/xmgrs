package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cxuhua/xginx"

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
			rv.Meta = 102
			return fmt.Errorf("get user info error %w", err)
		}
		if !user.CheckPass(args.Pass) {
			rv.Meta = 103
			return errors.New("password error")
		}
		tk := app.GenToken()
		err = db.SetUserToken(user.Id, tk)
		if err != nil {
			rv.Meta = 104
			return err
		}
		err = app.SetUserId(tk, user.Id.Hex(), time.Hour*24*7)
		if err != nil {
			rv.Meta = 105
			return err
		}
		rv.Token = app.EncryptToken(tk)
		return nil
	})
	if err != nil {
		c.Error(NewError(rv.Meta, err))
		return
	}
	c.JSON(http.StatusOK, rv)
}

func userInfoApi(c *gin.Context) {
	uid := GetAppUserId(c)
	app := db.GetApp(c)
	type result struct {
		Meta   int          `json:"meta"`
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"` //可用余额
		Locks  xginx.Amount `json:"locks"` //锁定的
	}
	res := result{Meta: 0}
	err := app.UseDb(func(sdb db.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
			res.Meta = 100
			return err
		}
		//获取用户余额
		bi := xginx.GetBlockIndex()
		coins, err := user.ListCoins(sdb, bi)
		if err != nil {
			res.Meta = 101
			return err
		}
		res.Coins = coins.Coins.Balance()
		res.Locks = coins.Locks.Balance()
		res.Mobile = user.Mobile
		return nil
	})
	if err != nil {
		c.Error(NewError(res.Meta, err))
		return
	}

	c.JSON(http.StatusOK, res)
}
