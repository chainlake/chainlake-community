# Identity and Authorization

状态：DECIDED（模型）/ Logto 联调与 SSO 为 PENDING。日期：2026-11-22。

## Auth Adapter 现状（已核实代码，非假设）

Adapter 真实存在：`chainlake-gateway/src/identity/`。
接口 `IdentityProvider`（`types.ts:65-143`）：`resolveIdentity`、`getUser`、
workspace CRUD、成员 CRUD、`getPlatformRole`、`hasMfa`、user/session/TOTP 管理、
`listAuditLogs`、workspace customData。按 `AUTH_PROVIDER` env 选择
`logto | zitadel`（`index.ts:14-24`；**代码两处默认仍为 zitadel**：
`identity/index.ts:15`、`server.ts:80`）。

职责划分（已核实）：
- Adapter 内（provider 相关）：token 校验与签名（Logto `logto-provider.ts:196-247`）、
  OIDC discovery + JWKS 缓存（`:169-194`）、org/membership 解析
  （`parseOrganizationClaims :108-156`，Management API 回退 `:345-355`）、
  MFA 因子查询、成员/用户/审计的 provider 侧实现。
- 应用层（provider 无关）：权限门只读 `Identity.organizations[].role`
  （`org:admin | org:member` 二值契约，`org-admin.ts:268-282`）、capabilities 纯函数
  （`permissions/capabilities.ts:31-40`）、MFA 编排、计费/席位、组织自动开通编排。

**HISTORICAL 说明**：`chainlake-console` 的 Zitadel 代码（`zitadel-config.ts` 等）是旧架构遗留，
不得作为 SaaS 身份现状引用。网关内 Zitadel provider 为遗留 fallback（新能力均抛
`not_supported`，`zitadel-provider.ts:221-255`）。Community 架构只认 Logto 路径；
任何 Zitadel 引用一律标 HISTORICAL。

## 身份契约（IdP 无关）

```mermaid
flowchart LR
    IdP[Logto OIDC / 未来 IdP] --> AD[Auth Adapter\ngateway/src/identity/]
    AD --> ID[Identity{sub, orgId=user:sub,\norganizationId, organizations[]\nrole=org:admin|org:member,\nplatformRole}]
    ID --> APP[App: gates/capabilities/billing]
```

- Authentication（你是谁）：IdP token → adapter 校验。
- Identity mapping（内部表示）：adapter 翻译为 `Identity` + 域 Role
  （owner/admin/member/viewer，`types.ts:10`，`mapWorkspaceRole :161-168`）。
  业务 handler 只见域概念（`types.ts:6-9` 注释即此要求）。
- Workspace membership（能进哪些空间）：`TenantMember` 行 + token org 声明 + Management 回退。
- Authorization（能做什么）：`RequireRole/RequireOwnershipOrRole/KBAccess` 矩阵（复用）。
- Resource permissions（能碰哪些数据）：**认证不自动给**。数据源/资产级授权独立判定
  （连接可见性 + SLayer 模型可见性 + 行级策略），见 `data-engineering.md` §安全。

Community 与 SaaS 共享同一应用层契约（`Identity` 形状 + 二值 role + capabilities），
部署配置可不同（Community 直连 Logto；SaaS 经网关 adapter）。业务逻辑禁止直接调
Logto Management/Zitadel API（破坏层已存在：`org-billing.ts` 按 provider kind 分支——
新代码禁止新增此类分支，统一走 adapter 方法）。

## Workspace 模型（4.2）

`User → Workspace(=Tenant) → Resources`。Workspace 即 `types.Tenant`
（含配额/存储/检索/记忆等 JSONB 配置）；成员即 `TenantMember`（role 四级）；
跨空间共享沿用 `Organization`（不引入 Project 抽象；新抽象需 design 论证）。

- 所有权：资源行含 `workspace_id` + `created_by`；跨空间访问只经 Organization 共享或分享 token。
- 个人 vs 共享：默认 workspace 内共享（role 门控）；个人空间 = 以个人为唯一成员的 workspace；
   secret 归用户所有（vault owner 模型），数据资产归 workspace 所有。
- 凭证归属：用户级 secret（vault）vs workspace 级连接凭证（连接行归 workspace，读连接需成员身份；
  写操作走审批）。**连接凭证表结构为 PENDING（P1 设计时定稿）。**
- 生命周期：workspace 删除级联清理 sessions/compute 会话/schedules（软删 + 保留期，审计留痕）；
  **级联清单为 PENDING（P1 定稿）。**

## 企业 SSO 与未来 IdP

现状：网关与 console 均无 SSO 实现（SAML/OIDC connection 代码为零；console 为占位文案）。
路径：adapter 新增 provider 实现（如 `saml-provider.ts` 实现同一 `IdentityProvider` 接口），
应用层零改。企业 SSO 属 enterprise track，Community 本期 DEFERRED（只保留接口可扩展性）。
