# PATCHES — 上游源码修改登记

规则：优先加法（新文件 + 注册行）；允许最小、有文档的上游源码修改（见总纲 §8）。
每条记录：文件、行范围、原因、上游 issue/PR 链接、同步时处理方式。

## 登记表

| ID | 文件 | 说明 | 原因 | 上游链接 | 同步处理 |
|---|---|---|---|---|---|
| P-01 | `internal/types/agent.go` | `ToolResult` 新增 `Artifacts MessageArtifacts`（`json:"-"`，仅存活期） | 让工具能在不依赖 sandbox 的情况下交付产物 | 待提 | 加性字段；冲突时保留本字段并重新挂到新结构体 |
| P-02 | `internal/agent/engine.go` | `AgentEngine` 新增 `turnArtifacts` 字段（每轮构建，不跨轮） | 同上，累积本轮的产物 | 待提 | 同上 |
| P-03 | `internal/agent/act.go` | `emitToolOutcome` 增加一处累积（3 行） | 工具产物汇入本轮结果 | 待提 | 函数体重命名时按语义重新插入 |
| P-04 | `internal/agent/finalize.go` | `Artifacts` 改为 `mergeTurnArtifacts(collaboration, turnArtifacts)` | 完成事件带出两层产物 | 待提 | 单行替换，保留 merge |
| P-05 | `internal/modelcontext/tool_policy.go` | `toolHandlePolicies` 新增 `"save_chart": {}` | 内置工具必须声明 handle 策略（`TestAvailableToolDefinitionsDeclareModelHandlePolicies` 强制） | 待提 | 加性 map 条目 |
| P-06 | `internal/application/service/custom_agent.go` | 新增 `builtinToolRollout` + `applyBuiltinToolRollout`，并在 `GetAgentByID`/`ListAgents` 的内置分支各调用一次 | 内置 Agent 被租户改过后 DB 行覆盖 YAML，新工具需对存量租户生效 | 待提 | 加性函数 + 2 处调用；与上游 config 版本机制合并时删除本补丁 |
| — | （其余改动为新增文件） | `internal/agent/tools/save_chart.go`、`internal/agent/artifacts.go` 及测试 | — | — | — |

## 注册行清单（加法，非 PATCHES，但同步时需复核位置）

| 位置 | 用途 | 状态 |
|---|---|---|
| `internal/datasource/connector.go`（Registry 组装处） | 新 Connector 注册 | 未开始 |
| `internal/agent/tools/definitions.go` | 新 Tool 条目 | **已用 1 处**：`save_chart` 常量 + `AvailableToolDefinitions` 条目 |
| `internal/agent/tools/capabilities.go` | 新 Tool 的 KB 能力要求 | **已用 1 处**：`save_chart`（沿用 `data_analysis` 要求） |
| `internal/application/service/agent_service.go`（registerTools） | 新 Tool case | **已用 1 处**：`ToolSaveChart` 注册（`s.fileService`） |
| `config/builtin_agents.yaml` | 内置 Agent 的 `allowed_tools` | **已用 1 处**：`builtin-data-analyst` 加 `save_chart` |
| `config/prompt_templates/agent_system_prompt.yaml` | `data_analyst` 输出契约（即席 / 保存两模式） | **已用 1 处** |
| `internal/router/router.go` | 新路由组挂载 | 未开始 |
| `frontend/src/router/index.ts` | 新页面路由 | 未开始 |
| `frontend/src/stores/menu.ts` | 新菜单项 | 未开始 |
