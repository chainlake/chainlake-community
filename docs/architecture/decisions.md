# Architecture Decision Records

日期：2026-11-22。状态以各条 Status 为准（Accepted / Pending / Deferred）。

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
- Status：Accepted（模型）；表结构 Pending（P0/P1 定稿）。

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
- Status：Accepted（模型）；字段定稿 Pending（P0）。

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
- Status：Accepted（剖面）；参数矩阵 Pending（P7）。
