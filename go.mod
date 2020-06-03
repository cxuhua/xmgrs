module github.com/cxuhua/xmgrs

go 1.13

replace github.com/cxuhua/xginx v0.0.1 => ../xginx

replace github.com/cxuhua/gopher-lua v1.0.1 => ../gopher-lua

require (
	github.com/cxuhua/xginx v0.0.1
	github.com/gin-gonic/gin v1.6.3
	github.com/go-playground/form/v4 v4.1.1
	github.com/go-playground/validator/v10 v10.2.0
	github.com/go-redis/redis/v7 v7.2.0
	github.com/json-iterator/go v1.1.9
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.0
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.3
)
