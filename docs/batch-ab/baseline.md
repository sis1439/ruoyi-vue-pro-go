# 批次 A / B 固定基线

执行日期：2026-09-07。需求来源：商城前端根目录 `ruoyi_go_astra_handoff.md` 的批次 A、B。

| 仓库 | 固定提交 | 用途 |
|---|---|---|
| wxlbd/ruoyi-vue-pro-go | `4a240b8b15d54f77d439b631417d35630c27ad4a` | Go 原始基线，与任务书一致 |
| YunaiV/ruoyi-vue-pro | `01a0eaf573a962eda54c83e46551cdd6bfbf207c` | Java Controller / VO 正式参照 |
| yudaocode/yudao-ui-admin-vue3 | `aab14fb0e74720dd09e964ae066f8bbde9f9012e` | 管理后台契约参照 |
| yudaocode/yudao-ui-admin-vue3（实际生产构建） | `2e001992486e69a464c7ba0b22f110040debe9a2` | 官方可构建父提交；隔离提交 `fbd0e3333f6fac63942b2af44a8e104191b23a73` 仅增加依赖构建策略 |
| yudaocode/yudao-mall-uniapp | `3c4bf3864415054a88fe616a414e098972329412` | 实际商城前端，页面业务代码保留 |

参照文件与许可证哈希见 `contracts/references.json`。本机 `dm-admin` 是其他 Go 管理后台，未将其冒认为芋道 Vue 前端；未修改该仓库。Java 与后台选择固定提交以获得可追溯参照，**尚未用页面端到端测试证明配套兼容**。

## 隔离与已有修改

原 Go 仓库 `/Users/macmini/Desktop/ruoyi-vue-pro-go` 工作树干净；原前端只有未跟踪的交接文档。本次使用前端 `.work/ruoyi-vue-pro-go` 中的独立 Git 副本，分支 `codex/batch-ab`，原仓库未覆盖。Go module 继续使用 `github.com/wxlbd/ruoyi-mall-go`。

商城前端构建在 `.work/uniapp-build` 隔离副本完成。其构建专用修改、提交及补丁见 `frontend-build.md`；原前端页面不修改。

## 初始构建复核

1. 初次命令遇到宿主默认 Go cache 的沙箱写权限，调整为独立临时缓存；这不是项目编译缺陷。
2. DAO 路径 `internal/repo/query` 被忽略，干净 checkout 必须先运行 `go run cmd/gen/generate.go`。不应 `go get` 自己的未生成 internal 包。
3. 在另外一个未修改的基线副本生成 DAO 后，`go test ./...` 通过，12 个包包含测试；`go build ./...` 通过。证据 `evidence/baseline-generated-tests.log` / `baseline-build.log`。
4. `go mod verify` 通过。原查询生成器与 GORM Gen 已固定在 go.mod；Makefile 改为构建/测试先生成 DAO，Wire 命令固定 `v0.7.0`，依赖检查不再隐式 tidy。

现有测试通过不等于库存、认证或多租户正确。新增集成证据和各项限制见 `verification-report.md`。

## 管理后台实际构建与契约等价

管理后台完整生产构建、清空 node_modules 后的 frozen-lock 安装及重建均通过，2,468 个产物文件字节一致。Node `25.8.1`、pnpm `11.19.0`；运行手册和日志见 [frontend-admin-build.md](frontend-admin-build.md)。原契约参照 `aab14fb` 保留；实际官方构建基线为 `2e001992`。两者 package.json、pnpm-lock.yaml、148 条已选 API 对应的 26 个源码文件及数据库六个菜单组件全部字节一致，故无需更改既有 API 引用或 SQL 菜单路径。逐文件哈希及完整差异见 [契约等价证据](evidence/t00-admin/contract-equivalence.json)。
