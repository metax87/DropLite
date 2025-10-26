# Repository Guidelines

## 项目结构与模块组织
项目在 `backend/` 与 `frontend/` 分治：前者包含 Go API、数据库迁移在 `backend/db/migrations/`，后者集中在 `frontend/src/`；基础设施脚本与 Compose 文件位于 `infra/`，设计文档与运营说明放入 `docs/`。开发样例数据置于 `seed/`，临时产物请写入 `tmp/` 或忽略于 `.gitignore`。

## 构建、测试与开发命令
首次克隆后执行 `make bootstrap` 拉取 Go、Node 依赖；本地开发可运行 `make dev` 同时启动后端与前端代理。仅验证后端时使用 `go run ./cmd/server`，前端调试则执行 `pnpm dev --filter frontend`。发布前统一运行 `make build` 和 `make e2e` 确认产物可部署。

## 代码风格与命名约定
Go 文件保存前需通过 `gofmt`、`gofumpt`，并确保 `golangci-lint run` 通过；错误处理务必显式返回 `error`。TypeScript 使用 ESLint `recommended` + `react` 规则，格式化依赖 Prettier 默认配置。命名约定：Go 导出符号采用 UpperCamelCase，JSON 字段与 SQL 列维持 snake_case，前端组件与路由文件使用 kebab-case。

## 测试规范
后端单元测试放置于与源码同目录的 `*_test.go`，数据库相关测试请通过 `testcontainers` 启动临时 PostgreSQL；前端组件测试使用 Vitest + React Testing Library，并以 `ComponentName.spec.tsx` 命名。提交前运行 `make test` 与 `make coverage`，目标语句覆盖率保持 80% 以上。

## 提交与合并请求指南
提交信息遵循 Conventional Commits，例如 `feat: support resumable upload`，确保一次提交只覆盖单一议题。Pull Request 描述需列出变更摘要、测试结果、潜在风险，并通过 `Closes #issue` 关联任务；界面改动请附前后对比截图。合并前确认 CI 成功且无待处理评论。

## 安全与配置提示
环境变量模板记录在 `.env.example`，新增配置请同步文档并使用 `infra/secrets/` 管理敏感凭据。上线状态必须启用速率限制、CORS 白名单与结构化审计日志，不得随意关闭；频繁上传任务建议使用固定 API Key 并定期轮换。
