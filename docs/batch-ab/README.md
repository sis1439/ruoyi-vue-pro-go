# 批次 A / B 交付入口

任务范围、逐项结果与限制：[验收报告](verification-report.md)。原始依据为商城前端根目录的ruoyi_go_astra_handoff.md。

- [固定版本](baseline.md) / [实际环境](environment.md)
- [缺陷复核与失败证据](audit-findings.md) / [技术决策](decisions.md)
- [接口清单](api-compatibility.csv) / [契约](contracts/wire-contract.md)
- [数据库基线](data-baseline.md) / [映射清单](schema-manifest.yaml) / [数据库验证](database-verification.md)
- [本地交易事务](t04-transactions.md) / [认证及租户边界](t09-t10-security-boundaries.md)
- [运行与测试手册](runbook.md) / [迁移及恢复](database-runbook.md)
- [后续范围和外部输入](blockers.md)

## 本地提交

| 任务 | 提交 |
|---|---|
| T00 构建基线及前端构建证明 | 61a2f36 |
| T01 契约盘点与消费端测试 | 467ddcb |
| T02/T03 PG迁移、类型和查询 | 8185d1f |
| T04 下单及商品事务 | 49150f0 |
| T09/T10 认证、租户、权限、缓存/文件/任务 | b258005 |

这些提交存在明确依赖，验收对象为完整分支，勿只挑选单个任务提交当作可独立部署版本。最终证据与汇总另有docs提交；工作流报告中的“该工作流未提交/父任务待整合”属于当时记录，以这里和最终验收报告为准。

前端独立构建提交：uniapp `51dbba93527830846618c9b319f6163de4d11ba5`；admin `fbd0e3333f6fac63942b2af44a8e104191b23a73`，细节及补丁见对应构建报告。未修改页面业务逻辑。
