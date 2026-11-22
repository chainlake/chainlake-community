# Data Ingestion Architecture

状态：DECIDED（架构）/ 实现为 PROPOSED（P2）。日期：2026-11-22。

## 统一流程

```
Source → Connector → Extraction → Transformation → Validation → Storage → Catalog Registration
```

四路来源：文件（Volume/上传）、数据库（连接器采样/整表）、API（分页/增量键）、网页抓取
（沿用 docreader 抓取 + SSRF allowlist，不另建抓取器）。

## 关键决策

- **全量 vs 增量**：首次全量 + 主键/时间戳增量为默认；CDC **不是**初始必备能力
  （DEFERRED；CDC / 定时增量 / 流式是三种不同机制，不得混用名头，Redpanda Connect 为可选 Advanced 组件）。
- **校验与检疫**：非空/类型/行数阈值；不过进 quarantine（状态可见、可重试、可废弃），不静默丢数。
- **重试与恢复**：指数退避 + 死信记录；run 级幂等（重跑不重入）。
- **Schema 演进**：新列加法兼容并版本化；破坏性变更建新版本表 + 通知订阅者（PENDING 细节，P2 定稿）。
- **Provenance**：每次 ingest 登记 artifact（源 URI + hash + 转换版本 + run_id）。

## 双通道（与 RAG 的边界）

同一源文件两条独立处理线（见 `resource-and-catalog-model.md`）：
Knowledge Processing（解析→分块→embedding→索引→KB）沿用现有流水线；
Data Processing（抽取→schema 推断→校验→注册为 Table/Dataset）为新链。
文件 hash 去重，产物分别登记，互不阻塞。**RAG 文档处理与结构化 ingestion 不共享执行路径，
只共享源文件与 Catalog 发现层。**
