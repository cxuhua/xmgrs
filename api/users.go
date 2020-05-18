package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cxuhua/xginx"

	"github.com/cxuhua/xmgrs/core"

	"github.com/gin-gonic/gin"
)

// 设置用户推送id
func setUserPushIDAPI(c *gin.Context) {
	args := struct {
		PushID string `form:"pid" binding:"required"` //推送id
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	err := app.UseDb(func(db core.IDbImp) error {
		return db.SetPushID(uid, args.PushID)
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, NewModel(0, "OK"))
}

//退出登陆
func quitLoginAPI(c *gin.Context) {
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	app.UseDb(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		return db.DelUserID(user.Token)
	})
	c.JSON(http.StatusOK, NewModel(0, "OK"))
}

//签名一个交易
func signTxAPI(c *gin.Context) {
	args := struct {
		ID   string `form:"id" binding:"HexHash256"` //交易id hex格式
		Pass string `form:"pass"`                    //私钥密码
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	id := xginx.NewHASH256(args.ID)
	bi := xginx.GetBlockIndex()
	err := app.UseTx(func(db core.IDbImp) error {
		ttx, err := db.GetTx(id.Bytes())
		if err != nil {
			return err
		}
		//如果不是新交易
		if ttx.State != core.TTxStateNew {
			return nil
		}
		//获取需要我签名的信息
		sigs, err := db.ListUserSigs(uid, id)
		if err != nil {
			return err
		}
		//开始签名
		for _, sig := range sigs {
			if sig.IsSign {
				continue
			}
			err := sig.Sign(db, args.Pass)
			if err != nil {
				return err
			}
		}
		//再次查询交易信息
		ttx, err = db.GetTx(id.Bytes())
		if err != nil {
			return err
		}
		//如果签名验证成功,更新为已经签名，否则需要等待所有签名执行完成
		if ttx.Verify(db, bi) {
			err = ttx.SetTxState(db, core.TTxStateSign)
		} else {
			err = fmt.Errorf("ttx verify error %v", args.ID)
		}
		return err
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, NewModel(0, "SignOK"))
}

//TTxModel 交易model
type TTxModel struct {
	ID       string        `json:"id"`
	Ver      uint32        `json:"ver"`
	Ins      []TxInModel   `json:"ins"`
	Outs     []TxOutModel  `json:"outs"`
	LockTime uint32        `json:"lt"`
	Time     int64         `json:"time"`
	Desc     string        `json:"desc"`
	State    core.TTxState `json:"state"`
}

//NewTTxModel 创建交易model
func NewTTxModel(ttx *core.TTx, bi *xginx.BlockIndex) TTxModel {
	m := TTxModel{
		ID:    xginx.NewHASH256(ttx.ID).String(),
		Ver:   ttx.Ver,
		Ins:   []TxInModel{},
		Outs:  []TxOutModel{},
		Time:  ttx.Time,
		Desc:  ttx.Desc,
		State: ttx.State,
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
			Addr:  addr,
			Value: out.Value,
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
func listUserSignTxsAPI(c *gin.Context) {
	app := core.GetApp(c)
	bi := xginx.GetBlockIndex()
	uid := GetAppUserID(c)
	ttxs := []*core.TTx{}
	err := app.UseDb(func(db core.IDbImp) error {
		txs, err := db.ListUserTxs(uid, false)
		if err != nil {
			return err
		}
		for _, ttx := range txs {
			//如果已经签名
			if ttx.Verify(db, bi) {
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
func listUserAccountsAPI(c *gin.Context) {
	//账户管理
	type item struct {
		ID   xginx.Address `json:"id"`   //账号地址id
		Tags []string      `json:"tags"` //标签，分组用
		Num  uint8         `json:"num"`  //总的密钥数量
		Less uint8         `json:"less"` //至少通过的签名数量
		Arb  bool          `json:"arb"`  //是否仲裁
		Kid  []string      `json:"kid"`  //相关的私钥
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
	uid := GetAppUserID(c)
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
			ID:   v.ID,
			Tags: v.Tags,
			Num:  v.Num,
			Less: v.Less,
			Arb:  v.Arb != xginx.InvalidArb,
			Desc: v.Desc,
			Kid:  v.Kid,
		}
		res.Items = append(res.Items, i)
	}
	c.JSON(http.StatusOK, res)
}

//注册
func registerAPI(c *gin.Context) {
	args := struct {
		Mobile   string `form:"mobile" binding:"required"` //手机号
		UserPass string `form:"upass" binding:"required"`  //用户登陆密码
		KeyPass  string `form:"kpass"`                     //私钥加密密码
		Code     string `form:"code" binding:"required"`   //手机验证码
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	if args.Mobile == "" || args.UserPass == "" {
		c.JSON(http.StatusOK, NewModel(101, "mobile or pass args error"))
		return
	}
	if args.KeyPass != "" && args.KeyPass == args.UserPass {
		c.JSON(http.StatusOK, NewModel(102, "login pass == key pass"))
		return
	}
	if args.Code != "9527" {
		c.JSON(http.StatusOK, NewModel(103, "code error"))
		return
	}
	rv := Model{}
	app := core.GetApp(c)
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfoWithMobile(args.Mobile)
		if err == nil {
			rv.Code = 104
			return errors.New("mobile exists")
		}
		user = core.NewUser(args.Mobile, args.UserPass, args.KeyPass)
		err = sdb.InsertUser(user)
		if err != nil {
			rv.Code = 105
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(rv.Code, err))
		return
	}
	c.JSON(http.StatusOK, rv)
}

func loginAPI(c *gin.Context) {
	args := struct {
		Mobile string `form:"mobile" binding:"required"`
		Pass   string `form:"pass" binding:"required"`
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
		err = db.SetUserToken(user.ID, tk)
		if err != nil {
			rv.Code = 104
			return err
		}
		err = db.SetUserID(tk, user.ID, core.TokenTime)
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
func listCoinsAPI(c *gin.Context) {
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	type item struct {
		ID     xginx.Address `json:"id"`     //所属账号地址
		Locked bool          `json:"locked"` //是否被锁定，锁定的不可用
		Pool   bool          `json:"pool"`   //是否是内存池中的
		Value  xginx.Amount  `json:"value"`  //数量
		TxID   string        `json:"tx"`     //交易id
		Index  uint32        `json:"index"`  //输出索引
		Height uint32        `json:"height"` //所在区块高度
	}
	type result struct {
		Model
		Height uint32 `json:"height"` //当前区块高度
		Items  []item `json:"items"`
	}
	bi := xginx.GetBlockIndex()
	res := result{
		Items:  []item{},
		Height: bi.Height(),
	}
	//判断消费高度下金额是否可用
	spent := bi.NextHeight()
	err := app.UseDb(func(sdb core.IDbImp) error {
		user, err := sdb.GetUserInfo(uid)
		if err != nil {
			res.Code = 101
			return err
		}
		//获取用户余额
		coins, err := user.ListCoins(sdb, bi)
		if err != nil {
			res.Code = 102
			return err
		}
		for _, coin := range coins.All {
			i := item{}
			id, err := xginx.EncodeAddress(coin.CPkh)
			if err != nil {
				continue
			}
			i.ID = id
			//未成熟的金额将被锁定
			i.Locked = !coin.IsMatured(spent)
			i.Pool = coin.IsPool()
			i.Value = coin.Value
			i.TxID = coin.TxID.String()
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

func userInfoAPI(c *gin.Context) {
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	type result struct {
		Model
		Mobile string       `json:"mobile"`
		Coins  xginx.Amount `json:"coins"` //可用余额
		Locks  xginx.Amount `json:"locks"` //锁定的
	}
	res := result{}
	err := app.UseDb(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		//获取用户余额
		bi := xginx.GetBlockIndex()
		coins, err := user.ListCoins(db, bi)
		if err != nil {
			res.Code = 101
			return err
		}
		//可用的
		res.Coins = coins.Coins.Balance()
		//锁定的
		res.Locks = coins.Locks.Balance()
		//
		res.Mobile = user.Mobile
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(res.Code, err))
		return
	}
	c.JSON(http.StatusOK, res)
}
