# 芋道商城 Go 版本 - 深度学习指南

> 本文档旨在帮助开发者快速理解项目架构、核心流程和配置机制，为项目扩展和维护奠定基础。

## 📚 文档导航

- [项目概览](#项目概览)
- [架构设计](#架构设计)
- [核心流程](#核心流程)
- [配置机制](#配置机制)
- [依赖注入](#依赖注入)
- [关键模块深度解析](#关键模块深度解析)
- [开发实践](#开发实践)
- [常见问题](#常见问题)

---

## 项目概览

### 项目定位

**芋道商城 Go 版本** 是 Java 版本 `ruoyi-vue-pro` 的 Go 语言实现，采用 **Clean Architecture** 设计原则，提供企业级电商 API 服务。

### 核心特点

| 特点 | 说明 |
|------|------|
| **高度对齐** | 97% 与 Java 版本 API 兼容，确保无缝迁移 |
| **Clean Architecture** | 清晰的四层架构（Handler → Service → Repository → Database） |
| **类型安全** | 使用 GORM Gen 生成类型安全的 DAO 代码 |
| **完善权限** | JWT + RBAC + 租户隔离三层权限体系 |
| **业务完整** | 会员、商品、交易、支付、促销等全业务链 |

### 技术栈

```
语言框架：Go 1.26.6 + Gin 1.11.0
数据访问：GORM 1.25.12 + GORM Gen 0.3.27
数据库：MySQL 8.0+ + Redis 6.0+
依赖注入：Google Wire 0.7.0
日志管理：Zap 1.27.1 + Lumberjack
配置管理：Viper 1.21.0
认证授权：JWT + OAuth2
```

---

## 架构设计

### 1. 分层架构（Clean Architecture）

```
┌─────────────────────────────────────────────────────────┐
│                   HTTP Request                          │
└────────────────────────┬────────────────────────────────┘
                         │
┌─────────────────────────▼────────────────────────────────┐
│              Handler Layer (API 层)                      │
│  • 请求参数绑定与验证                                   │
│  • 调用 Service 处理业务逻辑                            │
│  • 返回统一格式的响应                                   │
└────────────────────────┬────────────────────────────────┘
                         │
┌─────────────────────────▼────────────────────────────────┐
│              Service Layer (业务层)                      │
│  • 核心业务逻辑实现                                     │
│  • 事务管理                                             │
│  • 跨模块业务协调                                       │
│  • 缓存策略实现                                         │
└────────────────────────┬────────────────────────────────┘
                         │
┌─────────────────────────▼────────────────────────────────┐
│            Repository Layer (数据访问层)                │
│  • GORM 数据库操作                                      │
│  • Redis 缓存操作                                       │
│  • 数据查询、保存、删除                                 │
└────────────────────────┬────────────────────────────────┘
                         │
┌─────────────────────────▼────────────────────────────────┐
│            Database Layer (数据存储层)                  │
│  • MySQL 关系数据库                                     │
│  • Redis 缓存存储                                       │
└─────────────────────────────────────────────────────────┘
```

### 2. 项目目录结构

```
yudao-backend-go/
├── cmd/                              # 应用入口
│   ├── server/
│   │   ├── main.go                  # 启动文件：初始化配置、日志、数据库、Redis
│   │   ├── wire.go                  # Wire 配置：定义依赖注入规则
│   │   └── wire_gen.go              # Wire 生成的代码（自动生成）
│   └── gen/
│       └── generate.go              # GORM Gen 代码生成器
│
├── config/                          # 配置文件
│   └── config.local.yaml            # 本地配置（数据库、Redis、日志等）
│
├── internal/                        # 内部代码（不对外暴露）
│   ├── api/                         # HTTP API 层
│   │   ├── handler/                # 请求处理器（业务逻辑入口）
│   │   │   ├── admin/              # 后台管理 API
│   │   │   │   ├── member/         # 会员管理
│   │   │   │   ├── pay/            # 支付管理
│   │   │   │   ├── product/        # 商品管理
│   │   │   │   ├── promotion/      # 促销管理
│   │   │   │   └── trade/          # 交易管理
│   │   │   ├── app/                # 移动端/用户端 API
│   │   │   │   ├── member/
│   │   │   │   ├── product/
│   │   │   │   ├── promotion/
│   │   │   │   └── trade/
│   │   │   ├── auth.go             # 认证处理
│   │   │   ├── user.go             # 用户管理处理
│   │   │   └── ...                 # 其他系统模块处理
│   │   ├── req/                    # 请求对象 (VO - Value Object)
│   │   │   └── *.go                # 各模块请求参数定义
│   │   ├── resp/                   # 响应对象 (VO)
│   │   │   └── *.go                # 各模块响应数据定义
│   │   └── router/                 # 路由定义
│   │       ├── router.go           # 主路由初始化
│   │       └── ...                 # 各模块路由注册
│   │
│   ├── middleware/                 # 中间件
│   │   ├── auth.go                # JWT 认证中间件
│   │   ├── error.go               # 错误处理中间件
│   │   ├── recovery.go            # Panic 恢复中间件
│   │   ├── apilog.go              # API 访问日志中间件
│   │   └── validator.go           # 参数验证中间件
│   │
│   ├── model/                     # 数据模型 (DO - Data Object)
│   │   ├── member/                # 会员模块数据模型
│   │   ├── pay/                   # 支付模块数据模型
│   │   ├── product/               # 商品模块数据模型
│   │   ├── promotion/             # 促销模块数据模型
│   │   ├── trade/                 # 交易模块数据模型
│   │   ├── system_*.go            # 系统模块数据模型
│   │   └── types.go               # 通用类型定义
│   │
│   ├── service/                   # 业务服务层
│   │   ├── member/                # 会员服务
│   │   ├── pay/                   # 支付服务
│   │   ├── product/               # 商品服务
│   │   ├── promotion/             # 促销服务
│   │   ├── trade/                 # 交易服务
│   │   ├── auth.go                # 认证服务
│   │   ├── user.go                # 用户服务
│   │   └── ...                    # 其他系统服务
│   │
│   ├── repo/                      # 数据访问层 (Repository)
│   │   ├── query/                # GORM Gen 生成的查询代码
│   │   │   └── *.go              # 自动生成的 DAO 代码
│   │   └── *.go                  # 自定义 Repository 实现
│   │
│   └── pkg/                       # 内部工具包
│       ├── core/                  # 核心包
│       │   ├── context.go         # 上下文管理（用户信息）
│       │   ├── result.go          # 统一响应结果
│       │   ├── error.go           # 错误码定义
│       │   ├── db.go              # 数据库初始化
│       │   ├── redis.go           # Redis 初始化
│       │   ├── page.go            # 分页工具
│       │   └── consts.go          # 常量定义
│       ├── utils/                 # 工具函数
│       │   ├── jwt.go             # JWT 生成和解析
│       │   ├── pwd.go             # 密码加密和验证
│       │   ├── date.go            # 日期时间工具
│       │   └── ...
│       ├── file/                  # 文件处理
│       ├── excel/                 # Excel 操作
│       ├── area/                  # 地区数据
│       ├── statistics/            # 统计工具
│       └── websocket/             # WebSocket 管理
│
├── pkg/                           # 公共包（可对外暴露）
│   ├── config/                    # 配置管理
│   │   └── config.go              # 配置加载和结构定义
│   └── logger/                    # 日志管理
│       └── logger.go              # 日志初始化和使用
│
├── logs/                          # 日志文件输出目录
├── Makefile                       # 构建脚本
├── README.md                      # 项目说明
└── LEARNING_GUIDE.md              # 本文档
```

### 3. 路由设计

```
/
├── /admin-api/                    # 后台管理 API
│   ├── /system/                  # 系统管理
│   │   ├── /auth/               # 认证（登录、登出）
│   │   ├── /user/               # 用户管理
│   │   ├── /role/               # 角色管理
│   │   ├── /menu/               # 菜单管理
│   │   ├── /dept/               # 部门管理
│   │   └── ...
│   ├── /member/                 # 会员管理
│   ├── /product/                # 商品管理
│   ├── /trade/                  # 交易管理
│   ├── /pay/                    # 支付管理
│   └── /promotion/              # 促销管理
│
└── /app-api/                     # 移动端/用户端 API
    ├── /member/                 # 会员中心
    ├── /product/                # 商品中心
    ├── /trade/                  # 交易中心
    └── /promotion/              # 营销中心
```

---

## 核心流程

### 1. 项目启动流程

```
main.go 启动
    ↓
1. 加载配置文件 (config.local.yaml)
    ↓
2. 初始化日志系统 (Zap + Lumberjack)
    ↓
3. 初始化地区数据 (area.csv)
    ↓
4. 通过 Wire 初始化应用
    ├─ 初始化数据库连接 (MySQL)
    ├─ 初始化 Redis 连接
    ├─ 初始化所有 Repository
    ├─ 初始化所有 Service
    └─ 初始化所有 Handler
    ↓
5. 注册路由和中间件
    ├─ 注册系统路由
    ├─ 注册业务路由
    └─ 注册地区路由
    ↓
6. 启动 Gin 服务器 (监听指定端口)
    ↓
服务就绪，接收请求
```

**关键代码** (`cmd/server/main.go`)：

```go
func main() {
    // 1. 初始化配置
    if err := config.Load(); err != nil {
        panic(err)
    }
    
    // 2. 初始化日志
    logger.Init()
    
    // 3. 初始化地区数据
    if err := area.Init("configs/area.csv"); err != nil {
        logger.Log.Warn("Failed to init area data", zap.Error(err))
    }
    
    // 4. 通过 Wire 初始化应用（自动注入依赖）
    r, err := InitApp()
    if err != nil {
        logger.Log.Fatal("failed to init app", zap.Error(err))
    }
    
    // 5. 注册地区路由
    areaHandler := handler.NewAreaHandler()
    router.RegisterAreaRoutes(r, areaHandler)
    
    // 6. 启动服务
    addr := config.C.HTTP.Port
    logger.Info("Server starting...", zap.String("addr", addr))
    if err := r.Run(addr); err != nil {
        logger.Log.Fatal("failed to start server", zap.Error(err))
    }
}
```

### 2. 请求处理流程

```
HTTP Request
    ↓
Gin Router 匹配路由
    ↓
中间件链执行
├─ Recovery 中间件 (捕获 Panic)
├─ ErrorHandler 中间件 (统一错误处理)
├─ Auth 中间件 (JWT 认证)
├─ APIAccessLog 中间件 (记录访问日志)
└─ Validator 中间件 (参数验证)
    ↓
Handler 处理请求
├─ 绑定请求参数 (c.ShouldBindJSON)
├─ 参数验证 (Validator)
├─ 调用 Service 处理业务逻辑
└─ 返回响应
    ↓
Service 执行业务逻辑
├─ 调用 Repository 查询数据
├─ 业务规则检查
├─ 数据处理和转换
├─ 调用其他 Service 协调
└─ 返回结果
    ↓
Repository 执行数据操作
├─ GORM 查询数据库
├─ Redis 缓存操作
└─ 返回数据
    ↓
Handler 返回统一格式响应
    ↓
HTTP Response
```

### 3. 认证与授权流程

```
用户登录请求
    ↓
AuthHandler.Login()
    ↓
AuthService.Login()
├─ 验证用户名和密码
├─ 检查用户状态
└─ 生成 JWT Token
    ↓
Token 存储到 Redis (白名单)
    ↓
返回 Token 给客户端
    ↓
客户端在后续请求中携带 Token
    ↓
Auth 中间件验证
├─ 从请求头/参数中获取 Token
├─ 验证 JWT 签名和有效期
├─ 检查 Redis 白名单
└─ 提取用户信息到上下文
    ↓
Handler 通过 core.GetLoginUser(c) 获取用户信息
    ↓
业务逻辑执行
```

**用户信息获取方式**：

```go
// 在 Handler 或 Service 中获取登录用户信息
loginUser := core.GetLoginUser(c)
if loginUser == nil {
    // 未登录
    return
}

// 访问用户信息
userID := loginUser.UserID          // 用户 ID
userType := loginUser.UserType      // 用户类型 (0=Member, 1=Admin)
tenantID := loginUser.TenantID      // 租户 ID
nickname := loginUser.Nickname      // 用户昵称
```

### 4. 数据库操作流程

```
Service 需要查询数据
    ↓
调用 Repository (GORM Gen 生成的 Query)
    ↓
GORM 构建 SQL 语句
├─ 参数绑定（防止 SQL 注入）
├─ 条件拼接
└─ 排序、分页等
    ↓
执行 SQL 查询
    ↓
MySQL 返回结果
    ↓
GORM 将结果映射到 Model
    ↓
Repository 返回数据
    ↓
Service 处理数据
    ↓
Handler 返回响应
```

**GORM Gen 使用示例**：

```go
// 在 Service 中使用 GORM Gen 生成的代码
func (s *UserService) GetUserByID(ctx context.Context, id int64) (*model.SystemUser, error) {
    u := s.q.SystemUser  // 获取 User 表的查询对象
    
    // 使用类型安全的查询
    user, err := u.WithContext(ctx).
        Where(u.ID.Eq(id)).
        First()
    
    if err != nil {
        return nil, err
    }
    return user, nil
}

// 分页查询
func (s *UserService) GetUserPage(ctx context.Context, pageNo, pageSize int) ([]model.SystemUser, int64, error) {
    u := s.q.SystemUser
    
    count, err := u.WithContext(ctx).Count()
    if err != nil {
        return nil, 0, err
    }
    
    users, err := u.WithContext(ctx).
        Offset(int((pageNo - 1) * pageSize)).
        Limit(pageSize).
        Find()
    
    return users, count, err
}
```

---

## 配置机制

### 1. 配置文件结构

配置文件位置：`config/config.local.yaml`

```yaml
# 应用配置
app:
  name: "yudao-backend-go"      # 应用名称
  env: "local"                  # 运行环境: local/dev/prod

# HTTP 服务配置
http:
  port: ":48080"                # 服务端口
  mode: "debug"                 # Gin 模式: debug/release

# 日志配置
log:
  level: "debug"                # 日志级别: debug/info/warn/error
  filename: "logs/app.log"      # 日志文件路径
  max_size: 100                 # 单个文件最大大小 (MB)
  max_age: 7                    # 文件保留天数
  max_backups: 10               # 保留文件数量

# MySQL 数据库配置
mysql:
  dsn: "user:password@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle: 10                  # 最大空闲连接数
  max_open: 100                 # 最大打开连接数
  max_lifetime: 3600            # 连接最大存活时间 (秒)

# Redis 缓存配置
redis:
  addr: "localhost:6379"        # Redis 地址
  password: ""                  # Redis 密码
  db: 0                         # Redis 数据库编号

# 业务配置示例
trade:
  express:
    client: "kd100"             # 快递查询客户端
    kd100:
      customer: "xxx"           # 快递100客户ID
      key: "xxx"                # 快递100密钥
```

### 2. 配置加载机制

**配置加载流程**：

```go
// pkg/config/config.go
type Config struct {
    App    AppConfig
    HTTP   HTTPConfig
    Log    LogConfig
    MySQL  MySQLConfig
    Redis  RedisConfig
    Trade  TradeConfig
    // ... 其他配置
}

var C *Config  // 全局配置对象

func Load() error {
    // 1. 使用 Viper 加载配置文件
    viper.SetConfigName("config.local")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("config/")
    
    // 2. 读取配置文件
    if err := viper.ReadInConfig(); err != nil {
        return err
    }
    
    // 3. 解析到结构体
    if err := viper.Unmarshal(&C); err != nil {
        return err
    }
    
    // 4. 环境变量覆盖（可选）
    // viper.BindEnv("mysql.dsn", "MYSQL_DSN")
    
    return nil
}
```

### 3. 环境变量覆盖

配置项支持通过环境变量覆盖：

```bash
# 设置环境变量
export HTTP_PORT=:18080
export MYSQL_DSN=user:pass@tcp(db:3306)/yudao
export REDIS_ADDR=redis:6379
export LOG_LEVEL=info

# 启动应用时会自动读取这些环境变量
go run cmd/server/main.go
```

### 4. 数据库初始化

**MySQL 连接初始化** (`internal/pkg/core/db.go`)：

```go
func InitDB() *gorm.DB {
    cfg := config.C.MySQL
    
    // 创建 GORM Logger
    newLogger := gormlogger.New(
        ZapGormWriter{},
        gormlogger.Config{
            SlowThreshold:             200 * time.Millisecond,
            LogLevel:                  gormlogger.Info,
            IgnoreRecordNotFoundError: true,
            Colorful:                  false,
        },
    )
    
    // 连接数据库
    db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
        Logger: newLogger,
    })
    if err != nil {
        logger.Log.Fatal("failed to connect database", zap.Error(err))
    }
    
    // 配置连接池
    sqlDB, _ := db.DB()
    sqlDB.SetMaxIdleConns(cfg.MaxIdle)      // 最大空闲连接
    sqlDB.SetMaxOpenConns(cfg.MaxOpen)      // 最大打开连接
    sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
    
    DB = db
    return db
}
```

### 5. Redis 初始化

**Redis 连接初始化** (`internal/pkg/core/redis.go`)：

```go
func InitRedis() *redis.Client {
    cfg := config.C.Redis
    
    // 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })
    
    // 测试连接
    if err := rdb.Ping(context.Background()).Err(); err != nil {
        logger.Log.Fatal("failed to connect redis", zap.Error(err))
    }
    
    RDB = rdb
    return rdb
}
```

---

## 依赖注入

### 1. Wire 依赖注入框架

项目使用 **Google Wire** 实现依赖注入，自动管理对象的创建和依赖关系。

### 2. Wire 配置文件

**文件位置**：`cmd/server/wire.go`

```go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/google/wire"
    // ... 导入所有需要注入的包
)

// InitApp 初始化应用
func InitApp() (*gin.Engine, error) {
    wire.Build(
        // 1. 配置和日志层
        config.Module,
        logger.Module,
        
        // 2. 核心基础设施
        core.InitDB,
        core.InitRedis,
        
        // 3. Repository 层
        repository.Module,
        
        // 4. Service 层
        service.Module,
        
        // 5. Handler 层
        handler.Module,
        
        // 6. 路由初始化
        router.InitRouter,
    )
    return nil, nil
}
```

### 3. 依赖注入流程

```
Wire 分析代码
    ↓
识别所有 Provider（提供者）
├─ 构造函数 (NewXxx)
├─ 全局变量
└─ 接口实现
    ↓
构建依赖关系图
├─ 分析函数参数
├─ 匹配提供者
└─ 检测循环依赖
    ↓
生成初始化代码 (wire_gen.go)
    ↓
按依赖顺序初始化对象
├─ 初始化基础设施 (DB, Redis)
├─ 初始化 Repository
├─ 初始化 Service
├─ 初始化 Handler
└─ 初始化路由
    ↓
返回初始化完成的应用
```

### 4. 如何添加新的依赖注入

**步骤 1**：创建构造函数

```go
// internal/service/my_service.go
type MyService struct {
    repo *query.Query
}

func NewMyService(repo *query.Query) *MyService {
    return &MyService{
        repo: repo,
    }
}
```

**步骤 2**：在 Module 中注册

```go
// internal/service/module.go
var Module = wire.NewSet(
    NewMyService,
    // ... 其他 Service
)
```

**步骤 3**：重新生成 Wire 代码

```bash
make wire
# 或
go run github.com/google/wire/cmd/wire@latest ./cmd/server
```

---

## 关键模块深度解析

### 1. 认证模块 (Auth Module)

#### 1.1 认证流程

```
用户登录
    ↓
POST /admin-api/system/auth/login
    ↓
AuthHandler.Login()
    ├─ 绑定请求参数 (username, password)
    ├─ 验证参数
    └─ 调用 AuthService.Login()
    ↓
AuthService.Login()
    ├─ 查询用户 (Repository)
    ├─ 验证密码 (utils.CheckPassword)
    ├─ 检查用户状态
    ├─ 生成 JWT Token (utils.GenerateTokenWithInfo)
    ├─ 存储 Token 到 Redis (白名单)
    ├─ 记录登录日志
    └─ 返回 Token 和用户信息
    ↓
Handler 返回响应
{
    "code": 0,
    "msg": "success",
    "data": {
        "token": "eyJhbGc...",
        "user": {
            "id": 1,
            "username": "admin",
            "nickname": "管理员"
        }
    }
}
```

#### 1.2 JWT Token 结构

```go
type Claims struct {
    UserID   int64  `json:"userId"`      // 用户 ID
    UserType int    `json:"userType"`    // 用户类型: 0=Member, 1=Admin
    TenantID int64  `json:"tenantId"`    // 租户 ID
    Nickname string `json:"nickname"`    // 用户昵称
    jwt.RegisteredClaims                 // 标准 JWT 字段
}
```

#### 1.3 Token 验证机制

**双重验证**：JWT 签名验证 + Redis 白名单检查

```go
// middleware/auth.go
func Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取 Token
        token := obtainAuthorization(c)
        
        // 2. 验证 JWT 签名和有效期
        claims, err := utils.ParseToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, core.Error(401, "Token无效"))
            return
        }
        
        // 3. 检查 Redis 白名单（Token 是否已登出）
        if core.RDB != nil {
            redisKey := fmt.Sprintf(RedisKeyAccessToken, token)
            exists, err := core.RDB.Exists(c.Request.Context(), redisKey).Result()
            if err == nil && exists == 0 {
                c.AbortWithStatusJSON(401, core.Error(401, "Token已失效，请重新登录"))
                return
            }
        }
        
        // 4. 提取用户信息到上下文
        loginUser := &core.LoginUser{
            UserID:   claims.UserID,
            UserType: claims.UserType,
            TenantID: claims.TenantID,
            Nickname: claims.Nickname,
        }
        core.SetLoginUser(c, loginUser)
        c.Next()
    }
}
```

#### 1.4 Token 获取方式

支持三种方式传递 Token：

```go
// 1. Header 方式（推荐）
Authorization: Bearer eyJhbGc...

// 2. Query 参数方式
GET /api/user?Authorization=eyJhbGc...

// 3. Form 参数方式
POST /api/user
Content-Type: application/x-www-form-urlencoded
Authorization=eyJhbGc...
```

### 2. 用户管理模块 (User Module)

#### 2.1 用户创建流程

```
POST /admin-api/system/user/create
    ↓
UserHandler.CreateUser()
    ├─ 绑定请求参数
    ├─ 参数验证
    └─ 调用 UserService.CreateUser()
    ↓
UserService.CreateUser()
    ├─ 检查用户名唯一性
    ├─ 检查手机号唯一性
    ├─ 检查邮箱唯一性
    ├─ 加密密码 (utils.HashPassword)
    ├─ 构造 User 对象
    ├─ 保存到数据库 (Repository)
    ├─ 关联角色和岗位
    └─ 返回用户 ID
    ↓
Handler 返回成功响应
```

#### 2.2 用户查询

```go
// 获取用户列表
func (s *UserService) GetUserPage(ctx context.Context, req *req.UserPageReq) ([]resp.UserRespVO, int64, error) {
    u := s.q.SystemUser
    
    // 构建查询条件
    query := u.WithContext(ctx)
    
    if req.Username != "" {
        query = query.Where(u.Username.Like("%" + req.Username + "%"))
    }
    if req.Status != nil {
        query = query.Where(u.Status.Eq(int32(*req.Status)))
    }
    
    // 统计总数
    count, err := query.Count()
    if err != nil {
        return nil, 0, err
    }
    
    // 分页查询
    users, err := query.
        Offset(int((req.PageNo - 1) * req.PageSize)).
        Limit(req.PageSize).
        Find()
    
    // 转换为响应对象
    result := make([]resp.UserRespVO, 0, len(users))
    for _, user := range users {
        result = append(result, resp.UserRespVO{
            ID:       user.ID,
            Username: user.Username,
            Nickname: user.Nickname,
            // ... 其他字段
        })
    }
    
    return result, count, nil
}
```

### 3. 权限控制模块 (Permission Module)

#### 3.1 权限体系

项目采用 **RBAC (Role-Based Access Control)** 权限模型：

```
用户 (User)
    ↓
关联多个角色 (Role)
    ↓
每个角色拥有多个权限 (Permission)
    ↓
权限对应菜单和操作
```

#### 3.2 权限检查流程

```go
// 在 Handler 中检查权限
func (h *UserHandler) DeleteUser(c *gin.Context) {
    // 1. 获取登录用户
    loginUser := core.GetLoginUser(c)
    
    // 2. 检查权限
    hasPermission, err := h.permissionService.CheckPermission(
        c.Request.Context(),
        loginUser.UserID,
        "system:user:delete",  // 权限标识
    )
    
    if !hasPermission {
        core.WriteError(c, core.ForbiddenCode, "无权限执行此操作")
        return
    }
    
    // 3. 执行业务逻辑
    // ...
}
```

#### 3.3 租户隔离

项目支持多租户隔离，确保不同租户的数据完全隔离：

```go
// 在查询时添加租户过滤
func (s *UserService) GetUserByID(ctx context.Context, id int64) (*model.SystemUser, error) {
    loginUser := core.GetLoginUser(c)
    tenantID := loginUser.TenantID
    
    u := s.q.SystemUser
    user, err := u.WithContext(ctx).
        Where(u.ID.Eq(id)).
        Where(u.TenantID.Eq(tenantID)).  // 租户隔离
        First()
    
    return user, err
}
```

### 4. 响应格式标准化

#### 4.1 统一响应结构

```go
// 成功响应
type Result[T any] struct {
    Code int    `json:"code"`  // 错误码
    Msg  string `json:"msg"`   // 错误信息
    Data T      `json:"data"`  // 业务数据
}

// 成功示例
{
    "code": 0,
    "msg": "success",
    "data": {
        "id": 1,
        "username": "admin"
    }
}

// 错误示例
{
    "code": 400,
    "msg": "参数错误",
    "data": null
}
```

#### 4.2 分页响应

```go
type PageResult[T any] struct {
    List  []T   `json:"list"`   // 数据列表
    Total int64 `json:"total"`  // 总记录数
}

// 分页响应示例
{
    "code": 0,
    "msg": "success",
    "data": {
        "list": [
            {"id": 1, "username": "admin"},
            {"id": 2, "username": "user"}
        ],
        "total": 100
    }
}
```

#### 4.3 错误码体系

| 错误码 | 含义 | 使用场景 |
|--------|------|---------|
| `0` | 成功 | 请求成功 |
| `400` | 参数错误 | 请求参数验证失败 |
| `401` | 未授权 | 未登录或 Token 无效 |
| `403` | 禁止访问 | 无权限访问资源 |
| `404` | 资源不存在 | 请求的资源不存在 |
| `409` | 冲突 | 资源冲突（如重复创建） |
| `500` | 系统异常 | 服务器内部错误 |
| `501` | 未实现 | 功能未实现 |
| `503` | 服务不可用 | 服务暂时不可用 |

---

## 开发实践

### 1. 添加新功能的完整步骤

#### 步骤 1：定义数据模型

```go
// internal/model/my_entity.go
package model

import "time"

type MyEntity struct {
    ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
    Name      string    `gorm:"column:name;type:varchar(100)"`
    Status    int32     `gorm:"column:status;type:int"`
    CreateTime time.Time `gorm:"column:create_time;type:datetime"`
    UpdateTime time.Time `gorm:"column:update_time;type:datetime"`
}

func (MyEntity) TableName() string {
    return "my_entity"
}
```

#### 步骤 2：生成 GORM DAO 代码

```bash
# 编辑 cmd/gen/generate.go，添加新模型
make gen
```

#### 步骤 3：定义请求和响应对象

```go
// internal/api/req/my_entity.go
package req

type MyEntityCreateReq struct {
    Name   string `json:"name" binding:"required,min=1,max=100"`
    Status int32  `json:"status" binding:"min=0,max=1"`
}

// internal/api/resp/my_entity.go
package resp

type MyEntityRespVO struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Status    int32  `json:"status"`
    CreateTime string `json:"CreateTime"`
}
```

#### 步骤 4：实现 Service 层

```go
// internal/service/my_entity.go
package service

import (
    "context"
    "github.com/wxlbd/ruoyi-mall-go/internal/api/req"
    "github.com/wxlbd/ruoyi-mall-go/internal/api/resp"
    "github.com/wxlbd/ruoyi-mall-go/internal/model"
    "github.com/wxlbd/ruoyi-mall-go/internal/pkg/core"
    "github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
)

type MyEntityService struct {
    q *query.Query
}

func NewMyEntityService(q *query.Query) *MyEntityService {
    return &MyEntityService{q: q}
}

// 创建
func (s *MyEntityService) Create(ctx context.Context, req *req.MyEntityCreateReq) (int64, error) {
    entity := &model.MyEntity{
        Name:   req.Name,
        Status: req.Status,
    }
    
    if err := s.q.MyEntity.WithContext(ctx).Create(entity); err != nil {
        return 0, err
    }
    
    return entity.ID, nil
}

// 查询
func (s *MyEntityService) GetByID(ctx context.Context, id int64) (*resp.MyEntityRespVO, error) {
    entity, err := s.q.MyEntity.WithContext(ctx).
        Where(s.q.MyEntity.ID.Eq(id)).
        First()
    
    if err != nil {
        return nil, err
    }
    
    return &resp.MyEntityRespVO{
        ID:        entity.ID,
        Name:      entity.Name,
        Status:    entity.Status,
        CreateTime: entity.CreateTime.Format("2006-01-02 15:04:05"),
    }, nil
}

// 更新
func (s *MyEntityService) Update(ctx context.Context, id int64, req *req.MyEntityCreateReq) error {
    _, err := s.q.MyEntity.WithContext(ctx).
        Where(s.q.MyEntity.ID.Eq(id)).
        Updates(&model.MyEntity{
            Name:   req.Name,
            Status: req.Status,
        })
    
    return err
}

// 删除
func (s *MyEntityService) Delete(ctx context.Context, id int64) error {
    _, err := s.q.MyEntity.WithContext(ctx).
        Where(s.q.MyEntity.ID.Eq(id)).
        Delete()
    
    return err
}
```

#### 步骤 5：实现 Handler 层

```go
// internal/api/handler/admin/my_entity.go
package admin

import (
    "github.com/wxlbd/ruoyi-mall-go/internal/api/req"
    "github.com/wxlbd/ruoyi-mall-go/internal/pkg/core"
    "github.com/wxlbd/ruoyi-mall-go/internal/service"
    "github.com/gin-gonic/gin"
)

type MyEntityHandler struct {
    svc *service.MyEntityService
}

func NewMyEntityHandler(svc *service.MyEntityService) *MyEntityHandler {
    return &MyEntityHandler{svc: svc}
}

// 创建
func (h *MyEntityHandler) Create(c *gin.Context) {
    var req req.MyEntityCreateReq
    if err := c.ShouldBindJSON(&req); err != nil {
        core.WriteError(c, core.ParamErrCode, err.Error())
        return
    }
    
    id, err := h.svc.Create(c.Request.Context(), &req)
    if err != nil {
        core.WriteBizError(c, err)
        return
    }
    
    core.WriteSuccess(c, gin.H{"id": id})
}

// 获取
func (h *MyEntityHandler) GetByID(c *gin.Context) {
    id := c.GetInt64("id")
    
    data, err := h.svc.GetByID(c.Request.Context(), id)
    if err != nil {
        core.WriteBizError(c, err)
        return
    }
    
    core.WriteSuccess(c, data)
}

// 更新
func (h *MyEntityHandler) Update(c *gin.Context) {
    id := c.GetInt64("id")
    
    var req req.MyEntityCreateReq
    if err := c.ShouldBindJSON(&req); err != nil {
        core.WriteError(c, core.ParamErrCode, err.Error())
        return
    }
    
    if err := h.svc.Update(c.Request.Context(), id, &req); err != nil {
        core.WriteBizError(c, err)
        return
    }
    
    core.WriteSuccess(c, nil)
}

// 删除
func (h *MyEntityHandler) Delete(c *gin.Context) {
    id := c.GetInt64("id")
    
    if err := h.svc.Delete(c.Request.Context(), id); err != nil {
        core.WriteBizError(c, err)
        return
    }
    
    core.WriteSuccess(c, nil)
}
```

#### 步骤 6：注册路由

```go
// internal/api/router/router.go
func InitRouter(...) *gin.Engine {
    // ... 其他路由
    
    // 注册 MyEntity 路由
    myEntityGroup := r.Group("/admin-api/my-entity")
    myEntityGroup.Use(middleware.Auth())  // 需要认证
    {
        myEntityGroup.POST("/create", myEntityHandler.Create)
        myEntityGroup.GET("/:id", myEntityHandler.GetByID)
        myEntityGroup.PUT("/:id", myEntityHandler.Update)
        myEntityGroup.DELETE("/:id", myEntityHandler.Delete)
    }
    
    return r
}
```

#### 步骤 7：注册依赖注入

```go
// internal/service/module.go
var Module = wire.NewSet(
    NewMyEntityService,
    // ... 其他 Service
)

// internal/api/handler/module.go
var Module = wire.NewSet(
    admin.NewMyEntityHandler,
    // ... 其他 Handler
)
```

#### 步骤 8：重新生成 Wire 代码

```bash
make wire
```

### 2. 参数验证

使用 Gin 的 binding 标签进行参数验证：

```go
type CreateUserReq struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Email    string `json:"email" binding:"required,email"`
    Age      int    `json:"age" binding:"min=18,max=120"`
    Phone    string `json:"phone" binding:"omitempty,len=11"`
}
```

常用验证标签：

| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填字段 | `binding:"required"` |
| `min=N` | 最小值 | `binding:"min=18"` |
| `max=N` | 最大值 | `binding:"max=120"` |
| `len=N` | 固定长度 | `binding:"len=11"` |
| `email` | 邮箱格式 | `binding:"email"` |
| `url` | URL 格式 | `binding:"url"` |
| `omitempty` | 可选字段 | `binding:"omitempty,email"` |
| `dive` | 嵌套结构体验证 | `binding:"dive"` |

### 3. 错误处理

统一使用错误码体系：

```go
// 参数错误
if req.Username == "" {
    core.WriteError(c, core.ParamErrCode, "用户名不能为空")
    return
}

// 业务错误
user, err := h.svc.GetUserByID(c.Request.Context(), id)
if user == nil {
    core.WriteError(c, core.NotFoundCode, "用户不存在")
    return
}

// 系统错误
if err != nil {
    core.WriteBizError(c, err)
    return
}

// 自定义业务异常
if user.Status != 0 {
    core.WriteError(c, 1001001001, "用户已被禁用")
    return
}
```

### 4. 事务处理

使用 GORM 的事务功能：

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *req.CreateOrderReq) (int64, error) {
    // 开启事务
    tx := s.q.WithContext(ctx).Begin()
    
    // 创建订单
    order := &model.Order{
        UserID:    req.UserID,
        TotalAmount: req.TotalAmount,
    }
    if err := tx.Create(order).Error; err != nil {
        tx.Rollback()
        return 0, err
    }
    
    // 创建订单项
    for _, item := range req.Items {
        orderItem := &model.OrderItem{
            OrderID:   order.ID,
            ProductID: item.ProductID,
            Quantity:  item.Quantity,
        }
        if err := tx.Create(orderItem).Error; err != nil {
            tx.Rollback()
            return 0, err
        }
    }
    
    // 提交事务
    if err := tx.Commit().Error; err != nil {
        return 0, err
    }
    
    return order.ID, nil
}
```

### 5. 缓存策略

使用 Redis 缓存热数据：

```go
func (s *UserService) GetUserByID(ctx context.Context, id int64) (*model.SystemUser, error) {
    // 1. 先查缓存
    cacheKey := fmt.Sprintf("user:%d", id)
    val, err := core.RDB.Get(ctx, cacheKey).Result()
    if err == nil {
        // 缓存命中，反序列化
        var user model.SystemUser
        if err := json.Unmarshal([]byte(val), &user); err == nil {
            return &user, nil
        }
    }
    
    // 2. 缓存未命中，查数据库
    u := s.q.SystemUser
    user, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存（有效期 1 小时）
    data, _ := json.Marshal(user)
    core.RDB.Set(ctx, cacheKey, data, time.Hour)
    
    return user, nil
}
```

---

## 常见问题

### Q1: 如何获取登录用户信息？

```go
// 在 Handler 或 Service 中
loginUser := core.GetLoginUser(c)
if loginUser == nil {
    // 未登录
    return
}

userID := loginUser.UserID
userType := loginUser.UserType
tenantID := loginUser.TenantID
```

### Q2: 如何添加新的错误码？

```go
// internal/pkg/core/error.go
const MyCustomErrorCode = 1001001001

var ErrMyCustom = NewBizError(MyCustomErrorCode, "自定义错误信息")

// 使用
core.WriteError(c, MyCustomErrorCode, "自定义错误信息")
```

### Q3: 如何实现权限检查？

```go
// 在 Handler 中
loginUser := core.GetLoginUser(c)
hasPermission, err := h.permissionService.CheckPermission(
    c.Request.Context(),
    loginUser.UserID,
    "system:user:delete",
)

if !hasPermission {
    core.WriteError(c, core.ForbiddenCode, "无权限")
    return
}
```

### Q4: 如何实现租户隔离？

```go
// 在查询时添加租户过滤
loginUser := core.GetLoginUser(c)
tenantID := loginUser.TenantID

u := s.q.SystemUser
users, err := u.WithContext(ctx).
    Where(u.TenantID.Eq(tenantID)).
    Find()
```

### Q5: 如何使用 GORM 进行复杂查询？

```go
// 多条件查询
u := s.q.SystemUser
users, err := u.WithContext(ctx).
    Where(u.Status.Eq(0)).
    Where(u.DeptID.Eq(deptID)).
    Where(u.Username.Like("%" + keyword + "%")).
    Order(u.CreateTime.Desc()).
    Offset(offset).
    Limit(limit).
    Find()

// 统计
count, err := u.WithContext(ctx).
    Where(u.Status.Eq(0)).
    Count()

// 更新
_, err := u.WithContext(ctx).
    Where(u.ID.Eq(id)).
    Updates(&model.SystemUser{
        Nickname: "新昵称",
        Status:   1,
    })

// 删除
_, err := u.WithContext(ctx).
    Where(u.ID.Eq(id)).
    Delete()
```

### Q6: 如何处理数据库事务？

```go
// 开启事务
tx := s.q.WithContext(ctx).Begin()

// 执行多个操作
if err := tx.Create(&entity1).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Create(&entity2).Error; err != nil {
    tx.Rollback()
    return err
}

// 提交事务
return tx.Commit().Error
```

### Q7: 如何调试和查看 SQL 语句？

```go
// 在配置中启用 SQL 日志
log:
  level: "debug"  # 设置为 debug 级别

// 在代码中
u := s.q.SystemUser
users, err := u.WithContext(ctx).
    Where(u.Status.Eq(0)).
    Debug().  // 启用调试模式，会打印 SQL
    Find()
```

### Q8: 如何扩展中间件？

```go
// 创建新中间件
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 前置处理
        c.Set("key", "value")
        
        // 继续处理
        c.Next()
        
        // 后置处理
        status := c.Writer.Status()
        // ...
    }
}

// 注册中间件
r.Use(MyMiddleware())
```

### Q9: 如何使用 Redis 缓存？

```go
// 设置缓存
core.RDB.Set(ctx, "key", "value", time.Hour)

// 获取缓存
val, err := core.RDB.Get(ctx, "key").Result()

// 删除缓存
core.RDB.Del(ctx, "key")

// 设置过期时间
core.RDB.Expire(ctx, "key", time.Hour)

// 检查是否存在
exists, err := core.RDB.Exists(ctx, "key").Result()
```

### Q10: 如何处理并发请求？

```go
// GORM 天生支持并发
// 但需要注意以下几点：

// 1. 使用连接池
// 在 config.yaml 中配置
mysql:
  max_open: 100  # 最大打开连接数
  max_idle: 10   # 最大空闲连接数

// 2. 使用上下文超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := s.q.SystemUser.WithContext(ctx).First()

// 3. 使用 Redis 分布式锁（可选）
lock := core.RDB.SetNX(ctx, "lock:user:1", "1", time.Second)
if lock.Val() {
    // 获取锁成功，执行业务逻辑
    defer core.RDB.Del(ctx, "lock:user:1")
}
```

---

## 学习路径建议

### 初级阶段（1-2 周）

1. **理解项目架构**
   - 阅读本文档的"架构设计"部分
   - 理解 Clean Architecture 的四层设计
   - 了解项目的目录结构

2. **掌握基础流程**
   - 理解请求处理流程
   - 学习认证和授权机制
   - 熟悉统一的响应格式

3. **动手实践**
   - 修改现有的简单 API（如获取用户列表）
   - 理解 Handler → Service → Repository 的调用链
   - 学会使用 GORM 进行基本的 CRUD 操作

### 中级阶段（2-4 周）

1. **深入学习关键模块**
   - 研究认证模块的实现细节
   - 学习权限控制的实现方式
   - 理解租户隔离的机制

2. **掌握开发技能**
   - 学会添加新功能（按照"添加新功能的完整步骤"）
   - 理解依赖注入（Wire）的工作原理
   - 学会使用 GORM Gen 生成 DAO 代码

3. **实践项目**
   - 添加一个简单的新模块（如"分类管理"）
   - 实现完整的 CRUD 功能
   - 添加参数验证和错误处理

### 高级阶段（4+ 周）

1. **优化和扩展**
   - 学习缓存策略的实现
   - 理解事务处理的最佳实践
   - 学会处理并发场景

2. **性能优化**
   - 学习数据库查询优化
   - 理解 Redis 缓存的使用
   - 学会使用连接池和连接复用

3. **深度研究**
   - 研究其他模块的实现（如订单、支付等）
   - 理解复杂的业务逻辑
   - 学会设计和实现复杂的功能

---

## 总结

本文档从多个维度深度解析了芋道商城 Go 版本的架构、流程和配置机制：

- **架构设计**：Clean Architecture 四层设计，清晰的职责划分
- **核心流程**：启动流程、请求处理流程、认证流程、数据库操作流程
- **配置机制**：配置文件结构、加载机制、环境变量覆盖
- **依赖注入**：Wire 框架的使用和依赖关系管理
- **关键模块**：认证、用户管理、权限控制、响应标准化
- **开发实践**：添加新功能的完整步骤、参数验证、错误处理、事务处理、缓存策略
- **常见问题**：快速参考和解决方案

通过学习本文档，你应该能够：

✅ 快速理解项目的整体架构和设计思想
✅ 掌握项目的核心运行流程
✅ 学会如何在项目基础上添加新功能
✅ 理解项目的配置和依赖注入机制
✅ 能够独立解决开发中的常见问题
✅ 为项目的扩展和维护奠定坚实基础

祝你学习愉快！🚀
