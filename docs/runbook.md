# C/D 运行与验证

本分支基于 `codex/batch-ab` 的 `6548c9e`。迁移、初始化、租户域名与账号引导、备份恢复沿用 `docs/batch-ab/runbook.md`、`docs/batch-ab/database-runbook.md`；不要使用旧 MySQL 初始化路径。

## 配置

- 数据库：`database.driver=postgres`、`database.dsn`；支持 `RUOYI_DATABASE_DRIVER`、`RUOYI_DATABASE_DSN`。
- JWT：`security.jwt_secret` 或 `RUOYI_JWT_SECRET`，至少 32 字节且满足已有强度检查。旧 `app.jwt_secret` 不再生效。
- Redis：`redis.addr`，不能在认证或查单限频失败时放行。
- 渠道回调：`pay.order_notify_url`、`pay.refund_notify_url` 为 HTTP 完整地址，生产类环境必须 HTTPS。渠道使用的域名必须匹配 A/B 登记的可信租户域名。
- 内部通知：独立强密钥 `pay.notify_token`；`pay.trusted_notify_urls` 列出允许携带令牌的**完整业务回调 URL**，包含实际路径和必要查询参数。禁止 URL 用户信息、片段和重定向。业务接收端配置相同通知密钥；没有令牌时拒绝调用。

例如白名单可列入应用实际配置的 `/app-api/trade/order/update-paid` 与 `/admin-api/trade/after-sale/update-refunded` 完整 URL；应以实际注册路由和租户域名核对，不能直接照搬示例主机。

## 本地验证

工具链最低 Go 1.26.6（`go.mod` 与 CI 固定）；当前依赖锁文件包含漏洞扫描要求的修复版本。

```sh
rtk proxy make gen
rtk proxy make wire
rtk proxy go mod verify
rtk proxy go build ./...
rtk proxy go vet ./...
# 设置 TEST_POSTGRES_DSN 和 T09_REDIS_ADDR 指向隔离测试依赖后运行：
rtk proxy make test-integration
```

`make test-integration` 执行 `go test -race -count=1 -json ./...`，然后检查输出中没有测试跳过、没有失败，且关键数据库与 Redis 用例确实通过。输出为 `integration-results.jsonl`；它不是生产业务数据。

## 通知故障与恢复

1. 渠道回调验签或本地数据库提交失败返回 HTTP 500，由渠道重试。支付、退款、转账的本地结果与通知任务同事务提交。
2. 内部投递只有 HTTP 200 且 JSON `code=0` 才成功。任务与投递日志同事务保存；请求成功后进程崩溃可能重放，接收端必须幂等。
3. 微信退款超时或 `PROCESSING`、`ABNORMAL` 保留待处理状态与退款额度，定时查单按原退款号恢复。不能为绕过待处理限制再生成一笔退款。
4. 已取消订单收到迟到付款返回失败，交易订单保持取消；支付单与通知任务保留，重试耗尽后进入失败任务。使用支付通知管理接口查看任务、日志与关联支付单，人工对账后处理退款；不得直接把已取消订单置为待发货，也不得将任务强行标成功。当前没有自动退款或外部告警连接。
5. 如果历史支付单的 `merchant_order_id` 是展示单号，先盘点与核对关联关系再设计迁移，不应直接重放新主键契约。

真实支付/退款、生产部署和真实短信没有在本次验证中执行。
