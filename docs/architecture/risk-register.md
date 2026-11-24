# Decision & Risk Register

日期：2026-11-22；修订：2026-11-24（语义层升级为必需件，三档剖面取消，迁移改单轨）。
随验证进展更新；关闭 PENDING 需回写对应文档。

## 可执行决策（Ready）

- R1：fork WeKnora v0.9.0 为基座，RAG 原样复用（ADR-001）。
- R2：单 PUBLIC 仓 + 独立 PRIVATE 仓；本仓禁商业代码（ADR-002）。
- R3：User→Workspace(=Tenant)→Resources；复用 Role/Organization（ADR-004）。
- R4：双通道（Knowledge/Data）分离（ADR-006）；Notebook≠Pipeline（ADR-007）。
- R5：执行安全统一模型（校验+最小权限+超时+配额+审计），纠正注入假设（ADR-008）。
- R6：Intent→Compiler→ECharts 唯一渲染（ADR-011）；三级可视化模型。
- R7：单一完整剖面 + 可选加速件（ADR-012′，取代 ADR-012）。
- R8：Artifact 三层模型：呈现层 `message_artifacts`（不改）/ 注册层 `artifacts`（新表）/
  字节层 `resources` + `resource_bindings`（ADR-013）。
- R9：迁移单轨 PostgreSQL + 独立 source `migrations/chainlake/pg` + 独立版本表（ADR-014）。
- R10：语义层只经 `SemanticCompiler` 接口接入，模型以 Ossie 兼容为目标序列化（ADR-018）。
- R11：AI Context 不进入 SQL 生成路径（ADR-019）。

## 待验证决策（Pending：事项 / 为何重要 / 关闭条件）

- P1：Logto **经本仓内置 OIDC** 的兼容性（discovery / PKCE / userinfo email / JIT / 默认租户）。
  重要性：登录是 P0 门禁；路径是 `OIDC_AUTH_*` 配置，不是网关 adapter。关闭：联调通过 + 回写 identity 文档。
- P2：生产 Logto tenant 实际配置（endpoint / claim 映射 / 默认租户 / 单点登出）。
  重要性：Community 与 SaaS 身份栈不同（EXTERNAL adapter）。关闭：部署配置确认。
- P3：Jupyter Server 票据/WS/隔离/生命周期设计。重要性：notebook 是 P3 核心，不设计不能开工。
  关闭：`data-engineering.md` 更新 + 安全评审。
- P4：连接凭证表结构与 workspace 删除级联清单。重要性：涉及数据归属与清理正确性。
  关闭：P1 设计评审。
- P5：Duckle MCP HTTP 桥方案。重要性：决定 sidecar 集成成本。关闭：P4 前原型验证。
- P6：Asynq 调度负载。重要性：决定是否引入 K8s CronJob。关闭：P7 压测。
- P7：密钥轮换流程。重要性：生产必备。关闭：P7 手册。
- P8：语义层分发形态（镜像来源 / 许可 / 源码可得性）。重要性：语义层是**必需件**，
  P1 起全链路依赖，是发行阻断项。关闭：P0 结束前给出书面结论（分发方式 + 版本 pin）；
  与 ADR-018 的候选（Cube Core / SLayer）一并定。
- P9：语义层**程序化建模**能力（Go 写模型 → 不重启即时生效）。重要性：数据持续 ingestion、
  schema 事先未知，模型必须能被 Agent 生成；若不过，语义层只能做"人工建模的治理层"。
  关闭：实测记录（Cube `repositoryFactory`/reload 或 SLayer 写模型后生效）。
- P10：语义层与 in-process DuckDB 的并发拓扑。重要性：DuckDB 单写者，`data_analysis` 已在
  Go 进程内独占，独立进程再连同一库文件会冲突。关闭：确定"Parquet 只读 + 语义层走 PG"
  或等价方案，并给出延迟与一致性结论。
- P11：开源版多租户 RLS 可用性（access policies 是否在 Core 免费）。重要性：决定是否自写
  `queryRewrite`。关闭：行级/列级/多租户实测矩阵通过，或给出替代实现与工作量。
- P12：语义层"只编译不执行"能力（ADR-018 拓扑 A 判定）。重要性：决定凭证与审计边界归属
  （ADR-008）。关闭：`sql()` / schema-compiler 仅产出 SQL 的可行性结论；不可行则正式切
  拓扑 B 并改写执行安全文档。

## 技术风险

- T1：上游高速迭代导致同步冲突（0.x，每周 PR 量大）。缓解：只加文件+注册行；`sync/*` + 门禁。
- T2：ParadeDB 版本漂移（compose v0.25.11 vs helm v0.18.9）。缓解：P7 前统一并回归。
- T3：前端无 echarts，新增依赖增包体积。缓解：路由懒加载 + 按需引入。
- T4：DuckDB in-process 资源尖峰。缓解：并发上限 + 大查询走远程引擎。
- T5（2026-11-23 实证）：KnowledgeQA 回答预算取全局 `summary.max_completion_tokens`
  默认值（曾为 1024），不随模型行/目录值变化（行级值只进 AgentQA 路径）。后果：
  任何长回答或带推理的回答都可能撞上限 `length`。缓解：已将默认值提至 16384；
  长期修复为将 `modelMaxOutputTokens` 接入 KnowledgeQA `prepareChatModel`
  （需改上游文件，走 PATCHES 流程，P1 范畴）。
- T6：语义层成为 SQL 正确性单点（sidecar 不可用则指标问答全断）。缓解：降级为直连 + `ExecutionGuard`
  并在结果上显式标注"未经语义层治理"；`SemanticCompiler` 契约测试保证可替换。
- T7：语义层与 `data_analysis` 的 DuckDB 资源争用（单写者）。缓解：见 P10；默认让交互层与
  语义层不共享同一 DuckDB 文件，交换走 Parquet。

## 外部依赖风险

- E1：语义层为外部项目（Cube Core 或 SLayer），版本/接口/许可均可能漂移；SLayer 尚处 0.7.x 预 1.0。
  缓解：`SemanticCompiler` 接口隔离 + 版本 pin + 契约测试 + 分发形态结论（P8）。
- E2：Duckpad/Duckle 为外部项目（版本/接口漂移）。缓解：pin 版本 + MCP/HTTP 契约测试。
- E3：Logto 配置漂移（Community 经内置 OIDC 接入，不再是网关 adapter）。缓解：配置契约测试 +
  启动校验（`OIDC_AUTH_*` 必填项）。
- E4：Ossie（原 OSI）标准仍在 Apache 孵化期，字段与语义可能调整。缓解：内部 spec 与 Ossie
  之间用 adapter 转换，模型文件不直接等同于 Ossie 文件（ADR-018）。

## 安全风险

- S1：Jupyter 内核逃逸（ mitigations 见执行安全层；设计未关闭前禁上线）。
- S2：密钥泄露面（cell 日志/产物/分享链接）。缓解：别名制 + 输出脱敏 + 分享 token 审计。
- S3：MCP 外部工具权限放大。缓解：沿用审批卡 + capability 白名单。
- S4：默认密钥（JWT/AES/SEARXNG）未轮换。缓解：部署清单强制项（P7）。
- S5：语义层成为绕过 `ExecutionGuard` 的取数后门（拓扑 B 风险）。缓解：优先拓扑 A；若落 B，
  必须把 guard 前移为语义层前置网关，并为语义层单独配只读凭证、行数上限与配额。
- S6：语义层自身配置面风险：`CUBEJS_DEV_MODE=true` 是**认证绕过**（官方明确警告：无 token、
  任意 SQL、模型可被改写）、CORS 默认 `*`、匿名使用统计默认开启。缓解：部署清单强制项 +
  启动自检（检测到 dev mode 直接拒绝启动）。

## 许可风险

- L1：容器镜像许可未逐个核实（searxng/minio/neo4j/redis 谱系/clickhouse/langfuse/busybox 等）。
  **CE 发版前必须逐个清理**，否则不得发布制品。
- L2：SheetJS xlsx 走 CDN tarball（非 npm 源），确认 Community/Pro 范围与可再分发性，考虑 vendoring。
- L3：MPL-2.0（go-sql-driver 等）要求保留 LICENSE + notices + sources 压缩包（已有脚本，保持进 CI）。
- L4：参考项禁区：Sling GPL-3.0 / Steampipe 核心 AGPL / Metabase AGPL / havn BSL——
  只许借鉴思路，禁抄代码禁嵌入（havn 特别注意 hosted 竞品限制）。
- L5：**AGPL-3.0 禁入清单（已核实，2026-11-24）**：Wren AI、Sidemantic、Metabase。
  语义层选型时这两个项目只可借鉴设计（Schema Linking / 多格式转换），禁止嵌入或与其共进程分发。
- L6：语义层候选许可：Cube Core 后端 Apache-2.0、client MIT；SLayer MIT。
  容器镜像内部依赖仍需逐个核实（并入 L1）。

## 部署风险

- D1：sidecar sprawl（越加越多）。缓解：ADR-012′ 收敛为"必需集 + 可选件清单"，
  健康检查按必需/可选两档分级，可选件缺失不阻塞核心链路。
- D2：SQLite 系组件（SLayer/Lite DB）的单副本与备份。缓解：备份矩阵 + RWO 约束文档化。
- D3：升级/回滚演练缺失。缓解：P7 必演练项。
