# Agent Runtime & Multi-Agent Architecture

状态：引擎与协作框架 IMPLEMENTED；Chainlake 智能体与委派协议 PROPOSED。日期：2026-11-22。

## 复用结论（先说死）

WeKnora ReAct 引擎（`engine.go:860`）、`ToolRegistry`（`registry.go:18`）、
allowlist（含 nil/empty 语义，`allowed.go:12`）、协作框架
（`collaboration/runtime.go:70` `Spawn` ≤16 child、leases/recovery）**全部复用，不重写**。
Chainlake 只做：新 Tool（注册制）、新 Agent 预设（配置）、委派协议（prompt + artifact 约定）。

## 智能体分工（4.9）

| Agent | 职责 | 关键 Tool（新/复用） |
|---|---|---|
| Coordinator | 接 NL、拆任务、委派、汇总、版本管理 | 复用协作 `Spawn` + artifact 引用 |
| Ingestion | 四路接入、schema 发现、增量、校验、入库注册 | 新 ingestion tools（P2） |
| Engineering | notebook/pipeline 构建与运行、备数 | papermill 运行器、Duckpad/Duckle MCP（外部，SSE） |
| Visualization | 选图、编译查询、生成 Spec、迭代 | `nl2sql_query`、`chart_catalog`、`save_dive`（P5） |
| Knowledge | 口径解释、引用综合 | 复用 `search_knowledge`/`data_schema` |

**不建通用多智能体平台**：目标是"Agents 用共享资源+上下文+artifact 一起干活"，
不是任务编排框架。跨 Agent 状态只经三处：Workspace 上下文、Artifact Registry 记录、协作 Group/Run。

## 通用契约

- Workspace Context：tenant_id + 成员 role + 可见资源集合（每次调用装配）。
- Execution Context：session_id + run_id + sandbox 会话绑定；子任务继承父归属。
- Input/Output Contract：Tool 输入输出 schema 化；大结果只回引用（artifact URI + 预览行数上限）。
- Artifact References：Agent 间传递 artifact_id，不传字节；版本 pin 死（可复现）。
- Delegation Policy：Coordinator 拆分→子 Agent 认领→产物登记→父汇总；子失败可重试/改道/上报三选一。
- Retry and Resume：协作框架 leases + recovery；长任务检查点落 artifact。
- Provenance：ingest→notebook→Dive 全链 artifact 版本链可查（P6 落地视图）。

## MCP 约束（已核实）

`client.go:277-279` 禁用 stdio（安全原因），只用 SSE/HTTP Streamable。
Duckpad（HTTP MCP）可直连；Duckle（stdio）必须经 HTTP 桥（PENDING 选型，
见 `risk-register.md`）。
