# 通知系统改造：禁用 OS 通知 + JNPM 关联 + 飞书机器人通知

> 制定时间：2026-07-08
> 关联调研：`plan/ai-summarize-jnpm-lark-research.md`
> 目标仓库：`C:\Users\Admin\Documents\GitHub\pcraft`（Kandev/pcraft）

---

## 0. 需求拆解

用户共三条需求：

1. **永远不触发操作系统桌面通知**（相关代码保留，但功能不再触发）。
2. **New Task 创建弹窗新增一个选填项 `JNPM ID`**，placeholder 为「JNPM单号，需要带#」，该 `jnpm_id` 与创建的任务关联。
3. **接入飞书机器人通知**：当系统要发通知时，先看任务是否关联 `jnpm_id`；若有，则查询该 JNPM 单号对应的「指派人」，并通过飞书机器人把消息私信给该指派人。

---

## 1. 现状梳理（探索结论）

### 1.1 通知系统架构

pcraft 的「桌面通知」由 **会话进入 `WAITING_FOR_INPUT` 状态** 驱动（permission 请求只是众多触发源之一），分两条通道：

- **前端浏览器 Notification API**：`apps/web/lib/ws/handlers/notifications.ts` 里的 handler 收到 WS `session.waiting_for_input` 后 `new Notification(...)`。
- **后端 OS 原生 toast**：`apps/backend/internal/notifications/providers/system_provider.go`（macOS osascript / Linux notify-send / Windows PowerShell toast）。

统一分发入口（后端）：

- `apps/backend/internal/notifications/service/service.go`
  - `HandleTaskSessionStateChanged(ctx, taskID, sessionID, newState)`（L160）— 仅 `newState == "WAITING_FOR_INPUT"` 时分发。
  - `HandleInboxItem(ctx, itemType, title)`（L206）— Office inbox。
  - Provider 类型：`local`（WS→浏览器）、`system`（OS toast）、`apprise`（外部 CLI）。
  - `ensureDefaultProviders()`（L286）首次启动自动创建 **enabled 的** `local` + `system` provider。

事件订阅接线：`apps/backend/internal/backendapp/gateway.go`（约 L253-284）订阅 `TaskSessionStateChanged` → `HandleTaskSessionStateChanged`，`OfficeInboxItem` → `HandleInboxItem`。

Permission → 通知链路：
```
agent request_permission
 → orchestrator handlePermissionRequest (event_handlers_git.go)
 → setSessionWaitingForInput → TaskSessionStateChanged(WAITING_FOR_INPUT)
 → notificationSvc.HandleTaskSessionStateChanged
 → LocalProvider(WS→浏览器 Notification) + SystemProvider(OS toast)
```

### 1.2 任务创建数据流

```
TaskCreateDialog (apps/web/components/task-create-dialog.tsx)
 → useTaskSubmitHandlers.buildDynamicPrompt()  // 生成 description + metadata
 → buildCreateTaskPayload (task-create-dialog-helpers.ts)  // 组 payload，含 metadata
 → createTask() (apps/web/lib/api/domains/kanban-api.ts:47)  POST /api/v1/tasks
 → httpCreateTask (task_http_handlers.go:495)  // httpCreateTaskRequest.Metadata
 → service.CreateTask (service_tasks.go:64) → buildTask (:254 写入 task.Metadata)
 → sqlite.CreateTask (repository/sqlite/task.go:92)  // metadata 列存 JSON
```

关键：**`tasks` 表已有 `metadata TEXT` JSON 列**（`base_schema.go:247`），`Task.Metadata map[string]interface{}`（`models/models.go:315`）。存 `jnpm_id` **无需新增数据库列**，走 metadata 即可。前端 create payload 已支持 `metadata?: Record<string, unknown>`。

New Task 弹窗现有字段：`Task name`（必填）+ workspace 可配置的 `DynamicTaskForm`（默认只有一个 `Prompt`）。字段渲染在 `CreateModeBody`（`task-create-dialog.tsx:110`）。metadata 在 `buildDynamicPrompt`（`task-create-dialog-submit.tsx:123`）里组装，透传到 `performCreate` → `buildCreateTaskPayload`。

### 1.3 集成模式（Jira/Linear 为模板）

- 每个集成一个 Go 包：`service.go / store.go / client.go / provider.go / handlers.go / models.go / poller.go`。
- `Provide(writer, reader, secrets, eventBus, log)` DI 入口；`initXService` 在 `apps/backend/internal/backendapp/services.go`（L335+）注册；路由在 `helpers.go`。
- 密钥用 `internal/integrations/secretadapter` + `internal/secrets` `SecretStore`（固定 key，如 `jira:singleton:token`）。
- 外部 HTTP 无通用封装，各集成自建 `Client` 接口 + 私有 `do()`（30s 超时、限流读 body、非 2xx → `*APIError`）。
- **仓库内暂无任何飞书 / JNPM 代码**，需新建。

### 1.4 参考项目（ai_summarize）关键实现

- **JNPM 指派人提取**（`integrations/jnpm/client.py` `_extract_assignee`）：从 issue 详情的 `assignee` / `assignedTo` / `owner`（dict）取 `email` / `fullName` / `username`。issue 详情接口：`GET v1/open-api/projects/issues/{id}`，响应体在 `payload` 字段，认证头 `PRIVATE-TOKEN`。
- **飞书发私信**（`integrations/lark/client.py`）：`POST im/v1/messages?receive_id_type=email`，body `{receive_id: 邮箱, msg_type: "text"|"interactive", content: JSON字符串}`。需要 `app_id`/`app_secret` 换 `tenant_access_token`。

---

## 2. 设计决策（默认取值，如需调整请指出）

| 决策点 | 选择 | 理由 |
|--------|------|------|
| `jnpm_id` 存储 | task `metadata["jnpm_id"]`（固定键） | 无需 DB migration，前后端 metadata 已贯通 |
| `jnpm_id` 归一化 | 前端保存用户原始输入（带 `#`）；查询 JNPM 时后端 strip `#` 得纯数字 | placeholder 要求带 `#`，但 JNPM API 用数字 ID |
| 禁用 OS 通知的实现 | **保留全部 provider 代码**，在 `system_provider.Send` + 前端 `notifications.ts` handler 做 no-op；并在 `HandleTaskSessionStateChanged`/`HandleInboxItem` 顶部改为「不再走 local/system 通道，改走飞书通道」 | 满足「代码留着、功能不触发」 |
| 飞书 SDK | **直接调飞书 REST API**（自建 `Client` + `do()`），不引入 larksuite Go SDK | 与 Jira/Linear 现有模式一致，避免重依赖 |
| 飞书投递方式 | 按 **email** 私信（`receive_id_type=email`），消息类型 **纯文本 `text`** | 指派人 email 来自 JNPM，最简闭环 |
| 通知触发点 | 覆盖 **所有** `WAITING_FOR_INPUT`（含 permission / turn 完成 / clarification）+ `HandleInboxItem` | 用户已确认：覆盖全部 |
| 收件人解析 | 有 `jnpm_id` → JNPM assignee email；assignee 缺失或无 email → **admin（env）**；无 `jnpm_id` → **admin（env）** | 用户已确认降级到 admin |
| 全部配置来源 | **env**：JNPM base_url/token、飞书 base_domain/app_id/app_secret、admin 邮箱 | 用户已确认全走 env |
| 飞书/JNPM 凭据 | 本期直接读 **env**（不强制走 secret store；如需可后续迁移 `secretadapter`） | 用户已确认 env |
| 新增 provider 类型 | 新增 `models.ProviderTypeLark`，飞书作为一个 notification provider；收件人 email 在 service 层解析后经 `Message.TargetEmail` 传入 | 见 §5.3 |

> 开放问题已由用户确认，见 §7。

---

## 3. 需求一：永远不触发 OS 桌面通知

**原则**：代码保留，仅让「触发」变为 no-op，并新增开关常量便于回滚。

### 3.1 后端

1. `apps/backend/internal/notifications/providers/system_provider.go`
   - `Send()` 开头直接 `return nil`（不再 shell-out）。保留 `sendDarwinNotification` 等实现函数不动。
   - 可加包级常量 `const systemNotificationsDisabled = true` 作为显式开关。
2. `apps/backend/internal/notifications/providers/local.go`
   - `Send()` 开头 `return nil`（不再向浏览器广播 WS `session.waiting_for_input`）。保留实现。
3. `apps/backend/internal/notifications/service/service.go`
   - `HandleTaskSessionStateChanged`：保留 delivery 去重逻辑，但 **跳过 local/system provider 的实际投递**，改为调用新的飞书通知路径（见 §5）。最简做法：在遍历 provider 时对 `ProviderTypeLocal`/`ProviderTypeSystem` `continue`；或在 `dispatchProvider` 内对这两类型 no-op。
   - `ensureDefaultProviders` / `ensureSystemProvider`：**不再默认创建 enabled 的 local/system provider**（或创建但 `Enabled: false`），避免首启动又把 OS 通道打开。

### 3.2 前端

4. `apps/web/lib/ws/handlers/notifications.ts`
   - `NOTIFICATION_EVENT_TASK_SESSION_WAITING_FOR_INPUT` handler 开头直接 `return`（保留函数与注册）。
5. （可选）`apps/web/components/settings/notifications-settings-actions.ts`
   - `handleTestNotification` 的 `new Notification(...)` no-op 或在 UI 隐藏「桌面通知」区块（保留代码）。

### 3.3 测试

- 后端：`system_provider_test.go` / `service` 相关测试断言 `Send` 为 no-op、`HandleTaskSessionStateChanged` 不再调用 local/system。
- 前端：`notifications.ts` 对应单测（若有）断言不再 `new Notification`。

---

## 4. 需求二：New Task 弹窗新增 JNPM ID 选填字段

### 4.1 前端 UI

1. `apps/web/components/task-create-dialog-state.ts`（`useDialogFormState`）
   - 新增 `jnpmId` state + `setJnpmId`。
2. `apps/web/components/task-create-dialog.tsx`（`CreateModeBody`）
   - 在 `DynamicTaskForm` 下方新增一个 `Input`：
     - label：`JNPM ID`
     - placeholder：`JNPM单号，需要带#`
     - 非必填（无红色星号）
     - `value={fs.jnpmId}` / `onChange`
   - 可复用 `task-create-dialog-p4-fields.tsx` 的 `FieldLabel` 样式。
3. 通过 `SubmitHandlersDeps` 把 `jnpmId` 传入 `useTaskSubmitHandlers`。

### 4.2 提交链路（写入 metadata["jnpm_id"]）

4. `apps/web/components/task-create-dialog-submit.tsx`
   - `buildDynamicPrompt()` 返回的 `metadata` 中，追加 `if (jnpmId.trim()) metadata.jnpm_id = jnpmId.trim();`
   - 影响 `handleCreateSubmit` / `handleCreateWithPlanMode` / `handleCreateWithoutAgent`（都调用 `buildDynamicPrompt`）。
   - 注意：`handleCreatePlanMode`（description 为空的纯 plan 分支）目前不带 metadata，如需 plan 模式也保存 jnpm_id，需额外把 metadata 传进去。
5. `metadata` 已由 `buildCreateTaskPayload`（`task-create-dialog-helpers.ts:135`）透传，无需改后端即可落库。

### 4.3 后端（可选强化）

6. `apps/backend/internal/task/models/models.go`
   - 新增常量 `MetaKeyJnpmID = "jnpm_id"`，供后端引用（避免魔法字符串）。
7. metadata 已自动持久化，**无需 DB migration**。
   - 若未来需要在 kanban 卡片展示 JNPM 号或按其查询，再考虑加索引列（本期不做）。

### 4.4 测试

- `task-create-dialog-submit.test.tsx`：断言填入 JNPM ID 后 payload.metadata.jnpm_id 正确；留空时不写入。

---

## 5. 需求三：接入飞书机器人通知（JNPM 指派人）

分三个模块：**JNPM 客户端**、**飞书客户端 + notifier**、**通知路径接线**。

### 5.1 新增 `internal/jnpm` 包（查询指派人）

参照 `internal/jira` 的最小子集（不需要 poller/issue-watch）：

```
apps/backend/internal/jnpm/
├── models.go     // Config{BaseURL, Token(env)}, IssueDetail, Assignee
├── client.go     // Client 接口 + APIError
├── http_client.go// HTTPClient 实现：do(ctx, GET, path, out) + PRIVATE-TOKEN 头
├── service.go    // Service: ResolveAssigneeEmail(ctx, rawJnpmID)
├── provider.go   // Provide(log)：读 env；PCRAFT_MOCK_JNPM → mock
├── mock_client.go
└── *_test.go
```

关键方法：

- `Client.GetIssue(ctx, issueID int) (*IssueDetail, error)` → `GET {base}/v1/open-api/projects/issues/{id}`，解析 `payload`。
- `Service.ResolveAssigneeEmail(ctx, rawJnpmID string) (email, name string, err error)`：
  1. strip `#`/空白 → 数字 ID。
  2. `GetIssue` → 从 `assignee`/`assignedTo`/`owner` 取 email（对齐 ai_summarize `_extract_assignee`）。
  3. 返回 email（**可能为空**；由上层 service 降级到 admin，见 §5.3）。

base_url：**env** `PCRAFT_JNPM_BASE_URL`（默认 `https://jn-p-api.bytedance.net/jnpm/`，见调研文档 §3.1）。token：**env** `PCRAFT_JNPM_TOKEN`（`PRIVATE-TOKEN` 头）。

### 5.2 新增 `internal/lark` 包（飞书机器人）

```
apps/backend/internal/lark/
├── models.go     // Config{AppID, AppSecret, BaseDomain} 全部来自 env
├── client.go     // Client 接口
├── http_client.go// tenant_access_token 缓存/刷新 + SendTextByEmail
├── notifier.go   // Notifier: NotifyByEmail(ctx, email, title, body)  // 纯文本
├── provider.go   // Provide(log)：读 env；PCRAFT_MOCK_LARK → mock
├── mock_client.go
└── *_test.go
```

关键方法（直接 REST，对齐 ai_summarize 逻辑）：

- 取 token：`POST {base}/open-apis/auth/v3/tenant_access_token/internal` body `{app_id, app_secret}` → `tenant_access_token`（带 `expire` 秒，做内存缓存 + 提前刷新）。
- 发私信：`POST {base}/open-apis/im/v1/messages?receive_id_type=email`，header `Authorization: Bearer {token}`，body `{receive_id: email, msg_type: "text", content: "{\"text\":\"...\"}"}`。
- base_domain：**env** `PCRAFT_LARK_BASE_DOMAIN`，默认 `https://open.feishu.cn`，可配 `https://open.larksuite.com`。

app_id/app_secret：**env** `PCRAFT_LARK_APP_ID` / `PCRAFT_LARK_APP_SECRET`。

### 5.3 通知路径接线（把「发通知」改为「发飞书给收件人」）

采用 **方案 A：新增 `ProviderTypeLark` provider + 在 service 层解析收件人**。

1. `apps/backend/internal/notifications/models/models.go`：新增 `ProviderTypeLark ProviderType = "lark"`。
2. `apps/backend/internal/notifications/providers/provider.go`：`Message` 新增字段 `TargetEmail string`（向后兼容）。
3. 新建 `apps/backend/internal/notifications/providers/lark_provider.go`：
   - 实现 `Provider` 接口，构造时注入 `lark.Notifier`。
   - `Send(ctx, msg)`：`lark.Notifier.NotifyByEmail(ctx, msg.TargetEmail, msg.Title, msg.Body)`（纯文本）。收件人已在 service 层解析好，provider 只负责投递。
4. **收件人解析（service 层，`HandleTaskSessionStateChanged` + `HandleInboxItem` 共用一个 helper）**：

   ```
   resolveRecipientEmail(ctx, taskID) string:
     admin := env PCRAFT_NOTIFY_ADMIN_EMAIL
     if taskID == "": return admin
     task := taskRepo.GetTask(taskID)
     jnpmID := task.Metadata["jnpm_id"]  // 字符串，带 #
     if jnpmID == "": return admin                       // 无 jnpm_id → admin
     email := jnpmSvc.ResolveAssigneeEmail(ctx, jnpmID)  // 有 jnpm_id → 查 assignee
     if email == "": return admin                        // assignee 缺失/无 email → admin
     return email
   ```

   解析出的 email 写入 `providers.Message.TargetEmail` 传给 lark provider。

5. **触发范围 = 全部 `WAITING_FOR_INPUT`**：`HandleTaskSessionStateChanged` 现有的 `newState != "WAITING_FOR_INPUT" → return` 门槛保留（本就覆盖所有 WAITING_FOR_INPUT 触发源，含 permission / turn 完成 / clarification）。delivery 去重逻辑保留。
6. `notifications/service.NewService` 注册 `ProviderTypeLark`（构造函数新增入参 `jnpm.Service` + `lark.Notifier`）。
7. `ensureDefaultProviders`：默认创建一个 **enabled 的 Lark provider**（订阅 `session.waiting_for_input` + `office.inbox_item`），取代原来默认创建的 local/system。

### 5.4 DI 接线

5. `apps/backend/internal/backendapp/services.go`
   - 新增 `initJnpmService(log)`、`initLarkService(log)`（读 env，对齐 `initJiraService` 的组织方式，但无 db/secrets 入参）。
6. `apps/backend/internal/backendapp/gateway.go`（或 services 组装处）
   - 把 `jnpm.Service` + `lark.Notifier` 注入 `notificationservice.NewService(...)`（扩展其构造函数签名）。
   - 本期 **无需** `RegisterRoutes`（无配置端点，全 env）。

### 5.5 配置（全部 env）

本期所有配置走环境变量（无 Settings UI、不强制 secret store）：

| env | 作用 | 默认 |
|-----|------|------|
| `PCRAFT_JNPM_BASE_URL` | JNPM 基址 | `https://jn-p-api.bytedance.net/jnpm/` |
| `PCRAFT_JNPM_TOKEN` | JNPM `PRIVATE-TOKEN` | 空（空则跳过 JNPM 查询，直接 admin fallback）|
| `PCRAFT_LARK_BASE_DOMAIN` | 飞书基址 | `https://open.feishu.cn` |
| `PCRAFT_LARK_APP_ID` | 飞书 app_id | 空 |
| `PCRAFT_LARK_APP_SECRET` | 飞书 app_secret | 空 |
| `PCRAFT_NOTIFY_ADMIN_EMAIL` | 降级/无 jnpm_id 时的收件人 | 空（空则记 warn 日志、不发）|
| `PCRAFT_MOCK_JNPM` / `PCRAFT_MOCK_LARK` | E2E mock 开关 | false |

- 读取集中在各包 `provider.go` 的 `Provide()`，用 `os.Getenv`；可选在 `profiles.yaml` 补默认值。
- 飞书 app_id/app_secret / JNPM token 属敏感信息，本期按用户要求直接 env；日志中禁止打印。

### 5.6 测试

- `jnpm`：`http_client_test.go`（mock server 返回 issue 详情 → 断言 assignee email 解析）、strip `#` 归一化测试。
- `lark`：token 缓存/刷新、`SendTextByEmail` 请求构造（mock server）。
- `notifications`：`HandleTaskSessionStateChanged` 在有/无 jnpm_id、有/无 assignee email 时的分支（用 mock jnpm/lark）。
- goleak：新包若有后台 goroutine（token 刷新定时器）需遵守 owner start/stop + `goleak.VerifyTestMain`。

---

## 6. 实施顺序（建议分 PR / 分阶段）

1. **阶段 1｜禁用 OS 通知**（§3）：最小、独立、可先合。
2. **阶段 2｜JNPM ID 字段**（§4）：前端为主，后端仅加常量。
3. **阶段 3｜JNPM 客户端**（§5.1）：可独立测试（给 issue ID 拿 assignee）。
4. **阶段 4｜飞书客户端 + notifier**（§5.2）：可独立测试（给 email 发消息）。
5. **阶段 5｜通知路径接线**（§5.3-5.4）：串起来，替换 OS 通道为飞书通道。
6. **阶段 6｜配置/密钥/文档**（§5.5）+ 更新 `AGENTS.md`（新增 `internal/jnpm`、`internal/lark` 包说明）。

每阶段后：`make fmt` → `make typecheck test lint`（后端）/ `pnpm --filter @pcraft/web lint typecheck test`（前端）。

---

## 7. 已确认的决策（用户 2026-07-08 拍板）

1. **触发范围**：覆盖 **所有** `WAITING_FOR_INPUT`（permission / turn 完成 / clarification 等全部）。
2. **指派人**：= JNPM issue 的 `assignee`。若无 assignee 或无 email → **fallback 到 admin**（env `PCRAFT_NOTIFY_ADMIN_EMAIL`）。
3. **无 jnpm_id 的任务**：直接 **飞书通知到 admin**（同一 admin env）。
4. **全部配置来源**：**env**（JNPM base_url/token、飞书 base_domain/app_id/app_secret、admin 邮箱）。
5. **消息形态**：**纯文本**（先不做交互卡片）。

### 仍建议本期不做（可后续迭代）

- 飞书交互卡片 + WS 卡片按钮回执同步（ai_summarize 里的多消息 PATCH 同步）。
- Settings UI 配置页（本期纯 env）。
- 迁移凭据到 secret store（本期直接 env）。

---

## 8. 关键文件索引

| 用途 | 路径 |
|------|------|
| 通知分发 | `apps/backend/internal/notifications/service/service.go` |
| OS toast provider | `apps/backend/internal/notifications/providers/system_provider.go` |
| WS→浏览器 provider | `apps/backend/internal/notifications/providers/local.go` |
| Provider 接口 + Message | `apps/backend/internal/notifications/providers/provider.go` |
| Provider 类型枚举 | `apps/backend/internal/notifications/models/models.go` |
| 通知接线 | `apps/backend/internal/backendapp/gateway.go` |
| 前端浏览器通知 handler | `apps/web/lib/ws/handlers/notifications.ts` |
| New Task 弹窗 | `apps/web/components/task-create-dialog.tsx`（`CreateModeBody`） |
| 弹窗表单 state | `apps/web/components/task-create-dialog-state.ts` |
| 弹窗动态字段渲染 | `apps/web/components/task-create-dialog-p4-fields.tsx` |
| 提交/metadata 组装 | `apps/web/components/task-create-dialog-submit.tsx`（`buildDynamicPrompt`）|
| create payload | `apps/web/components/task-create-dialog-helpers.ts`（`buildCreateTaskPayload`）|
| createTask API | `apps/web/lib/api/domains/kanban-api.ts:47` |
| HTTP handler | `apps/backend/internal/task/handlers/task_http_handlers.go:495` |
| Service.CreateTask | `apps/backend/internal/task/service/service_tasks.go:64` |
| Task model / metadata | `apps/backend/internal/task/models/models.go:315` |
| tasks 表 schema | `apps/backend/internal/task/repository/sqlite/base_schema.go:247` |
| 集成模板 | `apps/backend/internal/jira/`、`apps/backend/internal/linear/` |
| 集成注册 | `apps/backend/internal/backendapp/services.go:335` |
| secret 接口 | `apps/backend/internal/secrets/store.go` |
| secretadapter | `apps/backend/internal/integrations/secretadapter/secretadapter.go` |
| JNPM 参考实现 | `C:\Users\Admin\Documents\GitHub\ai_summarize\src\app\integrations\jnpm\client.py` |
| 飞书参考实现 | `C:\Users\Admin\Documents\GitHub\ai_summarize\src\app\integrations\lark\client.py` |

---

## 9. 实现完成记录（2026-07-08）

三条需求均已实现并通过 `go build ./...` / `go vet` / 前端 `typecheck` / 相关单测。

### 9.1 阶段一：禁用 OS 桌面通知（代码保留）
- `providers/system_provider.go`：新增 `const systemNotificationsDisabled = true`；`Available()` 与 `Send()` 顶部短路返回，OS toast 永不触发。
- `providers/local.go`：新增 `const localNotificationsDisabled = true`；`Send()` 顶部短路，WS→浏览器广播永不触发。
- `apps/web/lib/ws/handlers/notifications.ts`：新增 `const DESKTOP_NOTIFICATIONS_DISABLED: boolean = true`；handler 顶部 early-return，`new Notification` 永不触发。

### 9.2 阶段二：New Task 弹窗 JNPM ID 字段
- `task-create-dialog.tsx`：`CreateModeBody` 新增 `JNPM ID` 输入框（placeholder「JNPM单号，需要带#」，`data-testid="task-jnpm-id-input"`）；`useDialogDynamicFormState` 持有 `jnpmId` 并在开窗时重置。
- `task-create-dialog-submit.tsx`：`buildDynamicPrompt` 把非空 `jnpmId`（trim 后）写入 `metadata.jnpm_id`。
- `task/models/models.go`：新增 `MetaKeyJnpmID = "jnpm_id"`。metadata 走既有 JSON 列，无需 DB 迁移。

### 9.3 阶段三/四：JNPM + 飞书客户端
- `internal/jnpm/`：`Client.GetIssue` → `Service.ResolveAssigneeEmail`（assignee→assignedTo→owner 优先级，`#`/文本前缀解析成 issue id）。
- `internal/lark/`：`Client.SendTextByEmail`（tenant_access_token 缓存 + email 收件人）→ `Notifier.NotifyByEmail`（纯文本 title+body）。
- 两者均支持未配置降级（`Enabled()=false`）与 `PCRAFT_MOCK_JNPM/PCRAFT_MOCK_LARK` mock。

### 9.4 阶段五：通知路径接线
- `notifications/models`：新增 `ProviderTypeLark = "lark"`。
- `providers/lark_provider.go`：新增 `LarkProvider`（按 `Message.TargetEmail` 投递）。`Message` 新增 `TargetEmail` 字段。
- `notifications/service/service.go`：`NewService` 增参 `larkSender / jnpmResolver / adminEmail`；新增 `resolveRecipientEmail`（有 `jnpm_id` 且解析成功→指派人邮箱，否则→admin）；`ensureLarkProvider` 首启自动创建 enabled 的 Lark provider（订阅两类事件）。
- `backendapp/gateway.go`：`jnpm.Provide(log)` + `lark.Provide(log)` + `PCRAFT_NOTIFY_ADMIN_EMAIL` 注入 `NewService`。

### 9.5 环境变量一览

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PCRAFT_JNPM_BASE_URL` | `https://jn-p-api.bytedance.net/jnpm/` | JNPM Open API base |
| `PCRAFT_JNPM_TOKEN` | 空 | JNPM `PRIVATE-TOKEN`；为空则禁用指派人解析 |
| `PCRAFT_LARK_BASE_DOMAIN` | `https://open.feishu.cn` | 飞书开放平台域名（国际版用 `https://open.larksuite.com`）|
| `PCRAFT_LARK_APP_ID` | 空 | 飞书自建应用 app_id |
| `PCRAFT_LARK_APP_SECRET` | 空 | 飞书自建应用 app_secret；与 app_id 任一为空则禁用飞书投递 |
| `PCRAFT_NOTIFY_ADMIN_EMAIL` | 空 | 无 `jnpm_id` / 无指派人邮箱时的兜底收件人 |
| `PCRAFT_MOCK_JNPM` / `PCRAFT_MOCK_LARK` | — | 置 `true` 使用内存 mock（e2e/测试）|
