# 批次 A / B 审查问题修复（2026-09-08）

修复对象：对提交 `851969f` 的审查所发现的 F1/P1、F2/P1、F3/P2。实现提交为 `59e857e`，分支 `codex/batch-ab`。三项均已修复并通过真实 PostgreSQL、全量竞态检测及完整服务 HTTP 验证；T10、A26、A27 在下述 A/B 范围内重新验收通过。

## 修复与回归对应

| 问题 | 最终行为 | 验证 |
|---|---|---|
| F1/P1：A 可停止 B 的实际调度任务 | 调度器唯一公开配置变更入口 `SyncJob(ctx,id)` 先按可信租户和任务 ID 查验归属，再修改定时器。删除、停用必须确认数据库影响行数；混合租户批量删除在事务中整体拒绝，提交前不修改定时器。 | `TestReviewCrossTenantSchedulerMutation` 覆盖删除/停用并断言 B 定时器仍存在；`TestJobMutationLifecycleAndRecovery` 覆盖混合批删、调度器直接调用、手动触发越权、SQL 失败及 B 任务不受影响；HTTP 单删、停用、批删均返回非零错误。 |
| F2/P1：普通管理员可读取其他租户管理资料 | 完整详情、分页、Excel 导出均显式限定当前租户；跨租户详情返回业务码 403。公开发现仍只返回 ID/名称，跨租户平台汇总继续使用原来独立授权并审计的检查接口。 | `TestReviewTenantAdminCannotReadOtherTenantManagement` 使用迁移生成的普通 tenant_admin 权限；`TestTenantManagementOwnScopeAndPublicDiscovery` 覆盖 A/B 正常读取、缺少可信租户、筛选不能扩大范围及公开字段。HTTP 检查真实 Excel 只有表头与一条本租户数据，其他租户联系人/手机号均不出现。原平台检查及保留角色回归继续通过。 |
| F3/P2：正常新建任务读取未提交记录而失败 | 新建直接保存开启状态，数据库提交后再注册；更新先提交新配置，再替换定时器。调度失败记录任务 ID、返回明确错误，保留持久化目标状态供再次同步。 | `TestReviewCreateJobRegistersAfterCommit`；`TestJobMutationLifecycleAndRecovery` 将连接池限制为单连接，验证创建、更新后执行时间/参数、重复与并发同步无重复定时器；`TestJobCreateRegistrationFailureCanRecover` 覆盖数据库插入失败与调度注册失败后恢复。HTTP 创建、修改、停用、恢复、删除及同步全流程通过。 |

`TestJobStaleTimerSkipsStoppedAndDeletedRows` 还验证执行前重新检查持久化状态：旧定时器不能启动已停用或删除的任务；本租户仍可手动执行已停用任务，已删除任务两条执行路径均拒绝。任务执行使用独立、有时限的租户上下文，不保留 HTTP 上下文。

## 权限与迁移决策

沿用普通管理员读取自身租户资料的 query/export 权限，在服务查询中强制限定自身租户。这样已经安装的角色关联也立即受约束，不需要撤销合法的自身读取权限，因此本次没有新增权限迁移，也没有改写六版已存在迁移。完整管理读取接口不根据平台身份放宽范围；平台跨租户访问仍仅限既有授权、审计的汇总入口。

`system_tenant` 仍可用于公开 ID/名称发现及内部有效租户检查；表级共享读取不能代替完整管理接口的资源授权。120 个模型的字段/约束保持一致，来源清单更新到 359 个源码哈希并验证相符。

## 实际验证

环境：Go 1.26.2，PostgreSQL 16.10，Redis 8.2.1。数据库测试使用随机临时 schema，自动应用六版迁移并清理。HTTP 测试暂停该 schema 的种子任务，新增任务使用未来调度时间，不执行真实支付或通知处理器。

- 修复前的三个顶层回归、四个场景失败证据：[review-fixes-red.log](evidence/review-fixes-red.log)。修复过程的定向结果：[initial-green](evidence/review-fixes-initial-green.log)、[focused-race](evidence/review-fixes-focused-race.log)。最终结果以全量日志为准。
- `go test -race -count=1 -json ./...`：104 个顶层测试、含子测试共 315 项通过，0 失败、0 测试跳过；28 个含测试包通过。77 个无测试文件的包不是测试覆盖。[汇总](evidence/review-fixes-go-summary.json)、[完整原始事件](evidence/review-fixes-go-race.jsonl)。
- `go build ./...`、`go vet ./...`、`go mod verify` 和服务/迁移/初始化三个二进制构建退出码均为 0。[检查记录](evidence/review-fixes-checks.json)。
- 最新完整服务 HTTP 测试通过，共 46 条检查记录，包括新增租户/任务验证和原登录、商品、地址、购物车、刷新与登出流程。[HTTP 结果](evidence/http-smoke.json)、[运行结果](evidence/review-fixes-http.log)、[服务日志](evidence/http-server.log)。日志检查未发现测试口令或令牌；服务与 schema 已清理。首次新增测试曾因脚本把 Go 字段名误作数据库列名而在准备数据时失败，已修正为实际 `expire_time` 并完整重跑。
- 六个迁移 SQL 文件与审查基线逐字节一致；来源清单中的模型字段/约束未改变，仅更新查询来源和源码哈希。[检查记录](evidence/review-fixes-checks.json)。

上述测试使用 `runbook.md` 的 `TEST_POSTGRES_DSN`、`T09_REDIS_ADDR` 以及隔离的 Go 缓存环境。不能省略依赖后将 skip 当作通过。前端源码未改变，本轮没有重跑前端构建；原构建证据保持适用。

## 故障恢复与边界

数据库和进程内调度器不具备共同事务。修复采用先提交目标状态、再同步调度的方式：同步失败明确返回“任务 ID 已保存，调度同步失败，请重试同步”并写错误日志。恢复调度器后，通过 `POST /admin-api/infra/job/sync` 重试当前租户任务，包含已停用和软删除记录，清除未成功移除的旧定时器；反复调用不会累积重复定时器。失败后先刷新任务列表，不应盲目重复创建同一处理器。

当前调度所有权仍是单进程；服务实例内串行化变更，多实例共同调度需要另行选主。执行前状态复核会阻止尚未进入处理器的旧任务，已经开始的处理器不会被强制终止。支付/退款状态机、真实渠道与终端 E2E 继续按 C/D 范围处理。
