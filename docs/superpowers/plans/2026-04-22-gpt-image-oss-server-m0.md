# gpt-image OSS Server (M0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在当前仓库中新建 `gpt-image/server/`（Go）并实现 OSS 单租户版本后端：`/v1/models`、`/v1/images/generations`、`/v1/images/edits`、`/v1/images/tasks/:id`，带全局 `AUTH_KEY` 鉴权、全局并发上限（默认 20）、账号池调度（单实例 lease）与最小管理接口（accounts/stats）。

**Architecture:** Gin 路由层（Edition=OSS）依赖 `core`（limiter、task store、OpenAI 兼容 response types）与 `oss`（SQLite 账号池、ChatGPT 协议上游适配器、全局 Auth middleware）。图片接口支持同步（默认）与 `async=true` 任务模式；OSS 任务存储为内存 TTL（重启丢失）。

**Tech Stack:** Go（标准 testing）、Gin、GORM + pure-go SQLite（glebarez/sqlite）、google/uuid、refraction-networking/utls（上游 TLS 指纹）。

---

## Scope / Plan 拆分

本计划只覆盖 **OSS 后端（M0 可跑通 + 最小管理接口）**。不包含：

- PRO（多用户、邮箱验证、MySQL、API 分发/专属池）——单独再写一份 plan。
- 前端迁移与 UI 完整可用化 ——单独 plan。

---

## 目标目录结构（将要创建/修改）

> 约定：所有命令在仓库根目录执行；Go 相关命令在 `gpt-image/server/` 下执行。

### Server（OSS）

- `gpt-image/server/go.mod`：OSS server 独立 Go module
- `gpt-image/server/cmd/gpt-image-oss/main.go`：OSS 启动入口（加载配置、初始化 DB、构建依赖、启动 Gin）

#### core（可复用，不含私有逻辑）

- `gpt-image/server/internal/core/openai/types.go`：OpenAI 兼容请求/响应结构（images/models/errors）
- `gpt-image/server/internal/core/limiter/limiter.go`：全局并发门闩（semaphore）
- `gpt-image/server/internal/core/tasks/{task.go,store.go,memory_store.go}`：任务模型 + 存储接口 + OSS 内存实现（TTL）

#### oss（单租户 + 账号池上游）

- `gpt-image/server/internal/oss/config/config.go`：读取 env（PORT、AUTH_KEY、DB_PATH、MAX_INFLIGHT、TASK_TTL…）
- `gpt-image/server/internal/oss/http/router.go`：Gin router 构建（注入 deps，便于测试）
- `gpt-image/server/internal/oss/http/middleware/auth.go`：全局 AUTH_KEY 鉴权
- `gpt-image/server/internal/oss/http/handlers/`：
  - `models.go`：`GET /v1/models`
  - `images.go`：`POST /v1/images/generations` + `POST /v1/images/edits`
  - `tasks.go`：`GET /v1/images/tasks/:id`
  - `admin.go`：`GET /v1/admin/stats`、`GET /v1/admin/accounts`、`POST /v1/admin/accounts/import`
- `gpt-image/server/internal/oss/storage/sqlite/db.go`：SQLite 初始化 + automigrate
- `gpt-image/server/internal/oss/storage/sqlite/models.go`：Account 等表结构（GORM）
- `gpt-image/server/internal/oss/pool/scheduler.go`：账号 lease + 选择策略（least-used + cooldown）
- `gpt-image/server/internal/oss/upstream/chatgpt/`：ChatGPT 协议上游适配器（从现有 `backend/` 迁移并改造）
  - `client.go`：核心调用（requirements、prepare、conversation、poll、download_url）
  - `tls.go`：uTLS transport（从 `backend/utils/tls.go` 迁移）
  - `pow.go`：PoW（从 `backend/utils/pow.go` 迁移）
  - `uuid.go`：UUID helper（从 `backend/utils/uuid.go` 迁移，或直接用 google/uuid）

#### tests

- `gpt-image/server/internal/oss/http/router_test.go`：端到端 handler 测试（使用 fake upstream + fake downloader，不出网）
- `gpt-image/server/internal/core/tasks/memory_store_test.go`：任务 TTL/读写测试
- `gpt-image/server/internal/oss/pool/scheduler_test.go`：lease/选择策略测试（用 in-memory repo stub）

---

## Task 0: 初始化 Go module + 最小可跑 server（/health）

**Files:**
- Create: `gpt-image/server/go.mod`
- Create: `gpt-image/server/cmd/gpt-image-oss/main.go`
- Create: `gpt-image/server/internal/oss/config/config.go`
- Create: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 创建目录结构**

Run (PowerShell):
```powershell
New-Item -ItemType Directory -Force gpt-image/server/cmd/gpt-image-oss | Out-Null
New-Item -ItemType Directory -Force gpt-image/server/internal/oss/config | Out-Null
New-Item -ItemType Directory -Force gpt-image/server/internal/oss/http | Out-Null
```

- [ ] **Step 2: 初始化 Go module + 依赖**

Run:
```powershell
cd gpt-image/server
go mod init gpt-image
go get github.com/gin-gonic/gin@v1.12.0
go get github.com/google/uuid@v1.6.0
```

Expected: `go: creating new go.mod: module gpt-image`

- [ ] **Step 3: 写一个失败的路由测试（/health）**

Create `gpt-image/server/internal/oss/http/router_test.go`:
```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 4: 运行测试，确认失败**

Run:
```powershell
go test ./... -v
```

Expected: FAIL（`NewRouter` 未定义 / 包不存在等编译错误）

- [ ] **Step 5: 写最小实现让测试通过**

Create `gpt-image/server/internal/oss/http/router.go`:
```go
package httpx

import "github.com/gin-gonic/gin"

type Deps struct{}

func NewRouter(_ Deps) *gin.Engine {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	return r
}
```

Create `gpt-image/server/internal/oss/config/config.go`:
```go
package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{Port: port}
}
```

Create `gpt-image/server/cmd/gpt-image-oss/main.go`:
```go
package main

import (
	"log"

	"gpt-image/internal/oss/config"
	httpx "gpt-image/internal/oss/http"
)

func main() {
	cfg := config.Load()
	r := httpx.NewRouter(httpx.Deps{})
	log.Printf("gpt-image-oss listening on :%s", cfg.Port)
	_ = r.Run(":" + cfg.Port)
}
```

- [ ] **Step 6: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

Expected: PASS

- [ ] **Step 7: Commit（如果你希望有 commit 轨迹）**

> 当前仓库根目录没有 `.git`。建议仅在 `gpt-image/` 下初始化一个独立 git，避免影响你已有的参考仓库子目录。

Run:
```powershell
cd ..
git init
git add .
git commit -m "chore(oss): bootstrap gpt-image server with health endpoint"
```

---

## Task 1: OSS 全局 AUTH_KEY 鉴权中间件（保护 /v1 与 /v1/admin）

**Files:**
- Create: `gpt-image/server/internal/oss/http/middleware/auth.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Modify: `gpt-image/server/internal/oss/config/config.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写失败测试：未带 Authorization 访问 /v1/models 必须 401**

Append to `router_test.go`:
```go
func TestAuth_Required(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{AuthKey: "testkey"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:
```powershell
go test ./... -v
```

Expected: FAIL（当前没有 `/v1/models`，或者未鉴权）

- [ ] **Step 3: 实现 Auth middleware + 配置读取**

Create `internal/oss/http/middleware/auth.go`:
```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireBearerKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server auth key not configured"})
			c.Abort()
			return
		}
		auth := c.GetHeader("Authorization")
		scheme, value, _ := strings.Cut(auth, " ")
		if !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(value) == "" || strings.TrimSpace(value) != expected {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is invalid"})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

Update `internal/oss/config/config.go`:
```go
type Config struct {
	Port    string
	AuthKey string
}

func Load() Config {
	// ...
	authKey := os.Getenv("AUTH_KEY")
	return Config{Port: port, AuthKey: authKey}
}
```

Update `internal/oss/http/router.go`（引入 deps + group）:
```go
package httpx

import (
	"github.com/gin-gonic/gin"

	"gpt-image/internal/oss/http/middleware"
)

type Deps struct {
	AuthKey string
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	v1 := r.Group("/v1")
	v1.Use(middleware.RequireBearerKey(d.AuthKey))
	{
		v1.GET("/models", func(c *gin.Context) { c.JSON(200, gin.H{"object": "list", "data": []any{}}) })
	}

	admin := r.Group("/v1/admin")
	admin.Use(middleware.RequireBearerKey(d.AuthKey))
	{
		admin.GET("/stats", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	}

	return r
}
```

Update `cmd/gpt-image-oss/main.go` 传入 `AuthKey`:
```go
cfg := config.Load()
r := httpx.NewRouter(httpx.Deps{AuthKey: cfg.AuthKey})
```

- [ ] **Step 4: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

Expected: PASS（`TestAuth_Required` 为 401）

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(oss): add global AUTH_KEY auth middleware for /v1 and /v1/admin"
```

---

## Task 2: `GET /v1/models`（返回 gpt-image-1/2）

**Files:**
- Create: `gpt-image/server/internal/core/openai/types.go`
- Create: `gpt-image/server/internal/oss/http/handlers/models.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写失败测试：/v1/models 返回包含 gpt-image-1/2**

Append to `router_test.go`:
```go
func TestModels_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{AuthKey: "testkey"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer testkey")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "gpt-image-1") || !strings.Contains(w.Body.String(), "gpt-image-2") {
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:
```powershell
go test ./... -v
```

Expected: FAIL（当前 data 为空或 handler 不存在）

- [ ] **Step 3: 定义 OpenAI types + models handler**

Create `internal/core/openai/types.go`:
```go
package openai

type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ListModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}
```

Create `internal/oss/http/handlers/models.go`:
```go
package handlers

import (
	"net/http"

	"gpt-image/internal/core/openai"
	"github.com/gin-gonic/gin"
)

func ListModels(c *gin.Context) {
	c.JSON(http.StatusOK, openai.ListModelsResponse{
		Object: "list",
		Data: []openai.ModelItem{
			{ID: "gpt-image-1", Object: "model", Created: 0, OwnedBy: "gpt-image"},
			{ID: "gpt-image-2", Object: "model", Created: 0, OwnedBy: "gpt-image"},
		},
	})
}
```

Modify `internal/oss/http/router.go` 使用 handler（并移除临时 inline handler）:
```go
import "gpt-image/internal/oss/http/handlers"
// ...
v1.GET("/models", handlers.ListModels)
```

- [ ] **Step 4: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(oss): implement GET /v1/models"
```

---

## Task 3: 任务系统（OSS 内存 TaskStore + `GET /v1/images/tasks/:id`）

**Files:**
- Create: `gpt-image/server/internal/core/tasks/task.go`
- Create: `gpt-image/server/internal/core/tasks/store.go`
- Create: `gpt-image/server/internal/core/tasks/memory_store.go`
- Create: `gpt-image/server/internal/core/tasks/memory_store_test.go`
- Create: `gpt-image/server/internal/oss/http/handlers/tasks.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写一个失败的 TaskStore 测试（Create/Get）**

Create `internal/core/tasks/memory_store_test.go`:
```go
package tasks

import (
	"testing"
	"time"
)

func TestMemoryStore_CreateGet(t *testing.T) {
	s := NewMemoryStore(10 * time.Minute)
	task := Task{ID: "t1", Status: StatusQueued, CreatedAt: time.Now()}
	if err := s.Create(task); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, ok := s.Get("t1")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got.ID != "t1" || got.Status != StatusQueued {
		t.Fatalf("unexpected task: %+v", got)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:
```powershell
go test ./... -v
```

Expected: FAIL（Task/Store 未实现）

- [ ] **Step 3: 实现 tasks 基础类型与内存存储**

Create `internal/core/tasks/task.go`:
```go
package tasks

import "time"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Task struct {
	ID         string     `json:"id"`
	Status     Status     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
```

Create `internal/core/tasks/store.go`:
```go
package tasks

type Store interface {
	Create(task Task) error
	Update(task Task) error
	Get(id string) (Task, bool)
}
```

Create `internal/core/tasks/memory_store.go`:
```go
package tasks

import (
	"errors"
	"sync"
	"time"
)

type memoryItem struct {
	task      Task
	expiresAt time.Time
}

type MemoryStore struct {
	ttl time.Duration
	mu  sync.RWMutex
	m   map[string]memoryItem
}

func NewMemoryStore(ttl time.Duration) *MemoryStore {
	return &MemoryStore{ttl: ttl, m: make(map[string]memoryItem)}
}

func (s *MemoryStore) Create(task Task) error {
	if task.ID == "" {
		return errors.New("task id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[task.ID] = memoryItem{task: task, expiresAt: time.Now().Add(s.ttl)}
	return nil
}

func (s *MemoryStore) Update(task Task) error {
	if task.ID == "" {
		return errors.New("task id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.m[task.ID]
	if !ok {
		return errors.New("task not found")
	}
	cur.task = task
	s.m[task.ID] = cur
	return nil
}

func (s *MemoryStore) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.m[id]
	if !ok {
		return Task{}, false
	}
	if s.ttl > 0 && time.Now().After(item.expiresAt) {
		return Task{}, false
	}
	return item.task, true
}
```

- [ ] **Step 4: 运行 TaskStore 测试，确认通过**

Run:
```powershell
go test ./... -v
```

Expected: PASS

- [ ] **Step 5: 写一个失败的 tasks handler 测试（404 / 200）**

Append to `router_test.go`:
```go
func TestTasks_Get_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{AuthKey: "testkey"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/images/tasks/does_not_exist", nil)
	req.Header.Set("Authorization", "Bearer testkey")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 6: 实现 tasks handler + 在 router 注入 store**

Create `internal/oss/http/handlers/tasks.go`:
```go
package handlers

import (
	"net/http"

	"gpt-image/internal/core/tasks"
	"github.com/gin-gonic/gin"
)

type TaskDeps struct {
	Store tasks.Store
}

func GetTask(d TaskDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		task, ok := d.Store.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, task)
	}
}
```

Modify `internal/oss/http/router.go`：
```go
import (
	"time"
	"gpt-image/internal/core/tasks"
	"gpt-image/internal/oss/http/handlers"
)

type Deps struct {
	AuthKey   string
	TaskStore tasks.Store
}

func NewRouter(d Deps) *gin.Engine {
	// ...
	store := d.TaskStore
	if store == nil {
		store = tasks.NewMemoryStore(24 * time.Hour)
	}

	v1 := r.Group("/v1")
	// ...
	v1.GET("/images/tasks/:id", handlers.GetTask(handlers.TaskDeps{Store: store}))
	// ...
}
```

Update tests 构造 `Deps` 时无需传 TaskStore（走默认内存 store）。

- [ ] **Step 7: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 8: Commit**

```powershell
git add .
git commit -m "feat(core): add in-memory task store and GET /v1/images/tasks/:id"
```

---

## Task 4: 全局并发限制（Limiter）+ async 任务执行骨架

**Files:**
- Create: `gpt-image/server/internal/core/limiter/limiter.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写一个失败的 limiter 单测**

Create `internal/core/limiter/limiter_test.go`:
```go
package limiter

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_AcquireTimeout(t *testing.T) {
	l := New(1)
	ctx := context.Background()
	release, err := l.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire 1: %v", err)
	}
	defer release()

	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = l.Acquire(ctx2)
	if err == nil {
		t.Fatalf("expected timeout err")
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 3: 实现 limiter**

Create `internal/core/limiter/limiter.go`:
```go
package limiter

import (
	"context"
	"errors"
)

type Limiter struct {
	ch chan struct{}
}

func New(max int) *Limiter {
	if max <= 0 {
		max = 1
	}
	return &Limiter{ch: make(chan struct{}, max)}
}

func (l *Limiter) Acquire(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {}, nil
	}
	select {
	case l.ch <- struct{}{}:
		return func() { <-l.ch }, nil
	case <-ctx.Done():
		return nil, errors.New("acquire timeout")
	}
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(core): add inflight limiter"
```

---

## Task 5: `POST /v1/images/generations`（sync + async=true）——先用 fake upstream 打通（不出网）

**Files:**
- Modify: `gpt-image/server/internal/core/openai/types.go`
- Create: `gpt-image/server/internal/core/openai/images.go`
- Create: `gpt-image/server/internal/core/openai/errors.go`
- Create: `gpt-image/server/internal/oss/http/handlers/images.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写失败测试：sync 默认 b64_json，并能返回 base64**

Append to `router_test.go`:
```go
// fake deps used only in tests
type fakeImageEngine struct{}

func (f fakeImageEngine) Generate(_ *gin.Context, reqBody string) (any, error) { return nil, nil }

func TestImagesGenerations_Sync_DefaultB64(t *testing.T) {
	// 该用例在本 Task 的 Step 3 会改成真实的 fake engine 接口
	t.Skip("placeholder until images handler is implemented")
}
```

> 说明：为了让计划可执行且不引入“先写大段实现再补测试”，在 Step 2 先把 types 与接口定型，然后回填这条测试为真正的 failing test（见 Step 2/3）。

- [ ] **Step 2: 定义 OpenAI images types + error 格式（先让测试可编译）**

Create `internal/core/openai/errors.go`:
```go
package openai

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
```

Create `internal/core/openai/images.go`:
```go
package openai

type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size"`
	ResponseFormat string `json:"response_format"`
	Async          bool   `json:"async"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageResponse struct {
	Created int64      `json:"created"`
	Data    []ImageData `json:"data"`
}
```

- [ ] **Step 3: 实现 images handler（注入 Engine + Downloader），并完善测试为真正 failing test**

Create `internal/oss/http/handlers/images.go`（最小可用：仅支持 fake engine 的“返回 url”，handler 再按 response_format 转 b64）:
```go
package handlers

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"gpt-image/internal/core/limiter"
	"gpt-image/internal/core/openai"
	"gpt-image/internal/core/tasks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ImageEngine interface {
	Generate(ctx context.Context, req openai.ImageGenerationRequest) ([]openai.ImageData, error)
}

type Downloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
}

type ImageDeps struct {
	Engine     ImageEngine
	Downloader Downloader
	Limiter    *limiter.Limiter
	TaskStore  tasks.Store
}

type httpDownloader struct{ client *http.Client }

func (d httpDownloader) Download(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func PostImageGenerations(d ImageDeps) gin.HandlerFunc {
	if d.Downloader == nil {
		d.Downloader = httpDownloader{client: http.DefaultClient}
	}
	if d.TaskStore == nil {
		d.TaskStore = tasks.NewMemoryStore(24 * time.Hour)
	}
	return func(c *gin.Context) {
		var req openai.ImageGenerationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "invalid json"}})
			return
		}
		if req.Prompt == "" {
			c.JSON(http.StatusBadRequest, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "prompt is required"}})
			return
		}
		if req.N <= 0 {
			req.N = 1
		}
		if req.ResponseFormat == "" {
			req.ResponseFormat = "b64_json"
		}

		run := func(ctx context.Context) (openai.ImageResponse, *tasks.Error) {
			release, err := d.Limiter.Acquire(ctx)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: "server busy", Code: "server_busy"}
			}
			defer release()

			items, err := d.Engine.Generate(ctx, req)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: err.Error(), Code: "upstream_error"}
			}

			out := make([]openai.ImageData, 0, len(items))
			for _, it := range items {
				if req.ResponseFormat == "b64_json" && it.URL != "" {
					b, derr := d.Downloader.Download(ctx, it.URL)
					if derr == nil {
						it.B64JSON = base64.StdEncoding.EncodeToString(b)
						it.URL = ""
					}
				}
				out = append(out, it)
			}
			return openai.ImageResponse{Created: time.Now().Unix(), Data: out}, nil
		}

		if !req.Async {
			resp, terr := run(c.Request.Context())
			if terr != nil {
				c.JSON(http.StatusBadGateway, openai.ErrorResponse{Error: openai.ErrorDetail{Message: terr.Message, Code: terr.Code}})
				return
			}
			c.JSON(http.StatusOK, resp)
			return
		}

		taskID := "imgtsk_" + uuid.NewString()
		now := time.Now()
		_ = d.TaskStore.Create(tasks.Task{ID: taskID, Status: tasks.StatusQueued, CreatedAt: now})

		go func() {
			start := time.Now()
			_ = d.TaskStore.Update(tasks.Task{ID: taskID, Status: tasks.StatusRunning, CreatedAt: now, StartedAt: &start})
			resp, terr := run(context.Background())
			finish := time.Now()
			if terr != nil {
				_ = d.TaskStore.Update(tasks.Task{ID: taskID, Status: tasks.StatusFailed, CreatedAt: now, StartedAt: &start, FinishedAt: &finish, Error: terr})
				return
			}
			_ = d.TaskStore.Update(tasks.Task{ID: taskID, Status: tasks.StatusSucceeded, CreatedAt: now, StartedAt: &start, FinishedAt: &finish, Result: resp})
		}()

		c.JSON(http.StatusOK, gin.H{"task_id": taskID})
	}
}
```

Update `internal/oss/http/router.go` 注入 limiter + task store + handler：
```go
type Deps struct {
	AuthKey   string
	TaskStore tasks.Store
	Limiter   *limiter.Limiter
	Image     handlers.ImageDeps
}
// 在 NewRouter 中：
lim := d.Limiter
if lim == nil { lim = limiter.New(20) }
store := d.TaskStore
if store == nil { store = tasks.NewMemoryStore(24 * time.Hour) }

v1.POST("/images/generations", handlers.PostImageGenerations(handlers.ImageDeps{
	Engine:     d.Image.Engine,
	Downloader: d.Image.Downloader,
	Limiter:    lim,
	TaskStore:  store,
}))
```

回填 `router_test.go` 里的生成接口测试（真正 failing test）：
```go
type fakeEngine struct{}

func (fakeEngine) Generate(_ context.Context, _ openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	return []openai.ImageData{{URL: "http://example.invalid/image.png"}}, nil
}

type fakeDownloader struct{}

func (fakeDownloader) Download(_ context.Context, _ string) ([]byte, error) {
	return []byte("hello"), nil
}

func TestImagesGenerations_Sync_DefaultB64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{
		AuthKey: "testkey",
		Limiter: limiter.New(20),
		TaskStore: tasks.NewMemoryStore(24 * time.Hour),
		Image: handlers.ImageDeps{
			Engine:     fakeEngine{},
			Downloader: fakeDownloader{},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"prompt":"hi"}`))
	req.Header.Set("Authorization", "Bearer testkey")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"b64_json"`) {
		t.Fatalf("expected b64_json, body=%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "aGVsbG8=") { // base64("hello")
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(oss): implement POST /v1/images/generations (sync + async tasks) with limiter and task store"
```

---

## Task 6: `POST /v1/images/edits`（multipart）——同样先用 fake upstream 打通

**Files:**
- Modify: `gpt-image/server/internal/oss/http/handlers/images.go`
- Test: `gpt-image/server/internal/oss/http/router_test.go`

- [ ] **Step 1: 写失败测试：multipart edits 返回 200 并包含 data**

Add to `router_test.go`（构造 multipart，上传 1 张图片）：
```go
func TestImagesEdits_Multipart_OK(t *testing.T) {
	// 本用例在实现 Edit 相关接口后补齐
	t.Skip("placeholder until edits handler is implemented")
}
```

- [ ] **Step 2: 扩展 ImageEngine 接口 + 实现 edits handler（最小支持 image + prompt）**

Update `ImageEngine` interface:
```go
type ImageEngine interface {
	Generate(ctx context.Context, req openai.ImageGenerationRequest) ([]openai.ImageData, error)
	Edit(ctx context.Context, prompt string, images [][]byte, mask []byte, responseFormat string) ([]openai.ImageData, error)
}
```

在 `images.go` 增加 `PostImageEdits(d ImageDeps)`：
```go
func PostImageEdits(d ImageDeps) gin.HandlerFunc {
	if d.Downloader == nil {
		d.Downloader = httpDownloader{client: http.DefaultClient}
	}
	if d.TaskStore == nil {
		d.TaskStore = tasks.NewMemoryStore(24 * time.Hour)
	}
	return func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
			c.JSON(http.StatusBadRequest, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "invalid multipart"}})
			return
		}
		prompt := c.PostForm("prompt")
		if prompt == "" {
			c.JSON(http.StatusBadRequest, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "prompt is required"}})
			return
		}
		responseFormat := c.PostForm("response_format")
		if responseFormat == "" {
			responseFormat = "b64_json"
		}

		readFiles := func(keys ...string) ([][]byte, error) {
			var out [][]byte
			for _, k := range keys {
				fhs := c.Request.MultipartForm.File[k]
				for _, fh := range fhs {
					f, err := fh.Open()
					if err != nil {
						return nil, err
					}
					b, err := io.ReadAll(f)
					_ = f.Close()
					if err != nil {
						return nil, err
					}
					out = append(out, b)
				}
			}
			return out, nil
		}

		images, err := readFiles("image", "image[]")
		if err != nil || len(images) == 0 {
			c.JSON(http.StatusBadRequest, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "image is required"}})
			return
		}

		var mask []byte
		if fh, err := c.FormFile("mask"); err == nil && fh != nil {
			f, _ := fh.Open()
			mask, _ = io.ReadAll(f)
			_ = f.Close()
		}

		// 同步实现（先不做 async，等 generations 稳了再补齐）
		release, err := d.Limiter.Acquire(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, openai.ErrorResponse{Error: openai.ErrorDetail{Message: "server busy"}})
			return
		}
		defer release()

		items, err := d.Engine.Edit(c.Request.Context(), prompt, images, mask, responseFormat)
		if err != nil {
			c.JSON(http.StatusBadGateway, openai.ErrorResponse{Error: openai.ErrorDetail{Message: err.Error()}})
			return
		}
		c.JSON(http.StatusOK, openai.ImageResponse{Created: time.Now().Unix(), Data: items})
	}
}
```

Update router 增加路由：
```go
v1.POST("/images/edits", handlers.PostImageEdits(handlers.ImageDeps{ /* same deps */ }))
```

补齐 `TestImagesEdits_Multipart_OK`（用 fake engine 返回 1 条 url，然后 fake downloader 转 b64）：
```go
type fakeEngineWithEdit struct{ fakeEngine }

func (fakeEngineWithEdit) Edit(_ context.Context, _ string, _ [][]byte, _ []byte, _ string) ([]openai.ImageData, error) {
	return []openai.ImageData{{B64JSON: "aGVsbG8="}}, nil
}
```

- [ ] **Step 3: 运行测试，确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 4: Commit**

```powershell
git add .
git commit -m "feat(oss): add POST /v1/images/edits (multipart) minimal implementation"
```

---

## Task 7: SQLite 账号池（Account 模型 + import/list/stats）+ Scheduler lease（单实例）

**Files:**
- Create: `gpt-image/server/internal/oss/storage/sqlite/db.go`
- Create: `gpt-image/server/internal/oss/storage/sqlite/models.go`
- Create: `gpt-image/server/internal/oss/pool/scheduler.go`
- Create: `gpt-image/server/internal/oss/pool/scheduler_test.go`
- Modify: `gpt-image/server/cmd/gpt-image-oss/main.go`
- Modify: `gpt-image/server/internal/oss/http/handlers/admin.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`

- [ ] **Step 1: 写 scheduler 选择策略 failing test（least-used + lease）**

Create `internal/oss/pool/scheduler_test.go`（用 stub repo，不连 DB）：
```go
package pool

import (
	"testing"
	"time"
)

type stubRepo struct {
	accounts []Account
}

func (r *stubRepo) ListAvailable(now time.Time) ([]Account, error) { return r.accounts, nil }
func (r *stubRepo) BumpUse(id uint) error                          { return nil }

func TestScheduler_LeaseLeastUsed(t *testing.T) {
	repo := &stubRepo{accounts: []Account{
		{ID: 1, UseCount: 10, Status: "active"},
		{ID: 2, UseCount: 1, Status: "active"},
	}}
	s := NewScheduler(repo)
	acc, err := s.Acquire(false)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if acc.ID != 2 {
		t.Fatalf("expected id=2 got=%d", acc.ID)
	}
	s.Release(acc.ID)
}
```

- [ ] **Step 2: 实现 sqlite models + db init（pure-go sqlite + gorm）**

Run:
```powershell
go get github.com/glebarez/sqlite@v1.11.0
go get gorm.io/gorm@v1.31.1
```

Create `internal/oss/storage/sqlite/models.go`:
```go
package sqlite

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	AccessToken string `gorm:"type:text;uniqueIndex;not null"`
	Proxy       string `gorm:"type:text"`
	Status      string `gorm:"index;default:active"`

	UseCount      int
	SuccessCount  int
	FailCount     int
	CooldownUntil *time.Time `gorm:"index"`
	LastUsedAt    *time.Time
}
```

Create `internal/oss/storage/sqlite/db.go`:
```go
package sqlite

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Account{}); err != nil {
		return nil, err
	}
	log.Printf("sqlite connected: %s", path)
	return db, nil
}
```

- [ ] **Step 3: 实现 Scheduler（lease + least-used）**

Create `internal/oss/pool/scheduler.go`：
```go
package pool

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type Account struct {
	ID       uint
	Status   string
	UseCount int
}

type Repo interface {
	ListAvailable(now time.Time) ([]Account, error)
	BumpUse(id uint) error
}

type Scheduler struct {
	repo Repo
	mu   sync.Mutex
	inUse map[uint]bool
}

func NewScheduler(repo Repo) *Scheduler {
	return &Scheduler{repo: repo, inUse: make(map[uint]bool)}
}

func (s *Scheduler) Acquire(_ bool) (Account, error) {
	now := time.Now()
	items, err := s.repo.ListAvailable(now)
	if err != nil {
		return Account{}, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UseCount < items[j].UseCount })

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range items {
		if s.inUse[it.ID] {
			continue
		}
		s.inUse[it.ID] = true
		_ = s.repo.BumpUse(it.ID)
		return it, nil
	}
	return Account{}, errors.New("no available accounts")
}

func (s *Scheduler) Release(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inUse, id)
}
```

- [ ] **Step 4: 运行 scheduler 测试，确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 5: 增加 admin handlers（import/list/stats）并接 SQLite**

Create `internal/oss/http/handlers/admin.go`（最小可用）：
```go
package handlers

import (
	"net/http"
	"strings"

	"gpt-image/internal/oss/storage/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminDeps struct {
	DB *gorm.DB
}

func GetStats(d AdminDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var total, active int64
		d.DB.Model(&sqlite.Account{}).Count(&total)
		d.DB.Model(&sqlite.Account{}).Where("status = ?", "active").Count(&active)
		c.JSON(http.StatusOK, gin.H{"total": total, "active": active})
	}
}

func GetAccounts(d AdminDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []sqlite.Account
		_ = d.DB.Order("id desc").Limit(200).Find(&items).Error
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func ImportAccounts(d AdminDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct{ Tokens []string `json:"tokens"` }
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		added := 0
		for _, t := range req.Tokens {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			acc := sqlite.Account{AccessToken: t, Status: "active"}
			if err := d.DB.Create(&acc).Error; err == nil {
				added++
			}
		}
		c.JSON(http.StatusOK, gin.H{"added": added})
	}
}
```

Modify router 注入 `DB *gorm.DB` 并注册 admin 路由：
```go
type Deps struct {
	AuthKey string
	DB      *gorm.DB
	// ...
}
admin := r.Group("/v1/admin")
admin.Use(middleware.RequireBearerKey(d.AuthKey))
admin.GET("/stats", handlers.GetStats(handlers.AdminDeps{DB: d.DB}))
admin.GET("/accounts", handlers.GetAccounts(handlers.AdminDeps{DB: d.DB}))
admin.POST("/accounts/import", handlers.ImportAccounts(handlers.AdminDeps{DB: d.DB}))
```

Modify `cmd/gpt-image-oss/main.go` 初始化 DB 并传入 deps：
```go
db, err := sqlite.Open("data/gpt-image-oss.db")
if err != nil { log.Fatal(err) }
r := httpx.NewRouter(httpx.Deps{AuthKey: cfg.AuthKey, DB: db, /* ... */})
```

- [ ] **Step 6: 手工冒烟（跑起来导入 token）**

Run:
```powershell
go run ./cmd/gpt-image-oss
```

Then:
```powershell
curl -Method Post http://localhost:8080/v1/admin/accounts/import `
  -Headers @{ Authorization="Bearer <AUTH_KEY>"; "Content-Type"="application/json" } `
  -Body '{"tokens":["<access_token_1>","<access_token_2>"]}'
```

Expected: `{"added":2}`（或略有差异）

- [ ] **Step 7: Commit**

```powershell
git add .
git commit -m "feat(oss): add sqlite account pool + admin import/list/stats + scheduler lease"
```

---

## Task 8: 接入真实 ChatGPT 协议上游（迁移并改造现有 backend 代码）

**Files:**
- Create: `gpt-image/server/internal/oss/upstream/chatgpt/*`
- Modify: `gpt-image/server/internal/oss/http/handlers/images.go`（Engine 实现从 fake 换成 chatgpt engine）
- Modify: `gpt-image/server/cmd/gpt-image-oss/main.go`（构建真实 Engine：scheduler + chatgpt client）

- [ ] **Step 1: 迁移现有实现（复制文件）**

Run（按需改路径）：
```powershell
New-Item -ItemType Directory -Force internal/oss/upstream/chatgpt | Out-Null
Copy-Item -Force ..\\..\\backend\\utils\\tls.go internal\\oss\\upstream\\chatgpt\\tls.go
Copy-Item -Force ..\\..\\backend\\utils\\pow.go internal\\oss\\upstream\\chatgpt\\pow.go
Copy-Item -Force ..\\..\\backend\\core\\openai.go internal\\oss\\upstream\\chatgpt\\client.go
```

- [ ] **Step 2: 改造 `client.go` 成为 `ImageEngine` 的实现（支持 Generate）**

在 `client.go` 中：
- 改 package 名为 `chatgpt`
- 移除对旧 module 的 import（`evo-image-api/...`），改为新路径
- 让 `Generate` 返回 `[]openai.ImageData`（至少填 `URL`，可选 `RevisedPrompt`）
- 把“最多 3 turn 重试”改为读取 env 配置 `IMG_MAX_TURNS`（默认 1）

完成后，在 `cmd/gpt-image-oss/main.go` 里用真实 engine 替换 fake：

```go
// 伪代码示例：main 中组装
// engine := oss.NewChatGPTImageEngine(db, scheduler, ...)
// routerDeps.Image.Engine = engine
```

- [ ] **Step 3: 提供最小离线单测：asset_pointer 提取器**

为了避免 CI/本地测试依赖外网：
- 在 `client.go` 抽出 `extractAssetPointers(mappingJson []byte) []string`（纯函数）
- 写 `client_test.go` 用固定 json fixture 测 `file-service://...` 的提取正确。

- [ ] **Step 4: 本地端到端冒烟（需要你真实 token）**

用 `/v1/images/generations` 发起真实请求，确认：
- 能拿到 `b64_json`
- `async=true` 能拿到 task_id，并能通过 tasks 查询到 `succeeded`

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(oss): integrate chatgpt upstream engine (protocol mode)"
```

---

## Verification Checklist（每个 Task 完成后都要做）

- `go test ./... -v` 通过
- `go run ./cmd/gpt-image-oss` 可启动
- `GET /health` 200
- `GET /v1/models`（带 auth）200
- `POST /v1/images/generations`（fake engine 测试）单测覆盖
- `GET /v1/images/tasks/:id` 404/200 行为正确

