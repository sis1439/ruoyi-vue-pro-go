# 环境与隔离

| 项目 | 本次值 |
|---|---|
| 系统 | macOS / arm64 |
| Go 实际工具链 | go1.26.2 darwin/arm64；模块最低 go1.25.4 |
| Go 构建缓存 | `/private/tmp/ruoyi-ab-go-build` |
| Go 模块缓存 | `/private/tmp/ruoyi-ab-gopath` |
| PostgreSQL | PostgreSQL 16.10，独立实例 |
| PostgreSQL 数据目录 | `/private/tmp/ruoyi-ab-pg/data` |
| PostgreSQL 监听 | `127.0.0.1:55439`，独立数据库 `ruoyi_ab` |
| Redis | 8.2.1，独立实例 `127.0.0.1:56397` |
| Node 初始环境 | v25.8.1；前端实际构建固定版本见 `frontend-build.md` |
| npm 初始环境 | 11.12.1 |

系统默认 `psql --version` 是 PostgreSQL 14.19 客户端，**不等于测试服务器版本**。未采纳交接文档示例中的 PostgreSQL 15，也未使用业务数据库。

真实 PostgreSQL 集成测试由 `internal/testutil.PostgreSQL` 创建独立随机 schema；每个连接的 search_path 都固定到该 schema，自动应用已提交迁移并清理。Redis 测试使用独立端口及命名空间，不发送短信或真实支付。

```sh
export GOCACHE=/private/tmp/ruoyi-ab-go-build
export GOPATH=/private/tmp/ruoyi-ab-gopath
export TEST_POSTGRES_DSN='host=127.0.0.1 port=55439 user=macmini dbname=ruoyi_ab sslmode=disable'
export T09_REDIS_ADDR=127.0.0.1:56397
make verify
```

上面的用户和端口仅为本机测试值。实际部署使用商城专用数据库、权限和 Redis 命名空间；连接口令及 JWT 密钥从环境注入，不写入种子或报告。
