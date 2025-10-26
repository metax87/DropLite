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
