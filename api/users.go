package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cxuhua/xginx"

	"github.com/cxuhua/xmgrs/core"

	"github.com/gin-gonic/gin"
)

//注册
func registerApi(c *gin.Context) {
	args := struct {
		Mobile string `form:"mobile"`
		Pass   string `form:"pass"`
		Code   string `form:"code"` //手机验证码
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	if args.Mobile == "" || args.Pass == "" {
		c.JSON(http.StatusOK, NewModel(101, "mobile or pass args error"))
		return
	}
	if args.Code != "9527" {
		c.JSON(http.StatusOK, NewModel(102, "code error"))
		return
	}
	type result struct {
		Code int `json:"meta"`
	}
	rv := result{
		Code: 0,
	}
	app := core.GetApp(c)
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfoWithMobile(args.Mobile)
		if err == nil {
			rv.Code = 103
			return errors.New("mobile exists")
		}
		user = core.NewUser(args.Mobile, []byte(args.Pass))
		rv.Code = 104
		return sdb.InsertUser(user)
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(rv.Code, err))
		return
	}
	c.JSON(http.StatusOK, rv)
}

func loginApi(c *gin.Context) {
	args := struct {
		Mobile string `form:"mobile"`
		Pass   string `form:"pass"`
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	if args.Mobile == "" || args.Pass == "" {
		c.JSON(http.StatusOK, NewModel(101, "mobile or pass args error"))
		return
	}
	type result struct {
		Code  int    `json:"code"`
		Token string `json:"token"`
	}
	rv := result{
		Code: 0,
	}
	app := core.GetApp(c)
	err := app.UseDb(func(db core.IDbImp) error {
		user, err := db.GetUserInfoWithMobile(args.Mobile)
		if err != nil {
			rv.Code = 102
			return fmt.Errorf("get user info error %w", err)
		}
		if !user.CheckPass(args.Pass) {
			rv.Code = 103
			return errors.New("password error")
		}
		tk := app.GenToken()
		err = db.SetUserToken(user.Id, tk)
		if err != nil {
			rv.Code = 104
			return err
		}
		err = app.SetUserId(tk, user.Id.Hex(), time.Hour*24*7)
		if err != nil {
			rv.Code = 105
			return err
		}
		rv.Token = app.EncryptToken(tk)
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(rv.Code, err))
		return
	}
	c.JSON(http.StatusOK, rv)
}

//获取可用的金额列表
func listCoinsApi(c *gin.Context) {
	uid := GetAppUserId(c)
	app := core.GetApp(c)
	type item struct {
		Id      xginx.Address `json:"id"`      //所属账号地址
		Matured bool          `json:"matured"` //是否成熟
		Pool    bool          `json:"pool"`    //是否是内存池中的
		Value   xginx.Amount  `json:"value"`   //数量
		TxId    xginx.HASH256 `json:"tx"`      //交易id
		Index   uint32        `json:"index"`   //输出索引
		Height  uint32        `json:"height"`  //所在区块高度
	}
	type result struct {
		Code  int    `json:"code"`
		Items []item `json:"items"`
	}
	res := result{Code: 0}
	bi := xginx.GetBlockIndex()
	spent := bi.NextHeight()
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
			res.Code = 100
			return err
		}
		//获取用户余额
		bi := xginx.GetBlockIndex()
		coins, err := user.ListCoins(sdb, bi)
		if err != nil {
			res.Code = 101
			return err
		}
		for _, coin := range coins.All {
			i := item{}
			id, err := xginx.EncodeAddress(coin.CPkh)
			if err != nil {
				continue
			}
			i.Id = id
			i.Matured = coin.IsMatured(spent)
			i.Pool = coin.IsPool()
			i.Value = coin.Value
			i.TxId = coin.TxId
			i.Index = coin.Index.ToUInt32()
			i.Height = coin.Height.ToUInt32()
			res.Items = append(res.Items, i)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(res.Code, err))
		return
	}
	c.JSON(http.StatusOK, res)
}

func userInfoApi(c *gin.Context) {
	uid := GetAppUserId(c)
	app := core.GetApp(c)
	type result struct {
		Code   int          `json:"code"`
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"` //可用余额
		Locks  xginx.Amount `json:"locks"` //锁定的
	}
	res := result{Code: 0}
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
			res.Code = 100
			return err
		}
		//获取用户余额
		bi := xginx.GetBlockIndex()
		coins, err := user.ListCoins(sdb, bi)
		if err != nil {
			res.Code = 101
			return err
		}
		res.Coins = coins.Coins.Balance()
		res.Locks = coins.Locks.Balance()
		res.Mobile = user.Mobile
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(res.Code, err))
		return
	}
	c.JSON(http.StatusOK, res)
}
