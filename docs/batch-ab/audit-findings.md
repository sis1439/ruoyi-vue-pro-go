# 批次 A / B 复核与修复证据

A = 固定源码确认，B = 调用链风险，经测试后再确认，C = 工程新增要求。原始版本见 baseline.md；以下“通过”只覆盖关联测试，完整矩阵见 verification-report.md。

| 编号 / 类型 | 源码与复核 | 最小失败场景 | 修复与证据 |
|---|---|---|---|
| T00 / A | `.gitignore` 忽略 `internal/repo/query`；Makefile 没有生成依赖 | 干净副本直接构建缺 internal 包 | 构建/测试先生成 DAO；Wire 固定 v0.7.0。基线生成后原测试通过，见 baseline-generated-tests.log |
| T01 / A | 固定前端、Java Controller/VO、Go 路由逐项盘点 | 18 处方法/路径无精确对应；支付消费端拒绝只有 prepayId 的 displayContent | CSV 216 调用点 / 215 方法路径；7 盘点测试、19 消费端测试。完整接口适配按 C/D 批次执行，不伪报兼容 |
| T02 / A/C | 原 scripts/migration.sql 非完整空库基线 | 未生成任何商城完整 DDL | 120 模型源清单、6 次版本迁移、权限与页面菜单、显式管理员初始化。真实空库、升级、重复执行、恢复验证 |
| T03 / A/B | `pkg/database` 原 MySQL 唯一入口；模型存在 BIT/tinyint、嵌入重名字段 | 初次生成 DDL 重复 update_time；pgx 无法把裸 bool 默认值写入 int2 | 按有效 GORM DBNames 建表；BitBool 保留 0/1 小整数与 Scanner/Valuer，SQL 默认表达式避免 bool 默认转换。真实120表查询与类型往返通过 |
| T03 / B | `internal/repo/product/statistics.go` MySQL IFNULL / Raw 子查询 | PostgreSQL 方言不兼容且 Raw 不带可信租户 | 改为带租户的模型查询/事务；3,000,000,001 分累计、重复统计、租户隔离通过 |
| T03 / A/B | 品牌 sort 的 required int；地址 defaultStatus 的 required bool | sort=0 / defaultStatus=false 被当作缺失值；地址 false 更新遗漏 | 使用必填指针区分缺失/null/零值；地址默认切换与写入共用事务，显式更新 false。zero-binding-red.log / green.log + HTTP smoke |
| T03 / B | `CartService.GetCartCount` SUM 扫描到 int | 空购物车 PostgreSQL SUM 返回 NULL，接口500 | 使用 sql.NullInt64；空、新增、他人/其他租户、删除后的数量全部验证。cart-empty-red.log / green.log |
| T04 / A/B | 原 CreateOrder 提交后库存/券/积分后置，PayOrder 用根 Query | 后项库存、积分、账本或支付单失败留下部分写入 | repo.InTransaction 显式传播同一 Query；创建/取消资源及本地支付单同事务，条件扣减与状态检查。t04-postgres.log |
| T04 / B | SPU事务内 SKU 服务仍用根连接；结构体更新忽略零值 | SKU后项失败留下部分商品；库存/价格改0丢失 | SPU/SKU CRUD 传播事务，显式选择可写字段，拒绝其他SPU或租户SKU ID。t04-product-postgres.log |
| T09 / A/B | JWT固定密钥、算法/种类缺少严格限定、Redis错误可放行 | 伪造/过期/撤销/refresh当access、Redis宕机 | 环境密钥校验、签名算法/issuer/种类校验、真实Redis白名单、刷新轮换/撤销、类型/权限分离。security-focused-race.log |
| T09 / A/B | captcha Get然后Del；短信随机数/键/消费非原子 | 同一验证码并发重放、多租户相同手机号串用、发送失败仍存在代码 | 原子验证码和凭据消费、按租户隔离、crypto随机、发送确认后激活、失败拒绝。t09-login-security-race.log |
| T09 / C | 无共享登录限流，CORS允许所有Origin，默认信任转发头 | 端点切换/XFF伪造可绕过孤立限制，跨站读取 | 按可信租户/直连IP共享预算、明确Origin环境白名单、默认不信任代理；访问日志只记路径不记Query令牌 |
| T10 / A/B | AuditPlugin只赋值租户，无统一读写约束 | 其他租户ID、批量更新、Raw/子查询、缺租户后台操作 | TenantPlugin统一范围与拒绝不安全SQL；可信身份/域名解析；支付缓存先验证数据库租户/渠道/应用；平台RBAC加载独立只读审计 |
| T10 / A/B | InfraFile/Config没有租户；本地路径拼接不限制根目录 | ../与符号链接越界读删写；跨租户资源路径 | 租户模型/路径/记录查验；os.Root限制文件系统根；物理操作错误不吞；真实PG文件CRUD+跨租户拒绝。file-red.log、file-green.log、infra-integration.log |
| T10 / B | Scheduler保留HTTP或无租户Background，启动错误仅后台日志 | 请求结束后任务丢上下文/写入默认租户 | 按持久化job.TenantID恢复独立有时限上下文，校验活跃租户；日志落库失败拒绝执行；缺租户拒绝。真实PG任务测试 |

## 不将新安全边界当作所有业务已完成

TenantPlugin 对不受支持的 Raw SQL、表覆盖、关联查询和危险 Upsert 采用失败拒绝。首期实际测试的商品、地址、购物车、下单资源、支付缓存、文件、统计和任务路径有单独证据；未经过真实测试的旁支接口不应因全局拦截器存在而标记验收。

平台管理写操作没有启用跨租户豁免。平台只读租户汇总接口要求保留角色、显式目标租户并记录操作审计；离线初始化之外的用户/角色接口不得提权或接管平台账号。服务启动的RBAC加载另有专门只读范围。不能开放任意visit-tenant-id。

## 联调与独立复查补充

- 交易后台原先只有认证、没有逐操作权限：新增交易权限检查及迁移6的38条权限/租户管理员绑定；普通管理员拒绝用例见安全报告。
- 真实前端刷新请求会携带过期access头：只在精确刷新端点让refresh凭据完成校验，不能扩大匿名例外。
- H5预检包含Platform头：精确CORS白名单允许该头；不允许其他Origin。完整服务HTTP smoke验证。
- 同租户不同会员的支付单访问/提交必须核实订单或充值钱包归属；非首期转账同步拒绝。
- 验证码不再写入SQL明文；短信日志页面及Excel导出对历史OTP/手机号/内容/渠道诊断脱敏，见t09-sms-audit-security.md。
- 邮件和站内通知模板全局缓存会跨租户或在默认拒绝查询后为空：移除缓存，使用当前租户查询，邮件证据见t10-mail-tenant-cache.md。
- 登录审计丢弃租户、通知调度函数提前返回导致子任务context被取消：复制可信租户、独立有时限审计；通知执行有界批次并等待写回。对应login-audit-tenant.log、notify-lifetime-red.log/green.log。
