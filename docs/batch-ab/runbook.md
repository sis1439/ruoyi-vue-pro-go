# 批次 A / B 使用手册

## 交付位置

Go修改位于 `/Users/macmini/Desktop/yudao-mall-uniapp/.work/ruoyi-vue-pro-go`，分支`codex/batch-ab`。本批次未向原 `/Users/macmini/Desktop/ruoyi-vue-pro-go` 写入修复；2026-09-08收尾时原目录已有其他未提交改动，后续整合需保留这些内容。前端构建副本、固定提交和可应用补丁见frontend-build.md及frontend-admin-build.md；不要把生成的dist放入后端源码。

## 生成、构建、检查

使用baseline/environment记录的Go工具链。在Go副本根目录执行：

```sh
rtk go version
rtk make deps
rtk make gen
rtk make wire
rtk make verify
```

`make build`和`make test`会先生成被忽略的DAO。无需私有包；手写事务帮助代码位于`internal/repo/transaction.go`，不放入可重新生成的query目录。沙箱环境可把GOCACHE和GOPATH设为environment.md的独立临时路径。

## 数据库初始化及启动

按database-runbook.md准备专用PostgreSQL/Redis，运行版本迁移并用环境变量显式创建管理员。复制config/config.example.yaml到被忽略的config/config.local.yaml，修改专用Redis地址/DB。设置RUOYI_DATABASE_DSN、独立随机RUOYI_JWT_SECRET及初始化用RUOYI_BOOTSTRAP_USERNAME/PASSWORD/WEBSITE；不把口令写进Git或命令输出。

```sh
rtk go run ./cmd/migrate
rtk go run ./cmd/bootstrap
rtk make build
rtk proxy ./server
```

bootstrap不会重置已有账号密码。host映射缺失时匿名登录返回租户未授权，这是失败拒绝，不应通过信任任意tenant-id绕过。http.port须为`:48080`形式。跨域访问仅在MALL_CORS_ORIGINS中列出精确Origin；缺省不发送允许跨域的响应头，禁止`*`。反向代理上线前配置可信代理策略，否则登录限流按代理的直连IP合并计算。

## 真实依赖回归

先配置TEST_POSTGRES_DSN、T09_REDIS_ADDR（本轮值见environment.md），测试会创建独立schema，不把缺少环境导致的skip算作通过。

```sh
rtk go test ./...
rtk go test -race ./internal/middleware ./internal/pkg/permission ./pkg/database ./pkg/utils ./internal/service/system ./internal/service/member ./internal/service/mall/product ./internal/service/mall/trade ./internal/service/pay ./internal/service/infra
rtk python3 -m unittest discover -s docs/batch-ab/contracts/tests
rtk node docs/batch-ab/contracts/tests/frontend-contracts.cjs
```

HTTP smoke创建独立schema、随机临时管理员，启动完整服务，验证登录/权限/商品/地址/购物车/租户拒绝/刷新/登出，并检查服务日志没有口令或令牌。新增租户详情/分页/Excel导出和任务创建/更新/停用/恢复/删除/同步验证；暂停临时schema中的种子任务，测试任务只使用未来调度时间。其测试配置只监听本机58091；运行前保证该端口空闲。

```sh
rtk proxy mkdir -p tmp/batch-ab
rtk go build -o tmp/batch-ab/server ./cmd/server
rtk go build -o tmp/batch-ab/migrate ./cmd/migrate
rtk go build -o tmp/batch-ab/bootstrap ./cmd/bootstrap
rtk python3 scripts/batch_ab_smoke.py
```

备份/恢复和升级演练命令见database-runbook.md。恢复必须使用匹配主版本的pg_dump/pg_restore；不要使用本机默认PG14工具处理PG16测试库。

## 排障与停止

- 任务接口报告“已保存，调度同步失败”时，数据库目标状态已经提交；错误包含任务ID。刷新列表确认状态，恢复调度器后用当前租户管理员调用`POST /admin-api/infra/job/sync`重试。同步包括停用和软删除记录，可清除失败残留的定时器，重复同步不会增加定时器。不要重复创建同一处理器。当前实现适用单进程调度，不能据此宣称已支持多实例选主或强制终止正在运行的处理器。
- 启动失败先检查密钥、数据库driver/DSN、迁移版本、Redis及域名映射；不得临时关闭认证/租户检查。
- 发现迁移dirty时保留数据库和错误日志，定位失败SQL后按备份恢复或经审核的前向修复处理。禁止盲目force版本后当作成功。
- 登录/验证码遇到503需恢复Redis；429需等待Retry-After。不要转为无白名单登录。
- 库存/积分/券或本地支付失败应整笔回滚；查询相关表确认不变式，见T04测试断言。不直接把订单改为已支付。
- 测试PG/Redis临时目录可重建。确认不再需要本轮实例后按PID停止各自进程；不要停掉机器上其他业务数据库或Redis。
- 应用回退用前一提交重新构建；已应用迁移不逆向删表。破坏性恢复使用独立演练通过的备份流程。

后续C/D批次及真实渠道资料见blockers.md。这里提供工程复现流程，不授权生产发布。
