module github.com/liujitcn/kratos-admin/backend

go 1.26.5

require (
	github.com/go-kratos/kratos/v3 v3.0.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/liujitcn/go-utils v0.0.35
	github.com/liujitcn/go-utils/crypto v0.0.12
	github.com/liujitcn/go-utils/http v0.0.5
	github.com/liujitcn/go-utils/translator v0.0.2
	github.com/liujitcn/gorm-kit v0.0.32
	github.com/liujitcn/kratos-core v0.0.2
	github.com/liujitcn/kratos-kit v0.0.66
	github.com/liujitcn/kratos-kit/api v0.0.26
	github.com/liujitcn/kratos-kit/auth v0.0.23
	github.com/liujitcn/kratos-kit/auth/authn v0.0.21
	github.com/liujitcn/kratos-kit/auth/authz v0.0.20
	github.com/liujitcn/kratos-kit/auth/authz/engine/casbin v0.0.18
	github.com/liujitcn/kratos-kit/bootstrap v0.0.18
	github.com/liujitcn/kratos-kit/captcha v0.0.19
	github.com/liujitcn/kratos-kit/database/gorm v0.0.38
	github.com/liujitcn/kratos-kit/database/gorm/migration v0.0.11
	github.com/liujitcn/kratos-kit/oauth v0.0.8
	github.com/liujitcn/kratos-kit/oss v0.0.15
	github.com/liujitcn/kratos-kit/transport/mcp v0.0.12
	github.com/liujitcn/kratos-kit/transport/sse v0.0.10
	github.com/liujitcn/kratos-kit/utils v0.0.17
	github.com/modelcontextprotocol/go-sdk v1.6.0
	github.com/redis/go-redis/v9 v9.19.0
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gen v0.3.27
	gorm.io/gorm v1.31.2
	gorm.io/plugin/dbresolver v1.6.2
	gorm.io/plugin/soft_delete v1.2.1
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
	gorm.io/datatypes v1.2.4 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/hints v1.1.0 // indirect
)

replace github.com/liujitcn/kratos-core => ../../kratos-core_bak
