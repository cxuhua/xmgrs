package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/cxuhua/xmgrs/config"

	"github.com/cxuhua/xmgrs/api"

	"github.com/cxuhua/xmgrs/db"

	"github.com/gin-gonic/gin"

	"github.com/cxuhua/xginx"
)

//实现自己的监听器
type mylis struct {
	xginx.Listener
	ctx    context.Context
	cancel context.CancelFunc
	xhttp  *http.Server
	app    *db.App
}

func (lis *mylis) OnLinkBlock(blk *xginx.BlockInfo) {
	//当一个区块连接到链上
}

func (lis *mylis) OnUnlinkBlock(blk *xginx.BlockInfo) {
	//当一个区块从链断开
}

func (lis *mylis) runHttp() {
	lis.ctx, lis.cancel = xginx.GetContext()
	db.RedisURI = config.Redis
	db.MongoURI = config.Mongo
	//创建一个全局连接
	lis.app = db.InitApp(lis.ctx)
	//
	handler := api.InitHandler(lis.ctx)
	lis.xhttp = &http.Server{
		Addr:    config.HttpAddr,
		Handler: handler,
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
	go lis.runHttp()
}

func (lis *mylis) OnStop(sig os.Signal) {
	if lis.app != nil {
		lis.app.Close()
	}
	ctx, cancel := context.WithTimeout(lis.ctx, time.Second*15)
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
