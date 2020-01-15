package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/cxuhua/xmgrs/config"

	"github.com/cxuhua/xmgrs/api"

	"github.com/cxuhua/xmgrs/core"

	"github.com/gin-gonic/gin"

	"github.com/cxuhua/xginx"
)

//实现自己的监听器
type mylis struct {
	xginx.Listener
	ctx    context.Context
	cancel context.CancelFunc
	xhttp  *http.Server
	app    *core.App
}

func (lis *mylis) OnLinkBlock(blk *xginx.BlockInfo) {
	//当一个区块连接到链上
	//更新交易交易状态
	lis.app.UseDb(func(db core.IDbImp) error {
		for _, tx := range blk.Txs {
			id, err := tx.ID()
			if err != nil {
				continue
			}
			ttx, err := db.GetTx(id[:])
			if err != nil {
				continue
			}
			err = ttx.SetTxState(db, core.TTxStateBlock)
			if err != nil {
				xginx.LogError("set tx state error", err)
			}
		}
		return nil
	})
}

func (lis *mylis) OnUnlinkBlock(blk *xginx.BlockInfo) {
	//当一个区块从链断开
	lis.app.UseDb(func(db core.IDbImp) error {
		for _, tx := range blk.Txs {
			id, err := tx.ID()
			if err != nil {
				continue
			}
			ttx, err := db.GetTx(id[:])
			if err != nil {
				continue
			}
			err = ttx.SetTxState(db, core.TTxStateCancel)
			if err != nil {
				xginx.LogError("set tx state error", err)
			}
		}
		return nil
	})
}

func (lis *mylis) run() {
	lis.ctx, lis.cancel = xginx.GetContext()
	core.RedisURI = config.Redis
	core.MongoURI = config.Mongo
	//创建一个全局连接
	lis.app = core.InitApp(lis.ctx)
	//
	m := api.InitEngine(lis.ctx)

	lis.xhttp = &http.Server{
		Addr:    config.HttpAddr,
		Handler: m,
	}
	//启动http服务
	if err := lis.xhttp.ListenAndServe(); err != nil {
		xginx.LogError("run serve info", err)
	}
}

func (lis *mylis) OnStart() {
	conf := xginx.GetConfig()
	file := conf.GetLogFile()
	if *xginx.IsDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = file
	gin.DefaultErrorWriter = file
	go lis.run()
}

func (lis *mylis) OnStop(sig os.Signal) {
	if lis.app != nil {
		lis.app.Close()
	}
	ctx, cancel := context.WithTimeout(lis.ctx, time.Second*5)
	defer cancel()
	err := lis.xhttp.Shutdown(ctx)
	if err != nil {
		xginx.LogError(err)
	}
}

func main() {
	flag.Parse()
	xginx.Run(&mylis{})
}
