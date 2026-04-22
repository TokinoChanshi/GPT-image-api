# gpt-image OSS MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在同一仓库中新建 `gpt-image/` 全新项目，先交付 **OSS（单租户）MVP**：支持 OpenAI 官方 Images API 的 Key 池化分发、全局并发上限 20、同步/异步任务模式，以及一个可用的基础后台前端骨架（登录写入 auth-key + Dashboard/Keys/Tasks 三页）。

**Architecture:** Go(Gin) 作为 HTTP 网关 + TaskManager(worker pool + queue) 负责并发与异步任务；Scheduler 负责从 OpenAI Key 池中选择可用 key（lease + cooldown）；Provider(OpenAI Official) 负责调用 `api.openai.com/v1/images/*`。OSS 版本任务只存内存（TTL），重启丢失。

**Tech Stack:** Go（Gin、net/http、uuid）、Vue3（Vite、TypeScript、Element Plus、Axios）。

---

## 0. 目录与文件结构（锁定分解）

> 本计划 **不修改** 现有 `backend/`、`frontend/`；所有新代码落到 `gpt-image/`。

### 后端（Go）

- Create: `gpt-image/go.mod`
- Create: `gpt-image/cmd/gpt-image-oss/main.go`
- Create: `gpt-image/cmd/gpt-image-pro/main.go`（先做可编译的占位入口：与 OSS 共用 wiring；Pro 功能另起 plan）
- Create: `gpt-image/internal/config/config.go`
- Create: `gpt-image/internal/app/app.go`
- Create: `gpt-image/internal/httpapi/router.go`
- Create: `gpt-image/internal/httpapi/middleware/auth_oss.go`
- Create: `gpt-image/internal/httpapi/handlers/handlers.go`
- Create: `gpt-image/internal/httpapi/handlers/models.go`
- Create: `gpt-image/internal/httpapi/handlers/images_generations.go`
- Create: `gpt-image/internal/httpapi/handlers/images_edits.go`
- Create: `gpt-image/internal/httpapi/handlers/tasks.go`
- Create: `gpt-image/internal/httpapi/handlers/admin.go`（OSS 管理：stats、keys 列表、tasks 列表）
- Create: `gpt-image/internal/providers/provider.go`
- Create: `gpt-image/internal/providers/openai/images.go`
- Create: `gpt-image/internal/providers/openai/types.go`
- Create: `gpt-image/internal/scheduler/keypool.go`
- Create: `gpt-image/internal/tasks/types.go`
- Create: `gpt-image/internal/tasks/store_memory.go`
- Create: `gpt-image/internal/tasks/manager.go`

### 后端测试

- Create: `gpt-image/internal/providers/openai/images_test.go`
- Create: `gpt-image/internal/scheduler/keypool_test.go`
- Create: `gpt-image/internal/tasks/manager_test.go`
- Create: `gpt-image/internal/httpapi/handlers/images_generations_test.go`
- Create: `gpt-image/internal/httpapi/handlers/images_edits_test.go`

### 前端（Vue）

> 直接复用你现有 `frontend/` 的风格与布局，但放到新目录，避免把旧项目“拽乱”。

- Create (copy from existing): `gpt-image/web/`（复制 `frontend/`，**不要复制** `node_modules/`）
- Modify: `gpt-image/web/src/router/index.ts`（增加 `/login` + 路由守卫）
- Create: `gpt-image/web/src/api/http.ts`（Axios 实例 + 注入 Authorization）
- Create: `gpt-image/web/src/views/Login.vue`
- Modify: `gpt-image/web/src/views/Dashboard.vue`（对接 `/v1/admin/stats`）
- Modify: `gpt-image/web/src/views/AccountPool.vue` → 改成 `ProviderKeys.vue`（对接 `/v1/admin/keys`）
- Modify: `gpt-image/web/src/views/APIKeys.vue` → 改成 `Tasks.vue`（对接 `/v1/admin/tasks`）
- Modify: `gpt-image/web/src/layout/MainLayout.vue`（文案与菜单）

---

## Task 1: 初始化 gpt-image Go 工程骨架（可编译可运行）

**Files:**
- Create: `gpt-image/go.mod`
- Create: `gpt-image/cmd/gpt-image-oss/main.go`
- Create: `gpt-image/cmd/gpt-image-pro/main.go`
- Create: `gpt-image/internal/config/config.go`
- Create: `gpt-image/internal/app/app.go`

- [ ] **Step 1: 创建 go.mod**

`gpt-image/go.mod`：
```go
module gpt-image

go 1.22

require (
  github.com/gin-gonic/gin v1.12.0
  github.com/google/uuid v1.6.0
)
```

- [ ] **Step 2: 写最小可运行入口（OSS）**

`gpt-image/cmd/gpt-image-oss/main.go`：
```go
package main

import (
  "log"
  "os"

  "gpt-image/internal/app"
  "gpt-image/internal/config"
)

func main() {
  cfg := config.Load()
  server := app.NewServer(cfg, app.ModeOSS)

  addr := ":" + cfg.Port
  if v := os.Getenv("ADDR"); v != "" {
    addr = v
  }

  log.Printf("[gpt-image][oss] listening on %s", addr)
  if err := server.Run(addr); err != nil {
    log.Fatalf("server error: %v", err)
  }
}
```

- [ ] **Step 3: 写 Pro 入口（先同 wiring，保证能编译）**

`gpt-image/cmd/gpt-image-pro/main.go`：
```go
package main

import (
  "log"

  "gpt-image/internal/app"
  "gpt-image/internal/config"
)

func main() {
  cfg := config.Load()
  server := app.NewServer(cfg, app.ModePro)
  addr := ":" + cfg.Port
  log.Printf("[gpt-image][pro] listening on %s", addr)
  if err := server.Run(addr); err != nil {
    log.Fatalf("server error: %v", err)
  }
}
```

- [ ] **Step 4: 写配置加载**

`gpt-image/internal/config/config.go`：
```go
package config

import (
  "os"
  "strconv"
  "strings"
)

type Config struct {
  Port string

  // OSS 单租户：下游请求 Bearer 校验
  AuthKey string

  // 上游 OpenAI 官方 keys：用逗号分隔
  OpenAIAPIKeys []string
  OpenAIBaseURL  string

  // 并发与队列
  MaxInflight int
  MaxQueue    int
  TaskTTLMin  int

  // Scheduler 策略
  PerKeyMaxInflight int
  Cooldown429Sec    int
}

func Load() Config {
  cfg := Config{
    Port: "8080",
    AuthKey: os.Getenv("AUTH_KEY"),
    OpenAIBaseURL: strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
    MaxInflight: 20,
    MaxQueue: 200,
    TaskTTLMin: 24 * 60,
    PerKeyMaxInflight: 1,
    Cooldown429Sec: 60,
  }

  if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
    cfg.Port = v
  }
  if v := strings.TrimSpace(os.Getenv("MAX_INFLIGHT")); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n > 0 {
      cfg.MaxInflight = n
    }
  }
  if v := strings.TrimSpace(os.Getenv("MAX_QUEUE")); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n >= 0 {
      cfg.MaxQueue = n
    }
  }
  if v := strings.TrimSpace(os.Getenv("TASK_TTL_MIN")); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n > 0 {
      cfg.TaskTTLMin = n
    }
  }
  if v := strings.TrimSpace(os.Getenv("PER_KEY_MAX_INFLIGHT")); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n > 0 {
      cfg.PerKeyMaxInflight = n
    }
  }
  if v := strings.TrimSpace(os.Getenv("COOLDOWN_429_SEC")); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n >= 0 {
      cfg.Cooldown429Sec = n
    }
  }

  rawKeys := strings.TrimSpace(os.Getenv("OPENAI_API_KEYS"))
  if rawKeys != "" {
    parts := strings.Split(rawKeys, ",")
    for _, p := range parts {
      k := strings.TrimSpace(p)
      if k != "" {
        cfg.OpenAIAPIKeys = append(cfg.OpenAIAPIKeys, k)
      }
    }
  }

  if cfg.OpenAIBaseURL == "" {
    cfg.OpenAIBaseURL = "https://api.openai.com"
  }
  return cfg
}
```

- [ ] **Step 5: 写最小 Server wiring（先只提供 /ping）**

`gpt-image/internal/app/app.go`：
```go
package app

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/config"
)

type Mode string

const (
  ModeOSS Mode = "oss"
  ModePro Mode = "pro"
)

func NewServer(cfg config.Config, mode Mode) *gin.Engine {
  r := gin.New()
  r.Use(gin.Recovery())

  r.GET("/ping", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "pong", "mode": mode})
  })

  return r
}
```

- [ ] **Step 6: 拉取依赖并验证能跑起来**

Run:
```powershell
cd gpt-image
go mod tidy
go run .\cmd\gpt-image-oss
```

Expected:
- 终端输出 listening
- 访问 `GET http://localhost:8080/ping` 返回 `{"message":"pong","mode":"oss"}`

- [ ] **Step 7:（可选）初始化 git 并提交**

```powershell
cd gpt-image
git init
git add .
git commit -m "chore: bootstrap gpt-image skeleton (oss/pro entrypoints)"
```

---

## Task 2: 定义 Provider 接口与 OpenAI Images Provider（含单测）

**Files:**
- Create: `gpt-image/internal/providers/provider.go`
- Create: `gpt-image/internal/providers/openai/types.go`
- Create: `gpt-image/internal/providers/openai/images.go`
- Test: `gpt-image/internal/providers/openai/images_test.go`

- [ ] **Step 1: 定义 Provider 接口与通用类型**

`gpt-image/internal/providers/provider.go`：
```go
package providers

import "context"

type ImagesResponse struct {
  Created int64 `json:"created"`
  Data []ImageData `json:"data"`

  // 透传字段（OpenAI images 可能返回）
  Background   string      `json:"background,omitempty"`
  OutputFormat string      `json:"output_format,omitempty"`
  Quality      string      `json:"quality,omitempty"`
  Size         string      `json:"size,omitempty"`
  Usage        interface{} `json:"usage,omitempty"`
}

type ImageData struct {
  B64JSON       string `json:"b64_json,omitempty"`
  URL           string `json:"url,omitempty"`
  RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type GenerateRequest struct {
  Prompt         string `json:"prompt"`
  Model          string `json:"model,omitempty"`
  N              int    `json:"n,omitempty"`
  Size           string `json:"size,omitempty"`
  Quality        string `json:"quality,omitempty"`
  Background     string `json:"background,omitempty"`
  OutputFormat   string `json:"output_format,omitempty"`
  ResponseFormat string `json:"response_format,omitempty"` // url|b64_json（对 GPT image：url 不支持）
}

type EditRequest struct {
  Prompt         string `json:"prompt"`
  Model          string `json:"model,omitempty"`
  N              int    `json:"n,omitempty"`
  Size           string `json:"size,omitempty"`
  Quality        string `json:"quality,omitempty"`
  Background     string `json:"background,omitempty"`
  OutputFormat   string `json:"output_format,omitempty"`
  ResponseFormat string `json:"response_format,omitempty"`
}

type ProviderError struct {
  StatusCode int
  Message    string
  Type       string
}

func (e *ProviderError) Error() string { return e.Message }

type ImagesProvider interface {
  Name() string
  ListModels(ctx context.Context) []string

  Generate(ctx context.Context, upstreamKey string, req GenerateRequest) (*ImagesResponse, error)
  Edit(ctx context.Context, upstreamKey string, req EditRequest, images [][]byte, masks [][]byte) (*ImagesResponse, error)
}
```

- [ ] **Step 2: 写 OpenAI Provider 的请求/响应结构**

`gpt-image/internal/providers/openai/types.go`：
```go
package openai

type apiErrorResp struct {
  Error struct {
    Message string `json:"message"`
    Type    string `json:"type"`
    Param   string `json:"param"`
    Code    string `json:"code"`
  } `json:"error"`
}
```

- [ ] **Step 3: 为 OpenAI Provider 写一个 failing test（httptest 模拟 OpenAI）**

`gpt-image/internal/providers/openai/images_test.go`：
```go
package openai

import (
  "context"
  "net/http"
  "net/http/httptest"
  "testing"

  "gpt-image/internal/providers"
)

func TestProvider_Generate_ParsesB64(t *testing.T) {
  ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/v1/images/generations" {
      w.WriteHeader(404); return
    }
    if r.Header.Get("Authorization") != "Bearer sk-test" {
      w.WriteHeader(401); return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"created":123,"data":[{"b64_json":"AAAA"}]}`))
  }))
  defer ts.Close()

  p := New(ts.URL, nil)
  resp, err := p.Generate(context.Background(), "sk-test", providers.GenerateRequest{
    Prompt: "hi",
    Model: "gpt-image-1",
    N: 1,
  })
  if err != nil {
    t.Fatalf("unexpected err: %v", err)
  }
  if resp.Created != 123 || len(resp.Data) != 1 || resp.Data[0].B64JSON != "AAAA" {
    t.Fatalf("bad resp: %#v", resp)
  }
}
```

- [ ] **Step 4: 实现 OpenAI Provider（最小实现让测试通过）**

`gpt-image/internal/providers/openai/images.go`：
```go
package openai

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "io"
  "mime/multipart"
  "net/http"
  "strings"
  "time"

  "gpt-image/internal/providers"
)

type Provider struct {
  baseURL string
  client  *http.Client
}

func New(baseURL string, client *http.Client) *Provider {
  if client == nil {
    client = &http.Client{Timeout: 60 * time.Second}
  }
  baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
  return &Provider{baseURL: baseURL, client: client}
}

func (p *Provider) Name() string { return "openai-official" }

func (p *Provider) ListModels(ctx context.Context) []string {
  // OSS MVP：先静态列出常见 image 模型；后续可做配置化
  return []string{
    "gpt-image-1.5",
    "gpt-image-1",
    "gpt-image-1-mini",
    "chatgpt-image-latest",
    "dall-e-3",
    "dall-e-2",
  }
}

func (p *Provider) Generate(ctx context.Context, upstreamKey string, req providers.GenerateRequest) (*providers.ImagesResponse, error) {
  body, _ := json.Marshal(req)
  u := p.baseURL + "/v1/images/generations"

  httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
  httpReq.Header.Set("Authorization", "Bearer "+upstreamKey)
  httpReq.Header.Set("Content-Type", "application/json")

  resp, err := p.client.Do(httpReq)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()

  raw, _ := io.ReadAll(resp.Body)
  if resp.StatusCode >= 400 {
    var er apiErrorResp
    _ = json.Unmarshal(raw, &er)
    msg := strings.TrimSpace(er.Error.Message)
    if msg == "" {
      msg = string(raw)
    }
    return nil, &providers.ProviderError{StatusCode: resp.StatusCode, Message: msg, Type: er.Error.Type}
  }

  var out providers.ImagesResponse
  if err := json.Unmarshal(raw, &out); err != nil {
    return nil, err
  }
  return &out, nil
}

func (p *Provider) Edit(ctx context.Context, upstreamKey string, req providers.EditRequest, images [][]byte, masks [][]byte) (*providers.ImagesResponse, error) {
  buf := &bytes.Buffer{}
  mw := multipart.NewWriter(buf)

  // fields
  _ = mw.WriteField("prompt", req.Prompt)
  if req.Model != "" { _ = mw.WriteField("model", req.Model) }
  if req.N > 0 { _ = mw.WriteField("n", fmt.Sprintf("%d", req.N)) }
  if req.Size != "" { _ = mw.WriteField("size", req.Size) }
  if req.Quality != "" { _ = mw.WriteField("quality", req.Quality) }
  if req.Background != "" { _ = mw.WriteField("background", req.Background) }
  if req.OutputFormat != "" { _ = mw.WriteField("output_format", req.OutputFormat) }
  if req.ResponseFormat != "" { _ = mw.WriteField("response_format", req.ResponseFormat) }

  // images[] (OpenAI API 支持多张)
  for i, b := range images {
    fw, _ := mw.CreateFormFile("image[]", fmt.Sprintf("image_%d.png", i))
    _, _ = fw.Write(b)
  }
  // mask（可选，兼容旧客户端/部分模型；如果传多个，只取第一个）
  if len(masks) > 0 {
    fw, _ := mw.CreateFormFile("mask", "mask.png")
    _, _ = fw.Write(masks[0])
  }

  _ = mw.Close()

  u := p.baseURL + "/v1/images/edits"
  httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, buf)
  httpReq.Header.Set("Authorization", "Bearer "+upstreamKey)
  httpReq.Header.Set("Content-Type", mw.FormDataContentType())

  resp, err := p.client.Do(httpReq)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()

  raw, _ := io.ReadAll(resp.Body)
  if resp.StatusCode >= 400 {
    var er apiErrorResp
    _ = json.Unmarshal(raw, &er)
    msg := strings.TrimSpace(er.Error.Message)
    if msg == "" { msg = string(raw) }
    return nil, &providers.ProviderError{StatusCode: resp.StatusCode, Message: msg, Type: er.Error.Type}
  }

  var out providers.ImagesResponse
  if err := json.Unmarshal(raw, &out); err != nil {
    return nil, err
  }
  return &out, nil
}
```

- [ ] **Step 5: 运行单测（先看到 FAIL 再修）**

Run:
```powershell
cd gpt-image
go test .\internal\providers\openai -v
```

Expected:
- 第一次应 FAIL（缺 import / 未实现）
- 修正后 PASS

- [ ] **Step 6: 提交**
```powershell
cd gpt-image
git add .
git commit -m "feat: add official OpenAI images provider with tests"
```

---

## Task 3: KeyPool Scheduler（round-robin + per-key lease + 429 cooldown）

**Files:**
- Create: `gpt-image/internal/scheduler/keypool.go`
- Test: `gpt-image/internal/scheduler/keypool_test.go`

- [ ] **Step 1: 写 failing test（lease + cooldown）**

`gpt-image/internal/scheduler/keypool_test.go`：
```go
package scheduler

import (
  "testing"
  "time"
)

func TestKeyPool_Acquire_LeaseAndCooldown(t *testing.T) {
  p := NewKeyPool([]string{"k1", "k2"}, KeyPoolConfig{
    PerKeyMaxInflight: 1,
    Cooldown429: 1 * time.Second,
  })

  k, rel, err := p.Acquire()
  if err != nil || k == "" || rel == nil {
    t.Fatalf("acquire failed: k=%q rel=%v err=%v", k, rel, err)
  }
  // 同 key 不能并发租第二次（因为 PerKeyMaxInflight=1，且 pool 只有 2 keys）
  k2, rel2, err := p.Acquire()
  if err != nil || k2 == "" || rel2 == nil {
    t.Fatalf("acquire2 failed: %v", err)
  }
  // pool 已满（两个 key 都 in-use），第三次应失败
  if _, _, err := p.Acquire(); err == nil {
    t.Fatalf("expected error when pool exhausted")
  }

  // 释放一个 key，并标记 429 冷却
  rel(false, ErrRateLimited)
  // 释放另一个 key 正常成功
  rel2(true, nil)

  // 冷却中的 key 不应被立刻获取（只能拿另一个）
  got, rel3, err := p.Acquire()
  if err != nil || got != "k2" {
    t.Fatalf("expected k2 available, got=%q err=%v", got, err)
  }
  rel3(true, nil)

  time.Sleep(1100 * time.Millisecond)
  // 冷却过后 k1 可再次获取
  got2, rel4, err := p.Acquire()
  if err != nil || (got2 != "k1" && got2 != "k2") || rel4 == nil {
    t.Fatalf("expected acquire ok after cooldown, got=%q err=%v", got2, err)
  }
  rel4(true, nil)
}
```

- [ ] **Step 2: 实现 KeyPool（让测试通过）**

`gpt-image/internal/scheduler/keypool.go`：
```go
package scheduler

import (
  "errors"
  "sort"
  "sync"
  "time"
)

var (
  ErrNoKeyAvailable = errors.New("no upstream key available")
  ErrRateLimited    = errors.New("upstream rate limited")
)

type KeyPoolConfig struct {
  PerKeyMaxInflight int
  Cooldown429       time.Duration
}

type keyState struct {
  inUse         int
  cooldownUntil time.Time
}

type KeyPool struct {
  mu    sync.Mutex
  keys  []string
  idx   int
  cfg   KeyPoolConfig
  state map[string]*keyState
}

func NewKeyPool(keys []string, cfg KeyPoolConfig) *KeyPool {
  if cfg.PerKeyMaxInflight <= 0 {
    cfg.PerKeyMaxInflight = 1
  }
  st := make(map[string]*keyState, len(keys))
  for _, k := range keys {
    st[k] = &keyState{}
  }
  return &KeyPool{keys: append([]string{}, keys...), cfg: cfg, state: st}
}

// Acquire returns (key, releaseFn, error).
// releaseFn(success, errType): 用于更新 cooldown / inUse。
func (p *KeyPool) Acquire() (string, func(success bool, err error), error) {
  p.mu.Lock()
  defer p.mu.Unlock()

  if len(p.keys) == 0 {
    return "", nil, ErrNoKeyAvailable
  }

  now := time.Now()
  // round-robin 扫一圈
  for i := 0; i < len(p.keys); i++ {
    k := p.keys[p.idx%len(p.keys)]
    p.idx++
    st := p.state[k]
    if st == nil { st = &keyState{}; p.state[k] = st }
    if st.inUse >= p.cfg.PerKeyMaxInflight {
      continue
    }
    if !st.cooldownUntil.IsZero() && st.cooldownUntil.After(now) {
      continue
    }
    st.inUse++
    released := false
    return k, func(success bool, err error) {
      p.mu.Lock()
      defer p.mu.Unlock()
      if released { return }
      released = true
      st.inUse--
      if st.inUse < 0 { st.inUse = 0 }
      if errors.Is(err, ErrRateLimited) && p.cfg.Cooldown429 > 0 {
        st.cooldownUntil = time.Now().Add(p.cfg.Cooldown429)
      }
    }, nil
  }

  return "", nil, ErrNoKeyAvailable
}

type KeyInfo struct {
  MaskedKey     string     `json:"masked_key"`
  InUse         int        `json:"in_use"`
  CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
}

type KeyPoolStats struct {
  Total     int `json:"total"`
  Available int `json:"available"`
}

func (p *KeyPool) Snapshot() []KeyInfo {
  p.mu.Lock()
  defer p.mu.Unlock()

  now := time.Now()
  out := make([]KeyInfo, 0, len(p.keys))
  for _, k := range p.keys {
    st := p.state[k]
    if st == nil {
      st = &keyState{}
      p.state[k] = st
    }

    var cd *time.Time
    if !st.cooldownUntil.IsZero() && st.cooldownUntil.After(now) {
      t := st.cooldownUntil
      cd = &t
    }

    out = append(out, KeyInfo{
      MaskedKey: maskKey(k),
      InUse:     st.inUse,
      CooldownUntil: cd,
    })
  }
  return out
}

func (p *KeyPool) Stats() KeyPoolStats {
  p.mu.Lock()
  defer p.mu.Unlock()

  now := time.Now()
  total := len(p.keys)
  available := 0
  for _, k := range p.keys {
    st := p.state[k]
    if st == nil {
      st = &keyState{}
      p.state[k] = st
    }
    if st.inUse >= p.cfg.PerKeyMaxInflight {
      continue
    }
    if !st.cooldownUntil.IsZero() && st.cooldownUntil.After(now) {
      continue
    }
    available++
  }
  return KeyPoolStats{Total: total, Available: available}
}

func maskKey(key string) string {
  if len(key) <= 10 {
    return "****"
  }
  return key[:6] + "..." + key[len(key)-4:]
}
```

- [ ] **Step 3: 运行测试**

Run:
```powershell
cd gpt-image
go test .\internal\scheduler -v
```
Expected: PASS

- [ ] **Step 4: 提交**
```powershell
cd gpt-image
git add .
git commit -m "feat: add key pool scheduler with lease and 429 cooldown"
```

---

## Task 4: TaskManager（队列 + worker pool + 任务内存存储 + TTL）

**Files:**
- Create: `gpt-image/internal/tasks/types.go`
- Create: `gpt-image/internal/tasks/store_memory.go`
- Create: `gpt-image/internal/tasks/manager.go`
- Test: `gpt-image/internal/tasks/manager_test.go`

- [ ] **Step 1: 定义任务类型**

`gpt-image/internal/tasks/types.go`：
```go
package tasks

import "time"

type Status string

const (
  StatusQueued    Status = "queued"
  StatusRunning   Status = "running"
  StatusSucceeded Status = "succeeded"
  StatusFailed    Status = "failed"
  StatusCanceled  Status = "canceled"
)

type TaskType string

const (
  TaskImageGeneration TaskType = "image_generation"
  TaskImageEdit       TaskType = "image_edit"
)

type Task struct {
  ID        string   `json:"task_id"`
  Type      TaskType `json:"type"`
  Status    Status   `json:"status"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`

  Error  string      `json:"error,omitempty"`
  Result interface{} `json:"result,omitempty"`
}
```

- [ ] **Step 2: 写内存 Store（含 TTL cleanup）**

`gpt-image/internal/tasks/store_memory.go`：
```go
package tasks

import (
  "errors"
  "sync"
  "time"
)

var ErrNotFound = errors.New("task not found")

type MemoryStore struct {
  mu   sync.RWMutex
  ttl  time.Duration
  data map[string]*Task
}

func NewMemoryStore(ttl time.Duration) *MemoryStore {
  s := &MemoryStore{ttl: ttl, data: make(map[string]*Task)}
  if ttl > 0 {
    go s.cleanupLoop()
  }
  return s
}

func (s *MemoryStore) Put(t *Task) {
  s.mu.Lock()
  defer s.mu.Unlock()
  cp := *t
  s.data[t.ID] = &cp
}

func (s *MemoryStore) Get(id string) (*Task, error) {
  s.mu.RLock()
  defer s.mu.RUnlock()
  t := s.data[id]
  if t == nil {
    return nil, ErrNotFound
  }
  cp := *t
  return &cp, nil
}

func (s *MemoryStore) TTL() time.Duration { return s.ttl }

func (s *MemoryStore) ListRecent(limit int) []*Task {
  s.mu.RLock()
  defer s.mu.RUnlock()
  out := make([]*Task, 0, len(s.data))
  for _, t := range s.data {
    cp := *t
    out = append(out, &cp)
  }
  sort.Slice(out, func(i, j int) bool {
    return out[i].UpdatedAt.After(out[j].UpdatedAt)
  })
  if limit > 0 && len(out) > limit {
    out = out[:limit]
  }
  return out
}

func (s *MemoryStore) cleanupLoop() {
  ticker := time.NewTicker(1 * time.Minute)
  defer ticker.Stop()
  for range ticker.C {
    s.cleanupOnce()
  }
}

func (s *MemoryStore) cleanupOnce() {
  if s.ttl <= 0 {
    return
  }
  cutoff := time.Now().Add(-s.ttl)
  s.mu.Lock()
  defer s.mu.Unlock()
  for id, t := range s.data {
    if t.UpdatedAt.Before(cutoff) {
      delete(s.data, id)
    }
  }
}
```

- [ ] **Step 3: 先写 failing test（并发上限）**

`gpt-image/internal/tasks/manager_test.go`：
```go
package tasks

import (
  "context"
  "sync/atomic"
  "testing"
  "time"
)

func TestManager_MaxInflight(t *testing.T) {
  store := NewMemoryStore(10 * time.Minute)
  m := NewManager(store, ManagerConfig{Workers: 2, QueueSize: 10})
  defer m.Stop()

  var running int32
  var peak int32

  job := func(ctx context.Context) (interface{}, error) {
    cur := atomic.AddInt32(&running, 1)
    for {
      p := atomic.LoadInt32(&peak)
      if cur > p && atomic.CompareAndSwapInt32(&peak, p, cur) {
        break
      }
      if cur <= p { break }
    }
    time.Sleep(200 * time.Millisecond)
    atomic.AddInt32(&running, -1)
    return map[string]any{"ok": true}, nil
  }

  // 提交 5 个异步任务
  for i := 0; i < 5; i++ {
    _, err := m.SubmitAsync(context.Background(), TaskImageGeneration, job)
    if err != nil {
      t.Fatalf("submit: %v", err)
    }
  }
  time.Sleep(1200 * time.Millisecond)
  if peak > 2 {
    t.Fatalf("expected peak<=2, got %d", peak)
  }
}
```

- [ ] **Step 4: 实现 Manager（队列 + worker + sync/async 提交）**

`gpt-image/internal/tasks/manager.go`：
```go
package tasks

import (
  "context"
  "errors"
  "sync"
  "sync/atomic"
  "time"

  "github.com/google/uuid"
)

var ErrQueueFull = errors.New("task queue full")

type ManagerConfig struct {
  Workers   int
  QueueSize int
}

type job struct {
  taskID string
  fn     func(context.Context) (interface{}, error)
  done   chan struct{}
}

type Manager struct {
  store *MemoryStore
  cfg   ManagerConfig

  jobs chan *job
  stop chan struct{}
  wg   sync.WaitGroup
  running int32
}

func NewManager(store *MemoryStore, cfg ManagerConfig) *Manager {
  if cfg.Workers <= 0 { cfg.Workers = 1 }
  if cfg.QueueSize <= 0 { cfg.QueueSize = 1 }
  m := &Manager{
    store: store,
    cfg: cfg,
    jobs: make(chan *job, cfg.QueueSize),
    stop: make(chan struct{}),
  }
  for i := 0; i < cfg.Workers; i++ {
    m.wg.Add(1)
    go m.worker()
  }
  return m
}

func (m *Manager) Stop() {
  close(m.stop)
  m.wg.Wait()
}

func (m *Manager) Workers() int { return m.cfg.Workers }
func (m *Manager) QueueLen() int { return len(m.jobs) }
func (m *Manager) QueueCap() int { return cap(m.jobs) }
func (m *Manager) Running() int { return int(atomic.LoadInt32(&m.running)) }

func (m *Manager) SubmitAsync(ctx context.Context, typ TaskType, fn func(context.Context) (interface{}, error)) (string, error) {
  id := uuid.NewString()
  now := time.Now()
  m.store.Put(&Task{ID: id, Type: typ, Status: StatusQueued, CreatedAt: now, UpdatedAt: now})

  j := &job{taskID: id, fn: fn, done: nil}
  select {
  case m.jobs <- j:
    return id, nil
  default:
    // queue full
    t, _ := m.store.Get(id)
    t.Status = StatusFailed
    t.Error = ErrQueueFull.Error()
    t.UpdatedAt = time.Now()
    m.store.Put(t)
    return "", ErrQueueFull
  }
}

func (m *Manager) SubmitSync(ctx context.Context, typ TaskType, fn func(context.Context) (interface{}, error)) (string, interface{}, error) {
  id := uuid.NewString()
  now := time.Now()
  m.store.Put(&Task{ID: id, Type: typ, Status: StatusQueued, CreatedAt: now, UpdatedAt: now})
  done := make(chan struct{})
  j := &job{taskID: id, fn: fn, done: done}

  select {
  case m.jobs <- j:
  default:
    t, _ := m.store.Get(id)
    t.Status = StatusFailed
    t.Error = ErrQueueFull.Error()
    t.UpdatedAt = time.Now()
    m.store.Put(t)
    return "", nil, ErrQueueFull
  }

  select {
  case <-done:
    t, err := m.store.Get(id)
    if err != nil { return id, nil, err }
    if t.Status == StatusSucceeded { return id, t.Result, nil }
    return id, t.Result, errors.New(t.Error)
  case <-ctx.Done():
    // 不强杀 worker；只是标记 canceled（worker 仍可能完成）
    t, _ := m.store.Get(id)
    t.Status = StatusCanceled
    t.Error = ctx.Err().Error()
    t.UpdatedAt = time.Now()
    m.store.Put(t)
    return id, nil, ctx.Err()
  }
}

func (m *Manager) worker() {
  defer m.wg.Done()
  for {
    select {
    case <-m.stop:
      return
    case j := <-m.jobs:
      if j == nil { continue }
      t, err := m.store.Get(j.taskID)
      if err != nil { continue }
      t.Status = StatusRunning
      t.UpdatedAt = time.Now()
      m.store.Put(t)

      atomic.AddInt32(&m.running, 1)
      res, runErr := j.fn(context.Background())
      atomic.AddInt32(&m.running, -1)
      t, _ = m.store.Get(j.taskID)
      if runErr != nil {
        t.Status = StatusFailed
        t.Error = runErr.Error()
      } else {
        t.Status = StatusSucceeded
        t.Result = res
      }
      t.UpdatedAt = time.Now()
      m.store.Put(t)

      if j.done != nil {
        close(j.done)
      }
    }
  }
}
```

- [ ] **Step 5: 运行测试**

Run:
```powershell
cd gpt-image
go test .\internal\tasks -v
```
Expected: PASS

- [ ] **Step 6: 提交**
```powershell
cd gpt-image
git add .
git commit -m "feat: add in-memory task store and worker-pool task manager"
```

---

## Task 5: HTTP API（/v1/models + images/generations + images/edits + tasks + admin）

**Files:**
- Create: `gpt-image/internal/httpapi/router.go`
- Create: `gpt-image/internal/httpapi/middleware/auth_oss.go`
- Create: `gpt-image/internal/httpapi/handlers/handlers.go`
- Create: `gpt-image/internal/httpapi/handlers/models.go`
- Create: `gpt-image/internal/httpapi/handlers/images_generations.go`
- Create: `gpt-image/internal/httpapi/handlers/images_edits.go`
- Create: `gpt-image/internal/httpapi/handlers/tasks.go`
- Create: `gpt-image/internal/httpapi/handlers/admin.go`
- Modify: `gpt-image/internal/app/app.go`
- Test: `gpt-image/internal/httpapi/handlers/images_generations_test.go`
- Test: `gpt-image/internal/httpapi/handlers/images_edits_test.go`

### 5.1 Middleware：OSS Auth

- [ ] **Step 1: 写 OSS Bearer Middleware**

`gpt-image/internal/httpapi/middleware/auth_oss.go`：
```go
package middleware

import (
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
)

func OSSAuth(authKey string) gin.HandlerFunc {
  return func(c *gin.Context) {
    if strings.TrimSpace(authKey) == "" {
      // 未配置 authKey：为避免误开放，直接拒绝
      c.JSON(http.StatusUnauthorized, gin.H{"error": "AUTH_KEY not set"})
      c.Abort()
      return
    }

    h := c.GetHeader("Authorization")
    scheme, _, token := strings.Cut(h, " ")
    if strings.ToLower(strings.TrimSpace(scheme)) != "bearer" || strings.TrimSpace(token) == "" {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization"})
      c.Abort()
      return
    }
    if strings.TrimSpace(token) != strings.TrimSpace(authKey) {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization"})
      c.Abort()
      return
    }
    c.Next()
  }
}
```

### 5.2 Router/Wiring

- [ ] **Step 2: 定义依赖容器与路由注册**

`gpt-image/internal/httpapi/router.go`：
```go
package httpapi

import (
  "github.com/gin-gonic/gin"

  "gpt-image/internal/httpapi/handlers"
  "gpt-image/internal/httpapi/middleware"
)

type Deps struct {
  AuthKey string
  H handlers.Handlers
}

func RegisterOSS(r *gin.Engine, d Deps) {
  v1 := r.Group("/v1")
  v1.Use(middleware.OSSAuth(d.AuthKey))
  {
    v1.GET("/models", d.H.ListModels)
    v1.POST("/images/generations", d.H.ImagesGenerations)
    v1.POST("/images/edits", d.H.ImagesEdits)
    v1.GET("/images/tasks/:id", d.H.GetTask)

    // admin（先跟 v1 同 auth）
    v1.GET("/admin/stats", d.H.AdminStats)
    v1.GET("/admin/keys", d.H.AdminKeys)
    v1.GET("/admin/tasks", d.H.AdminTasks)
  }
}
```

- [ ] **Step 3: 写 Handlers 结构体**

`gpt-image/internal/httpapi/handlers/handlers.go`：
```go
package handlers

import (
  "gpt-image/internal/providers"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

type Handlers struct {
  Provider providers.ImagesProvider
  Pool     *scheduler.KeyPool
  Tasks    *tasks.Manager
  Store    *tasks.MemoryStore
}
```

> 如果目录不存在，记得先创建 `gpt-image/internal/httpapi/handlers/`。

- [ ] **Step 4: 修改 app.NewServer 装配依赖并注册路由**

`gpt-image/internal/app/app.go`（替换为完整 wiring）：
```go
package app

import (
  "net/http"
  "time"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/config"
  "gpt-image/internal/httpapi"
  "gpt-image/internal/httpapi/handlers"
  "gpt-image/internal/providers/openai"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

type Mode string

const (
  ModeOSS Mode = "oss"
  ModePro Mode = "pro"
)

func NewServer(cfg config.Config, mode Mode) *gin.Engine {
  r := gin.New()
  r.Use(gin.Recovery())

  r.GET("/ping", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "pong", "mode": mode})
  })

  // OSS MVP：先统一用 OpenAI Official Provider + 内存任务
  store := tasks.NewMemoryStore(time.Duration(cfg.TaskTTLMin) * time.Minute)
  tm := tasks.NewManager(store, tasks.ManagerConfig{
    Workers: cfg.MaxInflight,
    QueueSize: cfg.MaxQueue,
  })
  pool := scheduler.NewKeyPool(cfg.OpenAIAPIKeys, scheduler.KeyPoolConfig{
    PerKeyMaxInflight: cfg.PerKeyMaxInflight,
    Cooldown429: time.Duration(cfg.Cooldown429Sec) * time.Second,
  })
  prov := openai.New(cfg.OpenAIBaseURL, nil)

  h := handlers.Handlers{
    Provider: prov,
    Pool: pool,
    Tasks: tm,
    Store: store,
  }

  httpapi.RegisterOSS(r, httpapi.Deps{
    AuthKey: cfg.AuthKey,
    H: h,
  })

  return r
}
```

### 5.3 Models / Admin / Tasks / Images Handlers

- [ ] **Step 5: ListModels handler**

`gpt-image/internal/httpapi/handlers/models.go`：
```go
package handlers

import (
  "net/http"

  "github.com/gin-gonic/gin"
)

func (h Handlers) ListModels(c *gin.Context) {
  models := h.Provider.ListModels(c.Request.Context())
  items := make([]gin.H, 0, len(models))
  for _, m := range models {
    items = append(items, gin.H{
      "id": m,
      "object": "model",
      "created": 0,
      "owned_by": "gpt-image",
    })
  }
  c.JSON(http.StatusOK, gin.H{"object": "list", "data": items})
}
```

- [ ] **Step 6: Task 查询 handler（/v1/images/tasks/:id）**

`gpt-image/internal/httpapi/handlers/tasks.go`：
```go
package handlers

import (
  "net/http"

  "github.com/gin-gonic/gin"
)

func (h Handlers) GetTask(c *gin.Context) {
  id := c.Param("id")
  t, err := h.Store.Get(id)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
    return
  }
  c.JSON(http.StatusOK, t)
}
```

- [ ] **Step 7: Admin handlers（stats/keys/tasks）**

`gpt-image/internal/httpapi/handlers/admin.go`：
```go
package handlers

import (
  "net/http"
  "time"

  "github.com/gin-gonic/gin"
)

func (h Handlers) AdminStats(c *gin.Context) {
  // OSS：给前端用的“运行态”指标（不泄露真实 key）
  ks := h.Pool.Stats()
  c.JSON(http.StatusOK, gin.H{
    "upstream": h.Provider.Name(),
    "keys_total": ks.Total,
    "keys_available": ks.Available,
    "max_inflight": h.Tasks.Workers(),
    "tasks_running": h.Tasks.Running(),
    "queue_len": h.Tasks.QueueLen(),
    "queue_cap": h.Tasks.QueueCap(),
    "task_ttl_min": int(h.Store.TTL().Minutes()),
    "now": time.Now().Format(time.RFC3339),
  })
}

func (h Handlers) AdminKeys(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{"items": h.Pool.Snapshot()})
}

func (h Handlers) AdminTasks(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{"items": h.Store.ListRecent(200)})
}
```

- [ ] **Step 8: images/generations handler（sync + async）**

`gpt-image/internal/httpapi/handlers/images_generations.go`：
```go
package handlers

import (
  "context"
  "net/http"
  "strings"
  "time"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/providers"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

type generationsIn struct {
  Prompt string `json:"prompt"`
  Model  string `json:"model"`
  N      int    `json:"n"`
  Size   string `json:"size"`
  Quality string `json:"quality"`
  Background string `json:"background"`
  OutputFormat string `json:"output_format"`
  ResponseFormat string `json:"response_format"`
  Async bool `json:"async"`
}

func (h Handlers) ImagesGenerations(c *gin.Context) {
  var in generationsIn
  if err := c.ShouldBindJSON(&in); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  in.Prompt = strings.TrimSpace(in.Prompt)
  if in.Prompt == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
    return
  }
  if in.N <= 0 { in.N = 1 }
  if in.N > 4 { in.N = 4 }
  if in.ResponseFormat == "" { in.ResponseFormat = "b64_json" }

  req := providers.GenerateRequest{
    Prompt: in.Prompt,
    Model: in.Model,
    N: in.N,
    Size: in.Size,
    Quality: in.Quality,
    Background: in.Background,
    OutputFormat: in.OutputFormat,
    ResponseFormat: in.ResponseFormat,
  }

  run := func(ctx context.Context) (interface{}, error) {
    key, release, err := h.Pool.Acquire()
    if err != nil {
      return nil, err
    }
    var relErr error
    defer func() { release(relErr == nil, relErr) }()

    out, err := h.Provider.Generate(ctx, key, req)
    if perr, ok := err.(*providers.ProviderError); ok && perr.StatusCode == 429 {
      relErr = scheduler.ErrRateLimited
      return nil, err
    }
    relErr = err
    return out, err
  }

  if in.Async {
    id, err := h.Tasks.SubmitAsync(c.Request.Context(), tasks.TaskImageGeneration, run)
    if err != nil {
      c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
      return
    }
    c.JSON(http.StatusAccepted, gin.H{"task_id": id, "status": "queued"})
    return
  }

  // 同步：允许排队等待（给一个上限）
  ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
  defer cancel()
  _, res, err := h.Tasks.SubmitSync(ctx, tasks.TaskImageGeneration, run)
  if err != nil {
    // scheduler no key
    if err == scheduler.ErrNoKeyAvailable {
      c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no upstream keys available"})
      return
    }
    c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, res)
}
```

- [ ] **Step 9: images/edits handler（multipart）**

`gpt-image/internal/httpapi/handlers/images_edits.go`：
```go
package handlers

import (
  "context"
  "io"
  "mime/multipart"
  "net/http"
  "strings"
  "time"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/providers"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

func (h Handlers) ImagesEdits(c *gin.Context) {
  // 兼容：image / image[]；mask / mask[]
  form, err := c.MultipartForm()
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "multipart/form-data required"})
    return
  }

  prompt := strings.TrimSpace(c.PostForm("prompt"))
  if prompt == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
    return
  }

  imagesFH := append(form.File["image"], form.File["image[]"]...)
  if len(imagesFH) == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "image is required"})
    return
  }
  masksFH := append(form.File["mask"], form.File["mask[]"]...)

  readAll := func(fh *multipart.FileHeader) ([]byte, error) {
    f, err := fh.Open()
    if err != nil { return nil, err }
    defer f.Close()
    return io.ReadAll(io.LimitReader(f, 20<<20)) // 20MB
  }

  var images [][]byte
  for _, fh := range imagesFH {
    b, err := readAll(fh)
    if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
    images = append(images, b)
  }
  var masks [][]byte
  for _, fh := range masksFH {
    b, err := readAll(fh)
    if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
    masks = append(masks, b)
  }

  inAsync := strings.ToLower(strings.TrimSpace(c.PostForm("async"))) == "true"
  req := providers.EditRequest{
    Prompt: prompt,
    Model: c.PostForm("model"),
    Size: c.PostForm("size"),
    Quality: c.PostForm("quality"),
    Background: c.PostForm("background"),
    OutputFormat: c.PostForm("output_format"),
    ResponseFormat: c.PostForm("response_format"),
  }
  if req.ResponseFormat == "" { req.ResponseFormat = "b64_json" }

  run := func(ctx context.Context) (interface{}, error) {
    key, release, err := h.Pool.Acquire()
    if err != nil { return nil, err }
    var relErr error
    defer func(){ release(relErr == nil, relErr) }()

    out, err := h.Provider.Edit(ctx, key, req, images, masks)
    if perr, ok := err.(*providers.ProviderError); ok && perr.StatusCode == 429 {
      relErr = scheduler.ErrRateLimited
      return nil, err
    }
    relErr = err
    return out, err
  }

  if inAsync {
    id, err := h.Tasks.SubmitAsync(c.Request.Context(), tasks.TaskImageEdit, run)
    if err != nil { c.JSON(503, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusAccepted, gin.H{"task_id": id, "status": "queued"})
    return
  }

  ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
  defer cancel()
  _, res, err := h.Tasks.SubmitSync(ctx, tasks.TaskImageEdit, run)
  if err != nil {
    if err == scheduler.ErrNoKeyAvailable {
      c.JSON(503, gin.H{"error": "no upstream keys available"})
      return
    }
    c.JSON(502, gin.H{"error": err.Error()})
    return
  }
  c.JSON(200, res)
}
```

### 5.4 Handler 单测（关键路径）

- [ ] **Step 10: generations handler test（fake provider + async 路径）**

`gpt-image/internal/httpapi/handlers/images_generations_test.go`：
```go
package handlers

import (
  "bytes"
  "context"
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"
  "time"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/providers"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

type fakeProvider struct{}
func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) ListModels(ctx context.Context) []string { return []string{"gpt-image-1"} }
func (fakeProvider) Generate(ctx context.Context, upstreamKey string, req providers.GenerateRequest) (*providers.ImagesResponse, error) {
  return &providers.ImagesResponse{Created: 1, Data: []providers.ImageData{{B64JSON: "AAAA"}}}, nil
}
func (fakeProvider) Edit(ctx context.Context, upstreamKey string, req providers.EditRequest, images [][]byte, masks [][]byte) (*providers.ImagesResponse, error) {
  return &providers.ImagesResponse{Created: 2, Data: []providers.ImageData{{B64JSON: "BBBB"}}}, nil
}

func TestImagesGenerations_AsyncReturnsTask(t *testing.T) {
  gin.SetMode(gin.TestMode)
  store := tasks.NewMemoryStore(10 * time.Minute)
  tm := tasks.NewManager(store, tasks.ManagerConfig{Workers: 1, QueueSize: 10})
  defer tm.Stop()
  pool := scheduler.NewKeyPool([]string{"k1"}, scheduler.KeyPoolConfig{PerKeyMaxInflight: 1})

  h := Handlers{Provider: fakeProvider{}, Pool: pool, Tasks: tm, Store: store}
  r := gin.New()
  r.POST("/v1/images/generations", h.ImagesGenerations)

  body, _ := json.Marshal(map[string]any{"prompt": "hi", "async": true})
  req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)

  if w.Code != http.StatusAccepted {
    t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
  }
}
```

- [ ] **Step 11: edits handler test（构造 multipart）**

`gpt-image/internal/httpapi/handlers/images_edits_test.go`：
```go
package handlers

import (
  "bytes"
  "context"
  "mime/multipart"
  "net/http"
  "net/http/httptest"
  "testing"
  "time"

  "github.com/gin-gonic/gin"

  "gpt-image/internal/providers"
  "gpt-image/internal/scheduler"
  "gpt-image/internal/tasks"
)

type fakeProvider2 struct{ fakeProvider }
func (fakeProvider2) Edit(ctx context.Context, upstreamKey string, req providers.EditRequest, images [][]byte, masks [][]byte) (*providers.ImagesResponse, error) {
  if len(images) != 1 {
    return nil, &providers.ProviderError{StatusCode: 400, Message: "need one image"}
  }
  return &providers.ImagesResponse{Created: 2, Data: []providers.ImageData{{B64JSON: "BBBB"}}}, nil
}

func TestImagesEdits_SyncOK(t *testing.T) {
  gin.SetMode(gin.TestMode)
  store := tasks.NewMemoryStore(10 * time.Minute)
  tm := tasks.NewManager(store, tasks.ManagerConfig{Workers: 1, QueueSize: 10})
  defer tm.Stop()
  pool := scheduler.NewKeyPool([]string{"k1"}, scheduler.KeyPoolConfig{PerKeyMaxInflight: 1})
  h := Handlers{Provider: fakeProvider2{}, Pool: pool, Tasks: tm, Store: store}
  r := gin.New()
  r.POST("/v1/images/edits", h.ImagesEdits)

  buf := &bytes.Buffer{}
  mw := multipart.NewWriter(buf)
  _ = mw.WriteField("prompt", "edit me")
  fw, _ := mw.CreateFormFile("image", "a.png")
  fw.Write([]byte("PNGDATA"))
  mw.Close()

  req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", buf)
  req.Header.Set("Content-Type", mw.FormDataContentType())
  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)

  if w.Code != 200 {
    t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
  }
}
```

- [ ] **Step 12: 跑全量后端测试**
```powershell
cd gpt-image
go test .\... -v
```
Expected: PASS

- [ ] **Step 13: 提交**
```powershell
cd gpt-image
git add .
git commit -m "feat: add OSS HTTP API (models/images/tasks/admin) with tests"
```

---

## Task 6: 前端 web 骨架迁移（登录写入 auth-key + Dashboard/Keys/Tasks）

**Files:**
- Create: `gpt-image/web/`（从现有 `frontend/` 复制）
- Create: `gpt-image/web/src/api/http.ts`
- Create: `gpt-image/web/src/views/Login.vue`
- Modify: `gpt-image/web/src/router/index.ts`
- Modify: `gpt-image/web/src/layout/MainLayout.vue`
- Modify: `gpt-image/web/src/views/Dashboard.vue`
- Rename/Modify: `gpt-image/web/src/views/AccountPool.vue` → `ProviderKeys.vue`
- Rename/Modify: `gpt-image/web/src/views/APIKeys.vue` → `Tasks.vue`

- [ ] **Step 1: 复制前端（不含 node_modules）**

Run（PowerShell）：
```powershell
cd "C:\Users\12562\创业网站项目？\代理\开源用的gpt"
New-Item -ItemType Directory -Path .\gpt-image\web -Force | Out-Null

# 复制 frontend 的必要文件（排除 node_modules）
Copy-Item -Recurse -Force .\frontend\public .\gpt-image\web\
Copy-Item -Recurse -Force .\frontend\src .\gpt-image\web\
Copy-Item -Force .\frontend\package.json .\gpt-image\web\
Copy-Item -Force .\frontend\package-lock.json .\gpt-image\web\
Copy-Item -Force .\frontend\vite.config.ts .\gpt-image\web\
Copy-Item -Force .\frontend\tsconfig*.json .\gpt-image\web\
Copy-Item -Force .\frontend\index.html .\gpt-image\web\
Copy-Item -Force .\frontend\README.md .\gpt-image\web\
Copy-Item -Force .\frontend\Dockerfile .\gpt-image\web\Dockerfile
Copy-Item -Force .\frontend\.gitignore .\gpt-image\web\.gitignore
```

- [ ] **Step 2: 创建 Axios 实例（注入 Authorization）**

`gpt-image/web/src/api/http.ts`：
```ts
import axios from 'axios'

const apiBase = import.meta.env.VITE_API_BASE_URL || ''

export const http = axios.create({
  baseURL: apiBase,
  timeout: 60_000,
})

http.interceptors.request.use((config) => {
  const key = localStorage.getItem('AUTH_KEY') || ''
  if (key) {
    config.headers = config.headers || {}
    config.headers['Authorization'] = `Bearer ${key}`
  }
  return config
})
```

- [ ] **Step 3: 添加登录页（只输入 AUTH_KEY）**

`gpt-image/web/src/views/Login.vue`：
```vue
<template>
  <div class="login">
    <el-card class="card" shadow="never">
      <div class="title">gpt-image OSS Login</div>
      <el-input v-model="key" placeholder="Enter AUTH_KEY" show-password />
      <div class="actions">
        <el-button type="primary" @click="save">Save</el-button>
        <el-button @click="clear">Clear</el-button>
      </div>
      <div class="hint">
        AUTH_KEY 仅保存在本地浏览器 LocalStorage，用于调用后端 Bearer 鉴权。
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const key = ref(localStorage.getItem('AUTH_KEY') || '')

const save = () => {
  localStorage.setItem('AUTH_KEY', key.value.trim())
  router.replace('/')
}
const clear = () => {
  localStorage.removeItem('AUTH_KEY')
  key.value = ''
}
</script>

<style scoped>
.login {
  min-height: 100vh;
  background: radial-gradient(circle at top left, #1a1a1a 0%, #0a0a0a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}
.card {
  width: 520px;
  background: rgba(0,0,0,.35);
  border: 1px solid rgba(255,255,255,.06);
}
.title {
  font-weight: 900;
  letter-spacing: 2px;
  margin-bottom: 16px;
}
.actions { margin-top: 16px; display:flex; gap:12px; }
.hint { margin-top: 12px; font-size: 12px; color: #888; line-height: 1.6; }
</style>
```

- [ ] **Step 4: 路由增加 /login + 守卫**

`gpt-image/web/src/router/index.ts`（整体替换为）：
```ts
import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '../layout/MainLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
    {
      path: '/',
      component: MainLayout,
      children: [
        { path: '', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
        { path: 'keys', name: 'ProviderKeys', component: () => import('../views/ProviderKeys.vue') },
        { path: 'tasks', name: 'Tasks', component: () => import('../views/Tasks.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.path === '/login') return true
  const key = localStorage.getItem('AUTH_KEY') || ''
  if (!key.trim()) return '/login'
  return true
})

export default router
```

- [ ] **Step 5: 更新 Layout 菜单与文案（/keys, /tasks）**

`gpt-image/web/src/layout/MainLayout.vue`：只改动 template 中的品牌与菜单（样式可不动）。把三条菜单改成 `/`、`/keys`、`/tasks`：
```vue
<template>
  <el-container class="layout-container">
    <el-aside width="280px" class="side-nav">
      <div class="brand">
        <div class="brand-logo">G</div>
        <div class="brand-text">
          <div class="name">GPT-IMAGE</div>
          <div class="version">OSS ADMIN</div>
        </div>
      </div>

      <nav class="nav-links">
        <router-link to="/" class="nav-item" active-class="active">
          <el-icon><Odometer /></el-icon>
          <span>DASHBOARD</span>
        </router-link>
        <router-link to="/keys" class="nav-item" active-class="active">
          <el-icon><Cpu /></el-icon>
          <span>PROVIDER KEYS</span>
        </router-link>
        <router-link to="/tasks" class="nav-item" active-class="active">
          <el-icon><Lock /></el-icon>
          <span>TASKS</span>
        </router-link>
      </nav>

      <div class="system-status">
        <div class="status-header">LOCAL KERNEL STATUS</div>
        <div class="status-bar">
          <div class="pulse-dot"></div>
          <span>SYNCHRONIZED</span>
        </div>
      </div>
    </el-aside>

    <el-container>
      <el-header class="top-header">
        <div class="header-left">
          <div class="breadcrumb">{{ $route.name }}</div>
        </div>
        <div class="header-right">
          <el-button link type="danger" @click="logout">Logout</el-button>
        </div>
      </el-header>

      <el-main class="viewport">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Cpu, Lock, Odometer } from '@element-plus/icons-vue'

const router = useRouter()
const logout = () => {
  localStorage.removeItem('AUTH_KEY')
  router.replace('/login')
}
</script>
```

- [ ] **Step 6: Dashboard 对接 /v1/admin/stats**

`gpt-image/web/src/views/Dashboard.vue`（整体替换为，保留你现有的视觉风格，但字段对接新 stats）：
```vue
<template>
  <div class="dashboard">
    <div class="hero-section">
      <h1 class="glitch" data-text="GPT-IMAGE">GPT-IMAGE</h1>
      <p class="subtitle">OSS Admin | Official OpenAI Upstream | Single Instance</p>
    </div>

    <el-row :gutter="24" class="stat-container">
      <el-col :span="6">
        <div class="stat-card total">
          <div class="label">Total Keys</div>
          <div class="value">{{ stats.keys_total }}</div>
          <div class="decor-bar"></div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card active">
          <div class="label">Available Keys</div>
          <div class="value">{{ stats.keys_available }}</div>
          <div class="decor-bar"></div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card img2">
          <div class="label">Running Tasks</div>
          <div class="value">{{ stats.tasks_running }}</div>
          <div class="decor-bar"></div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card banned">
          <div class="label">Queue Len</div>
          <div class="value">{{ stats.queue_len }} / {{ stats.queue_cap }}</div>
          <div class="decor-bar"></div>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="24" class="mt-40">
      <el-col :span="14">
        <div class="panel-card chart-panel">
          <div class="panel-header">Real-time Performance (mock)</div>
          <div class="mock-chart">
            <div
              v-for="(h, i) in chartHeights"
              :key="i"
              class="bar"
              :style="{ height: h + '%', animationDelay: i * 0.05 + 's' }"
            ></div>
          </div>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="panel-card activity-panel">
          <div class="panel-header">Kernel Events</div>
          <div class="event-list">
            <div v-for="(event, i) in events" :key="i" class="event-item">
              <span class="timestamp">{{ event.time }}</span>
              <span class="type" :class="event.type.toLowerCase()">[{{ event.type }}]</span>
              <span class="msg">{{ event.msg }}</span>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { http } from '../api/http'

type Stats = {
  upstream: string
  keys_total: number
  keys_available: number
  max_inflight: number
  tasks_running: number
  queue_len: number
  queue_cap: number
  task_ttl_min: number
  now: string
}

const stats = ref<Stats>({
  upstream: '',
  keys_total: 0,
  keys_available: 0,
  max_inflight: 0,
  tasks_running: 0,
  queue_len: 0,
  queue_cap: 0,
  task_ttl_min: 0,
  now: '',
})

const chartHeights = ref(Array.from({ length: 30 }, () => Math.random() * 80 + 20))
const events = ref([
  { time: '00:00:00', type: 'INFO', msg: 'Waiting for kernel events...' },
])

onMounted(async () => {
  try {
    const res = await http.get('/v1/admin/stats')
    stats.value = res.data
    events.value = [
      { time: new Date().toLocaleTimeString(), type: 'INFO', msg: `Upstream: ${stats.value.upstream}` },
      { time: new Date().toLocaleTimeString(), type: 'INFO', msg: `Max inflight: ${stats.value.max_inflight}` },
      { time: new Date().toLocaleTimeString(), type: 'INFO', msg: `Task TTL: ${stats.value.task_ttl_min} min` },
    ]
  } catch (e) {
    console.error('Failed to fetch stats', e)
    events.value = [{ time: new Date().toLocaleTimeString(), type: 'ERROR', msg: 'Failed to fetch /v1/admin/stats' }]
  }
})
</script>

<style scoped>
.dashboard {
  padding: 40px;
  background: radial-gradient(circle at top left, #1a1a1a 0%, #0a0a0a 100%);
  min-height: calc(100vh - 64px);
}

.hero-section {
  margin-bottom: 50px;
}

.glitch {
  font-size: 3rem;
  font-weight: 900;
  color: #fff;
  letter-spacing: 5px;
  position: relative;
  text-transform: uppercase;
}

.subtitle {
  color: #00ffa3;
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9rem;
  opacity: 0.8;
}

.stat-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
  padding: 24px;
  position: relative;
  overflow: hidden;
  transition: all 0.3s ease;
}

.stat-card:hover {
  background: rgba(255, 255, 255, 0.05);
  transform: translateY(-5px);
}

.stat-card .label {
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
  margin-bottom: 12px;
}

.stat-card .value {
  font-size: 28px;
  font-weight: bold;
  font-family: 'JetBrains Mono', monospace;
}

.decor-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 4px;
  width: 100%;
  background: #222;
}

.total .decor-bar { background: #3b82f6; }
.active .decor-bar { background: #00ffa3; }
.img2 .decor-bar { background: #f59e0b; }
.banned .decor-bar { background: #ef4444; }

.panel-card {
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.05);
  height: 400px;
  display: flex;
  flex-direction: column;
}

.panel-header {
  padding: 16px 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 14px;
  font-weight: bold;
  color: #666;
  text-transform: uppercase;
}

.mock-chart {
  flex: 1;
  display: flex;
  align-items: flex-end;
  padding: 40px;
  gap: 8px;
}

.bar {
  flex: 1;
  background: linear-gradient(to top, #00ffa3, rgba(0, 255, 163, 0.1));
  animation: grow 1s ease-out forwards;
  opacity: 0.6;
}

@keyframes grow {
  from { height: 0; }
}

.event-list {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.event-item {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  padding: 10px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.timestamp { color: #555; margin-right: 12px; }
.type.info { color: #00ffa3; }
.type.error { color: #ef4444; }
.msg { color: #ccc; margin-left: 8px; }

.mt-40 { margin-top: 40px; }
</style>
```

- [ ] **Step 7: ProviderKeys 页（替换 AccountPool.vue）**

创建 `gpt-image/web/src/views/ProviderKeys.vue`（全量内容如下）：
```vue
<template>
  <div class="keys">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span class="title">Provider Keys</span>
          <el-button type="primary" @click="fetchKeys">Refresh</el-button>
        </div>
      </template>

      <el-table :data="items" style="width: 100%" class="custom-table" size="large">
        <el-table-column prop="masked_key" label="Key" width="260" />
        <el-table-column prop="in_use" label="In Use" width="120" />
        <el-table-column label="Cooldown Until">
          <template #default="{ row }">
            <span v-if="row.cooldown_until">{{ row.cooldown_until }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { http } from '../api/http'

type KeyItem = {
  masked_key: string
  in_use: number
  cooldown_until?: string
}

const items = ref<KeyItem[]>([])

const fetchKeys = async () => {
  const res = await http.get('/v1/admin/keys')
  items.value = (res.data?.items || []) as KeyItem[]
}

onMounted(() => {
  fetchKeys().catch((e) => console.error(e))
})
</script>

<style scoped>
.table-card {
  background: #0d0d0d;
  border: 1px solid rgba(255, 255, 255, 0.05);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.title {
  font-family: 'JetBrains Mono', monospace;
  font-weight: bold;
  color: #fff;
  letter-spacing: 2px;
}
.custom-table {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: #151515;
  --el-table-border-color: #222;
  --el-table-text-color: #ccc;
}
</style>
```

- [ ] **Step 8: Tasks 页（替换 APIKeys.vue）**

创建 `gpt-image/web/src/views/Tasks.vue`（全量内容如下）：
```vue
<template>
  <div class="tasks">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span class="title">Tasks (Recent)</span>
          <el-button type="primary" @click="fetchTasks">Refresh</el-button>
        </div>
      </template>

      <el-table :data="items" style="width: 100%" class="custom-table" size="large">
        <el-table-column prop="task_id" label="Task ID" width="320" />
        <el-table-column prop="type" label="Type" width="160" />
        <el-table-column prop="status" label="Status" width="140" />
        <el-table-column prop="updated_at" label="Updated" width="220" />
        <el-table-column prop="error" label="Error" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { http } from '../api/http'

type TaskItem = {
  task_id: string
  type: string
  status: string
  created_at: string
  updated_at: string
  error?: string
}

const items = ref<TaskItem[]>([])

const fetchTasks = async () => {
  const res = await http.get('/v1/admin/tasks')
  items.value = (res.data?.items || []) as TaskItem[]
}

onMounted(() => {
  fetchTasks().catch((e) => console.error(e))
})
</script>

<style scoped>
.table-card {
  background: #0d0d0d;
  border: 1px solid rgba(255, 255, 255, 0.05);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.title {
  font-family: 'JetBrains Mono', monospace;
  font-weight: bold;
  color: #fff;
  letter-spacing: 2px;
}
.custom-table {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: #151515;
  --el-table-border-color: #222;
  --el-table-text-color: #ccc;
}
</style>
```

- [ ] **Step 9: 本地启动验证**

Run:
```powershell
cd gpt-image\web
npm install
npm run dev -- --port=3010
```
Expected:
- 打开 `http://localhost:3010` 自动跳转 `/login`
- 输入 AUTH_KEY 后进入 Dashboard（能请求到后端 `/v1/admin/stats`）

- [ ] **Step 10: 提交**
```powershell
cd gpt-image
git add .\web
git commit -m "feat(web): add OSS admin UI skeleton (login/dashboard/keys/tasks)"
```

---

## Task 7: 端到端手工验收（curl 示例）

**Goal:** 用真实 OpenAI API Key 验证 `/v1/images/*` 在 OSS 网关能跑通（同步 + 异步）。

- [ ] **Step 1: 启动后端（配置 AUTH_KEY + OPENAI_API_KEYS）**

Run:
```powershell
$env:AUTH_KEY="dev-auth"
$env:OPENAI_API_KEYS="sk-your-openai-key-1,sk-your-openai-key-2"
cd gpt-image
go run .\cmd\gpt-image-oss
```

- [ ] **Step 2: 同步 generations（默认 b64_json）**

Run:
```powershell
curl -s http://localhost:8080/v1/images/generations ^
  -H "Authorization: Bearer dev-auth" ^
  -H "Content-Type: application/json" ^
  -d "{\"model\":\"gpt-image-1\",\"prompt\":\"a cute cat\",\"n\":1}"
```

Expected:
- JSON 包含 `data[0].b64_json`

- [ ] **Step 3: 异步 generations**

Run:
```powershell
$r = curl -s http://localhost:8080/v1/images/generations ^
  -H "Authorization: Bearer dev-auth" ^
  -H "Content-Type: application/json" ^
  -d "{\"model\":\"gpt-image-1\",\"prompt\":\"a cute cat\",\"n\":1,\"async\":true}"
$r
```
Expected:
- 返回包含 `task_id`

然后查询：
```powershell
$id = (ConvertFrom-Json $r).task_id
curl -s http://localhost:8080/v1/images/tasks/$id -H "Authorization: Bearer dev-auth"
```
Expected:
- status 从 queued/running 变为 succeeded，并带 result

- [ ] **Step 4:（可选）edits 手工验收**

Run（准备一张本地 png `a.png`）：
```powershell
curl -s http://localhost:8080/v1/images/edits ^
  -H "Authorization: Bearer dev-auth" ^
  -F "prompt=make it cyberpunk" ^
  -F "image=@a.png"
```

- [ ] **Step 5: 最终提交（如果需要）**
```powershell
cd gpt-image
git status
```

---

## Self-Review（对照 spec 覆盖）

- spec 的 OSS MVP 核心：`/v1/models`、`/v1/images/generations`、`/v1/images/edits`、`/v1/images/tasks/:id` ✅（Task 5）
- sync + async ✅（Task 5）
- 全局并发 20 ✅（Task 4 workers=MaxInflight）
- Key 池 lease + 429 cooldown ✅（Task 3）
- OSS 单租户 Bearer auth ✅（Task 5 middleware）
- 前端 UI 骨架（登录 + 三页）✅（Task 6）

## Follow-ups（另起 plan，不在本文件实现）

- Pro（多用户、邮箱验证、MySQL 任务持久化、用户 API Keys、审计/用量）——单独出 `gpt-image-pro` 计划文件
