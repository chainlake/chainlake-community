# FORK — 上游同步纪律

基线：WeKnora v0.9.0，commit `7a3ca4990`。`upstream` =
`https://github.com/Tencent/WeKnora.git`（已验证连通）。

## 分支

- `origin/main`：Chainlake 开发主干。**不跟踪上游**，只接受 feature PR 与 sync PR。
- `feature/*`：功能分支。未合入前允许 rebase；合入后禁止改写历史。
- `sync/weknora-*`：专用同步分支。从 `origin/main` 建分支，merge 上游，打门禁，
  以 PR 合入。

## 同步流程（每次）

1. `git fetch upstream`，从 `origin/main` 建 `sync/weknora-<date>`。
2. `git merge upstream/main`（merge-based，保留合并点与冲突解决记录）。
3. 全量门禁：`go build ./...` + 相关单测 + 前端 `type-check` + `check-i18n`。
4. 回归重点：鉴权、RBAC、ingestion 队列、Agent 冒烟（notebook 问数一次）。
5. PR 描述记录：上游版本、冲突文件与解法、回归结果。合入后保留分支至下次同步成功。

## 回滚

同步 PR 出问题：revert merge commit。`sync/weknora-*` 分支保留备查。

## 本文件维护

每次同步更新"上次同步"记录：

| 日期 | 上游版本 | 同步分支 | 冲突数 | 门禁 | 备注 |
|---|---|---|---|---|---|
| — | v0.9.0（基线） | — | — | — | fork 起点，无同步 |
