# Deployment & Security

状态：DECIDED（剖面与矩阵）/ 具体参数随 P7 落定。日期：2026-11-22；修订：2026-11-24。

## 部署剖面：单一完整剖面（ADR-012′，取代 ADR-012）

Community 只有**一个剖面**：默认 `docker compose up` 起全部必需件，不提供
Lite / Standard / Advanced 裁剪（语义层是必需件，Lite 的"单应用零依赖"形态不成立）。

| 类别 | 组件 | 缺失时行为 |
|---|---|---|
| 必需 | Chainlake App(Go) + PostgreSQL(ParadeDB) + Redis + 对象存储（本地/MinIO 二选一）+ 语义层 sidecar（`SemanticCompiler`，见 ADR-018） | 启动自检失败、健康检查失败（不得带病启动） |
| 必需（内嵌） | DuckDB（交互查询，随 App 进程内运行，不单独部署） | — |
| 可选加速件 | Jupyter Server、Duckpad、Duckle、Spark/Flink Compute、Redpanda Connect（CDC/流）、外部向量库/图库/搜索引擎 | 不阻塞核心链路；对应 capability 置 false，前端入口隐藏 |

原则：可选件通过 capability 暴露"是否就绪"，**不再用于裁剪基础功能**；
SQLite 单机形态不支持 data platform 能力（整体 fail-closed）。
Kubernetes（helm）按必需/可选两档做健康分级。

## 配置、密钥与持久卷

- 配置：env + `config.yaml` 两层（沿用上游模式）；新增 Chainlake 段集中管理。
- 密钥：`JWT_SECRET`、`SYSTEM_AES_KEY`（丢则加密字段不可恢复，备份必含）、DB/Redis 密码、
  Logto client secret。**密钥轮换流程 P7 输出，当前为 PENDING。**
- 持久卷：Postgres 数据、Redis（AOF 可选）、对象存储桶、notebook/pipeline git 仓、
  SLayer SQLite（如部署）、DuckDB 备数区。sidecar 各自声明卷需求，部署矩阵统一列出。

## 备份矩阵

| 数据 | 位置 | 备份方式 | 恢复演练 |
|---|---|---|---|
| 业务库（ParadeDB） | pgdata 卷 | pg_dump 定时 + 卷快照 | P7 演练 |
| Redis（队列/会话） | 内存/AOF | 可重建（任务幂等），AOF 可选 | 文档说明 |
| 对象存储（文件/产物） | MinIO/本地/S3 | 桶复制或卷备份 | P7 演练 |
| notebook/pipeline 定义 | git 仓 | git 远端镜像 | 拉取即恢复 |
| 语义模型定义 | workspace git 仓（Ossie 兼容 YAML，ADR-018） | git 远端镜像 | 拉取即恢复 |
| 密钥 | 部署 secrets 管理 | 独立安全备份（禁入 git） | 轮换手册 |
| 审计日志 | DB 表（保留策略） | 随库备份；归档到对象存储（企业版长期归档不在本仓） | 抽查 |

## 升级与回滚

- 版本化发布（git tag + 镜像 tag）；迁移 `up/down` 成对；先备后升。
- 上游同步走 `sync/weknora-*` + merge（见 `../development/FORK.md`），门禁全绿才合入。

## 安全基线（摘要，细则见各域文档）

默认 deny 出站（allowlist 放行）；密钥不进日志/cell/产物；写操作必审批；
审计 append-only；默认关闭项：Docker 沙箱宿主挂载、公开注册、SSE 外网暴露、
语义层的开发/无鉴权模式（`CUBEJS_DEV_MODE` 一类开关，启动自检检测到即拒绝启动）、
供应商匿名使用统计。
首次部署检查清单随 P7 输出。

## 附录：SaaS 多租户 Notebook（DEFERRED，不进社区实现）

本节为 cloud/enterprise track 预留设计，社区版不实现。原则：
**同 workspace 共享数据、不共享执行状态；不同 workspace 什么都不共享**。

- 计算归属：kernel 粒度永远 per-user（独立变量空间与 DuckDB 内存）；共享的是数据、
  定义与存储。Compute 增加 `scope: personal | workspace-shared | ephemeral`
  （个人/共享池/定时一次性），spawner 按 scope 选 K8s 模板。
- 同 workspace 协作分档：L1 隔离协作（默认，各自 kernel + 共享存储/git，可见彼此文件与产物）；
  L2 在场感（光标/cell 锁）；L3 实时协同编辑（Yjs，明确延后）。
- 跨 workspace 硬隔离：进程隔离（namespace/标签 + NetworkPolicy）；
  存储隔离（独立卷/桶，不 cross-mount）；凭证隔离（别名按 workspace 注入）；
  配额隔离（CPU/内存/并发 kernel 数/DuckDB `memory_limit`，超限熔断+审计）。
- SaaS 运营：idle 回收（交互 30min、定时跑完即焚；warm pool 仅付费档）；
  成本归属（run 记录记 workspace/user/compute/时长，用量计量归 cloud 仓）；
  审计到会话级 + 查询级（cell 级不做）。
- 共享引擎（Spark Connect 集群）：per-user session + fair 调度 + workspace 配额，
  复用引擎原生机制。
