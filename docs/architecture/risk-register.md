# Decision & Risk Register

日期：2026-11-22。随验证进展更新；关闭 PENDING 需回写对应文档。

## 可执行决策（Ready）

- R1：fork WeKnora v0.9.0 为基座，RAG 原样复用（ADR-001）。
- R2：单 PUBLIC 仓 + 独立 PRIVATE 仓；本仓禁商业代码（ADR-002）。
- R3：User→Workspace(=Tenant)→Resources；复用 Role/Organization（ADR-004）。
- R4：双通道（Knowledge/Data）分离（ADR-006）；Notebook≠Pipeline（ADR-007）。
- R5：执行安全统一模型（校验+最小权限+超时+配额+审计），纠正注入假设（ADR-008）。
- R6：Intent→Compiler→ECharts 唯一渲染（ADR-011）；三级可视化模型。
- R7：Lite/Standard/Advanced 部署剖面（ADR-012）。

## 待验证决策（Pending：事项 / 为何重要 / 关闭条件）

- P1：Logto OIDC 与 WeKnora 兼容性（PKCE/userinfo email/JIT/默认租户）。重要性：登录是 P0 门禁。
  关闭：联调通过 + 回写 identity 文档。
- P2：生产 `AUTH_PROVIDER` 与 Logto tenant 实际配置（endpoint/roles/claim）。重要性：代码默认仍是 zitadel。
  关闭：部署配置确认。
- P3：Jupyter Server 票据/WS/隔离/生命周期设计。重要性：notebook 是 P3 核心，不设计不能开工。
  关闭：`data-engineering.md` 更新 + 安全评审。
- P4：连接凭证表结构与 workspace 删除级联清单。重要性：涉及数据归属与清理正确性。
  关闭：P1 设计评审。
- P5：Duckle MCP HTTP 桥方案。重要性：决定 sidecar 集成成本。关闭：P4 前原型验证。
- P6：Asynq 调度负载。重要性：决定是否引入 K8s CronJob。关闭：P7 压测。
- P7：密钥轮换流程。重要性：生产必备。关闭：P7 手册。
- P8：SLayer 社区分发形态（镜像来源/许可/源码可得性）。重要性：P1/P2 依赖外部服务。
  关闭：与 SLayer 提供方确认 + fallback（直连 SQL）演练。

## 技术风险

- T1：上游高速迭代导致同步冲突（0.x，每周 PR 量大）。缓解：只加文件+注册行；`sync/*` + 门禁。
- T2：ParadeDB 版本漂移（compose v0.25.11 vs helm v0.18.9）。缓解：P7 前统一并回归。
- T3：前端无 echarts，新增依赖增包体积。缓解：路由懒加载 + 按需引入。
- T4：DuckDB in-process 资源尖峰。缓解：并发上限 + 大查询走远程引擎。

## 外部依赖风险

- E1：SLayer 服务源码不在仓库（仅镜像引用）。缓解：pinned 契约 + fallback 演练（P8）。
- E2：Duckpad/Duckle 为外部项目（版本/接口漂移）。缓解：pin 版本 + MCP/HTTP 契约测试。
- E3：Logto（SaaS 侧）配置漂移。缓解：adapter 契约 + 集成测试。

## 安全风险

- S1：Jupyter 内核逃逸（ mitigations 见执行安全层；设计未关闭前禁上线）。
- S2：密钥泄露面（cell 日志/产物/分享链接）。缓解：别名制 + 输出脱敏 + 分享 token 审计。
- S3：MCP 外部工具权限放大。缓解：沿用审批卡 + capability 白名单。
- S4：默认密钥（JWT/AES/SEARXNG）未轮换。缓解：部署清单强制项（P7）。

## 许可风险

- L1：容器镜像许可未逐个核实（searxng/minio/neo4j/redis 谱系/clickhouse/langfuse/busybox 等）。
  **CE 发版前必须逐个清理**，否则不得发布制品。
- L2：SheetJS xlsx 走 CDN tarball（非 npm 源），确认 Community/Pro 范围与可再分发性，考虑 vendoring。
- L3：MPL-2.0（go-sql-driver 等）要求保留 LICENSE + notices + sources 压缩包（已有脚本，保持进 CI）。
- L4：参考项禁区：Sling GPL-3.0 / Steampipe 核心 AGPL / Metabase AGPL / havn BSL——
  只许借鉴思路，禁抄代码禁嵌入（havn 特别注意 hosted 竞品限制）。

## 部署风险

- D1：sidecar  sprawl（越加越多）。缓解：三档剖面 + 缺失不阻塞 + 健康分级。
- D2：SQLite 系组件（SLayer/Lite DB）的单副本与备份。缓解：备份矩阵 + RWO 约束文档化。
- D3：升级/回滚演练缺失。缓解：P7 必演练项。
