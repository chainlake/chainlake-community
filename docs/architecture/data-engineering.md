# Data Engineering Architecture

状态：DECIDED（概念与安全）/ Compute/Jupyter/Notebook UI/调度器为 PROPOSED。日期：2026-11-22。

## Notebook vs Pipeline（必须区分）

- **Notebook**：交互式 SQL/Python 开发与探索；支持单步执行与结果检查；可产出可复用代码与 artifact。**不是**调度单元。
- **Pipeline**：版本化、可重复的工作流；含节点、依赖、参数、执行策略；支持调度、重试、恢复、执行历史。notebook 可被封装为 pipeline 节点（papermill 参数化执行），反之不行。

## Pipeline Model（Chainlake 自有，引擎可替换）

定义（YAML/JSON，存 git）：nodes（sql|python|notebook|subpipeline|approval）、edges（含条件）、
triggers（cron|manual|event）、runs（run_id/状态/发起人/产物）。执行引擎为适配器：
DuckDB（单步/采样）、Papermill/nbclient（notebook）、Duckpad（交互备数）、
Duckle（大吞吐/定时）、Spark（Spark Connect）、Flink（SQL Gateway）。
**Pipeline 定义不得绑定任一第三方产品格式**；Duckpad/Duckle 以引擎适配器接入。

## Compute（执行目标抽象）

notebook attach 的是 Compute（registry + session + 凭证发放 + idle 回收），不是语言。
DuckDB 本地/远程、Spark Connect、Flink SQL Gateway 均为 Compute 实现。
新增引擎 = 新实现 + 会话管理，Pipeline/Notebook 前端不动。

## Compute 与执行引擎分工（已定案 DECIDED）

- **notebook 交互层 = DuckDB in-process**：零进程启动开销、DataFrame 进程内共享、
  延迟最低；WeKnora 现有 `data_analysis` 同构。覆盖即席查询、采样预览、小中量 Python 互通。
- **pipeline 调度层 = Duckle runner**：子进程隔离（崩只死一个 run）、流式执行、
  多核、锁死环境可复现；覆盖批量、定时、大吞吐。
- **两层经 Parquet/共享存储换数据**：备数区为交换格式，Catalog 登记表位置；
  notebook 与 pipeline 互引同一份表，不倒数据。

## 调度

默认 Asynq 队列 + **新增**周期触发器（现状只有 datasource scheduler 与 automation trigger 的
cron 模式可借鉴，`datasource/scheduler.go:35-96`；**通用 Asynq-periodic 不存在，需新建**）。
K8s CronJob 为备选（重型场景）。notebook 定时跑、pipeline 定时跑共用同一触发器与 run 记录格式。

## 执行安全（4.7，所有执行面统一）

适用于 Agent SQL Tool、notebook cell、pipeline 步骤、定时任务：

1. **Workspace 隔离**：会话归属 + 资源可见性门控。**纠正旧假设**：
   `tenant_id` 注入仅对 WeKnora 自有表有效（如 `database_query.go` 模式）；
   **对任意外部数据库不得假设注入即隔离**，必须用：DB 侧最小权限凭证 + 只读角色默认 +
   SQL 解析校验 + 全量审计。
2. **SQL 校验是必要非充分**：allowlist/危险函数/单语句之外，还需超时、行数上限、资源配额。
3. **Python 不靠 URL 白名单保底**：容器级隔离 + 非 root + 文件系统限制 + 出站 egress 策略
   + 资源 quotas + 可取消；Jupyter kernel 跑在受限 sidecar，kernel 故障域与主服务隔离。
4. **密钥**：vault/连接凭证不进 cell、不进日志；notebook 只见别名。
5. **审计与清理**：每次执行记（谁/哪台 compute/跑了什么/耗时/行数）；idle 会话回收；
   workspace 删除级联清理（清单 P1 定稿，PENDING）。

Jupyter Server 自身设计（认证票据/WS 代理/kernel 隔离/生命周期）见
`identity-and-authorization.md` Quant；细节在 P1/P3 关闭前均为 PENDING，不得先写代码依赖。
