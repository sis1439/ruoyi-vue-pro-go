# 项目协作指南

适用于本仓库；子目录有更具体指导时按其范围执行，用户明确要求优先。

## 项目入口

- 芋道商城 Go 后端，使用 Gin、GORM Gen、Wire、Redis；模块名为 `github.com/wxlbd/ruoyi-mall-go`，版本以 `go.mod` 为准。
- 入口：`cmd/server/main.go`；装配：`cmd/server/wire.go`；接口：`internal/api/`；业务：`internal/service/`；数据访问：`internal/repo/`；实体：`internal/model/`。
- 请求 / 响应放在 `internal/api/contract/`，复用 `pkg/response/`、`pkg/errors/`、`pkg/types/` 和 `internal/consts/`。沿用现有 Service / Query 模式，不新增仅透传的分层。
- 当前应用使用 MySQL。旧文档和隔离副本的实现、测试结果不代表当前检出状态，以源码和实际验证为准。

## 常用命令

在仓库根目录执行，其他命令查 `Makefile`：

```sh
rtk go mod download
rtk proxy make gen    # 生成 DAO，不连接数据库、不建表
rtk proxy make wire   # 生成依赖装配，需要已安装 Wire
rtk proxy make build
rtk proxy make test
rtk proxy make vet
```

- `internal/repo/query/` 被 Git 忽略；缺失或 model / 生成清单变更后先运行 `make gen`。构造函数、ProviderSet 或接口绑定变更后运行 `make wire`，不手改 `cmd/server/wire_gen.go`。
- `make ci` 执行 gen、build、vet、test。`make deps` 会执行 tidy 并可能改写依赖；仅下载时用 `go mod download`。
- 启动前核对 `pkg/config/config.go`：`GO_ENV` 默认 local，读取 `config/config.<env>.yaml`，本地配置需自行准备；不要假设所有字段都支持环境变量覆盖。

## 项目易错点

- API 主要前缀是 `/admin-api`、`/app-api`。普通响应为 `{ code, msg, data }`，成功码为 0，分页为 `data.list/total`；接口改动核对前端实际参数、类型与空值约定。
- 普通业务错误可能以 HTTP 200 返回；支付渠道回调必须按渠道协议应答，不能直接套用业务响应工具。
- `AuditPlugin` 填充审计字段不等于租户隔离。用户、租户和资源归属从可信上下文校验；同时核对逻辑删除过滤。认证依赖故障不能作为放行依据。
- 库存、券、积分、余额及订单联合变更要核对事务传播：相关查询和下游服务必须使用同一事务句柄，并保留请求上下文。
- 金额按契约使用整数“分”。区分交易主键、展示单号、支付单、扩展单及商户订单号；修改映射须检查通知 / 退款链路和历史数据兼容。
- 支付 / 退款成功须来自可信通知或查单，校验签名、商户、应用、归属、金额和状态。重复通知、并发及重试不得重复扣减或返还资产。
- 保留价格计算器和订单处理器的执行顺序。涉及支付时连同通知持久化、重试及故障恢复路径检查。
- `make gen` 不执行 Schema 迁移；`scripts/migration.sql` 含删表语句，不能直接作为通用初始化脚本运行。数据库变更需核对目标库及数据兼容。

## 验证与交付

- 保留已有修改和未跟踪文件；只格式化改动文件，不夹带生成产物、本地配置或真实凭证。
- 先运行受影响包 / 用例；共享接口或装配变更再扩大检查，并发修改按需使用 `-race`。纯文档只核对内容、路径与差异，无需构建或启动服务。
- 现有测试使用 `testing` / `testify`。`make test-integration` 当前未配套 integration tag 测试，命令成功不等于真实数据库验证通过；实现变化后更新此条。
- 事务、租户、并发与 Redis 行为用隔离真实依赖验证最终数据；缺少渠道凭证时继续本地测试，单列真实联调未覆盖项。真实交易或生产操作须有相应授权。
- 交付前检查 `rtk git diff --check` 及新增文件，说明实际验证范围、结果和限制；未运行、跳过及其他副本的结果不算通过。

## 按需参考

- 配置和运行：`pkg/config/config.go`、`docs/runbook.md`。
- 支付 / 订单：`internal/service/pay/`、`internal/service/mall/trade/`、`docs/decisions.md`；文档结论仍需与当前代码核对。
