# System Architecture

状态：DECIDED（结构）/ 各组件状态见表。日期：2026-11-22。

## 系统图

```mermaid
flowchart TB
    U[User / Agent Client] --> C[Chainlake App\nGo + Vue 3 + TDesign\n(IMPLEMENTED: WeKnora base)]
    C --> IDP[(Logto OIDC\nAuth Adapter: gateway/src/identity/)]
    C --> CAT[Resource Registry +\nData Catalog + Artifact Registry\n(PROPOSED, P0/P1)]
    C --> AGT[Agent Runtime\nReAct + ToolRegistry + Collaboration\n(IMPLEMENTED)]
    AGT --> SL[SemanticCompiler → SLayer sidecar :5143\n(IMPLEMENTED client, EXTERNAL service)]
    AGT --> ENG[Compute + Connectors\n(PROPOSED: Go native + RemoteConnector)]
    ENG --> DB[(User Databases\nPG/MySQL/ClickHouse...)]
    C --> NB[Jupyter Server sidecar\n(PROPOSED, P3)]
    NB --> ENG
    AGT --> PIPE[Pipeline Engine\n(PROPOSED: Papermill/Duckpad/Duckle/Spark/Flink adapters)]
    AGT --> VIZ[Dives / Dashboards\n(PROPOSED, P5)]
    C --> KB[(Knowledge: ParadeDB\nParadeDB+Redis, IMPLEMENTED)]
    C --> OBS[Audit Logs + Langfuse OTel\n(IMPLEMENTED core, exporter own code)]
    PIPE -.-> SCHED[Scheduler: Asynq queues + cron owner\n(IMPLEMENTED queues; periodic = new work)]
```

## 架构库存（3.1）

| Component | Existing Implementation | Responsibility | Status | Evidence |
|---|---|---|---|---|
| App Server | WeKnora Go (Gin) + Vue 3 SPA | REST API、会话、RAG 问答、Agent 编排 | IMPLEMENTED | `chainlake-community/cmd/server`, `internal/router/router.go:105`, `frontend/` |
| Identity Adapter | gateway `src/identity/` (`types.ts` IdentityProvider, `logto-provider.ts`, `zitadel-provider.ts`, `index.ts` env 选择) | token 校验/discovery/JWKS/org 解析/成员 CRUD，按 provider 实现 | IMPLEMENTED | `chainlake-gateway/src/identity/types.ts:65-143`, `logto-provider.ts:196-247`, `index.ts:14-24` |
| Tenant/Workspace | `types.Tenant` + `TenantMember` + 4 级 Role + Organization 跨租户共享 | workspace 建模、成员、角色 | IMPLEMENTED | `internal/types/tenant.go:86`, `tenant_member.go:16-32`, `organization.go:61` |
| RBAC enforcement | role/ownership/share 三类 guard + fail-open 开关 | 路由级鉴权 | IMPLEMENTED | `internal/router/rbac.go:16-113`, `internal/middleware/rbac.go:68,215` |
| AuthN | bcrypt + JWT 双 token；OIDC discovery/PKCE/JIT；capability API Key | 登录/注册/切换租户/刷新 | IMPLEMENTED | `application/service/user.go:161,306,483,543,1210`, `types/tenant_api_key.go:18` |
| Knowledge/RAG | KB CRUD（document/FAQ/Wiki)；混合检索 + rerank；多 RETRIEVE_DRIVER | 知识管理与检索 | IMPLEMENTED | `types/knowledgebase.go:59`, `container/engine_factory.go:156-390` |
| Ingestion pipeline | docreader gRPC 服务 + parser registry + Asynq 队列拓扑 | 文档解析入库 | IMPLEMENTED | `docreader/parser/registry.py:151`, `types/task.go:63-88` |
| Agent runtime | ReAct engine + ToolRegistry + allowlist(nil/empty 语义） | Agent 执行与工具调用 | IMPLEMENTED | `internal/agent/engine.go:860`, `tools/registry.go:18`, `tools/allowed.go:12` |
| Built-in tools | 检索/文档/ask_user/`data_analysis`(DuckDB)/`database_query`(自有三表)/web/MCP | Agent 能力面 | IMPLEMENTED | `tools/definitions.go:26-94`, `tools/data_analysis.go`, `tools/database_query.go` |
| MCP client | SSE/HTTP Streamable；**stdio 被禁用** | 连外部 MCP | IMPLEMENTED | `internal/mcp/client.go:277-279` |
| Collaboration | Runtime.Spawn（≤16 child/group）+ leases/recovery | 多智能体委派 | IMPLEMENTED | `internal/agent/collaboration/runtime.go:70,101` |
| Queues | Asynq 10 队列 + 6 worker 池；cron 目前只有 datasource scheduler 与 automation trigger | 后台任务 | IMPLEMENTED（周期调度为 PROPOSED 新活） | `types/task.go:31-88`, `datasource/scheduler.go:35-96` |
| History | sessions/messages/artifacts（含版本） | 对话与执行存档 | IMPLEMENTED | `types/session.go:89`, `types/message.go:473`, `message_artifact.go:12` |
| Storage | 8 后端抽象 + 租户绑定 + presigned 短链 | 文件存取 | IMPLEMENTED | `storageallowlist/allowlist.go:10`, `router/files.go:524-537` |
| Audit | 通用 `audit_logs`（action/target/details JSONB） | 审计 | IMPLEMENTED | `migrations/versioned/000044_audit_log.up.sql` |
| Secrets | login vault（AES-256-GCM，按用户）+ tenant JSONB 加密字段 | 凭证 | IMPLEMENTED（通用 secret 需适配器，PROPOSED） | `migrations/versioned/000136_credential_vault.up.sql` |
| Sandbox | docker/e2b/cube 后端 + capability 门 + 网络策略 + SSRF guard | 隔离执行 | IMPLEMENTED | `sandbox/capabilities.go`, `types/sandbox_network_policy.go:22` |
| SLayer client | TS BFF（dry_run/save/validate；方言映射 15 种；降级直连 SQL） | 语义编译调用 | IMPLEMENTED（client）；服务 EXTERNAL | `chainlake-gateway/src/api/slayer.ts`, `agents/slayer-convert.ts:85` |
| Semantic models/connectors | gateway TS：models、relationships、connectors(registry) | 模型管理与连接 | IMPLEMENTED | `api/semantic-models.ts:52`, `connectors/connector.ts` |
| Python connector svc | FastAPI（types/test/execute 只读）+ transforms/ssrf | 结构化执行（备选通道） | IMPLEMENTED | `chainlake-connector/src/chainlake_connector/main.py`, `server.py` |
| Observability | OTel 自研 Langfuse exporter（无第三方 SDK）；审计保留策略 | 追踪 | IMPLEMENTED | `internal/tracing/langfuse/exporter.go` |
| Jupyter/Compute/Catalog表/Dive/Notebook UI | — | — | PROPOSED（仓库中不存在） | 无（不得引用为既有） |

## 域边界（4.1 摘要，详见各域文档）

| Domain | Owns | Must NOT own |
|---|---|---|
| Application & Identity | 会话、租户上下文、IdP 适配调用 | IdP 内部逻辑（adapter 后） |
| Workspace & Authorization | 成员/角色/资源归属判定 | 资源内容本身 |
| Data Sources & Integration | 连接定义、连通性、采样执行通道 | 数据语义（归 SLayer）、数据存储 |
| Data Catalog & Resource Registry | 身份注册、元数据组织、发现 | 物理数据集（只存引用） |
| Knowledge & Retrieval | 解析/索引/混合检索 | 结构化查询执行 |
| Data Engineering & Compute | notebook/pipeline 定义与执行、计算会话 | 业务口径（归 SLayer） |
| Agent Runtime & Tool Execution | ReAct 循环、工具注册/调用、委派 | 工具背后的业务系统 |
| Semantic Layer | measures/dims/口径、NL→SQL 编译 | 执行与存储 |
| Visualization & Analytics | Intent→Spec→ECharts、Dive/Dashboard 生命周期 | 取数执行 |
| Artifact Registry | 产物元数据/版本/血缘/状态 | 产物字节（存对象存储/数仓/KB） |
| Workflow & Scheduling | 触发、重试、运行记录 | 执行语义（归各引擎） |
| Observability & Audit | 追踪、审计、保留策略 | 业务审计内容定义（各域 emit） |

重叠消除：Catalog 管"可发现资产"、Artifact 管"执行产出"（同一物化数据集以引用双挂，不拷贝）；
Pipeline 管"定义与调度"、Compute 管"执行会话"、Scheduler 只管"触发与记录"。
