module github.com/cxuhua/xmgrs

go 1.13

require (
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/cxuhua/xginx v0.1.0
	github.com/gin-gonic/gin v1.5.0
	github.com/go-redis/redis/v7 v7.0.0-beta.4
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/json-iterator/go v1.1.7
	github.com/pkg/errors v0.8.1 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/vmihailenco/taskq/v2 v2.2.3
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.2.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	gopkg.in/go-playground/validator.v9 v9.29.1
)

replace github.com/cxuhua/xginx => ../xginx
