# PATCHES — 上游源码修改登记

规则：优先加法（新文件 + 注册行）；允许最小、有文档的上游源码修改（见总纲 §8）。
每条记录：文件、行范围、原因、上游 issue/PR 链接、同步时处理方式。

## 登记表

| ID | 文件 | 说明 | 原因 | 上游链接 | 同步处理 |
|---|---|---|---|---|---|
| — | （暂无） | — | — | — | — |

## 注册行清单（加法，非 PATCHES，但同步时需复核位置）

| 位置 | 用途 | 状态 |
|---|---|---|
| `internal/datasource/connector.go`（Registry 组装处） | 新 Connector 注册 | 未开始 |
| `internal/agent/tools/definitions.go` | 新 Tool 条目 | 未开始 |
| `internal/agent/agent_service.go`（registerTools） | 新 Tool case | 未开始 |
| `internal/router/router.go` | 新路由组挂载 | 未开始 |
| `frontend/src/router/index.ts` | 新页面路由 | 未开始 |
| `frontend/src/stores/menu.ts` | 新菜单项 | 未开始 |
