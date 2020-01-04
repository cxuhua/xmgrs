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

//签名一个交易
func signTxApi(c *gin.Context) {
	args := struct {
		Id string `form:"id"` //交易id hex格式
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	id := xginx.NewHASH256(args.Id)
	err := app.UseTx(func(db core.IDbImp) error {
		_, err := db.GetTx(id.Bytes())
		if err != nil {
			return err
		}
		sigs, err := db.ListUserSigs(uid, id)
		if err != nil {
			return err
		}
		for _, sig := range sigs {
			if sig.IsSign {
				continue
			}
			err := sig.Sign(db)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, NewModel(0, "SignOK"))
}

type TTxModel struct {
	Id       string       `json:"id"`
	Ver      uint32       `json:"ver"`
	Ins      []TxInModel  `json:"ins"`
	Outs     []TxOutModel `json:"outs"`
	LockTime uint32       `json:"lt"`
	Time     int64        `json:"time"`
	Desc     string       `json:"desc"`
}

func NewTTxModel(ttx *core.TTx, bi *xginx.BlockIndex) TTxModel {
	m := TTxModel{
		Id:       xginx.NewHASH256(ttx.Id).String(),
		Ver:      ttx.Ver,
		Ins:      []TxInModel{},
		Outs:     []TxOutModel{},
		LockTime: ttx.LockTime,
		Time:     ttx.Time,
		Desc:     ttx.Desc,
	}
	for _, in := range ttx.Ins {
		out, err := in.ToTxIn().LoadTxOut(bi)
		if err != nil {
			panic(err)
		}
		addr, err := out.Script.GetAddress()
		if err != nil {
			panic(err)
		}
		inv := TxInModel{
			Addr:     addr,
			Value:    out.Value,
			Sequence: in.Sequence,
		}
		m.Ins = append(m.Ins, inv)
	}
	for _, out := range ttx.Outs {
		addr, err := out.Script.GetAddress()
		if err != nil {
			panic(err)
		}
		outv := TxOutModel{
			Addr:  addr,
			Value: xginx.Amount(out.Value),
		}
		m.Outs = append(m.Outs, outv)
	}
	return m
}

//获取待签名交易
func listUserSignTxsApi(c *gin.Context) {
	app := core.GetApp(c)
	bi := xginx.GetBlockIndex()
	uid := GetAppUserId(c)
	ttxs := []*core.TTx{}
	err := app.UseDb(func(db core.IDbImp) error {
		txs, err := db.ListUserTxs(uid, false)
		if err != nil {
			return err
		}
		for _, ttx := range txs {
			//如果已经签名成功忽略
			_, err := ttx.ToTx(db, bi)
			if err == nil {
				continue
			}
			ttxs = append(ttxs, ttx)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	type result struct {
		Code  int        `json:"code"`
		Items []TTxModel `json:"items"`
	}
	res := result{
		Code:  0,
		Items: []TTxModel{},
	}
	for _, ttx := range ttxs {
		res.Items = append(res.Items, NewTTxModel(ttx, bi))
	}
	c.JSON(http.StatusOK, res)
}

//获取用户的账号
func listUserAccountsApi(c *gin.Context) {
	//账户管理
	type item struct {
		Id   xginx.Address `json:"id"`   //账号地址id
		Tags []string      `json:"tags"` //标签，分组用
		Num  uint8         `json:"num"`  //总的密钥数量
		Less uint8         `json:"less"` //至少通过的签名数量
		Arb  bool          `json:"arb"`  //是否仲裁
		Pks  []string      `json:"pks"`  //相关的私钥
		Desc string        `json:"desc"` //描述
	}
	type result struct {
		Code  int    `json:"code"`
		Items []item `json:"items"`
	}
	res := result{
		Code:  0,
		Items: []item{},
	}
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	var accs []*core.TAccount = nil
	err := app.UseDb(func(db core.IDbImp) error {
		acc, err := db.ListAccounts(uid)
		if err != nil {
			return err
		}
		accs = acc
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	for _, v := range accs {
		i := item{
			Id:   v.Id,
			Tags: v.Tags,
			Num:  v.Num,
			Less: v.Less,
			Arb:  v.Arb != xginx.InvalidArb,
			Desc: v.Desc,
			Pks:  []string{},
		}
		for _, h := range v.Pkh {
			i.Pks = append(i.Pks, core.GetPrivateId(h))
		}
		res.Items = append(res.Items, i)
	}
	c.JSON(http.StatusOK, res)
}

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
	rv := Model{}
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
		Model
		Token string `json:"token"`
	}
	rv := result{}
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
		err = db.SetUserId(tk, user.Id.Hex(), time.Hour*24*7)
		if err != nil {
			rv.Code = 105
			return err
		}
		//返回加密的token
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
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	type item struct {
		Id      xginx.Address `json:"id"`      //所属账号地址
		Matured bool          `json:"matured"` //是否成熟
		Pool    bool          `json:"pool"`    //是否是内存池中的
		Value   xginx.Amount  `json:"value"`   //数量
		TxId    string        `json:"tx"`      //交易id
		Index   uint32        `json:"index"`   //输出索引
		Height  uint32        `json:"height"`  //所在区块高度
	}
	type result struct {
		Model
		Height uint32 `json:"height"` //当前区块高度
		Items  []item `json:"items"`
	}
	res := result{}
	bi := xginx.GetBlockIndex()
	spent := bi.NextHeight()
	res.Height = bi.Height()
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
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
			i.TxId = coin.TxId.String()
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
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	type result struct {
		Model
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"` //可用余额
		Locks  xginx.Amount `json:"locks"` //锁定的
	}
	res := result{}
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
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
