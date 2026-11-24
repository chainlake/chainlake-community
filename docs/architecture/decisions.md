# Architecture Decision Records

日期：2026-11-22；修订：2026-11-24（新增 ADR-012′/013/014/018/019，ADR-012 被取代）。
状态以各条 Status 为准（Accepted / Pending / Deferred / Superseded by ADR-xxx）。
编号不重排：被取代的 ADR 保留原文，只改 Status，避免历史决策丢失。

## ADR-001 WeKnora 复用策略

- Context：需要 RAG/知识/多租户/Agent 底座，自研成本高。
- Decision：fork WeKnora v0.9.0（commit `7a3ca4990`），RAG 引擎与现有能力原样复用、
  只做挂接；Chainlake 领域以新模块实现。
- Alternatives：自研全栈（否决：成本）/ 仅当库引用（否决：深度定制困难）。
- Consequences：承担上游同步成本；受上游架构约束；需 FORK/PATCHES 纪律。
- Status：Accepted。

## ADR-002 Community vs Enterprise vs Cloud

- Context：公开社区版与商业形态需隔离。
- Decision：单仓 `chainlake-community`（PUBLIC）承载全部社区能力；
  企业/云能力进独立 PRIVATE 仓，经扩展点集成，不在本仓出现计费/租户开通/托管代码。
- Alternatives：特性开关同仓（否决：许可证与商业污染风险）。
- Consequences：扩展点设计需前置；企业功能不得回流本仓。
- Status：Accepted。

## ADR-003 Logto/Auth Adapter（身份）

- Context：SaaS 已转向 Logto；网关有 `src/identity/` provider 抽象（Logto 已实现、
  Zitadel 为遗留 fallback；代码默认仍为 zitadel）。
- Decision：社区架构以 Logto + Auth Adapter 为准；业务只认 `Identity` 契约；
  不得新增 provider 分支；Zitadel 引用一律标 HISTORICAL。
- Alternatives：直连 Logto（否决：耦合 provider）；保留 Zitadel 默认（否决：与 SaaS 分叉）。
- Consequences：需实测 OIDC 兼容（PENDING）；生产默认需切 `AUTH_PROVIDER=logto`（部署项）。
- Status：Accepted（架构方向）；联调细节 Pending（见 P3）。

## ADR-004 Workspace 与租户模型

- Context：需要 User→Workspace→Resources 的协作边界。
- Decision：复用 `Tenant`（即 Workspace）+ `TenantMember` 四级 Role + Organization 跨空间共享；
  不引入 Project/新抽象。
- Alternatives：新建 Workspace 表（否决：重复现有模型）。
- Consequences：租户配置 JSONB 会继续膨胀，需治理；跨空间语义受 Organization 限制。
- Status：Accepted。

## ADR-005 Resource Registry 与 Data Catalog

- Context：资产（库/表/卷/KB）需要统一发现与归属。
- Decision：Registry 管身份归属、Catalog 管元数据组织发现；外部表零拷贝注册；
  Volume 双用途（存文件/入知识库）。
- Alternatives：单一大表（否决：职责混杂）；复用 KB 表存一切（否决：语义污染）。
- Consequences：P0/P1 新表；需 catalog-agent 保持元数据新鲜。
- Status：Accepted（模型）；**修订（2026-11-24）**：Resource Registry 已由上游实现
  （`migrations/versioned/000069_resource_registry.up.sql`），P0 只需增 Catalog 元数据表 +
  `artifacts` 表（ADR-013），不再新建 Registry。

## ADR-006 Knowledge vs Data 处理分离

- Context：同一文件可既做知识又做数据。
- Decision：双通道独立（Knowledge 走现有 ingestion；Data 走新链），共享源 blob 与 Catalog 发现层。
- Alternatives：统一处理管线（否决：两种语义硬捏会导致两边都难用）。
- Consequences：需 hash 去重与双登记一致性。
- Status：Accepted。

## ADR-007 Notebook vs Pipeline

- Context：交互探索与版本化调度的混淆是常见失败根源。
- Decision：Notebook 是交互单元，Pipeline 是版本化调度单元；notebook 可封装为 pipeline 节点，
  反之不行；共用执行与产物层。
- Alternatives：notebook 直接可调度（否决：参数/版本/审计缺失）。
- Consequences：需 papermill/nbclient 封装层与 run 记录。
- Status：Accepted。

## ADR-008 SQL/Python 执行安全

- Context：多个执行面需要统一安全。
- Decision：校验+最小权限凭证+超时+配额+审计全覆盖；**纠正旧假设**：
  `tenant_id` 注入仅对自有表有效，外部库必须用 DB 侧最小权限；Python 不靠 URL 白名单保底。
- Alternatives：纯 SQL 解析（否决：不充分）；纯容器隔离无治理（否决：不可审计）。
- Consequences：连接器必须支持只读角色；Jupyter 需独立加固设计。
- Status：Accepted（原则）；Jupyter/级联清单细节 Pending。

## ADR-009 Pipeline 执行与调度

- Context：需要定时/重试/恢复与执行历史。
- Decision：默认 Asynq 队列 + 新增周期触发器；K8s CronJob 为备选；引擎全适配器化。
- Alternatives：引入 Airflow/Temporal（否决：过重）；纯 K8s CronJob（否决：run 记录与重试语义弱）。
- Consequences：需新增通用周期调度器（现状只有 datasource/automation cron 可借鉴）。
- Status：Accepted（方向）；Asynq 负载验证 Pending（P7）。

## ADR-010 Artifact Registry

- Context：产物需要元数据/版本/血缘管理。
- Decision：注册表只存元数据与引用（URI+hash），字节留在原系统；版本链不可变；状态机管理生命周期。
- Alternatives：产物入库（否决：存储爆炸）；无注册表（否决：不可复现）。
- Consequences：所有执行面必须登记 artifact（Agent/notebook/pipeline/Dive 统一）。
- Status：Accepted（模型）；字段由 **ADR-013** 定稿（新建 `artifacts` 表 + 字节层复用 `resources`）。

## ADR-011 可视化模型

- Context：Agent 生成图表需要可靠、可编辑、可复现。
- Decision：Intent（~15 字段）→ Compiler → ECharts 唯一渲染；Agent/编辑器读写同一 Spec；
  Dive（交互）/Chart（组件）/Dashboard（组合）三级分立。
- Alternatives：直出 option（否决：幻觉+失控）；Vega-Lite（否决：与 ECharts 双栈）。
- Consequences：需 Compiler 与 Schema 校验；模板库持续建设。
- Status：Accepted（模型）；模板库建设中（P5）。

## ADR-012 部署剖面

- Context：自托管必须开箱可用，又不能强制重依赖。
- Decision：Lite（核心）/ Standard（+Jupyter/调度）/ Advanced（+重组件）三档；
  缺 sidecar 不得阻塞核心。
- Alternatives：单体全量（否决：重）；全微服务（否决：运维负担）。
- Consequences：compose/helm 需 profiles 矩阵与健康分级。
- Status：**Superseded by ADR-012′**（2026-11-24：语义层为必须件，Lite 单机零依赖形态不成立）。

## ADR-012′ 单一完整剖面（取代 ADR-012）

- Context：语义层（`SemanticCompiler` → sidecar）是"Agent 生成可靠 SQL"的必要件，
  而 ADR-012 的 Lite 档（单应用、零外部依赖）装不下它，两者不可同真。
  三档剖面还会在前端 capability、迁移轨、部署文档三处产生分支，维护成本高于收益。
- Decision：Community 只有**一个完整剖面**：App(Go) + PostgreSQL(ParadeDB) + Redis +
  对象存储 + 语义层 sidecar。可选加速件（Duckpad/Duckle/Spark/Flink/Redpanda Connect/
  Jupyter Server）默认关闭，缺失不阻塞核心链路。
  `deploymentCapabilities` 的语义从"裁剪基础功能"改为"可选件是否就绪"。
- Alternatives：保留三档（否决：语义层无降级形态）；取消 capability 机制
  （否决：可选件缺失时前端报错而非隐藏）。
- Consequences：SQLite 单机形态不支持 data platform 能力（capability fail-closed）；
  `docs/LITE.md` 需标注"上游 Lite 形态不含 data agent"；
  `frontend/src/config/deploymentCapabilities.ts` 的 liteMode 特判需清理。
- Status：Accepted。

## ADR-013 Artifact 三层模型（呈现层 / 注册层 / 字节层）

- Context：上游 `message_artifacts` 的语义是"一次 assistant turn 产出的文件"，
  生命周期随消息与会话，不能承载 Artifact Registry。证据：
  `internal/application/repository/message_artifact.go:43` 每次写消息 **先 DELETE 再全量重插**，
  行是易失的；`migrations/versioned/000103_message_artifacts.up.sql` 把
  `(message_id, position)` 设为唯一且 `position` 即下载地址；
  `000107_message_artifacts_deleted_at.up.sql` 注释说明 ArtifactCollector 依赖 sandbox mtime
  会"重新挂回"被删文件。以上与"不可变版本链 + 状态机 + workspace 级归属"直接冲突。
  上游另有 `resources` / `resource_bindings` / `resource_access_grants`
  （`migrations/versioned/000069_resource_registry.up.sql`）与
  `internal/application/service/resource.go:188 Bind()`、`:316 CreateAccessGrant()`，
  已提供内容寻址去重、引用计数、presigned 与分享 token。
- Decision：三层分立，各司其职，`message_artifacts` **一行不改**：
  1. 呈现层 = `message_artifacts`（"这轮回答给用户看的文件"，位置即地址，可被用户删除）；
  2. 注册层 = 新表 `artifacts`：`workspace_id, logical_id, version（UNIQUE(logical_id,version)）、
     artifact_type, status(draft/validated/published/deprecated), schema_version,
     resource_id, file_name/file_type/file_size, created_by, created_by_agent,
     session_id?, message_artifact_id?, run_id?, inputs, code_version, params, deleted_at`；
  3. 字节层 = 上游 `resources`（`content_hash` 去重）+ `resource_bindings`
     （新增 `owner_type=artifact` 常量，参与引用计数）+ `resource_access_grants`（分享/嵌入）。
  单向桥：只允许 `artifacts.message_artifact_id` 指向呈现层，禁止反向写入呈现层。
- Alternatives：扩展 `message_artifacts`（否决：易失行、无 message 场景无法插入、
  破坏 position 语义、必须改上游 Go 代码）；产物只发事件不入表（否决：不可复现、无血缘）。
- Consequences：新增 `ResourceOwnerArtifact` 常量与 GC 判定；
  删除会话只减绑定引用，**不得回收仍被 artifact 引用的字节**；表结构在 P0 定稿。
- Status：Accepted（字段定稿 Pending，P0）。

## ADR-014 迁移单轨 + 独立 migration source

- Context：上游维护两套迁移：`migrations/versioned`（PostgreSQL，已到 `000137`）与
  `migrations/sqlite`（已到 `000053`），同一张表要写两份 up/down 且方言不同
  （`JSONB`↔`TEXT`、`TIMESTAMPTZ`↔`DATETIME`，见 `000136` 与 `000052` 对照）。
  上游每周新增迁移，Chainlake 沿用 `000138` 必然撞号。
  `internal/database/migration.go:110` 按 DSN 前缀二选一加载。
- Decision：① data platform 新表**只支持 PostgreSQL**（ADR-012′ 的连带结论），
  SQLite 部署下整体 capability fail-closed；② Chainlake 使用**独立迁移源**
  `migrations/chainlake/pg/`，编号自 `000001` 起，独立版本表 `chainlake_schema_migrations`；
  ③ 用 `migrate.NewWithDatabaseInstance` + driver 的 `MigrationsTable` 字段
  （`golang-migrate` 的 postgres 与 sqlite3 driver 均支持）实现，
  **不修改 `internal/database/migration.go`**，零 PATCHES；
  ④ 在 `RunMigrations` 之后调用一次（外键依赖 `tenants`/`resources`，必须先跑上游）；
  ⑤ 独立脚本 `scripts/migrate-chainlake.sh`，不改上游 `scripts/migrate.sh`。
- Alternatives：保留双轨（否决：一半工作量耗在方言兼容）；沿用上游目录但占高位号段
  `000500+`（否决：需长期人工仲裁，`migrate down` 语义易混淆）；
  改上游 `migration.go` 合并两个 source（否决：PATCHES 面变大，同步冲突风险高）。
- Consequences：系统信息接口只显示上游版本，Chainlake 版本单独暴露；
  Chainlake 迁移失败须 fail-fast 并给独立的 dirty 恢复命令；
  门禁需覆盖"空库 → 上游 up → chainlake up → down 1 → up"与"上游新增迁移后两轨互不影响"。
- Status：Accepted。

## ADR-018 语义层选型与拓扑

- Context：需要语义层保证 Agent 生成 SQL 的正确性与可治理性。候选与判定：
  Cube Core（Apache-2.0，成熟度/预聚合/PG 线协议最强）、SLayer（MIT，模型形态最贴 Agent，
  但 0.7.x 预 1.0、单一公司）、Wren AI（**AGPL-3.0**，且整体是 GenBI 产品）、
  Sidemantic（**AGPL-3.0**）、MetricFlow（Apache-2.0，但强依赖 dbt 工程形态）、
  Malloy（Apache-2.0，语言级，Agent 生成成本高）。
  AGPL 项目撞 `risk-register` L4，只可借鉴设计。
  同时存在未被澄清的拓扑问题：语义层是"编译器"还是"取数入口"，
  决定凭证与执行安全边界归属（ADR-008）。
- Decision：
  1. 只经 `SemanticCompiler` 接口调用；**ADR 与代码中的接口命名不得出现第三方产品名**，
     模型以 semantic member（entity / dimension / metric / policy）表达。
  2. 模型序列化以 **Ossie**（Apache-2.0，原 Open Semantic Interchange，基于 MetricFlow）兼容为目标；
     内部 spec 不直接等同于 Ossie 文件，用 adapter 隔离标准演化。
  3. 拓扑二选一，由 P0 spike 判定（`risk-register` P9–P12）：
     **A = 编译器**（语义层只输出 SQL；凭证、执行、审计全在 Go 侧，目标形态）；
     **B = 受治理读路径**（语义层持只读凭证与预聚合；`ExecutionGuard` 前移为前置网关，
     探索式查询直连 + Guard）。默认目标为 A，A 不可行则落 B 并重写安全边界描述。
  4. AGPL / GPL / BSL 项目只可借鉴设计，禁止嵌入制品（沿用 L4）。
- Alternatives：直选 SLayer 作必须件（否决：0.7.x 不宜承载 SQL 正确性守门人）；
  直选 Cube 并接受其执行面（否决：与凭证/审计边界冲突，需先做拓扑决策）；
  自研语义引擎（否决：方言与预聚合的长期维护成本）。
- Consequences：拓扑 A 需验证语义层"只编译不执行"的能力，否则整条链路退化为 B；
  拓扑 B 需把多租户凭证注入语义层并把 SQL 审计前移；
  两者都必须有契约测试（同一 logical query 经不同 adapter 产出等价 SQL）。
- Status：Accepted（接口与序列化格式）；拓扑 A/B 待 spike 关闭。

## ADR-019 AI Context 与 Semantic Model 的边界

- Context：业务口径（"收入不含退款"）与指标定义（`revenue`）是两类信息，
  混在一起会导致模型无法校验。RAG 检索是概率性的，一旦进入 SQL 生成路径，
  结果不可复现、不可审计。
- Decision：分层并禁止交叉：
  - **Semantic Model**：确定性、可编译、进 git、**是 SQL 生成的唯一输入**；
  - **AI Context**：解释性、可检索（KB/RAG）、只用于对话解释与 `member.description`，
    **不得作为 SQL 生成的输入**，也不得替代语义模型拼进 prompt；
  - 关联方式：`member.id ↔ knowledge_id` 双向引用；description 由 KB 摘要生成后**人工确认**。
- Alternatives：把口径塞进 metric 定义（否决：不可维护、无法校验）；
  AI Context 上游化后再生成 SQL（否决：不可复现、无法归因）。
- Consequences：Agent 回答必须区分"查询结果"（来自语义层，可引用）与"口径解释"
  （来自 KB，必须带来源）；可视化 Intent 只允许引用 semantic member，不得引用 KB 片段。
- Status：Accepted。
