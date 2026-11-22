# Chainlake Community — Architecture Overview

状态：DECIDED（本文）/ 各领域细节见链接文档。日期：2026-11-22。
基线：WeKnora v0.9.0（commit `7a3ca4990`）；`origin/main` 为 Chainlake 开发主干。

标签说明（全文统一）：`IMPLEMENTED` 已在代码中实现 · `PROPOSED` 提议未实现 ·
`DECIDED` 已决策待实现 · `PENDING` 待验证 · `DEFERRED` 明确推迟 ·
`HISTORICAL` 历史遗留说明。

## 产品定位

Chainlake 是独立的 Agent-native Data Platform：**From raw data to trusted insights.**
覆盖 Data Ingestion、Data Catalog、Knowledge Management、Data Engineering、
Data Visualization、AI Agents、Multi-Agent Collaboration。
不是"WeKnora 换皮"，也不是传统 RAG 应用：RAG/Knowledge 是子系统（WeKnora 提供），
Data 平台能力是 Chainlake 自有领域。

## 文档地图

| 文档 | 内容 |
|---|---|
| `system-architecture.md` | 系统图、组件职责表（含证据）、域边界、数据归属 |
| `identity-and-authorization.md` | Auth Adapter 现状、身份契约、Workspace 模型、授权 |
| `resource-and-catalog-model.md` | Resource Registry、Data Catalog、Artifact Registry、KB 与 Volume 关系 |
| `agent-runtime.md` | Agent 引擎、工具、协作、多智能体契约 |
| `data-ingestion.md` | 统一 ingestion：四路来源、双通道、增量/重试/检疫 |
| `data-engineering.md` | Notebook vs Pipeline、Compute、调度、执行安全 |
| `visualization-and-artifacts.md` | Chart/Dive/Dashboard 三级模型、Intent→Compiler→ECharts |
| `deployment-and-security.md` | Lite/Standard/Advanced 部署剖面、备份矩阵、密钥与升级 |
| `decisions.md` | ADR-001…ADR-012（每个含 Context/Decision/Alternatives/Consequences/Status） |
| `risk-register.md` | 可执行决策 / 待验证决策 / 技术·外部·安全·许可·部署风险 |
| `../roadmap/community-data-agent-roadmap.md` | P0–P7 修订路线图 |
| `../development/FORK.md`、`PATCHES.md` | 上游同步纪律与修改登记 |

版本章程：`chainlake-community-edition.md`（本目录）。

## 基线事实（证据优先）

- Community Edition fork 自 WeKnora v0.9.0；`upstream` 远程已配并验证连通；
  `origin/main` 为开发主干，同步走 `sync/weknora-*` + merge（见 `../development/FORK.md`）。
- 网关层真实存在：`chainlake-gateway`（TS）：SLayer 客户端（pinned v1.5.0）、
  semantic models、connectors、agents（含 `slayer_query`）、`src/identity/` Auth Adapter
  （Logto provider + Zitadel provider，env 选择）。
- SLayer 服务本体**不在任何仓库内**：仅 infra 有镜像引用
  （`chainlake-slayer:latest`，ACR 私有仓库）。它是真实部署依赖，但源码/许可不在社区可控范围
  ——见 `risk-register.md` R-EXT-01。
- Jupyter（Server/Hub）、Go 原生 Connector、Catalog/Artifact 新表、Dive 相关表：
  **仓库中均不存在**，一律标 `PROPOSED`，不得当作既有能力引用。
