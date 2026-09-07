# 批次 A / B 验收报告

日期：2026-09-07。依据任务书2.1完成批次A（T00、T01契约盘点、T02/T03）和批次B（T09/T10安全边界、T04本地交易事务）。完整T01/T12及支付/退款属于C/D，不将自动化通过当作真实渠道或生产批准。

## 固定版本、实现与交付

仓库和提交见[baseline.md](baseline.md)，实际环境见[environment.md](environment.md)。Go工作分支为`codex/batch-ab`，位于原前端`.work/ruoyi-vue-pro-go`独立副本；原Go仓库保持基线。前端只有独立构建副本修改构建配置，页面业务代码未改；其本地提交及补丁见两份前端构建报告。

| 工作包 | 交付范围 | 状态 / 主要证据 |
|---|---|---|
| T00 | 固定4套版本、DAO/Wire依赖、后端与H5/小程序/后台可复现构建 | 验收通过；baseline、frontend-build、frontend-admin-build、最终干净源码验证记录 |
| T01（A部分） | 216调用点/215方法路径、Java/Go/前端证据、字段与行为清单、7盘点及19消费端测试 | 验收通过（盘点）；198精确路径匹配、18缺口，未宣称全部接口已对齐 |
| T02 | 120源模型、六版前向迁移、约束/种子/6个真实菜单组件、显式管理员初始化 | 验收通过；schema-manifest、migrations.sha256、database-v6-tests.log |
| T03 | 真实PG驱动、120表查询映射、codec/零值/软删除、统计金额和事务、空库/升级/恢复 | 验收通过（首期测试范围）；database-verification.md及完整测试记录 |
| T04 | 下单所有本地资源同事务，SPU/SKU事务传播，失败/竞争/COMMIT失败/SIGKILL回滚，未付款取消一次恢复 | 验收通过（本地边界）；t04-transactions.md与原始失败/成功证据 |
| T09 | 独立强JWT配置、access/refresh轮换撤销、Redis故障拒绝、身份/角色/归属、验证码/限流/日志/CORS | 验收通过（所列场景）；安全报告、真实Redis与完整服务HTTP记录 |
| T10 | 可信租户、统一ORM读写、支付/文件/缓存/任务边界、平台只读显式授权审计、保留角色防提权 | 验收通过（首期已测试路径）；security、infra、mail、login-audit和完整回归记录。真实支付回调/C批次另验 |

## A01—A28 追踪矩阵

状态遵守任务书统一定义。一个用例含本地工程与真实渠道两部分时分别说明，不把子项通过升级为整项通过。

| 编号 | 状态 | 环境与命令 | 结果、证据路径 / 未覆盖范围 |
|---|---|---|---|
| A01 | 验收通过  | PG16.10；`go test ./migrations ./cmd/bootstrap -count=1`；`scripts/batch_ab_smoke.py` | 空库迁移1..6、非敏感种子、显式bootstrap、完整服务HTTP商品/会员配置可用；证据`evidence/database-v6-tests.log`、`evidence/database-restore-v6.log` |
| A02 | 验收通过  | PG16.10；`go test ./migrations ./cmd/bootstrap -count=1`；`scripts/batch_ab_smoke.py` | 基线升级至6、二次执行无变更、dirty失败不伪报、121表及序列恢复后一致；证据`evidence/database-v6-tests.log`、`evidence/database-restore-v6.log` |
| A03 | 验收通过  | PG16.10；`go test ./migrations ./cmd/bootstrap -count=1`；`scripts/batch_ab_smoke.py` | 全120模型投影、BitBool/native bool、JSON/CSV/时间/空值/零值/软删除、超过32位整数分；绑定false/0与真实CRUD；证据`evidence/database-v6-tests.log`、`evidence/database-restore-v6.log` |
| A04 | 待处理（完整验收D）  | 固定前端/Node25.8.1；`python3 -m unittest discover -s docs/batch-ab/contracts/tests`、`node docs/batch-ab/contracts/tests/frontend-contracts.cjs` | A阶段盘点7测试、19实际消费端测试通过；18方法/路径缺口保留，未执行全页面E2E；证据`evidence/final-contract-inventory.log`、`evidence/final-frontend-contracts.log` |
| A05 | 验收通过（已列HTTP/服务范围）  | PG16.10/Redis8.2.1；`scripts/batch_ab_smoke.py`；`go test ./internal/service/mall/product` | 完整HTTP分类/品牌/SPU创建及零值更新、地址false、购物车计数/清理；PG SPU/SKU原子CRUD及跨租户SKU拒绝；证据`evidence/http-smoke.json`、`evidence/final-go-race.jsonl` |
| A06 | 验收通过  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 真实PriceService读取SKU/SPU，初始缺货、下架、回收站直接拒绝；定价后库存不足整笔回滚。unavailable-product.log及T04失败用例；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A07 | 验收通过  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 两个请求同时定价后竞争最后库存，最多一个订单、项、支付及资源组合成功；真实PG且race通过；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A08 | 验收通过  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 多SKU后项失败、SPU聚合和批次SKU写失败，精确比对所有本地资源与订单/支付表；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A09 | 验收通过（资源一致性）  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 同券及积分竞争使用不同SKU/SPU排除锁掩盖；不重复、不透支；余额/账本同事务。优惠算法另验；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A10 | 已修复待验收（渠道提交C）  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 本地支付INSERT/链接/COMMIT失败及进程中断均无孤儿单；远程渠道提交结果/恢复未完成；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A11 | 已修复待验收（重复请求D）  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 零元及未启用营销明确拒绝；已提交但响应丢失后的客户端幂等尚未实现；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A12 | 已修复待验收（超时任务C）  | PG16.10；`go test -race ./internal/service/mall/trade -count=1` | 未付款取消并发/重复仅一次资源返回；真实任务上下文隔离通过。远程支付关闭及正式超时业务调度C继续；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A13 | 待处理（C）  | 完整场景未运行，按C批次处理 | 支付回调与取消竞争未验收 |
| A14 | 待处理（C）  | 完整场景未运行，按C批次处理 | 操作权限已补齐，发货/自提/调价状态机及幂等仍需业务验收 |
| A15 | 已修复待验收（C）  | PG16.10；`go test ./internal/service/pay -run TestPaymentClientTenantBoundary`；真实渠道未运行 | 缓存前重新验证租户/渠道/应用/配置且拒绝停用已通过；微信工厂和真实渠道未验收；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A16 | 待处理（C）  | 完整场景未运行，按C批次处理 | 原版支付消费端测试证明签名对象要求；目前displayContent缺口保留 |
| A17 | 待处理（C）  | 完整场景未运行，按C批次处理 | 真实支付回调签名、商户、金额及状态变更尚未认证 |
| A18 | 待处理（C）  | 完整场景未运行，按C批次处理 | 通知租户/上下文生命周期已修复；空DTO与SUCCESS旧协议必须继续修复 |
| A19 | 待处理（C）  | 完整场景未运行，按C批次处理 | 重复/乱序支付通知状态与账本幂等未验收 |
| A20 | 待处理（C）  | PG16.10/Redis8.2.1；`go test ./internal/service/pay -run TestNotify`；恢复未运行 | 本轮通知锁隔离/持有者释放和有界执行已测；重启恢复、告警、人工重放仍待完成；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A21 | 待处理（C）  | 完整场景未运行，按C批次处理 | sync=true完整业务语义待实现；支付查询/提交的会员归属已加固 |
| A22 | 待处理（C）  | 完整场景未运行，按C批次处理 | 全额/部分/并发退款未验收；真实退款还需渠道配置和授权 |
| A23 | 待处理（C）  | 完整场景未运行，按C批次处理 | 退款失败/迟到回调/履约库存策略未验收 |
| A24 | 验收通过（记录场景）  | PG16.10/Redis8.2.1；`go test -race -count=1 ./...`；`scripts/batch_ab_smoke.py` | JWT篡改/算法/过期/用途/类型、真实Redis刷新竞争与撤销、过期access头刷新、普通角色拒绝、实时撤权、平台防提权；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A25 | 验收通过  | PG16.10/Redis8.2.1；`go test -race -count=1 ./...`；`scripts/batch_ab_smoke.py` | 默认/弱/缺失JWT配置拒绝；Redis未配置/错误/实际宕机拒绝；原子验证码及共享登录节流；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A26 | 验收通过（首期已测试路径）  | PG16.10/Redis8.2.1；`go test -race -count=1 ./...`；`scripts/batch_ab_smoke.py` | 两租户ORM增删改查/批量/危险Raw拒绝、统计、会员/地址/订单/支付缓存、模板；普通管理员拒绝平台检查，平台显式目标只读操作有审计；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A27 | 验收通过（A/B路径）  | PG16.10/Redis8.2.1；`go test -race -count=1 ./...`；`scripts/batch_ab_smoke.py` | 文件根路径/符号链接/租户校验，持久化job恢复租户，登录审计与通知不依赖存活HTTP；真实渠道回调与Outbox恢复仍属C；完整原始结果`evidence/final-go-race.jsonl`（测试名及输出可检索） |
| A28 | 待处理（完整E2E D）  | PG16.10；`python3 scripts/postgres_restore_check.py`；完整终端E2E未运行 | 备份/恢复已实测121表+序列一致、再次迁移无变化；下单至售后完整终端链路未验收；证据`evidence/database-restore-v6.log` |

## 验证方法与限制

全部数据库验收使用真实PostgreSQL16.10，Redis8.2.1；每个集成测试随机schema并固定所有连接search_path。不以SQLite、跳过测试或源码注释作为通过。原始最小失败日志与修复证据见[audit-findings.md](audit-findings.md)。完整命令见[runbook.md](runbook.md)。

前端生产构建从清空依赖后重建：商城H5/微信小程序共1056文件、后台2468文件两轮字节一致。它证明构建可复现，不证明真实浏览器/微信的交易E2E。后台构建选用可构建官方父提交，所选商城API和菜单组件与契约参照逐文件相同，有哈希证明。

工程修复完成、自动化验收通过、真实渠道验收通过、获准生产发布是不同状态。本批次仅提交前两者的范围内证据。没有真实支付/退款/短信/邮件、生产变更或远程推送。后续事项及唯一所需外部输入见[blockers.md](blockers.md)。

## 最终全量检查

- `go mod verify`、`go build ./...`、`go vet ./...`退出码均为0，DAO/Wire重新生成成功且Wire输出无额外变更。记录分别为`evidence/final-dependencies.log`、`final-build.log`、`final-vet.log`、`final-generation.log`。
- `go test -race -count=1 -json ./...`：97个顶层测试，含子测试共306项通过，0失败、0测试跳过；28个含测试包通过，77个无测试文件包由Go报告skip。后者不能计作测试覆盖。原始事件`evidence/final-go-race.jsonl`、汇总`final-go-summary.json`。
- 契约盘点7/7、真实前端消费端19/19通过，日志`final-contract-inventory.log`与`final-frontend-contracts.log`。

- 最新完整服务HTTP smoke通过，包含精确CORS Origin/Platform预检、初始化登录/菜单权限、商品/品牌零值、地址false、购物车清理、会员访问后台拒绝、跨租户拒绝、刷新旧access失效及登出撤销；`evidence/http-smoke.json`与`http-server.log`。脚本退出码0，临时服务与schema已清理。

## 修复提交

T00 `61a2f36`；T01 `467ddcb`；T02/T03 `8185d1f`；T04 `49150f0`；T09/T10 `b258005`。按工作包提交并保留原始失败/成功证据；有相互依赖，验证的是完整分支。

## 干净检出验收与最终工作树

从提交`8bc4c0004f0d21a994dc0bee8596196fb06a5536`通过git archive导出到全新目录，只包含Git跟踪文件。确认`internal/repo/query`不存在后，在真实PG16.10/Redis8.2.1环境执行`make verify`，DAO重新生成、依赖校验、全量编译、vet与全部测试退出码0。证据`evidence/clean-checkout.json`、`evidence/clean-checkout-verify.log`。之后只有文档/证据和Git格式属性更新，没有业务源码变更。

Git全分支差异检查通过。原始终端日志和git-format-patch保留原貌，仅对这些证据关闭空白样式检查；CSV使用标准CRLF且允许行尾CR。源码和报告仍执行正常格式检查。

原Go仓库工作树干净。原前端页面未修改，增加结果入口及隔离副本/缓存忽略规则；该仓库.git只读，未提交这些入口改动，用户原有未跟踪交接文档保留。后端独立副本全部按任务提交；前端两套构建副本也有独立提交。
