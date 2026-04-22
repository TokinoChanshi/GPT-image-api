# gpt-image（新项目）设计稿（v0.1）

日期：2026-04-22  
范围：在当前仓库中新建全新项目目录 `gpt-image/`，在不破坏现有 `backend/`、`frontend/` 的前提下，逐步迁移与重构。  
目标：同时产出 **开源版（单租户）** 与 **自用版（平台化、多用户）**，共享核心内核与 UI 资产，但开源发布时允许剔除自用版专有能力。

---

## 0. 当前代码现状（用于迁移参考）

当前仓库已有：

- `backend/`：Go + Gin，SQLite（但根 `docker-compose.yml` 同时起了 MySQL，存在不一致）。
- `frontend/`：Vue3 + Element Plus，已有 Dashboard / AccountPool / APIKeys 视图雏形。
- 参考项目（本地目录）：`basketikun_chatgpt2api/`、`gpt2api/`、`fran0220_chatgpt2api/`。

已发现的硬问题（迁移时优先修复/规避）：

- `backend/api/image.go` 里把 `GenerateImage()` 当成返回 `string` 使用，但 `backend/core/openai.go` 实际返回 `[]string`（编译/运行必炸）。
- `/v1/admin/*` 未做鉴权（前端还在裸调本地地址）。
- 部署形态不一致：compose 起 MySQL，但后端走 SQLite 文件库。

> 结论：不在旧代码上“硬修补”，而是在 `gpt-image/` 中重构出清晰分层后再迁移。

---

## 1. 产品形态与版本策略

### 1.1 开源版（OSS）

定位：单租户的“账号池图片 API 网关”，类似 `basketikun/chatgpt2api` 的体验。

- 认证：单一全局 `AUTH_KEY`（`Authorization: Bearer <AUTH_KEY>`）。
- 上游：仅支持 **账号池（chatgpt.com 协议请求）**。
- 任务：仅内存保存（重启丢失）。
- 数据：尽量轻部署（SQLite/文件存储皆可，偏向 SQLite 单文件）。
- UI：账号池管理 + 基础统计 + 在线测试（后续渐进）。

### 1.2 自用版（PRO / 私有）

定位：API 分发平台（用户可登录），同时支持共享池/专属池分发策略。

- 用户：支持注册、登录；注册需 **邮箱验证**；管理员可选开启/关闭“开放注册”。
- Key：用户可自助管理 API Keys（创建/禁用/删除/配额/绑定池）。
- 上游：支持 **账号池** + **官方 OpenAI API Key**（可配置优先级/路由策略）。
- 任务：落库（MySQL 8），可追溯、可审计。
- UI：用户侧（Keys/用量/任务）+ 管理员侧（用户/池/账号/审计/系统配置）。

### 1.3 “同一套代码”的落地方式（推荐）

为了既“同一套开发”，又能“发布开源版时放弃/剔除部分能力”，采用：

- **共享 core**：所有通用能力放 `gpt-image/server/internal/core/...`（不会涉及私有业务细节）。
- **按 Edition 装配**：
  - OSS：`gpt-image/server/cmd/gpt-image-oss`（仅引用 `core` + `edition/oss`）
  - PRO：`gpt-image/server/cmd/gpt-image-pro`（引用 `core` + `edition/pro`）
- **开源发布策略**：开源发布时仅发布 OSS 相关目录；PRO 目录不进入开源包（通过发布脚本/手动剔除）。

> 备注：若未来你希望“真正一个公开仓库 + build tag 隐藏私有能力”，在合规与保密层面并不现实；更稳的是“公开包中不包含私有代码”。

---

## 2. 非功能目标（NFR）

- 单实例高并发：先以 **并发 20 个进行中图片任务**为目标（可配置）。
- 稳定性：可控超时、取消、错误分类；避免账号被并发打爆。
- 可观测：结构化日志（请求/任务/账号维度）；PRO 版增加审计日志、关键操作记录。
- 可配置：核心行为（并发、重试轮数、冷却时间、限流）均可配置。

---

## 3. 对外 API（OpenAI 兼容子集 + 扩展）

### 3.1 认证

- OSS：请求头必须携带
  - `Authorization: Bearer <AUTH_KEY>`
- PRO：
  - OpenAI 兼容接口仍使用 `Authorization: Bearer <sk-...>`（API Key）
  - Web 登录接口使用 Cookie/JWT（实现细节在 PRO 版定义）

### 3.2 `GET /v1/models`

返回可用模型列表（最小可用集）：

- OSS：`gpt-image-1`、`gpt-image-2`（以及你需要兼容的别名）
- PRO：同上 + 可包含官方模型列表的子集（按配置）

### 3.3 `POST /v1/images/generations`

请求（JSON，兼容为主，允许扩展字段）：

```json
{
  "model": "gpt-image-2",
  "prompt": "a cyberpunk cat walking in rainy Tokyo street",
  "n": 1,
  "size": "1024x1024",
  "response_format": "b64_json",
  "async": false
}
```

行为：

- 默认 `async=false`：阻塞等待完成并返回结果。
- `async=true`：立即创建任务并返回 `task_id`，客户端使用 tasks 接口轮询。
- `response_format`：
  - 默认 `b64_json`：返回纯 base64（不带 `data:image/...` 前缀），默认 PNG 编码。
  - `url`：返回上游 `download_url` 透传（暂不做本地签名代理/缓存）。

响应（同步完成）：

```json
{
  "created": 1775366000,
  "data": [
    { "b64_json": "...." }
  ]
}
```

响应（异步）：

```json
{
  "task_id": "imgtsk_..."
}
```

### 3.4 `POST /v1/images/edits`

兼容 OpenAI 常见用法（multipart/form-data）：

- `image`：必填文件
- `mask`：可选文件
- `prompt`：必填
- `model` / `n` / `size` / `response_format` / `async`：同 generations

> OSS 第一阶段可先实现“最小可用 edits”（把图片作为附件上传给上游并触发编辑/重绘流程）；实现细节需结合上游协议与现有参考实现落地。

### 3.5 `GET /v1/images/tasks/:id`

返回任务状态与结果（OpenAI 官方没有该接口，这是扩展接口）。

建议状态机：

- `queued`：排队中（未获取到全局并发/账号 lease）
- `running`：生成中
- `succeeded`：成功
- `failed`：失败（含错误码/消息）

响应示例：

```json
{
  "id": "imgtsk_...",
  "status": "succeeded",
  "created_at": 1775366000,
  "started_at": 1775366010,
  "finished_at": 1775366042,
  "result": {
    "created": 1775366042,
    "data": [{ "url": "https://..." }]
  },
  "error": null
}
```

存储差异：

- OSS：内存保存（可配置 TTL，例如 24h；重启丢失）
- PRO：MySQL 持久化（可配置保留策略）

---

## 4. 内核架构（Server）

### 4.1 核心模块（core）

1) `Limiter`（全局并发门闩）
- 配置：`MAX_INFLIGHT=20`
- 实现：带超时的 semaphore（拿不到则排队或返回 503，按配置）

2) `TaskManager`
- 支持同步/异步两种调用路径
- 抽象存储接口：
  - OSS：`MemoryTaskStore`
  - PRO：`MySQLTaskStore`

3) `UpstreamProvider`（上游适配器接口）
- `Generate(prompt, opts) -> images`
- `Edit(image, mask, prompt, opts) -> images`
- `ListModels()`

4) `PoolScheduler`（仅账号池上游用）
- 目标：在高并发下稳定租约（lease）与公平分配
- 策略（第一期）：
  - 每账号同一时刻最多 1 个任务（lease）
  - 429/失败进入冷却（cooldown）
  - 选择算法：可用 + 最少使用优先（usage_count asc）+ 冷却/重置窗口过滤
  - `max_turns`（是否重试）可配置：默认建议 1（提速），必要时可调回 3

### 4.2 OSS 装配

- `Auth`: 全局 `AUTH_KEY`
- `Upstream`: 账号池适配器
- `TaskStore`: 内存
- `DB`: SQLite（用于账号池/基础配置/统计，具体表见 5.1）

### 4.3 PRO 装配

- `Auth`: 用户登录（Web）+ API Keys（OpenAI 兼容接口）
- `Upstream`: 账号池 + 官方 OpenAI，上游路由按 Key/分组/策略决定
- `TaskStore`: MySQL
- `DB`: MySQL 8（表见 5.2）

---

## 5. 数据模型（建议）

### 5.1 OSS（轻量）

最小表集合（SQLite）：

- `accounts`：上游账号（token/cookie/device/session/状态/额度/冷却/统计）
- `settings`：全局配置（重试轮数、并发、冷却、超时等）
- `audit_logs`（可选）：管理操作日志（导入/删除账号等）

说明：

- OSS 不引入“用户体系”，也不引入“多 Key 体系”（只有全局 AUTH_KEY）。

### 5.2 PRO（平台化，MySQL）

建议表集合：

- 用户与认证
  - `users`（email、password_hash、status、role）
  - `email_verification_codes`（email、code、expires_at、used_at）
  - `sessions`（可选，若用 session/cookie）
- API 分发
  - `api_keys`（key_hash、user_id、status、name、rate_limit、quota、pool_binding）
  - `usage_records`（按天/小时聚合、扣费、请求数、成功/失败）
- 上游资源池
  - `pools`（共享池/专属池）
  - `pool_members`（账号归属/绑定关系）
  - `accounts`（账号池账户信息，含能力标记、冷却、失败计数、最近使用时间）
  - `proxies`（可选，若实现代理池）
- 任务与审计
  - `image_tasks`（id、api_key_id、status、request、result、error、timing）
  - `audit_logs`（管理员/用户关键操作）

安全建议：

- token/cookie 等敏感字段：PRO 版建议加密存储（AES-GCM），密钥来自环境变量。
- password：Argon2id / bcrypt（实现阶段定）。

---

## 6. 前端 UI（Web）

### 6.1 迁移策略

在 `gpt-image/web/` 中沿用你现有 `frontend/`（Vue3 + Element Plus），把“看起来像 UI”的页面，改成真正可用的管理台。

### 6.2 OSS UI（最小）

- 登录：输入全局 AUTH_KEY（或在浏览器端存储后每次请求带上）
- 账号池管理：
  - 导入/批量导入 token
  - 列表：状态、额度、冷却、最近使用、成功/失败
  - 刷新/探测能力（可触发后台探测任务）
- 在线测试：输入 prompt，选择模型/尺寸，发起生成

### 6.3 PRO UI（平台）

用户侧：

- 注册（邮箱验证）、登录、找回密码（后续）
- API Keys：创建/禁用/删除，绑定“共享池/专属池”
- 用量：请求数、成功率、扣费/额度
- 任务：列表、详情、重试（可选）

管理员侧：

- 用户管理
- 资源池管理（共享池/专属池）
- 账号池导入与健康检查
- 系统配置（并发/重试/冷却/模型映射）
- 审计日志

---

## 7. 配置与部署

### 7.1 OSS

- 单容器即可（SQLite + 本地文件）
- 环境变量示例：
  - `PORT=8080`
  - `AUTH_KEY=...`
  - `MAX_INFLIGHT=20`
  - `TASK_TTL=24h`
  - `IMG_MAX_TURNS=1`（可调 3）

### 7.2 PRO

- MySQL 8 必需
- 环境变量示例：
  - `MYSQL_DSN=...`
  - `JWT_SECRET=...`
  - `EMAIL_SMTP_*`（邮箱验证码）
  - `ENCRYPTION_KEY=...`

---

## 8. 里程碑（建议）

M0（打底）：

- `gpt-image/server` 跑起来：路由、统一错误、日志、配置
- `GET /v1/models`
- `POST /v1/images/generations`（同步 + async=true 任务模式）
- `GET /v1/images/tasks/:id`（OSS 内存）
- 账号池最小调度：全局并发 20 + 每账号 lease + 冷却

M1（补齐 edits + UI 可用）：

- `POST /v1/images/edits`（最小可用）
- OSS 管理 UI：账号导入/列表/刷新、在线测试

M2（PRO 平台最小闭环）：

- 用户注册/登录（邮箱验证）
- API Keys 管理
- tasks 落 MySQL
- 共享池/专属池的绑定逻辑

---

## 9. 未决项（明确可配置/后续决定）

- 是否做图片缓存/签名代理（目前定为不做，透传上游 url）
- `b64_json` 的图片编码（目前定为 PNG；如需 webp/jpg 可配置）
- IMG2 “重试轮数”：目前做成可配置（默认 1；必要时调回 3）

