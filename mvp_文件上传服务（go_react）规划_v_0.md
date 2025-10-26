# 项目总览

**项目名**：DropLite（临时名）

**目标**：构建一个小而完整的「文件上传与下载」Web 服务，以学习并实践 Go 后端开发（HTTP API、存储、鉴权、部署、监控），前端使用 React 提供最小可用的上传 UI。

**核心价值**：可靠上传、可恢复、可查看/下载、基础权限控制与审计日志。

**MVP 范围**：

- 匿名/简易鉴权下的单文件上传（<= 100MB）。
- 前端展示上传进度，完成后显示文件卡片（名称、大小、类型、上传时间、下载链接）。
- 后端持久化文件**元数据**（PostgreSQL），**对象数据**存储至本地磁盘（开发）与 S3 兼容存储（生产）。
- 下载/删除文件的 API（仅资源拥有者可操作）。
- 基础限流与大小/类型校验。
- 观测与日志：结构化日志、请求指标、错误告警（本地先用控制台 + Prometheus/ Grafana 预留）。

**非目标（后续扩展）**：多租户、团队协作、复杂权限、预览转换、DLP/杀毒、WebHook、通知、版本回滚等。

---

# 技术选型

## 前端

- **框架**：React 18 + TypeScript + Vite。
- **UI**：Tailwind CSS + shadcn/ui（快速出 UI）。
- **状态**：React Query（请求/缓存）+ 受控表单。
- **上传**：原生 `XMLHttpRequest` 或 `fetch` + `onUploadProgress`（Axios 可选），支持断点续传时考虑 tus-js-client 或 S3 分片直传。
- **代码质量**：ESLint + Prettier + Vitest。

## 后端（Go）

- **语言/版本**：Go 1.22+。
- **Web 框架**：`chi`（轻量路由/中间件），或 `gin`（生态广）。本项目默认 **chi**。
- **配置**：`env` + `dotenv`（12-Factor）。
- **数据库**：PostgreSQL（元数据）+ `sqlc`（以 SQL 为中心的类型安全访问）。迁移：`golang-migrate`。
- **对象存储**：开发用本地磁盘或 **MinIO**，生产用 **AWS S3**（通过统一接口抽象）。
- **鉴权**：MVP 使用 **API Key**（简洁），后续升级为 JWT/OIDC。
- **日志**：`zap`/`zerolog`（结构化 JSON 日志）。
- **观测**：OpenTelemetry（otlp）+ Prometheus 指标（`promhttp`）。
- **安全**：`cors`、`rate-limit`（令牌桶/漏桶）、`Content-Type`/大小白名单校验、etag/sha256 校验可选。
- **打包与部署**：Docker + docker-compose（本地）、Fly.io/Render/Hetzner/AWS（生产）。

## 测试与质量

- **后端测试**：Go test（单测/集成测），`httptest`、`testcontainers`（起 Postgres/MinIO）。
- **前端测试**：Vitest + React Testing Library；E2E：Playwright。
- **CI/CD**：GitHub Actions（lint、test、构建镜像、发布）。

---

# 架构设计

```
[React App] --HTTPS--> [Go API Gateway (chi)] --> [Service Layer]
                                         |-> [PostgreSQL (metadata)]
                                         |-> [Object Storage: Local/MinIO/S3]
                                         |-> [Logger/Metrics]
```

**领域对象**：

- File：`id`、`owner_id`、`filename`、`mime_type`、`size`、`checksum`、`storage_key`、`created_at`、`deleted_at`。
- User（MVP 可选，若启用 API Key 则 `owner_id` 来自 key）。
- UploadSession（扩展：分片与断点续传）。

---

# API 设计（MVP）

**Base URL**：`/api/v1`

**鉴权**：HTTP Header：`Authorization: ApiKey <token>`（MVP 可设为固定测试 key）。

## Endpoints

- `POST /files`：上传文件（表单 `multipart/form-data` 字段名 `file`）。
  - 请求头：`Content-Type: multipart/form-data`
  - 响应：`{ id, filename, size, mime_type, download_url }`
- `GET /files`：列出当前 owner 的文件（分页）。
  - 查询：`?page=1&page_size=20`
- `GET /files/{id}`：获取文件元数据。
- `GET /files/{id}/download`：下载文件（302 到签名 URL，或 API 直接流式返回）。
- `DELETE /files/{id}`：删除（软删）。

> 未来扩展（非 MVP）：
>
> - `POST /uploads`（创建上传会话）/ `PATCH /uploads/{id}`（分片上报）/ `POST /uploads/{id}/complete`（合并）。
> - S3 直传（前端获取后端生成的 pre-signed URL，前端直传到 S3）。

## 错误码约定

- 4xx：`code` + `message` + `request_id`；
- 常见：`invalid_content_type`、`file_too_large`、`mime_not_allowed`、`not_found`、`unauthorized`、`forbidden`、`rate_limited`。

---

# 数据库与存储

## 表结构（sqlc 驱动）

```sql
-- files
CREATE TABLE files (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id     TEXT NOT NULL,
  filename     TEXT NOT NULL,
  mime_type    TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL CHECK (size_bytes >= 0),
  checksum     TEXT, -- sha256 可选
  storage_key  TEXT NOT NULL, -- 例如 s3 key 或本地路径
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_files_owner ON files(owner_id) WHERE deleted_at IS NULL;
```

## 存储策略

- 开发：`/var/data/uploads/<yyyy>/<mm>/<uuid>/<original_name>`（避免目录过大）。
- 生产：S3 bucket，私有；下载通过后端生成 **预签名 URL**（有效期如 5 分钟）。
- 文件名安全：保存原名但实际访问用 `storage_key`；防止路径穿越；严格校验 MIME 与扩展名映射。
- 配额（可选）：按 owner 限制总容量/文件数。

---

# 安全与合规（MVP 级）

- CORS 白名单、仅 HTTPS（本地自签）。
- 上传大小上限（如 100MB，Nginx/反代与应用双重限制）。
- MIME 白名单（如 image/\*, pdf, zip）。
- 反病毒（后续）：集成 ClamAV（异步扫描 + 隔离）。
- 审计日志：记录 `request_id`、来源 IP、owner、操作与结果。
- 速率限制：IP + owner 维度（如 10 req/s 突发 20）。

---

# 项目结构（后端）

```
backend/
  cmd/server/main.go
  internal/
    api/            # handlers + DTO + validation
    service/        # 用例/领域服务（上传、删除、签名URL）
    repo/           # sqlc 生成的查询接口 + 封装
    storage/        # S3/MinIO/Local 的统一接口与实现
    middleware/     # auth, cors, rate-limit, logging, recovery
    config/         # env 解析
    observability/  # metrics, tracing
    pkg/            # 公共工具（id, time, error）
  db/
    migrations/     # golang-migrate 脚本
    queries/        # sqlc 查询
  go.mod
  Makefile
  Dockerfile
  docker-compose.yml
```

**关键接口示例**（伪代码）：

```go
// storage.Interface
Store(ctx context.Context, r io.Reader, size int64, mime string) (key string, err error)
Load(ctx context.Context, key string) (rc io.ReadCloser, mime string, size int64, err error)
Delete(ctx context.Context, key string) error
PresignGet(ctx context.Context, key string, ttl time.Duration) (url string, err error) // 可选
```

---

# 项目结构（前端）

```
frontend/
  src/
    app/
      routes.tsx
    components/
      FileUpload.tsx
      FileList.tsx
      ProgressBar.tsx
    lib/
      api.ts        # 基于 fetch/react-query 的封装
    pages/
      Home.tsx
      Files.tsx
  index.html
  vite.config.ts
  package.json
  Dockerfile
```

**上传流程（前端）**：

1. 选择文件 → 客户端校验（大小/类型）。
2. `POST /files`（multipart）→ 监听进度 → 成功后刷新列表。
3. 下载：打开后端返回的 `download_url`。
4. 删除：`DELETE /files/{id}`，确认后刷新。

---

# 开发流程（工业化实践）

1. **需求冻结（本文档）** → 建立 issue 看板（GitHub Projects）。
2. **代码规范与工具**：Go fmt/vet、golangci-lint；前端 ESLint/Prettier；Commitlint + Conventional Commits；语义化版本。
3. **环境**：dev（本地）、staging（单人/预发）、prod（云）。配置通过 env 管理（`.env`）。
4. **数据库迁移**：PR 合并前必须包含对应迁移脚本，CI 执行 `migrate up` 校验。
5. **API 契约**：OpenAPI（`openapi.yaml`），前端通过 `openapi-typescript` 生成类型（保持前后端契约一致）。
6. **日志/指标**：每个请求注入 `request_id`，记录 handler 延迟、状态码、错误；暴露 `/metrics` 给 Prometheus。
7. **发布**：每次合并发布 Docker 镜像（标签：git sha + semver），staging 自动部署，prod 手动审批。

---

# 里程碑与任务拆分

## 里程碑 0：脚手架（1–2 天）

-

## 里程碑 1：上传&元数据（2–3 天）

-

## 里程碑 2：列表&下载&删除（2–3 天）

-

## 里程碑 3：鉴权与限流（2 天）

-

## 里程碑 4：S3/MinIO 抽象（2–3 天）

-

## 里程碑 5：CI/CD 与可观测（2 天）

-

---

# 可运行脚手架（代码清单）

> 复制这些文件即可本地 `docker-compose up --build` 起环境（Postgres/MinIO/API/Frontend）。

**backend/go.mod**

```go
module github.com/you/droplite

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.11
	github.com/go-chi/cors v1.2.1
	github.com/rs/zerolog v1.33.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/jackc/pgx/v5/stdlib v5.7.2
	github.com/minio/minio-go/v7 v7.0.66
	github.com/google/uuid v1.6.0
)
```

**backend/cmd/server/main.go**

```go
package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.NewConsoleWriter())

	db, err := sql.Open("pgx", getenv("DB_DSN", "postgres://postgres:postgres@localhost:5432/uploader?sslmode=disable"))
	if err != nil { log.Fatal().Err(err).Msg("open db") }
	defer db.Close()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET","POST","DELETE"}, AllowedHeaders: []string{"*"}}))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("ok")) })

	// TODO: mount /metrics
	// TODO: mount /api/v1 with handlers

	addr := ":" + getenv("PORT", "8080")
	log.Info().Str("addr", addr).Msg("listening")
	if err := http.ListenAndServe(addr, r); err != nil { log.Fatal().Err(err).Msg("server") }
}

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
```

**backend/db/migrations/0001\_init.sql**

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS files (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id     TEXT NOT NULL,
  filename     TEXT NOT NULL,
  mime_type    TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL CHECK (size_bytes >= 0),
  checksum     TEXT,
  storage_key  TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files(owner_id) WHERE deleted_at IS NULL;
```

**backend/sqlc.yaml**

```yaml
version: "2"
sql:
  - schema: "db/migrations"
    queries: "db/queries"
    engine: postgresql
    gen:
      go:
        package: "repo"
        out: "internal/repo"
```

**backend/db/queries/files.sql**

```sql
-- name: InsertFile :one
INSERT INTO files (owner_id, filename, mime_type, size_bytes, checksum, storage_key)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING *;

-- name: GetFile :one
SELECT * FROM files WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFiles :many
SELECT * FROM files WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SoftDeleteFile :exec
UPDATE files SET deleted_at = now() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL;
```

**backend/internal/middleware/auth.go**

```go
package middleware

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string
const OwnerKey ctxKey = "owner_id"

func APIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "ApiKey ") { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		owner := strings.TrimPrefix(auth, "ApiKey ")
		r = r.WithContext(context.WithValue(r.Context(), OwnerKey, owner))
		next.ServeHTTP(w, r)
	})
}
```

**backend/internal/storage/storage.go**

```go
package storage

import "io"

type Interface interface {
	Store(r io.Reader, size int64, mime string) (key string, err error)
	Load(key string) (rc io.ReadCloser, mime string, size int64, err error)
	Delete(key string) error
	PresignGet(key string, ttlSeconds int64) (string, error)
}
```

**backend/internal/storage/local/local.go**

```go
package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Local struct{ Root string }

func (l Local) Store(r io.Reader, size int64, mime string) (string, error) {
	p := filepath.Join(l.Root, time.Now().Format("2006/01"))
	if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
	f, err := os.CreateTemp(p, "upl_*")
	if err != nil { return "", err }
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil { return "", err }
	return f.Name(), nil
}
func (l Local) Load(key string) (io.ReadCloser, string, int64, error) {
	fi, err := os.Stat(key); if err != nil { return nil, "", 0, err }
	f, err := os.Open(key); if err != nil { return nil, "", 0, err }
	return f, "application/octet-stream", fi.Size(), nil
}
func (l Local) Delete(key string) error { return os.Remove(key) }
func (l Local) PresignGet(key string, ttlSeconds int64) (string, error) { return fmt.Sprintf("/api/v1/files/raw?key=%s", key), nil }
```

**backend/internal/api/files.go**（精简示例）

```go
package api

import (
	"database/sql"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/you/droplite/internal/middleware"
	"github.com/you/droplite/internal/storage"
)

type FilesHandler struct{
	DB *sql.DB
	Store storage.Interface
}

func (h *FilesHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(middleware.APIKey).Post("/files", h.upload)
	return r
}

func (h *FilesHandler) upload(w http.ResponseWriter, r *http.Request) {
	owner := r.Context().Value(middleware.OwnerKey).(string)
	file, hdr, err := r.FormFile("file")
	if err != nil { http.Error(w, "file required", http.StatusBadRequest); return }
	defer file.Close()

	key, err := h.Store.Store(file, hdr.Size, hdr.Header.Get("Content-Type"))
	if err != nil { log.Error().Err(err).Msg("store"); http.Error(w, "store error", 500); return }

	// 省略：INSERT 元数据到 DB（使用 sqlc 生成的 InsertFile）
	_ = owner; _ = key

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
}
```

**backend/Dockerfile**

```Dockerfile
FROM golang:1.22 as build
WORKDIR /app
COPY . .
RUN go build -o /out/api ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/api /api
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/api"]
```

**docker-compose.yml（根目录）**

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: uploader
    ports: ["5432:5432"]
  minio:
    image: minio/minio:RELEASE.2024-05-10T18-33-45Z
    command: server /data
    environment:
      MINIO_ACCESS_KEY: minio
      MINIO_SECRET_KEY: minio123
    ports: ["9000:9000"]
  api:
    build: ./backend
    environment:
      DB_DSN: postgres://postgres:postgres@postgres:5432/uploader?sslmode=disable
      LOCAL_STORAGE_ROOT: /data
      STORAGE_DRIVER: local
      API_KEY: dev-123456
    volumes:
      - ./data:/data
    ports: ["8080:8080"]
    depends_on: [postgres, minio]
  web:
    build: ./frontend
    ports: ["5173:5173"]
    depends_on: [api]
```

**frontend/package.json**

```json
{
  "name": "droplite-web",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "@tanstack/react-query": "^5.51.0"
  },
  "devDependencies": {
    "typescript": "^5.6.2",
    "vite": "^5.4.0",
    "@types/react": "^18.2.66",
    "@types/react-dom": "^18.2.22"
  }
}
```

**frontend/src/main.tsx**

```tsx
import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'

createRoot(document.getElementById('root')!).render(<App />)
```

**frontend/index.html**

```html
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>DropLite</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**frontend/src/App.tsx**

```tsx
import React, { useRef, useState } from 'react'

export default function App(){
  const inputRef = useRef<HTMLInputElement>(null)
  const [progress, setProgress] = useState<number>(0)
  const [msg, setMsg] = useState<string>('')

  const upload = async () => {
    const f = inputRef.current?.files?.[0]
    if(!f){ setMsg('请选择文件'); return }
    const form = new FormData()
    form.append('file', f)

    const xhr = new XMLHttpRequest()
    xhr.upload.onprogress = (e) => {
      if(e.lengthComputable){ setProgress(Math.round(e.loaded / e.total * 100)) }
    }
    xhr.onload = () => setMsg(xhr.status === 200 || xhr.status === 201 ? '上传成功' : '上传失败')
    xhr.onerror = () => setMsg('网络错误')
    xhr.open('POST', '/api/v1/files')
    xhr.setRequestHeader('Authorization', 'ApiKey dev-123456')
    xhr.send(form)
  }

  return (
    <div style={{maxWidth:600, margin:'40px auto', fontFamily:'system-ui'}}>
      <h1>DropLite</h1>
      <input ref={inputRef} type="file" />
      <button onClick={upload} style={{marginLeft:8}}>上传</button>
      <div style={{marginTop:12}}>进度：{progress}%</div>
      <div>{msg}</div>
    </div>
  )
}
```

---

# 配置示例（env）

```
APP_ENV=dev
PORT=8080
API_KEY=dev-123456
# Postgres
DB_DSN=postgres://postgres:postgres@postgres:5432/uploader?sslmode=disable
# 存储
STORAGE_DRIVER=local # local|s3
LOCAL_STORAGE_ROOT=/var/data/uploads
S3_BUCKET=droplite-prod
S3_REGION=us-west-2
S3_ENDPOINT=https://s3.us-west-2.amazonaws.com
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
# 安全
MAX_UPLOAD_BYTES=104857600
ALLOWED_MIME=image/*,application/pdf,application/zip
```

---

# OpenAPI 片段（简化）

```yaml
openapi: 3.0.3
info: { title: DropLite API, version: 0.1.0 }
paths:
  /files:
    post:
      security: [{ ApiKeyAuth: [] }]
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file: { type: string, format: binary }
              required: [file]
      responses:
        "201": { description: Created }
        "400": { description: Bad Request }
        "413": { description: Payload Too Large }
    get:
      security: [{ ApiKeyAuth: [] }]
      parameters:
        - in: query
          name: page
          schema: { type: integer, minimum: 1, default: 1 }
        - in: query
          name: page_size
          schema: { type: integer, minimum: 1, maximum: 100, default: 20 }
      responses:
        "200": { description: OK }
  /files/{id}/download:
    get:
      security: [{ ApiKeyAuth: [] }]
      responses:
        "302": { description: Redirect to presigned URL }
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: Authorization
```

---

# 风险与权衡

- **大文件/断点续传**：MVP 先不做，后续引入 tus 或 S3 分片直传；需要会话表与合并逻辑。
- **病毒扫描**：MVP 不做，后续可异步扫描 + 状态标记。
- **成本**：S3 与日志/监控流量需控制；可用生命周期/清理任务（如 30 天未访问自动归档/删除）。
- **并发/性能**：采用流式存储、避免一次性读入内存；Nginx 反代优化长连接；限制并发与连接池。

---

# 骨架代码（任务优先顺序）

-

---

# 下一步（请你确认）

1. 是否采用 **chi + sqlc + MinIO/S3** 的组合？若偏好 gin 或 GORM 可切换。
2. 是否接受 **API Key** 作为 MVP 鉴权？
3. 是否需要在 MVP 加入 **预签名下载**（更贴近生产）？
4. 是否希望我直接生成：后端最小可运行骨架 + 前端上传页（可 Docker Compose 一键起）？

