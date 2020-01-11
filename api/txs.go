package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cxuhua/xmgrs/util"

	"github.com/cxuhua/xmgrs/core"

	"github.com/cxuhua/xginx"

	"github.com/gin-gonic/gin"
)

type AddrValue struct {
	Addr  xginx.Address
	Value xginx.Amount
}

func (av AddrValue) String() string {
	return fmt.Sprintf("%s->%d", av.Addr, av.Value)
}

func NewAddrValue(s string) (AddrValue, error) {
	av := AddrValue{}
	v := strings.Split(s, "->")
	if len(v) != 2 {
		return av, errors.New("dst format error")
	}
	amt, err := xginx.ParseIntMoney(v[1])
	if err != nil {
		return av, err
	}
	if !amt.IsRange() {
		return av, errors.New("amount range error")
	}
	av.Addr = xginx.Address(v[0])
	av.Value = amt
	return av, nil
}

type TxInModel struct {
	Addr     xginx.Address `json:"addr"`  //coinbase地址是空的
	Value    xginx.Amount  `json:"value"` //coinbasevalue是空的
	Sequence uint32        `json:"sequence"`
}

type TxOutModel struct {
	Addr  xginx.Address `json:"addr"`
	Value xginx.Amount  `json:"value"`
}

type TxModel struct {
	Ver      uint32       `json:"ver"`
	Ins      []TxInModel  `json:"ins"` //为空是coinbase交易
	Outs     []TxOutModel `json:"outs"`
	LockTime uint32       `json:"lt"`
	Confirm  uint32       `json:"confirm"` //确认数
	BlkTime  uint32       `json:"time"`    //区块时间戳
	Pool     bool         `json:"pool"`    //是否来自交易池
}

func NewTxModel(tx *xginx.TX, blk *xginx.BlockInfo, bi *xginx.BlockIndex) TxModel {
	m := TxModel{
		Ver:      tx.Ver.ToUInt32(),
		Ins:      []TxInModel{},
		Outs:     []TxOutModel{},
		LockTime: tx.LockTime,
		Pool:     tx.IsPool(),
	}
	if blk != nil {
		m.Confirm = bi.Height() - blk.Meta.Height + 1
		m.BlkTime = blk.Meta.Time
	} else {
		m.Confirm = 0
		m.BlkTime = 0
	}
	for _, in := range tx.Ins {
		if in.IsCoinBase() {
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
		inv := TxInModel{
			Addr:     addr,
			Value:    out.Value,
			Sequence: in.Sequence,
		}
		m.Ins = append(m.Ins, inv)
	}
	for _, out := range tx.Outs {
		addr, err := out.Script.GetAddress()
		if err != nil {
			panic(err)
		}
		outv := TxOutModel{
			Addr:  addr,
			Value: out.Value,
		}
		m.Outs = append(m.Outs, outv)
	}
	return m
}

//获取区块链中的交易信息
func getTxInfoApi(c *gin.Context) {
	args := struct {
		Id string `uri:"id"`
	}{}
	if err := c.ShouldBindUri(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	id := xginx.NewHASH256(args.Id)
	bi := xginx.GetBlockIndex()
	txv, err := bi.LoadTxValue(id)
	if err != nil {
		c.JSON(http.StatusOK, NewModel(101, err))
		return
	}
	blk, err := bi.LoadBlock(txv.BlkId)
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
func submitTxApi(c *gin.Context) {
	args := struct {
		Id string `form:"id"` //交易id
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	id := xginx.NewHASH256(args.Id).Bytes()
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	bi := xginx.GetBlockIndex()
	var tx *xginx.TX = nil
	err := app.UseTx(func(db core.IDbImp) error {
		ttx, err := db.GetTx(id)
		if err != nil {
			return err
		}
		if !core.ObjectIDEqual(ttx.UserId, uid) {
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
func createTxApi(c *gin.Context) {
	args := struct {
		Dst  []string     `form:"dst"`  //addr->amount 向addr转amount个
		Fee  xginx.Amount `form:"fee"`  //交易费
		Desc string       `form:"desc"` //描述
		LT   uint32       `form:"lt"`   //locktime
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	if len(args.Dst) == 0 {
		c.JSON(http.StatusOK, NewModel(101, "dst args miss"))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserId(c)
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
			av, err := NewAddrValue(dst)
			if err != nil {
				return err
			}
			mi.Add(av.Addr, av.Value)
		}
		mi.Fee = args.Fee
		tx, err := mi.NewTx(args.LT)
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
func createAccountApi(c *gin.Context) {
	args := struct {
		Num  uint8    `form:"num"`  //私钥数量
		Less uint8    `form:"less"` //至少通过数量
		Arb  bool     `form:"arb"`  //启用仲裁
		Id   []string `form:"id"`   //为空将自动创建私钥
		Tags []string `form:"tags"` //标签
		Desc string   `form:"desc"` //描述
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	//去除重复的数据
	args.Id = util.RemoveRepeat(args.Id)
	args.Tags = util.RemoveRepeat(args.Tags)
	//
	type item struct {
		Id   xginx.Address `json:"id"`   //账号地址id
		Tags []string      `json:"tags"` //标签，分组用
		Num  uint8         `json:"num"`  //总的密钥数量
		Less uint8         `json:"less"` //至少通过的签名数量
		Arb  bool          `json:"arb"`  //是否仲裁
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
		acc, err := core.NewAccount(db, args.Num, args.Less, args.Arb, args.Id)
		if err != nil {
			return err
		}
		if len(args.Tags) > 0 {
			acc.Tags = args.Tags
		}
		acc.Desc = args.Desc
		err = db.InsertAccount(acc)
		if err != nil {
			return err
		}
		i := item{}
		i.Id = acc.Id
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
func createUserPrivateApi(c *gin.Context) {
	args := struct {
		Desc string `form:"desc"` //私钥描述
		Pass string `form:"pass"` //私钥密码
	}{}
	if err := c.ShouldBind(&args); err != nil {
		c.JSON(http.StatusOK, NewModel(100, err))
		return
	}
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	type item struct {
		Id     string `json:"id"`
		Desc   string `json:"desc"`
		Cipher int    `json:"cipher"`
		Time   int64  `json:"time"`
	}
	type result struct {
		Code int  `json:"code"`
		Item item `json:"item"`
	}
	pass := []string{}
	if args.Pass != "" {
		pass = []string{args.Pass}
	}
	m := result{}
	err := app.UseTx(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		pri, err := user.NewPrivate(db, args.Desc, pass...)
		if err != nil {
			return err
		}
		i := item{}
		i.Id = pri.Id
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
func listPrivatesApi(c *gin.Context) {
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	type item struct {
		Id     string `json:"id"`
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
				Id:     v.Id,
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
func listTxsApi(c *gin.Context) {
	args := struct {
		Addr xginx.Address `uri:"addr"`
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
			tx, err := txp.Get(v.TxId)
			if err != nil {
				c.JSON(http.StatusOK, NewModel(104, err))
				return
			}
			item := NewTxModel(tx, nil, bi)
			res.Items = append(res.Items, item)
		} else {
			txv, err := bi.LoadTxValue(v.TxId)
			if err != nil {
				c.JSON(http.StatusOK, NewModel(102, err))
				return
			}
			blk, err := bi.LoadBlock(txv.BlkId)
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
