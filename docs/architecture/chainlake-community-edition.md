# Chainlake Community Edition（总纲）

状态：已评审修订。日期：2026-11-22。基线：WeKnora v0.9.0（commit `7a3ca4990`）。
分阶段路线图见 `../roadmap/community-data-agent-roadmap.md`；
架构详见 `../architecture/`；同步纪律见 `../development/FORK.md`。

本文定义：命名体系、功能边界、Git 策略要点、核心模型索引、技术选型结论。
推理过程与 ADR 见 `../architecture/decisions.md`；风险见 `../architecture/risk-register.md`。

## 1. 品牌与命名体系

```
Chainlake — From raw data to trusted insights.（母品牌 / 产品品牌）
├── chainlake-community   PUBLIC   Open-source Community Edition（本仓）
├── chainlake-enterprise  PRIVATE  Commercial Enterprise Extensions
├── chainlake-cloud       PRIVATE  SaaS Control Plane & Services
└── Chainlake App（app.chainlake.io）Commercial SaaS entry point
```

## 2. Community 功能边界

包含：Data Agent（含多智能体协作）、RAG/Knowledge（上游原样复用+挂接）、
Data Catalog（含 Artifact Registry）、Data Engineering（notebook 即 pipeline+调度）、
Data Visualization（Intent→ECharts+Dive+双编辑器）、Self-hosted 单一完整剖面部署（ADR-012′）。

禁区（本仓不得出现）：计费/订阅/用量计量、自助租户开通与试用流转、托管运维面、
企业合规包（审计归档/KMS/DLP）、SaaS 形态白标。企业 seam 只预留接口。

## 3. Git 策略（要点，细则见 FORK.md）

- `upstream/main`：WeKnora 上游（只读）。
- `origin/main`：Chainlake 开发主干，**不跟踪上游**。
- `feature/*`：功能分支（未合入可 rebase）。
- `sync/weknora-*`：专用同步分支，merge-based + 门禁 + PR 合入；出问题 revert merge commit。

## 4. 核心模型索引

User → Workspace（= Tenant）→ Resources；四级 Role；Organization 仅跨空间共享；
Resource Registry / Data Catalog / Artifact Registry 三权分立；
Knowledge 与 Data 双通道；Notebook ≠ Pipeline；Intent→Compiler→ECharts。
详见 `../architecture/` 各域文档。

## 5. 身份架构（Logto 为准）

以网关 `src/identity/` Auth Adapter + Logto 为准；业务只认 `Identity` 契约。
`chainlake-console` 的 Zitadel 代码为 HISTORICAL 遗留，不得作为现状引用；
网关内 Zitadel provider 为遗留 fallback。企业 SSO 未实现（占位），属 enterprise track。

## 6. 技术选型结论（许可）

可用：MIT/Apache-2.0（ECharts、Jupyter 生态、Duckpad、Duckle 双许可其一、
Redpanda Connect 核心 MIT+多数连接器 Apache-2.0、MPL-2.0 保留 notices）。
排除嵌入：Sling GPL-3.0、Steampipe 核心 AGPL、Metabase AGPL、havn BSL（只看不抄；
注意 hosted 竞品限制）。容器镜像许可逐个核实后方可发版（见风险登记 L1）。
语义层候选：Cube Core（后端 Apache-2.0 / client MIT）、SLayer（MIT）；
Wren AI 与 Sidemantic 均为 **AGPL-3.0**，只可借鉴设计，禁止嵌入（见风险登记 L5）。
标注 PENDING 的项不得带着未验证假设进入实现。
