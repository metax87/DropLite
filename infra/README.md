# 基础设施脚本

此目录用于存放 docker-compose、Terraform 等部署相关文件。目前提供了最小化的 PostgreSQL 编排，方便本地联调：

```bash
cd infra
docker compose up -d postgres
```

默认会启动 `postgres:15`，用户名/密码/数据库均为 `droplite`，可通过 `.env.example` 中的 `DB_*` 配置覆盖。数据持久化在仓库根目录的 `tmp/postgres-data`，删除目录即可重置数据。
