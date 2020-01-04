package api

import (
	"net/http"

	"github.com/cxuhua/xmgrs/core"

	"github.com/cxuhua/xginx"

	"github.com/gin-gonic/gin"
)

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
	Height   uint32       `json:"height"`  //所在区块高度
	Confirm  uint32       `json:"confirm"` //确认数
	BlkTime  uint32       `json:"time"`    //区块时间戳
}

func NewTxModel(tx *xginx.TX, blk *xginx.BlockInfo, bi *xginx.BlockIndex) TxModel {
	m := TxModel{
		Ver:      tx.Ver.ToUInt32(),
		Ins:      []TxInModel{},
		Outs:     []TxOutModel{},
		LockTime: tx.LockTime,
		Height:   blk.Meta.Height,
		Confirm:  bi.Height() - blk.Meta.Height + 1,
		BlkTime:  blk.Meta.Time,
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
		Code int  `json:"code"`
		Item item `json:"item"`
	}
	res := result{
		Code: 0,
	}
	res.Item.Tags = []string{}
	res.Item.Pks = []string{}
	app := core.GetApp(c)
	uid := GetAppUserId(c)
	err := app.UseTx(func(db core.IDbImp) error {
		acc, err := core.NewAccount(db, uid, args.Num, args.Less, args.Arb, args.Id)
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
		res.Item.Id = acc.Id
		res.Item.Tags = acc.Tags
		res.Item.Num = acc.Num
		res.Item.Less = acc.Less
		res.Item.Arb = acc.Arb != xginx.InvalidArb
		res.Item.Desc = acc.Desc
		for _, h := range acc.Pkh {
			res.Item.Pks = append(res.Item.Pks, core.GetPrivateId(h))
		}
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
		Desc string `form:"desc"`
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
	m := result{}
	err := app.UseTx(func(db core.IDbImp) error {
		user, err := db.GetUserInfo(uid)
		if err != nil {
			return err
		}
		pri, err := user.NewPrivate(db, args.Desc)
		if err != nil {
			return err
		}
		m.Item.Id = pri.Id
		m.Item.Desc = pri.Desc
		m.Item.Cipher = int(pri.Cipher)
		m.Item.Time = pri.Time
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
	}
	for _, txv := range txs {
		txv, err := bi.LoadTxValue(txv.TxId)
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
	c.JSON(http.StatusOK, res)
}
