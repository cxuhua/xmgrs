package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cxuhua/xmgrs/util"

	"github.com/cxuhua/xmgrs/core"

	"github.com/cxuhua/xginx"

	"github.com/gin-gonic/gin"
)

//AddrValue 地址金额
type AddrValue struct {
	Addr      xginx.Address //目标地址
	Value     xginx.Amount  //金额
	OutScript string        //输出脚本
}

func (av AddrValue) String() string {
	return fmt.Sprintf("%s->%d,%s", av.Addr, av.Value, av.OutScript)
}

//解析 金额和定制的输出脚本,第一个,号之后的全是脚本
func parseValueScript(s string) (string, string) {
	p := strings.Index(s, ",")
	if p < 0 {
		return s, ""
	} else if p == len(s)-1 {
		return s[:p], ""
	} else if p < len(s) {
		return s[:p], s[p+1:]
	}
	return "", ""
}

//ParseAddrValue 解析addr->amount格式
func ParseAddrValue(s string) (AddrValue, error) {
	av := AddrValue{}
	v := strings.Split(s, "->")
	if len(v) != 2 {
		return av, errors.New("dst format error")
	}
	amts, outs := parseValueScript(v[1])
	if amts == "" {
		return av, errors.New("amount string miss")
	}
	//如果未设置，启用默认锁定脚本
	if outs == "" {
		outs = string(xginx.DefaultLockedScript)
	}
	amt, err := xginx.ParseIntMoney(amts)
	if err != nil {
		return av, err
	}
	if !amt.IsRange() {
		return av, errors.New("amount range error")
	}
	av.Addr = xginx.Address(v[0])
	err = av.Addr.Check()
	if err != nil {
		return av, err
	}
	av.Value = amt
	err = xginx.CheckScript([]byte(outs))
	if err != nil {
		return av, err
	}
	//设置输出脚本
	av.OutScript = outs
	return av, nil
}

//TxInModel 输出model
type TxInModel struct {
	Addr     xginx.Address `json:"addr"`   //coinbase地址是空的
	Value    xginx.Amount  `json:"value"`  //coinbasevalue是空的
	Script   string        `json:"script"` //输出脚本
	Sequence uint32        `json:"sequence"`
}

//TxOutModel 输出model
type TxOutModel struct {
	Addr   xginx.Address `json:"addr"`
	Value  xginx.Amount  `json:"value"`
	Script string        `json:"script"` //输出脚本
}

//TxModel 交易model
type TxModel struct {
	Ver     uint32       `json:"ver"`
	Ins     []TxInModel  `json:"ins"` //为空是coinbase交易
	Outs    []TxOutModel `json:"outs"`
	Script  string       `json:"script"`  //交易脚本
	Confirm uint32       `json:"confirm"` //确认数 =0 表示在交易池中
	BlkTime uint32       `json:"time"`    //区块时间戳
}

//NewTxModel 创建model
func NewTxModel(tx *xginx.TX, blk *xginx.BlockInfo, bi *xginx.BlockIndex) TxModel {
	m := TxModel{
		Ver:    tx.Ver.ToUInt32(),
		Ins:    []TxInModel{},
		Outs:   []TxOutModel{},
		Script: util.ScriptToStr(tx.Script),
	}
	if blk != nil {
		m.Confirm = bi.Height() - blk.Meta.Height + 1
		m.BlkTime = blk.Meta.Time
	} else {
		m.Confirm = 0
		m.BlkTime = 0
	}
	for _, in := range tx.Ins {
		inv := TxInModel{
			Script: util.ScriptToStr(in.Script),
		}
		if in.IsCoinBase() {
			m.Ins = append(m.Ins, inv)
			continue
		}
		out, err := in.LoadTxOut(bi)
		if err != nil {
			panic(err)
		}
		addr, err := out.Script.GetAddress()
		if err != nil {
			panic(err)
		}
		inv.Addr = addr
		inv.Value = out.Value
		m.Ins = append(m.Ins, inv)
	}
	for _, out := range tx.Outs {
		addr, err := out.Script.GetAddress()
		if err != nil {
			panic(err)
		}
		outv := TxOutModel{
			Addr:   addr,
			Value:  out.Value,
			Script: util.ScriptToStr(out.Script),
		}
		m.Outs = append(m.Outs, outv)
	}
	return m
}

//获取区块链中的交易信息
func getTxInfoAPI(c *gin.Context) {
	args := struct {
		ID string `uri:"id" binding:"HexHash256"`
	}{}
	if err := c.ShouldBindUri(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	id := xginx.NewHASH256(args.ID)
	bi := xginx.GetBlockIndex()
	txv, err := bi.LoadTxValue(id)
	if err != nil {
		c.JSON(http.StatusOK, NewModel(101, err))
		return
	}
	blk, err := bi.LoadBlock(txv.BlkID)
	if err != nil {
		c.JSON(http.StatusOK, NewModel(102, err))
		return
	}
	tx, err := blk.GetTx(txv.TxIdx.ToInt())
	if err != nil {
		c.JSON(http.StatusOK, NewModel(103, err))
		return
	}
	res := struct {
		Code   int     `json:"code"`
		Height uint32  `json:"height"` //区块链高度
		Item   TxModel `json:"item"`
	}{
		Height: bi.Height(),
		Item:   NewTxModel(tx, blk, bi),
	}
	c.JSON(http.StatusOK, res)
}

//发布交易
func submitTxAPI(c *gin.Context) {
	args := struct {
		ID string `form:"id" binding:"HexHash256"` //交易id
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	id := xginx.NewHASH256(args.ID)
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	bi := xginx.GetBlockIndex()
	var tx *xginx.TX = nil
	err := app.UseTx(func(db core.IDbImp) error {
		ttx, err := db.GetTx(id.Bytes())
		if err != nil {
			return err
		}
		//如果已经在链中
		if _, err := bi.LoadTX(id); err == nil {
			return nil
		}
		if !core.ObjectIDEqual(ttx.UserID, uid) {
			return errors.New("not mine ttx")
		}
		tx, err = ttx.ToTx(db, bi)
		if err != nil {
			return err
		}
		err = ttx.SetTxState(db, core.TTxStatePool)
		if err != nil {
			return err
		}
		txp := bi.GetTxPool()
		err = txp.PushTx(bi, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, NewModel(0, "OK"))
}

//创建交易
func createTxAPI(c *gin.Context) {
	args := struct {
		Dst    []string     `form:"dst" binding:"gt=0"`        //addr->amount 向addr转amount个,使用script脚本
		Fee    xginx.Amount `form:"fee" binding:"gte=0"`       //交易费
		Desc   string       `form:"desc"`                      //描述
		Script string       `form:"script" binding:"IsScript"` //交易脚本
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	bi := xginx.GetBlockIndex()
	var ttx *core.TTx = nil
	err := app.UseTx(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		lis := core.NewSignListener(db, user)
		mi := bi.NewTrans(lis)
		for _, dst := range args.Dst {
			av, err := ParseAddrValue(dst)
			if err != nil {
				return err
			}
			mi.Add(av.Addr, av.Value, xginx.Script(av.OutScript))
		}
		mi.Fee = args.Fee
		tx, err := mi.NewTx(0, []byte(args.Script))
		if err != nil {
			return err
		}
		ttx, err = user.SaveTx(db, tx, lis, args.Desc)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	res := struct {
		Code int      `json:"code"`
		Item TTxModel `json:"item"`
	}{
		Code: 0,
		Item: NewTTxModel(ttx, bi),
	}
	c.JSON(http.StatusOK, res)
}

//创建账号
func createAccountAPI(c *gin.Context) {
	args := struct {
		Num  uint8    `form:"num"`  //私钥数量
		Less uint8    `form:"less"` //至少通过数量
		Arb  bool     `form:"arb"`  //启用仲裁
		ID   []string `form:"id"`   //为空将自动创建私钥
		Tags []string `form:"tags"` //标签
		Desc string   `form:"desc"` //描述
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	//去除重复的数据
	args.ID = util.RemoveRepeat(args.ID)
	args.Tags = util.RemoveRepeat(args.Tags)
	//
	type item struct {
		ID   xginx.Address `json:"id"`   //账号地址id
		Tags []string      `json:"tags"` //标签，分组用
		Num  uint8         `json:"num"`  //总的密钥数量
		Less uint8         `json:"less"` //至少通过的签名数量
		Arb  bool          `json:"arb"`  //是否启用仲裁
		Kid  []string      `json:"kis"`  //相关的私钥
		Desc string        `json:"desc"` //描述
	}
	type result struct {
		Code int  `json:"code"`
		Item item `json:"item"`
	}
	res := result{
		Code: 0,
	}
	res.Item.Tags = []string{}
	res.Item.Kid = []string{}
	app := core.GetApp(c)
	err := app.UseTx(func(db core.IDbImp) error {
		acc, err := core.NewAccount(db, args.Num, args.Less, args.Arb, args.ID, args.Desc, args.Tags)
		if err != nil {
			return err
		}
		err = db.InsertAccount(acc)
		if err != nil {
			return err
		}
		i := item{}
		i.ID = acc.ID
		i.Tags = acc.Tags
		i.Num = acc.Num
		i.Less = acc.Less
		i.Arb = acc.Arb != xginx.InvalidArb
		i.Desc = acc.Desc
		i.Kid = acc.Kid
		res.Item = i
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, res)
}

//创建一个私钥
func createUserPrivateAPI(c *gin.Context) {
	args := struct {
		Desc string        `form:"desc"` //私钥描述
		Pass []string      `form:"pass"` //私钥密码,存在会加密私钥
		Exp  time.Duration `form:"exp"`  //过期时间 单位:秒
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	if args.Exp == 0 {
		args.Exp = core.DefaultExpTime
	}
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	type item struct {
		ID     string `json:"id"`
		Desc   string `json:"desc"`
		Cipher int    `json:"cipher"`
		Time   int64  `json:"time"`
	}
	type result struct {
		Code int  `json:"code"`
		Item item `json:"item"`
	}
	m := result{}
	err := app.UseTx(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		pri, err := user.NewPrivate(db, args.Exp, args.Desc, args.Pass...)
		if err != nil {
			return err
		}
		i := item{}
		i.ID = pri.ID
		i.Desc = pri.Desc
		i.Cipher = int(pri.Cipher)
		i.Time = pri.Time
		m.Item = i
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, m)
}

//获取用户的私钥
func listPrivatesAPI(c *gin.Context) {
	app := core.GetApp(c)
	uid := GetAppUserID(c)
	type item struct {
		ID     string `json:"id"`
		Desc   string `json:"desc"`
		Cipher int    `json:"cipher"`
		Time   int64  `json:"time"`
	}
	type result struct {
		Code  int    `json:"code"`
		Items []item `json:"items"`
	}
	res := result{
		Code:  0,
		Items: []item{},
	}
	err := app.UseDb(func(db core.IDbImp) error {
		pris, err := db.ListPrivates(uid)
		if err != nil {
			return err
		}
		for _, v := range pris {
			i := item{
				ID:     v.ID,
				Desc:   v.Desc,
				Cipher: int(v.Cipher),
				Time:   v.Time,
			}
			res.Items = append(res.Items, i)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, NewModel(200, err))
		return
	}
	c.JSON(http.StatusOK, res)
}

//获取区块中的用户交易
func listTxsAPI(c *gin.Context) {
	args := struct {
		Addr xginx.Address `uri:"addr" binding:"IsAddress"`
	}{}
	if err := c.ShouldBindUri(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	bi := xginx.GetBlockIndex()
	txs, err := bi.ListTxs(args.Addr)
	if err != nil {
		c.JSON(http.StatusOK, NewModel(101, err))
		return
	}
	res := struct {
		Code   int       `json:"code"`
		Height uint32    `json:"height"` //区块链高度
		Items  []TxModel `json:"items"`
	}{
		Height: bi.Height(),
		Items:  []TxModel{},
	}
	txp := bi.GetTxPool()
	for _, v := range txs {
		if v.IsPool() {
			tx, err := txp.Get(v.TxID)
			if err != nil {
				c.JSON(http.StatusOK, NewModel(104, err))
				return
			}
			item := NewTxModel(tx, nil, bi)
			res.Items = append(res.Items, item)
		} else {
			txv, err := bi.LoadTxValue(v.TxID)
			if err != nil {
				c.JSON(http.StatusOK, NewModel(102, err))
				return
			}
			blk, err := bi.LoadBlock(txv.BlkID)
			if err != nil {
				c.JSON(http.StatusOK, NewModel(103, err))
				return
			}
			tx, err := blk.GetTx(txv.TxIdx.ToInt())
			if err != nil {
				c.JSON(http.StatusOK, NewModel(104, err))
				return
			}
			item := NewTxModel(tx, blk, bi)
			res.Items = append(res.Items, item)
		}
	}
	c.JSON(http.StatusOK, res)
}
