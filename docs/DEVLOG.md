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
