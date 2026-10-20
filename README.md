# go-core

[![Go Version](https://img.shields.io/badge/go-1.27%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

通用 Go 后端组件库，按职责拆分为配置、日志、数据库、鉴权、中间件、对象存储和通用工具等独立包，用于快速搭建服务。

- 模块路径：`github.com/bigbigliu/go-core`
- Go 版本：`1.27+`
- License：MIT
- 设计原则：显式错误返回、安全默认值、按职责拆包、可直接复用

## 目录

- [安装](#安装)
- [Bootstrap（推荐）](#bootstrap推荐)
- [快速开始](#快速开始)
- [配置](#配置)
- [组件总览](#组件总览)
- [使用示例](#使用示例)
  - [日志](#日志)
  - [MySQL](#mysql)
  - [MongoDB](#mongodb)
  - [Redis](#redis)
  - [JWT](#jwt)
  - [Gin 中间件](#gin-中间件)
  - [HTTP 客户端](#http-客户端)
  - [对象存储](#对象存储)
  - [验证码](#验证码)
  - [通用工具](#通用工具)
- [目录结构](#目录结构)
- [设计约定](#设计约定)
- [开发与测试](#开发与测试)
- [改进路线图](#改进路线图)
- [版本升级](#版本升级)
- [License](#license)

## 安装

```bash
go get github.com/bigbigliu/go-core
```

## Bootstrap（推荐）

`bootstrap` 将配置加载、日志初始化、MySQL/Redis/MongoDB 连接和 Gin 标准中间件收敛为一次调用，业务服务不需要在每个项目里重复写这些初始化代码。

```go
package main

import (
	"context"
	"log"

	"github.com/bigbigliu/go-core/bootstrap"
	"github.com/bigbigliu/go-core/response"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.New(ctx, bootstrap.Options{
		ConfigPath:            "./config/settings.yml",
		ServiceName:           "order-service",
		InitMySQL:             true, // 初始化 default + databases 中的所有连接
		InitRedis:             true,
		InitMongo:             false,
		EnableMemoryRateLimit: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = app.Close() }()

	app.Router.GET("/health", func(c *gin.Context) {
		c.JSON(200, response.OK(gin.H{"status": "ok"}))
	})

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

| 选项 | 说明 |
|---|---|
| `ConfigPath` / `Config` | 配置文件路径或直接注入配置 |
| `Host` / `Port` | 覆盖配置中的监听地址 |
| `ServiceName` | 日志中的服务名 |
| `InitMySQL` | 初始化 `db` 与 `databases` 中的所有连接，可通过 `app.MySQL` 获取 Manager |
| `InitRedis` | 初始化 Redis，`app.Redis` |
| `InitMongo` | 初始化 MongoDB，`app.Mongo` |
| `EnableMemoryRateLimit` | 启用默认内存限流 |
| `Timeout` | 请求超时，默认 5 秒 |
| `DisableLogger` / `DisableConsoleLog` | 关闭日志或控制台输出 |

`bootstrap.New` 只初始化显式开启的组件；未使用的服务不会尝试连接。初始化失败时会自动关闭已创建的资源。

## 快速开始

以下是不使用 `bootstrap` 时的手动组装方式：

```go
package main

import (
	"log"
	"strconv"
	"time"

	"github.com/bigbigliu/go-core/config"
	"github.com/bigbigliu/go-core/logger"
	"github.com/bigbigliu/go-core/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.MustLoad("./config/settings.yml")

	if err := logger.InitializeLogger(&logger.CoreLog{
		LogDir:        cfg.Logger.Path,
		LogLevel:      cfg.Logger.Level,
		ConsoleOutPut: true,
		ServiceName:   "demo",
	}); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	router := gin.New()
	router.Use(
		gin.Recovery(),
		middleware.RequestIDMiddleware(),
		middleware.CorsMiddleware(),
		middleware.GinLogger([]string{"/health"}, nil),
		middleware.TimeoutMiddleware(3*time.Second),
	)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := cfg.App.Host + ":" + strconv.Itoa(cfg.App.Port)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}
```

完整可运行示例（基于 `bootstrap`，包含限流、优雅退出、Request ID 日志）见 [examples/basic/main.go](examples/basic/main.go)：

```bash
APP_PORT=8080 go run ./examples/basic
```

## 配置

配置默认按以下顺序查找：

1. `CONFIG` 环境变量指定的文件
2. `./config/settings.yml`
3. `./config/settings.yaml`

```go
cfg := config.MustLoadDefault()
```

也可以显式加载并处理错误：

```go
if err := config.Load("./config/settings.yml"); err != nil {
	log.Fatal(err)
}
cfg := config.GetConfig()
```

示例配置见 [config/settings-dev.yml](config/settings-dev.yml)：

```yaml
app:
  host: 0.0.0.0
  port: 8080
  mode: debug

db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: change-me
  name: core
  max_idle_connections: 10
  max_open_connections: 100

# 可选：任意数量的命名数据库
databases:
  orders:
    host: 127.0.0.1
    port: 3306
    user: root
    password: change-me
    name: orders
    max_idle_connections: 10
    max_open_connections: 100

redis:
  addr: 127.0.0.1
  port: 6379
  pwd:
  db: 0

mongo:
  # uri 非空时优先使用
  uri:
  host: 127.0.0.1
  port: 27017
  username:
  password:
  auth_source:
  database: core
  app_name: go-core
  max_pool_size: 100
  min_pool_size: 0
  connect_timeout: 10
  server_selection_timeout: 10

logger:
  path: ./log
  level: info

jwt:
  secret: change-me
  timeout: 7200
```

| 配置项 | 类型 | 说明 |
|---|---|---|
| `app.host` / `app.port` | string / int | 服务监听地址 |
| `app.mode` | string | Gin 模式，`release` 或 `debug` |
| `db.*` | - | 默认 MySQL 连接信息与连接池 |
| `databases.<name>.*` | map | 任意数量的命名 MySQL 连接，字段与 `db` 相同 |
| `redis.addr` / `redis.port` | string / int | Redis 地址 |
| `redis.pwd` / `redis.db` | string / int | Redis 密码与 DB |
| `mongo.*` | - | MongoDB 连接、认证与连接池 |
| `logger.path` / `logger.level` | string | 日志目录与级别 |
| `jwt.secret` / `jwt.timeout` | string / int | JWT 密钥与过期时间（秒） |

> 配置加载失败时库不会调用 `os.Exit`，由调用方决定如何处理；`GetConfig()` 在未加载成功时返回 `nil`。

可运行示例：[examples/config](examples/config)

## 组件总览

| 包 | 说明 | 常用 API |
|---|---|---|
| [`bootstrap`](bootstrap) | 一站式初始化配置、日志、数据库与 Gin | `New`、`Run`、`Close` |
| [`config`](config) | YAML 配置加载与默认值 | `Load`、`MustLoad`、`LoadDefault`、`GetConfig` |
| [`logger`](logger) | zap + lumberjack 日志 | `InitializeLogger`、`Sync`、`WithContext` |
| [`database/mysql`](database/mysql) | GORM MySQL 连接 | `GenerateDSNParam.InitDB` |
| [`database/mongodb`](database/mongodb) | MongoDB 客户端 | `NewClient`、`Init`、`DB` |
| [`database/redis`](database/redis) | Redis 客户端 | `InitRedisReq.InitRedis` |
| [`auth/jwt`](auth/jwt) | JWT 签发、解析、校验中间件 | `CoreJWT.NewToken`、`ParseToken`、`TokenVerify` |
| [`middleware`](middleware) | CORS、访问日志、Request ID、超时、限流 | `CorsMiddleware`、`GinLogger`、`NewMemoryLimiter` |
| [`httpx`](httpx) | HTTP 客户端 | `New`、`Do`、`Get`、`PostJSON`、`PostForm` |
| [`storage`](storage) | 统一对象存储接口 | `storage.Storage` |
| [`storage/aliyunoss`](storage/aliyunoss) | 阿里云 OSS | `NewClient`、`UploadLocalFile` |
| [`storage/qiniuoss`](storage/qiniuoss) | 七牛云 | `NewClient`、`UploadLocalFile` |
| [`storage/upyunoss`](storage/upyunoss) | 又拍云 | `NewClient`、`UploadLocalFile` |
| [`captcha`](captcha) | 基于 Redis 的验证码存储 | `NewRedisStore` |
| [`idgen`](idgen) | Snowflake ID / UUID | `GenUniversalID`、`NewGenerator`、`UUIDv7`、`UUIDv4` |
| [`netutil`](netutil) | 客户端 IP 解析 | `GetRemoteIP` |
| [`fileutil`](fileutil) | 文件大小与写入 | `IsFileGreaterThan`、`SaveToFile` |
| [`imageutil`](imageutil) | 图片转 Base64 | `EncodeImageToBase64` |
| [`response`](response) | 通用响应结构 | `OK`、`OKWithTotal`、`Fail`、`FailMsg` |
| [`errorutil`](errorutil) | GORM/Redis 错误判断 | `IsNoRowFoundError`、`IsRedisNilError` |
| [`strutil`](strutil) / [`timeutil`](timeutil) | 字符串/时间工具 | `IntsToString`、`String2Time` |
| [`termcolor`](termcolor) | 终端颜色 | `Red`、`Green`、`Bold` |

## 使用示例

### 日志

```go
if err := logger.InitializeLogger(&logger.CoreLog{
	LogDir:        "./log",
	LogLevel:      "info",
	ConsoleOutPut: true,
	ServiceName:   "demo",
}); err != nil {
	log.Fatal(err)
}
defer func() { _ = logger.Sync() }()

logger.Logger.Info("server started")

// 在 Gin handler 中携带 Request ID
logger.Logger.WithOptions(logger.WithContext(c.Request.Context())).
	Info("handle request")
```

`middleware.RequestIDMiddleware()` 会同时写入响应头、`gin.Context` 和 `request.Context()`，因此 `logger.WithContext` 可以取到链路 ID。

可运行示例：[examples/logger](examples/logger)

### MySQL

```go
if err := (&mysql.GenerateDSNParam{
	DbHost:      cfg.Db.Host,
	DbPort:      cfg.Db.Port,
	DbUser:      cfg.Db.User,
	DbPwd:       cfg.Db.Password,
	DbName:      cfg.Db.Name,
	MaxIdleConn: cfg.Db.MaxIdleConnections,
	MaxOpenConn: cfg.Db.MaxOpenConnections,
}).InitDB(); err != nil {
	log.Fatal(err)
}

// 全局客户端
db := mysql.DBClient
```

DSN 包含密码，不要直接输出到日志；需要日志时使用 `SafeDSN()`。

#### 连接多个 MySQL 数据库

`mysql.Manager` 按名称管理多个连接，适合分库或读写分离。连接数量不受限制，`db` 只是兼容单库的默认配置：

```go
manager := mysql.NewManager()
defer func() { _ = manager.CloseAll() }()

primary, err := manager.Open("primary", &mysql.Config{
	DbHost: cfg.Db.Host, DbPort: cfg.Db.Port,
	DbUser: cfg.Db.User, DbPwd: cfg.Db.Password, DbName: cfg.Db.Name,
	MaxIdleConn: cfg.Db.MaxIdleConnections, MaxOpenConn: cfg.Db.MaxOpenConnections,
})
if err != nil {
	log.Fatal(err)
}

// cfg.Databases 可以是任意数量
for name, dbCfg := range cfg.Databases {
	_, err := manager.Open(name, &mysql.Config{
		DbHost: dbCfg.Host, DbPort: dbCfg.Port,
		DbUser: dbCfg.User, DbPwd: dbCfg.Password, DbName: dbCfg.Name,
		MaxIdleConn: dbCfg.MaxIdleConnections, MaxOpenConn: dbCfg.MaxOpenConnections,
	})
	if err != nil {
		log.Fatal(err)
	}
}

orders, err := manager.DB("orders")
_ = primary
_ = orders
```

也可以直接调用 `(&mysql.Config{...}).Open()` 创建连接而不修改全局 `mysql.DBClient`。

可运行示例：[examples/mysql](examples/mysql)

### MongoDB

```go
client, err := mongodb.NewClient(ctx, mongodb.Config{
	Host:                   cfg.Mongo.Host,
	Port:                   cfg.Mongo.Port,
	Username:               cfg.Mongo.Username,
	Password:               cfg.Mongo.Password,
	AuthSource:             cfg.Mongo.AuthSource,
	Database:               cfg.Mongo.Database,
	AppName:                cfg.Mongo.AppName,
	MaxPoolSize:            cfg.Mongo.MaxPoolSize,
	MinPoolSize:            cfg.Mongo.MinPoolSize,
	ConnectTimeout:         time.Duration(cfg.Mongo.ConnectTimeout) * time.Second,
	ServerSelectionTimeout: time.Duration(cfg.Mongo.ServerSelectionTimeout) * time.Second,
})
if err != nil {
	log.Fatal(err)
}
defer func() { _ = client.Close(context.Background()) }()

db, err := client.Database()            // 配置中的默认数据库
orders, err := client.Database("orders") // 同一实例下的其他数据库
users := db.Collection("users")
_ = orders
_ = users
```

使用全局客户端：

```go
if err := mongodb.Init(ctx, mongoConfig); err != nil {
	log.Fatal(err)
}
defer func() { _ = mongodb.Close(context.Background()) }()

db, err := mongodb.DB()          // 默认数据库
logs, err := mongodb.DB("logs")  // 指定数据库
```

可运行示例：[examples/mongodb](examples/mongodb)

### Redis

```go
if err := (&redis.InitRedisReq{
	Addr: cfg.Redis.Addr,
	Port: cfg.Redis.Port,
	Pwd:  cfg.Redis.Pwd,
	Db:   cfg.Redis.Db,
}).InitRedis(); err != nil {
	log.Fatal(err)
}

client := redis.Redisclient
```

可运行示例：[examples/redis](examples/redis)

### JWT

```go
coreJWT := &jwt.CoreJWT{
	Secret:  cfg.Jwt.Secret,
	Timeout: cfg.Jwt.Timeout, // 秒
}

token, err := coreJWT.NewToken("alice")

api := router.Group("/api")
api.Use(coreJWT.TokenVerify())
```

`TokenVerify` 会校验：

- `Authorization: Bearer <token>`
- HS256 签名与过期时间
- Redis 中 `coreos:<username>:accesstoken` 是否与请求 token 一致

可运行示例：[examples/jwt](examples/jwt)

### Gin 中间件

推荐顺序：

```go
limiter := middleware.NewMemoryLimiter(100, time.Minute)
defer limiter.Close()

router.Use(
	gin.Recovery(),
	middleware.RequestIDMiddleware(),
	middleware.CorsMiddleware(),
	middleware.GinLogger([]string{"/health"}, nil),
	limiter.Middleware(),
	middleware.TimeoutMiddleware(5*time.Second),
)
```

- CORS：默认允许任意 Origin 但不允许携带凭证；需要 Cookie 时使用 `CorsMiddlewareWithConfig(DefaultCorsConfig())` 配置明确白名单并开启 `AllowCredentials`。
- 内存限流：固定窗口，使用完调用 `Close()` 释放清理 goroutine。
- Redis 限流：滑动窗口，适合多实例部署。
- 超时中间件不会自动取消业务 handler，业务内部仍应使用带超时的 context。

```go
// Redis 滑动窗口限流
router.Use(middleware.IPFilterWithRedisMiddleware(redis.Redisclient, 100, time.Minute))
```

可运行示例：[examples/basic](examples/basic)（完整 Gin 服务与优雅退出）

### HTTP 客户端

```go
client := httpx.New(
	httpx.WithTimeout(5*time.Second),
	httpx.WithRetry(2),               // 仅重试幂等请求
	httpx.WithMaxResponseSize(2<<20), // 2 MiB
	httpx.WithUserAgent("demo/1.0"),
)

body, status, err := client.Get(ctx, url, map[string]string{
	"Authorization": "Bearer <token>",
})

body, status, err = client.PostJSON(ctx, url, nil, payload)
body, status, err = client.PostForm(ctx, url, nil, form)
```

默认行为：

- 30 秒超时，复用连接池
- 响应体上限 10 MiB
- 幂等请求（GET/HEAD/OPTIONS/PUT/DELETE）对网络错误、429、5xx 自动重试
- 默认 `User-Agent: go-core/httpx`
- 默认校验 HTTPS 证书；仅测试环境可显式使用 `httpx.New(httpx.WithInsecureSkipVerify())`

可运行示例：[examples/httpx](examples/httpx)

### 对象存储

直接使用某家实现：

```go
uploader := &aliyunoss.AliyunOssUpload{
	AccessKeyId:     "ak",
	AccessKeySecret: "sk",
	Endpoint:        "https://oss-cn-hangzhou.aliyuncs.com", // 可省略
}

key, err := uploader.UploadLocalFile(&aliyunoss.UploadLocalFileReq{
	LocalPath: "./demo.png",
	SavePath:  "images",
	FileName:  "demo.png",
	Bucket:    "my-bucket",
})
```

使用统一接口：

```go
store, err := aliyunoss.NewClient(uploader, "my-bucket")
if err != nil {
	log.Fatal(err)
}

_, _ = store.Put(ctx, "images/demo.png", reader, size)
_ = store.Delete(ctx, "images/demo.png")
```

- `NewClient` 会校验 uploader 和 bucket，失败返回 error。
- `Put` 的 `size` 为内容字节数；未知时传 `-1`。七牛在 `-1` 时返回 `storage.ErrSizeRequired`。
- 阿里云、又拍云会缓存底层客户端，按 bucket 复用。

可运行示例：[examples/storage](examples/storage)

### 验证码

```go
store := captcha.NewRedisStore(redis.Redisclient, time.Minute)

// 配合 github.com/mojocn/base64Captcha 使用
cap := base64Captcha.NewCaptcha(driver, store)
id, b64s, answer, err := cap.Generate()
_ = cap.Verify(id, answer, true)
```

可运行示例：[examples/captcha](examples/captcha)

### 通用工具

```go
// 默认 Snowflake 数字 ID（时间有序、适合数据库主键）
id := idgen.GenUniversalID()

// 分布式部署时指定唯一节点号（0..1023）
_ = idgen.SetDefaultNodeID(1)

// 或者创建独立 Generator
generator, _ := idgen.NewGenerator(2)
id2 := generator.NextID()

// 不希望维护节点号时，可使用时间有序的 UUID v7
uuid7, _ := idgen.UUIDv7()

// 随机 UUID v4
uuid4 := idgen.UUIDv4()

ip := netutil.GetRemoteIP(c)

ok, err := fileutil.IsFileGreaterThan("./video.mp4", 300) // 是否大于 300MB
b64, err := imageutil.EncodeImageToBase64("./avatar.png")

c.JSON(200, response.OK(data))
c.JSON(200, response.OKWithTotal(list, total))
c.JSON(400, response.FailMsg("参数错误"))
```

> `netutil.GetRemoteIP` 会读取 `X-Forwarded-For` / `X-Real-IP`，仅在可信代理之后使用；更推荐配置 Gin 的 `SetTrustedProxies` 后直接使用 `c.ClientIP()`。

可运行示例：[examples/tools](examples/tools)、[examples/imageutil](examples/imageutil)

## 目录结构

```text
go-core
├── auth/jwt               # JWT 签发、解析、Gin 校验中间件
├── bootstrap              # 一站式初始化：配置/日志/数据库/Gin
├── captcha                # 基于 Redis 的验证码存储
├── config                 # 配置结构体与加载
├── database
│   ├── mongodb            # MongoDB 客户端
│   ├── mysql              # GORM 连接、多库管理、基础模型
│   └── redis              # Redis 客户端
├── doc.go                 # 模块根包文档
├── errorutil              # 错误判断辅助
├── examples
│   ├── basic              # Gin 服务综合示例
│   ├── captcha            # 验证码
│   ├── config             # 配置
│   ├── httpx              # HTTP 客户端
│   ├── imageutil          # 图片转 Base64
│   ├── jwt                # JWT
│   ├── logger             # 日志
│   ├── mongodb            # MongoDB
│   ├── mysql              # MySQL 多库
│   ├── redis              # Redis
│   ├── storage            # 对象存储
│   └── tools              # ID/IP/文件/字符串/时间工具
├── fileutil               # 文件操作
├── httpx                  # HTTP 客户端
├── idgen                  # Snowflake ID / UUID
├── imageutil              # 图片转 Base64
├── logger                 # zap 日志、GORM logger
├── middleware             # CORS、访问日志、RequestID、超时、限流
├── netutil                # 客户端 IP 解析
├── response               # 通用响应结构
├── storage
│   ├── storage.go         # 统一 Storage 接口
│   ├── aliyunoss          # 阿里云 OSS
│   ├── qiniuoss           # 七牛云
│   └── upyunoss           # 又拍云
├── strutil                # 字符串工具
├── termcolor              # 终端颜色
└── timeutil               # 时间工具
```

## 设计约定

- **显式错误**：初始化、加载、网络操作均返回 `error`，库内部不调用 `os.Exit`。
- **context 优先**：HTTP 客户端和统一存储接口首参为 `context.Context`；不支持 context 的第三方 SDK 会在接口层接受并忽略。
- **安全默认值**：默认校验 HTTPS 证书、CORS 默认不允许携带凭证、Redis 限流使用原子 Lua 脚本。
- **统一响应**：`response.ResultInfo` 结构为 `{code, msg, total, data}`，中间件默认使用 `code: "-1"` 表示失败。
- **可测试性**：核心逻辑拆分为可注入依赖的函数/类型，Redis 相关组件通过传入 client 使用。

## 开发与测试

```bash
make fmt      # gofmt + goimports
make vet      # go vet
make test     # go test ./...
make race     # go test -race ./...
make lint     # golangci-lint（需支持 Go 1.27，建议 v2.15+）
```

CI 会在 Go 1.27 上执行格式检查、构建、vet、`go test -race` 和 golangci-lint。

## 改进路线图

版本化、依赖隔离、可观测性、错误模型、数据库/缓存增强等改进项与优先级见：

- [docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md)

## 版本升级

包路径已按职责重新组织：

| 旧路径 | 新路径 |
|---|---|
| `pkgs/httputil` | `httpx` |
| `pkgs/captcha` | `captcha` |
| `pkgs/imageutil` | `imageutil` |
| `pkgs/storage/aliyunOss` | `storage/aliyunoss` |
| `pkgs/storage/qiniuOss` | `storage/qiniuoss` |
| `pkgs/storage/upyunOss` | `storage/upyunoss` |
| `pkgs`（ID/IP/文件等） | `idgen` / `netutil` / `fileutil` / `strutil` / `timeutil` / `response` / `errorutil` / `termcolor` |
| `web/jwt_token` | `auth/jwt` |
| `web/web_middleware` | `middleware` |

行为变更：

- `logger.InitializeLogger`、`mysql.InitDB`、`redis.InitRedis` 返回 `error`，不再退出进程
- `config` 不再在包初始化时加载或退出；使用 `Load` / `MustLoad` / `LoadDefault`
- `DbConf` 数值字段改为 `int`，YAML 使用 `max_idle_connections` / `max_open_connections`
- `go-redis/redis/v8` 升级为 `redis/go-redis/v9`
- 移除 `satori/go.uuid`，`GenUniversalId` 改为 `GenUniversalID`（旧名保留为 Deprecated 别名）
- `httpx` 移除 `IHttpClient`/`HttpClientOption`，改为 `New(Options...)` + `Do/Get/PostJSON/PostForm`
- 对象存储 `NewClient` 返回 `(*Client, error)`，并校验 uploader/bucket
- `IPFilterMiddleware` 仍可用；需要释放资源时使用 `NewMemoryLimiter` + `Close`

## License

[MIT](LICENSE)
