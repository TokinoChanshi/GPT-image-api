# gpt-image OSS Server (M0) Implementation Plan (v2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新建 `gpt-image/server/`（Go）并实现 OSS 单租户后端：`GET /v1/models`、`POST /v1/images/generations`、`POST /v1/images/edits`、`GET /v1/images/tasks/:id`，支持默认同步 + `async=true` 任务模式、默认 `response_format=b64_json`、全局并发上限（默认 20）、单实例账号池 lease 调度，并提供最小管理接口：`GET /v1/admin/stats`、`GET /v1/admin/accounts`、`POST /v1/admin/accounts/import`。

**Architecture:** Gin Router（OSS edition）= 全局 `AUTH_KEY` 鉴权 + OpenAI 兼容 images/models 接口 + tasks（OSS 内存 TTL）+ limiter（inflight 控制）。上游实现采用 **ChatGPT 协议请求**（复制并改造本地参考 Go 实现），账号来源为 SQLite（GORM），调度器负责选择可用账号并做 in-process lease（单实例）。

**Tech Stack:** Go + `testing`, Gin, GORM + pure-go SQLite（`github.com/glebarez/sqlite`）, uTLS（`github.com/refraction-networking/utls`）, `github.com/google/uuid`.

---

## 文件结构（本计划将创建/修改）

### Server

- Create: `gpt-image/server/go.mod`
- Create: `gpt-image/server/cmd/gpt-image-oss/main.go`

### core（可复用）

- Create: `gpt-image/server/internal/core/openai/types.go`（models/images/errors）
- Create: `gpt-image/server/internal/core/limiter/limiter.go`
- Create: `gpt-image/server/internal/core/tasks/{task.go,store.go,memory_store.go,memory_store_test.go}`

### oss（单租户 + 账号池）

- Create: `gpt-image/server/internal/oss/config/config.go`
- Create: `gpt-image/server/internal/oss/http/router.go`
- Create: `gpt-image/server/internal/oss/http/middleware/auth.go`
- Create: `gpt-image/server/internal/oss/http/handlers/{models.go,tasks.go,images.go,admin.go}`
- Create: `gpt-image/server/internal/oss/storage/sqlite/{db.go,models.go,repo.go}`
- Create: `gpt-image/server/internal/oss/pool/{scheduler.go,scheduler_test.go}`
- Create: `gpt-image/server/internal/oss/upstream/chatgpt/{client.go,transport.go,pow.go,engine.go}`（由本地参考实现复制并改造）

### HTTP tests（不出网）

- Create: `gpt-image/server/internal/oss/http/health_test.go`
- Create: `gpt-image/server/internal/oss/http/auth_test.go`
- Create: `gpt-image/server/internal/oss/http/models_test.go`
- Create: `gpt-image/server/internal/oss/http/tasks_http_test.go`
- Create: `gpt-image/server/internal/oss/http/images_generations_test.go`
- Create: `gpt-image/server/internal/oss/http/images_edits_test.go`
- Create: `gpt-image/server/internal/oss/http/admin_http_test.go`

---

## Task 0: 初始化 Go module + 最小 server（/health）

**Files:**
- Create: `gpt-image/server/go.mod`
- Create: `gpt-image/server/cmd/gpt-image-oss/main.go`
- Create: `gpt-image/server/internal/oss/config/config.go`
- Create: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/health_test.go`

- [ ] **Step 1: 创建目录**

Run (PowerShell, repo root):
```powershell
New-Item -ItemType Directory -Force gpt-image/server/cmd/gpt-image-oss | Out-Null
New-Item -ItemType Directory -Force gpt-image/server/internal/oss/config | Out-Null
New-Item -ItemType Directory -Force gpt-image/server/internal/oss/http | Out-Null
```

- [ ] **Step 2: 初始化 go.mod + 依赖**

Run:
```powershell
cd gpt-image/server
go mod init gpt-image
go get github.com/gin-gonic/gin@v1.12.0
```

- [ ] **Step 3: 写失败测试（/health）**

Create `internal/oss/http/health_test.go`:
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
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 4: 运行测试确认失败**

Run:
```powershell
go test ./... -v
```

Expected: FAIL（`NewRouter` 未定义）

- [ ] **Step 5: 实现最小 router + config + main**

Create `internal/oss/http/router.go`:
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

Create `internal/oss/config/config.go`:
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

Create `cmd/gpt-image-oss/main.go`:
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

- [ ] **Step 6: 运行测试确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 7: Commit（可选，但推荐）**

> 当前仓库根目录没有 `.git`。若你想要 commit 轨迹，建议在 `gpt-image/` 下 `git init`。

---

## Task 1: AUTH_KEY 鉴权 middleware（保护 /v1 与 /v1/admin）

**Files:**
- Create: `gpt-image/server/internal/oss/http/middleware/auth.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Modify: `gpt-image/server/internal/oss/config/config.go`
- Test: `gpt-image/server/internal/oss/http/auth_test.go`

- [ ] **Step 1: 写失败测试（未带 Bearer 必须 401）**

Create `internal/oss/http/auth_test.go`:
```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuth_Required(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{AuthKey: "k"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d got %d body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: 运行测试确认失败（404/200 都算失败）**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 3: 实现 middleware + 在 router 增加分组**

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AUTH_KEY not configured"})
			c.Abort()
			return
		}
		auth := c.GetHeader("Authorization")
		scheme, value, _ := strings.Cut(auth, " ")
		if !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(value) != expected {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is invalid"})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

Modify `internal/oss/config/config.go`:
```go
type Config struct {
	Port    string
	AuthKey string
}

func Load() Config {
	// ...
	auth := os.Getenv("AUTH_KEY")
	return Config{Port: port, AuthKey: auth}
}
```

Modify `cmd/gpt-image-oss/main.go` 传入 AuthKey:
```go
cfg := config.Load()
r := httpx.NewRouter(httpx.Deps{AuthKey: cfg.AuthKey})
```

Modify `internal/oss/http/router.go`（新增 deps 字段与 /v1 分组）：
```go
package httpx

import (
	"gpt-image/internal/oss/http/middleware"

	"github.com/gin-gonic/gin"
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

- [ ] **Step 4: 运行测试确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(oss): add AUTH_KEY middleware and protect /v1, /v1/admin"
```

---

## Task 2: OpenAI types + `GET /v1/models`

**Files:**
- Create: `gpt-image/server/internal/core/openai/types.go`
- Create: `gpt-image/server/internal/oss/http/handlers/models.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/models_test.go`

- [ ] **Step 1: 写失败测试（返回包含 gpt-image-1/2）**

Create `internal/oss/http/models_test.go`:
```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestModels_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Deps{AuthKey: "k"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer k")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "gpt-image-1") || !strings.Contains(w.Body.String(), "gpt-image-2") {
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 3: 实现 core openai types + models handler**

Create `internal/core/openai/types.go`:
```go
package openai

// --- errors ---

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// --- models ---

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

// --- images ---

type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size"`
	Quality        string `json:"quality"`
	Background     string `json:"background"`
	ResponseFormat string `json:"response_format"`
	Async          bool   `json:"async"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
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

Modify `internal/oss/http/router.go` 使用 handler（替换临时 inline）：
```go
import "gpt-image/internal/oss/http/handlers"
// ...
v1.GET("/models", handlers.ListModels)
```

- [ ] **Step 4: 运行测试确认通过**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 5: Commit**

```powershell
git add .
git commit -m "feat(core): add openai types; feat(oss): implement GET /v1/models"
```

---

## Task 3: tasks（内存 TTL store + `GET /v1/images/tasks/:id`）

**Files:**
- Create: `gpt-image/server/internal/core/tasks/{task.go,store.go,memory_store.go,memory_store_test.go}`
- Create: `gpt-image/server/internal/oss/http/handlers/tasks.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/tasks_http_test.go`

- [ ] **Step 1: 写失败单测（MemoryStore Create/Get）**

Create `internal/core/tasks/memory_store_test.go`:
```go
package tasks

import (
	"testing"
	"time"
)

func TestMemoryStore_CreateGet(t *testing.T) {
	s := NewMemoryStore(10 * time.Minute)
	now := time.Now()
	if err := s.Create(Task{ID: "t1", Status: StatusQueued, CreatedAt: now}); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, ok := s.Get("t1")
	if !ok || got.ID != "t1" {
		t.Fatalf("unexpected: ok=%v task=%+v", ok, got)
	}
}
```

- [ ] **Step 2: 实现 tasks（Task/Store/MemoryStore）**

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

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type Task struct {
	ID         string     `json:"id"`
	Status     Status     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
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

type item struct {
	task      Task
	expiresAt time.Time
}

type MemoryStore struct {
	ttl time.Duration
	mu  sync.RWMutex
	m   map[string]item
}

func NewMemoryStore(ttl time.Duration) *MemoryStore {
	return &MemoryStore{ttl: ttl, m: make(map[string]item)}
}

func (s *MemoryStore) Create(task Task) error {
	if task.ID == "" {
		return errors.New("task id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[task.ID] = item{task: task, expiresAt: time.Now().Add(s.ttl)}
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
	it, ok := s.m[id]
	if !ok {
		return Task{}, false
	}
	if s.ttl > 0 && time.Now().After(it.expiresAt) {
		return Task{}, false
	}
	return it.task, true
}
```

- [ ] **Step 3: 跑单测（应通过）**

Run:
```powershell
go test ./... -v
```

- [ ] **Step 4: 写失败 HTTP 测试（tasks 404）**

Create `internal/oss/http/tasks_http_test.go`:
```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gpt-image/internal/core/tasks"
	"github.com/gin-gonic/gin"
)

func TestTasks_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := tasks.NewMemoryStore(24 * time.Hour)
	r := NewRouter(Deps{AuthKey: "k", TaskStore: store})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/images/tasks/none", nil)
	req.Header.Set("Authorization", "Bearer k")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d got %d body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}
```

- [ ] **Step 5: 实现 tasks handler + router 注入 TaskStore**

Create `internal/oss/http/handlers/tasks.go`:
```go
package handlers

import (
	"net/http"

	"gpt-image/internal/core/tasks"
	"github.com/gin-gonic/gin"
)

type TaskDeps struct{ Store tasks.Store }

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
	// ...
	v1.GET("/images/tasks/:id", handlers.GetTask(handlers.TaskDeps{Store: store}))
	// ...
}
```

- [ ] **Step 6: 跑测试（通过）+ Commit**

Run:
```powershell
go test ./... -v
```

Commit:
```powershell
git add .
git commit -m "feat(core): add tasks store; feat(oss): add GET /v1/images/tasks/:id"
```

---

## Task 4: limiter（全局并发门闩）

**Files:**
- Create: `gpt-image/server/internal/core/limiter/limiter.go`
- Create: `gpt-image/server/internal/core/limiter/limiter_test.go`

- [ ] **Step 1: 写失败单测（Acquire 超时）**

Create `internal/core/limiter/limiter_test.go`:
```go
package limiter

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_Timeout(t *testing.T) {
	l := New(1)
	release, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = l.Acquire(ctx)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
```

- [ ] **Step 2: 实现 limiter**

Create `internal/core/limiter/limiter.go`:
```go
package limiter

import (
	"context"
	"errors"
)

type Limiter struct{ ch chan struct{} }

func New(max int) *Limiter {
	if max <= 0 {
		max = 1
	}
	return &Limiter{ch: make(chan struct{}, max)}
}

func (l *Limiter) Acquire(ctx context.Context) (func(), error) {
	select {
	case l.ch <- struct{}{}:
		return func() { <-l.ch }, nil
	case <-ctx.Done():
		return nil, errors.New("acquire timeout")
	}
}
```

- [ ] **Step 3: 跑测试 + Commit**

Run:
```powershell
go test ./... -v
```

Commit:
```powershell
git add .
git commit -m "feat(core): add inflight limiter"
```

---

## Task 5: `POST /v1/images/generations`（sync + async=true；默认 b64_json）

**Files:**
- Create: `gpt-image/server/internal/oss/http/handlers/images.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/images_generations_test.go`

- [ ] **Step 1: 写失败测试（sync 默认 b64_json）**

> 先确保 `internal/oss/http/handlers/images.go` 里已经定义了 `handlers.ImageDeps` / `handlers.ImageEngine`（哪怕先返回 501），否则这个测试文件会因为类型不存在而无法编译。

Create `internal/oss/http/images_generations_test.go`:
```go
package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gpt-image/internal/core/limiter"
	"gpt-image/internal/core/openai"
	"gpt-image/internal/core/tasks"
	"gpt-image/internal/oss/http/handlers"
	"github.com/gin-gonic/gin"
)

type fakeEngine struct{}

func (fakeEngine) Generate(_ context.Context, _ openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	return []openai.ImageData{{URL: "http://example.invalid/x.png"}}, nil
}

func (fakeEngine) Edit(_ context.Context, _ string, _ [][]byte, _ []byte, _ openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	return nil, nil
}

type fakeDownloader struct{}

func (fakeDownloader) Download(_ context.Context, _ string) ([]byte, error) { return []byte("hello"), nil }

func TestGenerations_Sync_DefaultB64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := tasks.NewMemoryStore(24 * time.Hour)
	r := NewRouter(Deps{
		AuthKey:   "k",
		TaskStore: store,
		Limiter:   limiter.New(20),
		Image: handlers.ImageDeps{
			Engine:     fakeEngine{},
			Downloader: fakeDownloader{},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"prompt":"hi"}`))
	req.Header.Set("Authorization", "Bearer k")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"b64_json"`) || !strings.Contains(w.Body.String(), "aGVsbG8=") {
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}
```

- [ ] **Step 2: 实现 images handler（Generate + async tasks）**

Create `internal/oss/http/handlers/images.go`:
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
	Edit(ctx context.Context, prompt string, images [][]byte, mask []byte, req openai.ImageGenerationRequest) ([]openai.ImageData, error)
}

type Downloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
}

type ImageDeps struct {
	Engine     ImageEngine
	Downloader Downloader
	TaskStore  tasks.Store
	Limiter    *limiter.Limiter
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

func normalizeGenReq(req *openai.ImageGenerationRequest) {
	if req.N <= 0 {
		req.N = 1
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "b64_json"
	}
}

func buildResponse(ctx context.Context, dl Downloader, responseFormat string, items []openai.ImageData) []openai.ImageData {
	out := make([]openai.ImageData, 0, len(items))
	for _, it := range items {
		if responseFormat == "b64_json" && it.URL != "" {
			b, err := dl.Download(ctx, it.URL)
			if err == nil {
				it.B64JSON = base64.StdEncoding.EncodeToString(b)
				it.URL = ""
			}
		}
		out = append(out, it)
	}
	return out
}

func PostImageGenerations(d ImageDeps) gin.HandlerFunc {
	if d.Downloader == nil {
		d.Downloader = httpDownloader{client: http.DefaultClient}
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
		normalizeGenReq(&req)

		run := func(ctx context.Context) (openai.ImageResponse, *tasks.Error) {
			release, err := d.Limiter.Acquire(ctx)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: "server busy", Code: "server_busy"}
			}
			defer release()

			raw, err := d.Engine.Generate(ctx, req)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: err.Error(), Code: "upstream_error"}
			}
			data := buildResponse(ctx, d.Downloader, req.ResponseFormat, raw)
			return openai.ImageResponse{Created: time.Now().Unix(), Data: data}, nil
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

Modify `internal/oss/http/router.go` 注入 limiter/taskstore/engine，并注册路由：
```go
import (
	"time"

	"gpt-image/internal/core/limiter"
	"gpt-image/internal/core/tasks"
	"gpt-image/internal/oss/http/handlers"
)

type Deps struct {
	AuthKey   string
	TaskStore tasks.Store
	Limiter   *limiter.Limiter
	Image     handlers.ImageDeps
}

func NewRouter(d Deps) *gin.Engine {
	// ...
	store := d.TaskStore
	if store == nil {
		store = tasks.NewMemoryStore(24 * time.Hour)
	}
	lim := d.Limiter
	if lim == nil {
		lim = limiter.New(20)
	}

	v1 := r.Group("/v1")
	// ...
	v1.POST("/images/generations", handlers.PostImageGenerations(handlers.ImageDeps{
		Engine:     d.Image.Engine,
		Downloader: d.Image.Downloader,
		TaskStore:  store,
		Limiter:    lim,
	}))
	// tasks 路由仍指向同一个 store
	// ...
}
```

- [ ] **Step 3: 跑测试 + 增加 async 用例**

Add async test to `images_generations_test.go`（同文件末尾）：
```go
func TestGenerations_Async_TaskLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := tasks.NewMemoryStore(24 * time.Hour)
	r := NewRouter(Deps{
		AuthKey:   "k",
		TaskStore: store,
		Limiter:   limiter.New(20),
		Image: handlers.ImageDeps{
			Engine:     fakeEngine{},
			Downloader: fakeDownloader{},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"prompt":"hi","async":true}`))
	req.Header.Set("Authorization", "Bearer k")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "task_id") {
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}
```

Run:
```powershell
go test ./... -v
```

- [ ] **Step 4: Commit**

```powershell
git add .
git commit -m "feat(oss): implement POST /v1/images/generations (sync + async) with limiter + tasks"
```

---

## Task 6: `POST /v1/images/edits`（multipart；sync + async=true）

**Files:**
- Modify: `gpt-image/server/internal/oss/http/handlers/images.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Test: `gpt-image/server/internal/oss/http/images_edits_test.go`

- [ ] **Step 1: 写失败测试（multipart sync 默认 b64_json）**

Create `internal/oss/http/images_edits_test.go`:
```go
package httpx

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gpt-image/internal/core/limiter"
	"gpt-image/internal/core/openai"
	"gpt-image/internal/core/tasks"
	"gpt-image/internal/oss/http/handlers"
	"github.com/gin-gonic/gin"
)

type fakeEditEngine struct{}

func (fakeEditEngine) Generate(_ context.Context, _ openai.ImageGenerationRequest) ([]openai.ImageData, error) { return nil, nil }
func (fakeEditEngine) Edit(_ context.Context, _ string, _ [][]byte, _ []byte, _ openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	return []openai.ImageData{{URL: "http://example.invalid/e.png"}}, nil
}

func TestEdits_Multipart_Sync_DefaultB64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := tasks.NewMemoryStore(24 * time.Hour)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("prompt", "edit it")
	fw, _ := w.CreateFormFile("image", "a.png")
	_, _ = fw.Write([]byte("pngbytes"))
	_ = w.Close()

	r := NewRouter(Deps{
		AuthKey:   "k",
		TaskStore: store,
		Limiter:   limiter.New(20),
		Image: handlers.ImageDeps{
			Engine:     fakeEditEngine{},
			Downloader: fakeDownloader{},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", &buf)
	req.Header.Set("Authorization", "Bearer k")
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"b64_json"`) {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}
```

- [ ] **Step 2: 实现 edits handler（复用 buildResponse + limiter + async）**

在 `internal/oss/http/handlers/images.go` 增加：
- `PostImageEdits(d ImageDeps) gin.HandlerFunc`
- 解析 multipart：读取 `prompt`、`image`/`image[]`、可选 `mask`、可选 `async`（`async=true` 走任务）
- 默认 `response_format=b64_json`

实现要点（直接贴可用代码，追加在同文件末尾即可）：
> 下面代码会用到 `io` / `strings` / `uuid` 等；如果 `images.go` 的 import 列表缺少 `strings`，记得补上：`"strings"`。

```go
func PostImageEdits(d ImageDeps) gin.HandlerFunc {
	if d.Downloader == nil {
		d.Downloader = httpDownloader{client: http.DefaultClient}
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
		var req openai.ImageGenerationRequest
		req.ResponseFormat = c.PostForm("response_format")
		if req.ResponseFormat == "" {
			req.ResponseFormat = "b64_json"
		}
		req.Async = strings.EqualFold(c.PostForm("async"), "true")

		readFiles := func(keys ...string) ([][]byte, error) {
			var out [][]byte
			for _, k := range keys {
				for _, fh := range c.Request.MultipartForm.File[k] {
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

		run := func(ctx context.Context) (openai.ImageResponse, *tasks.Error) {
			release, err := d.Limiter.Acquire(ctx)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: "server busy", Code: "server_busy"}
			}
			defer release()
			raw, err := d.Engine.Edit(ctx, prompt, images, mask, req)
			if err != nil {
				return openai.ImageResponse{}, &tasks.Error{Message: err.Error(), Code: "upstream_error"}
			}
			data := buildResponse(ctx, d.Downloader, req.ResponseFormat, raw)
			return openai.ImageResponse{Created: time.Now().Unix(), Data: data}, nil
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

Modify router 注册路由：
```go
v1.POST("/images/edits", handlers.PostImageEdits(handlers.ImageDeps{
	Engine:     d.Image.Engine,
	Downloader: d.Image.Downloader,
	TaskStore:  store,
	Limiter:    lim,
}))
```

- [ ] **Step 3: 跑测试 + Commit**

Run:
```powershell
go test ./... -v
```

Commit:
```powershell
git add .
git commit -m "feat(oss): implement POST /v1/images/edits (multipart, sync + async)"
```

---

## Task 7: SQLite 账号池 + admin 接口 + Scheduler lease

**Files:**
- Create: `gpt-image/server/internal/oss/storage/sqlite/{db.go,models.go,repo.go}`
- Create: `gpt-image/server/internal/oss/pool/{scheduler.go,scheduler_test.go}`
- Create: `gpt-image/server/internal/oss/http/handlers/admin.go`
- Modify: `gpt-image/server/internal/oss/http/router.go`
- Modify: `gpt-image/server/cmd/gpt-image-oss/main.go`
- Test: `gpt-image/server/internal/oss/http/admin_http_test.go`

- [ ] **Step 1: 加依赖（gorm + sqlite）**

Run:
```powershell
go get gorm.io/gorm@v1.31.1
go get github.com/glebarez/sqlite@v1.11.0
```

- [ ] **Step 2: 实现 sqlite db + Account 模型**

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

	UseCount int
	LastUsedAt *time.Time
}
```

Create `internal/oss/storage/sqlite/db.go`:
```go
package sqlite

import (
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
	return db, nil
}
```

- [ ] **Step 3: 实现 repo + scheduler + 单测**

Create `internal/oss/storage/sqlite/repo.go`:
```go
package sqlite

import (
	"time"

	"gorm.io/gorm"
)

type Repo struct{ DB *gorm.DB }

func (r Repo) ListActive() ([]Account, error) {
	var out []Account
	err := r.DB.Where("status = ?", "active").Order("use_count asc, id asc").Find(&out).Error
	return out, err
}

func (r Repo) BumpUse(id uint) error {
	return r.DB.Model(&Account{}).Where("id = ?", id).Updates(map[string]any{
		"use_count":   gorm.Expr("use_count + 1"),
		"last_used_at": time.Now(),
	}).Error
}
```

Create `internal/oss/pool/scheduler.go`:
```go
package pool

import (
	"errors"
	"sort"
	"sync"

	"gpt-image/internal/oss/storage/sqlite"
)

type Repo interface {
	ListActive() ([]sqlite.Account, error)
	BumpUse(id uint) error
}

type Scheduler struct {
	repo Repo
	mu   sync.Mutex
	inUse map[uint]bool
}

func New(repo Repo) *Scheduler {
	return &Scheduler{repo: repo, inUse: make(map[uint]bool)}
}

func (s *Scheduler) Acquire() (sqlite.Account, error) {
	items, err := s.repo.ListActive()
	if err != nil {
		return sqlite.Account{}, err
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
	return sqlite.Account{}, errors.New("no available accounts")
}

func (s *Scheduler) Release(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inUse, id)
}
```

Create `internal/oss/pool/scheduler_test.go`:
```go
package pool

import (
	"testing"

	"gpt-image/internal/oss/storage/sqlite"
	"gorm.io/gorm"
)

type stubRepo struct{ items []sqlite.Account }

func (r stubRepo) ListActive() ([]sqlite.Account, error) { return r.items, nil }
func (r stubRepo) BumpUse(uint) error                    { return nil }

func TestScheduler_LeastUsed(t *testing.T) {
	s := New(stubRepo{items: []sqlite.Account{
		{Model: gorm.Model{ID: 1}, UseCount: 9, Status: "active"},
		{Model: gorm.Model{ID: 2}, UseCount: 1, Status: "active"},
	}})
	acc, err := s.Acquire()
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if acc.ID != 2 {
		t.Fatalf("expected id=2 got=%d", acc.ID)
	}
	s.Release(acc.ID)
}
```

- [ ] **Step 4: admin handlers + http 单测（import + stats）**

Create `internal/oss/http/handlers/admin.go`:
```go
package handlers

import (
	"net/http"
	"strings"

	"gpt-image/internal/oss/storage/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminDeps struct{ DB *gorm.DB }

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
			if err := d.DB.Create(&sqlite.Account{AccessToken: t, Status: "active"}).Error; err == nil {
				added++
			}
		}
		c.JSON(http.StatusOK, gin.H{"added": added})
	}
}
```

Create `internal/oss/http/admin_http_test.go`（用 temp sqlite 文件）：
```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gpt-image/internal/core/limiter"
	"gpt-image/internal/core/tasks"
	"gpt-image/internal/oss/storage/sqlite"
	"github.com/gin-gonic/gin"
)

func TestAdmin_ImportAndStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	db, err := sqlite.Open(dir + "/t.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	r := NewRouter(Deps{AuthKey: "k", DB: db, TaskStore: tasks.NewMemoryStore(24 * time.Hour), Limiter: limiter.New(1)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/accounts/import", strings.NewReader(`{"tokens":["a","b"]}`))
	req.Header.Set("Authorization", "Bearer k")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("import expected 200 got %d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/v1/admin/stats", nil)
	req2.Header.Set("Authorization", "Bearer k")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("stats expected 200 got %d body=%s", w2.Code, w2.Body.String())
	}
}
```

Modify router 注入 DB + 注册 admin 路由：
```go
type Deps struct {
	AuthKey string
	DB      *gorm.DB
	// ...
}
admin.GET("/stats", handlers.GetStats(handlers.AdminDeps{DB: d.DB}))
admin.GET("/accounts", handlers.GetAccounts(handlers.AdminDeps{DB: d.DB}))
admin.POST("/accounts/import", handlers.ImportAccounts(handlers.AdminDeps{DB: d.DB}))
```

Modify main：打开 sqlite 并传入 deps（默认 `data/gpt-image-oss.db`）。

- [ ] **Step 5: go test + Commit**

Run:
```powershell
go test ./... -v
```

Commit:
```powershell
git add .
git commit -m "feat(oss): add sqlite account pool + admin endpoints + scheduler"
```

---

## Task 8: 接入真实 ChatGPT 上游（复制本地参考实现）并替换 fake engine

**Files:**
- Create: `gpt-image/server/internal/oss/upstream/chatgpt/{client.go,transport.go,pow.go,engine.go}`
- Modify: `gpt-image/server/cmd/gpt-image-oss/main.go`（组装真实 engine）

- [ ] **Step 1: 复制参考实现到新包（并改 package/import）**

从本地 `fran0220_chatgpt2api/handler/` 复制（在 `gpt-image/server` 目录执行）：
```powershell
New-Item -ItemType Directory -Force internal/oss/upstream/chatgpt | Out-Null
Copy-Item -Force ..\\..\\fran0220_chatgpt2api\\handler\\client.go internal\\oss\\upstream\\chatgpt\\client.go
Copy-Item -Force ..\\..\\fran0220_chatgpt2api\\handler\\transport.go internal\\oss\\upstream\\chatgpt\\transport.go
Copy-Item -Force ..\\..\\fran0220_chatgpt2api\\handler\\pow.go internal\\oss\\upstream\\chatgpt\\pow.go
```

然后把三个文件的 `package handler` 改为 `package chatgpt`，并修正 import 路径（保持文件内相互引用）。

补依赖：
```powershell
go get github.com/refraction-networking/utls@v1.8.2
go get golang.org/x/net@v0.53.0
go get golang.org/x/crypto@v0.50.0
go get github.com/google/uuid@v1.6.0
```

- [ ] **Step 2: 写 `engine.go`（用 scheduler 取账号，调用 chatgpt client）**

Create `internal/oss/upstream/chatgpt/engine.go`:
```go
package chatgpt

import (
	"context"
	"fmt"

	"gpt-image/internal/core/openai"
	"gpt-image/internal/oss/pool"
	"gpt-image/internal/oss/storage/sqlite"
)

type Engine struct {
	Scheduler *pool.Scheduler
}

func (e Engine) Generate(ctx context.Context, req openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	acc, err := e.Scheduler.Acquire()
	if err != nil {
		return nil, err
	}
	defer e.Scheduler.Release(acc.ID)

	client := NewChatGPTClient(acc.AccessToken, "")
	results, err := client.GenerateImage(ctx, req.Prompt, req.N, req.Size, req.Quality, req.Background)
	if err != nil {
		return nil, err
	}
	out := make([]openai.ImageData, 0, len(results))
	for _, it := range results {
		out = append(out, openai.ImageData{URL: it.URL, RevisedPrompt: it.RevisedPrompt})
	}
	return out, nil
}

func (e Engine) Edit(ctx context.Context, prompt string, images [][]byte, mask []byte, req openai.ImageGenerationRequest) ([]openai.ImageData, error) {
	acc, err := e.Scheduler.Acquire()
	if err != nil {
		return nil, err
	}
	defer e.Scheduler.Release(acc.ID)

	client := NewChatGPTClient(acc.AccessToken, "")
	results, err := client.EditImageByUpload(ctx, prompt, images, mask)
	if err != nil {
		return nil, fmt.Errorf("edit: %w", err)
	}
	out := make([]openai.ImageData, 0, len(results))
	for _, it := range results {
		out = append(out, openai.ImageData{URL: it.URL, RevisedPrompt: it.RevisedPrompt})
	}
	return out, nil
}

// compile-time check
var _ any = sqlite.Account{}
```

- [ ] **Step 3: 在 main.go 注入真实 engine（替换测试用 fake）**

在 `cmd/gpt-image-oss/main.go`：
- 打开 sqlite：`db, _ := sqlite.Open(<path>)`
- 构建 repo + scheduler：`repo := sqlite.Repo{DB: db}; sched := pool.New(repo)`
- 构建 engine：`eng := chatgpt.Engine{Scheduler: sched}`
- `NewRouter(Deps{..., Image: handlers.ImageDeps{Engine: eng}})`

- [ ] **Step 4: 本地冒烟（需要真实 token）**

1) 启动：
```powershell
$env:AUTH_KEY="k"; go run ./cmd/gpt-image-oss
```

2) 导入 token：
```powershell
curl -Method Post http://localhost:8080/v1/admin/accounts/import `
  -Headers @{ Authorization="Bearer k"; "Content-Type"="application/json" } `
  -Body '{"tokens":["<your_access_token>"]}'
```

3) 生图（同步 b64）：
```powershell
curl -Method Post http://localhost:8080/v1/images/generations `
  -Headers @{ Authorization="Bearer k"; "Content-Type"="application/json" } `
  -Body '{"prompt":"a cute cat, watercolor","response_format":"b64_json"}'
```

Expected: `data[0].b64_json` 有值。

- [ ] **Step 5: go test + Commit**

Run:
```powershell
go test ./... -v
```

Commit:
```powershell
git add .
git commit -m "feat(oss): integrate chatgpt upstream engine backed by sqlite account pool"
```

---

## 全局验收

- `go test ./... -v` 全绿
- `GET /health` 200
- `GET /v1/models` 200（带 AUTH_KEY）
- `POST /v1/images/generations`：sync + async 正常（async 能从 tasks 查到结果）
- `POST /v1/images/edits`：multipart sync + async 正常
- `GET /v1/admin/stats`、`/v1/admin/accounts`、`POST /v1/admin/accounts/import` 正常
