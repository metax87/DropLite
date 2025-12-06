# 开发日志

## 2024-03-24
- 初始化仓库骨架：创建 `backend/`、`frontend/`、`infra/`、`docs/` 目录，并补充顶层 `.gitignore` 与 `Makefile`。
- 后端新增 `cmd/server/main.go`，提供基础健康检查与优雅退出逻辑，为后续路由和依赖注入留出空间。
- 前端搭建 Vite + React + TypeScript 最小可运行结构，暂时实现文件选择占位 UI 以验证构建链路。
- 约定 `Makefile` 中的通用命令（bootstrap/dev/test/lint），后续完善时与 CI 流程保持一致。
- 调整后端结构：新增 `internal/` 分层目录与说明文档，接入 chi 路由与基础中间件；实现配置加载（端口、存储目录）与日志初始化，`main.go` 统一使用 router + graceful shutdown 架构。
- 明日计划：补充自定义中间件与 CORS 支持，完善配置项（数据库、对象存储），并草拟文件上传/列表接口的领域模型。

## 2024-03-25
- 扩展配置模块：新增 CORS 白名单、全局限流窗口/阈值等字段，统一通过环境变量注入并提供 `.env.example` 示例便于本地覆盖。
- 新增 `internal/middleware` 目录，提供自定义 CORS 与 IP 级限流实现，在 chi Router 中默认启用以满足跨域和安全要求。
- main 入口向路由传递配置，确保中间件能读取到用户自定义的白名单与速率限制参数。
- 配置与基础设施继续演进：补充 PostgreSQL 连接字段及 `PostgresDSN` 帮助方法，`.env.example` 与 `infra/docker-compose.yml` 提供本地数据库默认值与启动方式。
- 初步确定文件元数据 schema 与 Go 模型，补充 repository 接口及 Postgres 实现，并添加 `backend/db/migrations/0001_*` 迁移脚本，开始沉淀持久层基础。
- 新增 `internal/database` 负责建立 pgx 连接，`cmd/server` 注入 `FileRepository` + `FileService` 并暴露 `/files` 列表/登记接口，完成 HTTP → Service → Repository 的最小闭环。
- 内置迁移执行器：`internal/migrations` 读取 embed SQL，`cmd/migrate` + `make migrate` 提供本地 CLI，服务启动时也会自动执行待应用的 schema 变更。
- 新增存储抽象：`internal/storage` 定义 writer 接口，首个 `storage/local` 实现将文件写入 `STORAGE_DIR`；`FileService` 现会落盘文件后登记元数据，上传 API 正式具备最小可用流程。
- 为 `/files` 相关逻辑补充单元测试：`internal/service/files_test.go` 验证落盘与校验分支，`internal/api/files_handler_test.go` 覆盖创建与列表响应，确保基础链路可回归。

## 2024-03-26
- `/files` 上传端点改为 `multipart/form-data`，后端负责生成存储路径、检测 MIME、校验 100MB 大小限制，并支持表单字段传递 metadata/checksum/有效期。
- `FileService` 现在自动生成安全的存储路径，成功写入对象后会落库为 `stored` 状态，同时扩充单元测试覆盖路径生成与状态转移。
- handler 测试替换为真实 multipart 构造，移除旧的 base64 JSON 请求；DEV 验证步骤更新为新的调用方式。
- 前端基于 React Query 对接 `/files` 列表，并使用 `XMLHttpRequest + FormData` 实现实时进度的多文件上传体验，上传完成后自动刷新列表。

## 2025-12-06
- 新增文件下载 API：`GET /files/{id}/download` 端点，支持流式文件下载。
- 扩展存储抽象层：`internal/storage/storage.go` 新增 `Reader` 接口与 `Storage` 组合接口，`local.Writer` 实现 `Read` 方法打开本地文件。
- `FileService` 新增 `GetFile`（获取元数据）与 `GetFileContent`（获取文件内容流）方法。
- 下载 API 校验文件状态，仅 `stored` 状态的文件可下载；设置正确的 `Content-Type`、`Content-Disposition` 与 `Content-Length` 响应头。
- 更新测试 mock 以适配新接口，所有单元测试通过。
- 新增文件删除 API：`DELETE /files/{id}` 端点，实现软删除（将状态更新为 `deleted`）。
- 新增单个文件元数据 API：`GET /files/{id}` 端点，返回指定文件的详细信息。
- 更新 `FileRepository.List` 方法，默认排除 `deleted` 状态的文件。
- 前端新增下载和删除按钮：下载按钮打开新窗口下载文件，删除按钮触发确认对话框后执行软删除并刷新列表。
- 新增 API Key 鉴权功能：
  - 配置新增 `AUTH_ENABLED` 和 `API_KEYS` 环境变量，默认启用鉴权，开发环境默认 key 为 `dev-api-key-123456`。
  - 新增 `internal/middleware/auth.go` 实现 API Key 验证中间件，支持 `Authorization: ApiKey <token>` 格式。
  - 路由更新：`/healthz` 不需要鉴权，`/files` 相关端点需要有效 API Key。
  - 前端更新：所有 API 请求自动附带 `Authorization` 头，API Key 通过 `VITE_API_KEY` 环境变量配置。
- 新增 S3/MinIO 存储支持：
  - 添加 MinIO Go SDK 依赖（`github.com/minio/minio-go/v7`）。
  - 配置新增 `STORAGE_DRIVER`（local/s3）、`S3_ENDPOINT`、`S3_ACCESS_KEY`、`S3_SECRET_KEY`、`S3_BUCKET` 等环境变量。
  - 新增 `internal/storage/s3/s3.go` 实现 S3 兼容存储，支持自动创建 bucket。
  - `main.go` 根据 `STORAGE_DRIVER` 配置动态选择本地或 S3 存储后端。
  - `infra/docker-compose.yml` 新增 MinIO 服务，端口 9000（S3 API）和 9001（控制台）。
- 新增可观测性与 CI/CD 功能：
  - 添加 Prometheus 客户端依赖，新增 `internal/middleware/metrics.go` 收集 HTTP 请求指标（总数、耗时、大小）。
  - 新增 `/metrics` 端点暴露 Prometheus 指标。
  - 创建 `.github/workflows/ci.yml` GitHub Actions 工作流：后端 lint/test/build，前端 lint/build，Docker 镜像构建。
  - 新增 `backend/Dockerfile` 多阶段构建，使用 alpine 最小镜像，包含健康检查和非 root 用户。
- 集成 Supabase Auth：
  - 后端配置新增 `AUTH_PROVIDER`, `SUPABASE_URL`, `SUPABASE_JWT_SECRET` 等字段。
  - 新增 `middleware.SupabaseAuth` 验证 JWT 签发的 Bearer Token，并提取 Sub 作为 `owner_id`。
  - 更新 frontend 依赖：`@supabase/supabase-js`, `@supabase/auth-ui-react`。
  - 前端新增 `AuthProvider` 上下文管理 Session。
  - 新增 `LoginPage` 使用 Supabase Auth UI 组件。
  - `App.tsx` 集成登录状态检查，并自动为 API 请求附加 Bearer Token。
