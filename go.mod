module github.com/cxuhua/xmgrs

go 1.13

replace github.com/cxuhua/xginx v0.0.1 => ../xginx

replace github.com/cxuhua/gopher-lua v1.0.1 => ../gopher-lua

require (
	github.com/bsm/redislock v0.5.0
	github.com/cxuhua/xginx v0.0.1
	github.com/gin-gonic/gin v1.6.3
	github.com/go-playground/form/v4 v4.1.1
	github.com/go-playground/validator/v10 v10.3.0
	github.com/go-redis/redis/v7 v7.4.0
	github.com/hashicorp/go-memdb v1.2.1
	github.com/json-iterator/go v1.1.10
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.4
)
