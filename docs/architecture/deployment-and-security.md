# Deployment & Security

状态：DECIDED（剖面与矩阵）/ 具体参数随 P7 落定。日期：2026-11-22。

## 部署剖面（按需启用，不强制全套）

| 剖面 | 包含 | 适用 |
|---|---|---|
| Lite | Chainlake App + PostgreSQL(ParadeDB) + Redis + DuckDB(in-process) + 对象存储（本地/MinIO二选一） | 最小自托管；完整 Agent + Knowledge 可用，**不需要** Jupyter/外部引擎 |
| Standard | Lite + Jupyter Server（notebook 执行） + Asynq 周期调度 | 交互式数据工程；notebook 即 pipeline |
| Advanced | Standard + 可选：Duckpad（交互备数）、Duckle（大吞吐/定时）、Spark/Flink Compute、Redpanda Connect（CDC/流）、外部向量库/图库/搜索引擎 | 重负载与企业级场景，按 sidecar 开关逐个启用 |

原则：默认 `docker compose up` 只起 Lite；Standard/Advanced 为 profiles。
Kubernetes（helm）镜像映射；任一 sidecar 缺失不得阻塞核心启动（健康检查分级）。

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
| SLayer 模型库 | sidecar SQLite | 文件备份 | P7 演练 |
| 密钥 | 部署 secrets 管理 | 独立安全备份（禁入 git） | 轮换手册 |
| 审计日志 | DB 表（保留策略） | 随库备份；归档到对象存储（企业版长期归档不在本仓） | 抽查 |

## 升级与回滚

- 版本化发布（git tag + 镜像 tag）；迁移 `up/down` 成对；先备后升。
- 上游同步走 `sync/weknora-*` + merge（见 `../development/FORK.md`），门禁全绿才合入。

## 安全基线（摘要，细则见各域文档）

默认 deny 出站（allowlist 放行）；密钥不进日志/cell/产物；写操作必审批；
审计 append-only；默认关闭项：Docker 沙箱宿主挂载、公开注册、SSE 外网暴露。
首次部署检查清单随 P7 输出。
