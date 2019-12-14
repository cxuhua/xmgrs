package main

import (
	"context"
	"flag"
	"net/http"
	"os"

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
}

func (lis *mylis) runHttp() {
	lis.ctx, lis.cancel = context.WithCancel(context.Background())
	defer lis.cancel()

	m := gin.Default()

	app := db.InitApp(lis.ctx)

	m.Use(db.AppHandler(app))

	lis.xhttp = &http.Server{
		Addr:    ":9334",
		Handler: m,
	}
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
	err := lis.xhttp.Shutdown(lis.ctx)
	if err != nil {
		xginx.LogError(err)
	}
}

func main() {
	flag.Parse()
	xginx.Run(&mylis{})
}
