module github.com/cxuhua/xmgrs

go 1.13

replace github.com/cxuhua/xginx => ../xginx

require (
	github.com/cxuhua/xginx v0.0.1
	github.com/gin-gonic/gin v1.6.3
	github.com/go-playground/form/v4 v4.1.1
	github.com/go-redis/redis/v7 v7.2.0
	github.com/json-iterator/go v1.1.9
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.3
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
)
