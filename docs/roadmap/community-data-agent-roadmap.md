# Community Data Agent Roadmap (P0–P7, 修订版)

状态：DECIDED（阶段划分与依赖）/ 每阶段启动前确认 PENDING 关闭情况。日期：2026-11-22。
总纲与架构见 `../architecture/chainlake-community-edition.md` 及 `../architecture/`。

阶段组织：1 架构决策 → 2 基础平台 → 3 核心实现 → 4 Agent 能力 →
5 高级集成 → 6 生产就绪。技术只在被需要时引入；任一阶段可独立测试交付。

## P0 — Foundation & Integration（地基与联通）

- Objective：可同步、可登录、可注册资产的代码地基。
- Scope：sync 流程演练；Logto 联调；Catalog/Artifact 基础表与 API；FORK/PATCHES 建账。
  排除：任何 Agent/执行/可视化实现。
- Dependencies：上游基线可用；Logto tenant 可配。
- Deliverables：sync PR 模板；Logto 联调报告；`catalogs/schemas/tables/volumes` 元数据表；
  artifact 注册表（元数据+版本+状态）+ 读写 API；FORK.md/PATCHES.md。
- Acceptance criteria：一次完整上游 merge 门禁通过；Logto 登录成功；
  catalog 表建表成功；artifact 登记/查询/版本流转通。
- Upstream conflict risk：无（新表 + 配置 + 文档）。
- Rollback：down 迁移；`OIDC_AUTH_ENABLE=false` 回本地登录。

## P1 — Data Platform Core（平台核心）

- Objective：连得上、跑得出、记得住账的执行底座。
- Scope：Compute 注册表（DuckDB 先行）+ 会话管理；Go 原生三方言连接器；
  执行安全层落地（校验/最小权限/超时/配额/审计）；连接凭证表结构；
  SemanticCompiler 接口 + SLayer adapter（通路验证）；Jupyter Server 设计定稿。
  排除：notebook UI、Agent、调度触发器。
- Dependencies：P0（catalog/artifact/密钥模型）。
- Deliverables：compute/connectors/安全层代码；SLayer 编译通路；Jupyter 设计文档。
- Acceptance criteria：同查询双引擎执行+审计；非法 SQL 被拒；kernel/Jupyter 设计评审通过；
  SLayer dry-run 通；P3/P4 的 PENDING（凭证表、级联清单、Jupyter 设计）关闭。
- Upstream conflict risk：低（新文件 + `container.go` 注册行）。
- Rollback：down 迁移；开关摘除（尚无上游依赖方）。

## P2 — Data Ingestion Agent（ingestion 专职智能体）

- Objective：四路接入 + 双通道 + 增量 + 校验 + 入 catalog。
- Scope：文件/库表/API/网页（复用 docreader+SSRF）；schema 发现；主键/时间戳增量；
  校验→quarantine；重试/死信；run 幂等；成功注册 catalog + artifact。
  明确排除：CDC（DEFERRED，可选 Advanced）、流式摄入。
- Dependencies：P1（connectors/安全/密钥）。
- Deliverables：Ingestion Agent 预设 + ingestion tools + 双通道状态机。
- Acceptance criteria：四路各跑通一次；增量只处理新增；校验失败进检疫；
  同一文件双通道产物各自正确；失败可重放。
- Upstream conflict risk：低（新 Tool 文件 + `definitions.go` + `registerTools` case）。
- Rollback：Agent 预设下线；ingestion 产物按 artifact 状态机废弃（不删源数据）。

## P3 — Data Engineering Foundation（工程地基）

- Objective：notebook 可写可跑；pipeline 可定义可执行可调度。
- Scope：自研 Vue notebook UI（Jupyter 协议、语言下拉、TDesign 表格、Compute attach）；
  Pipeline Model（定义存 git）+ Papermill/nbclient 执行器；Asynq 通用周期触发器（新增）
  + `notebook:run` 队列；notebook↔pipeline 互调。排除：Agent 编排、可视化。
- Dependencies：P1（Compute/Jupyter 设计/安全）。
- Deliverables：Data Studio UI；pipeline 定义与执行器；调度底座与 run 记录。
- Acceptance criteria：语言切换/`%sql` 富表格可用；定时 notebook 产出版本化产物；
  pipeline 定义可 review 可回滚；失败重试/恢复有效。
- Upstream conflict risk：无到极低（fork 新文件 + sidecar）。
- Rollback：菜单隐藏；调度开关关闭；sidecar 摘除。

## P4 — Data Engineering Agent（工程智能体）

- Objective：Agent 驱动备数、建 pipeline、触发重跑。
- Scope：Engineering Agent 预设；Duckpad MCP 接入（交互备数，人审链路）；
  Duckle HTTP 桥接入（大吞吐/定时）；notebook↔pipeline 互调工具；
  SLayer model 生成/更新（备数完成触发）。
  排除：流式/CDC（Advanced 可选）。
- Dependencies：P2（ingestion 产物）、P3（pipeline/调度）。
- Deliverables：Agent 预设 + Duckpad/Duckle 配置 + 互调工具 + SLayer 建模链。
- Acceptance criteria：对话完成"备数→建模→重跑"；Excel 更新触发下游刷新；
  重型与轻量两路产物同格式可查；P5 桥方案 PENDING 关闭。
- Upstream conflict risk：低（新 Tool + MCP 配置 + 桥 sidecar）。
- Rollback：Agent 下线；sidecar 摘除不影响已有 pipeline 定义。

## P5 — Data Visualization Agent（可视化智能体）

- Objective：NL→Dive/Dashboard，Agent 与人共用同一 Spec。
- Scope：Intent 规范 + Compiler + 12 种模板 + JSON Schema 校验；
  `chart_catalog`/`save_dive`；dives/dashboards 表与版本；chart 卡 + Dive 查看器
  （过滤/下钻/分享/嵌入）+ Chart Editor + Dashboard Editor（布局库选型时核许可）。
  排除：可写回 App（延后）、第二渲染引擎。
- Dependencies：P1（SLayer 编译）、P3（artifact 链）。
- Deliverables：Spec+模板库+双编辑器+分享/嵌入。
- Acceptance criteria：12 种图全链路生成；校验失败 Agent 自修复；
  同一 Dive 建/改/回滚；嵌入页鉴权有效。
- Upstream conflict risk：低（新文件 + ToolResult 分发分支）。
- Rollback：编辑器菜单隐藏；Dive 按版本废弃（历史可读）。

## P6 — Multi-Agent Orchestration（多智能体编排）

- Objective：三智能体在共享资源上协同完成端到端任务。
- Scope：协调者 Agent + 委派协议（输入输出契约/artifact 引用/失败三策）；
  provenance 全链视图；Dive vs Dashboard 运营规范。不建通用编排平台。
- Dependencies：P2–P5（各 Agent 就绪）。
- Deliverables：协调者预设 + 协议文档 + lineage 视图。
- Acceptance criteria：一句话需求走完备数→建模→可视化且 lineage 可查；
  子任务失败可定位到具体 Agent/run；降级为单 Agent 仍可用。
- Upstream conflict risk：低（复用协作框架；新增协议文档）。
- Rollback：协调者下线，降级单 Agent 模式。

## P7 — Production Readiness（生产就绪）

- Objective：可部署、可备份、可升级、可同步。
- Scope：三档部署矩阵与参数；密钥轮换；加固复审；备份恢复演练；
  Asynq 负载验证（备选 K8s CronJob）；上游同步 SOP 常态化。
- Dependencies：P0–P6。
- Deliverables：部署矩阵文档；加固报告；备份恢复演练记录；同步 SOP。
- Acceptance criteria：全新机器按文档一次部署成功；恢复演练通过；
  一次上游大版本同步门禁全绿。
- Upstream conflict risk：本阶段以验证为主；深层冲突回退对应阶段。
- Rollback：版本化发布 + 数据恢复手册（演练覆盖）。
