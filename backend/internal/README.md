# internal 目录结构

- `api/`：HTTP 层的 handler、请求响应 DTO、校验逻辑。
- `service/`：领域服务，封装业务用例（如上传、删除、签名 URL）。
- `repository/`：数据访问层，负责数据库操作与缓存包装。
- `storage/`：对象存储适配层，屏蔽本地、S3、MinIO 等实现差异。
- `config/`：配置加载与环境变量解析工具。
- `middleware/`：HTTP 中间件（鉴权、日志、限流、CORS）。
- `logging/`：日志初始化与通用日志工具。

各目录内会按照需要进一步细分子包，遵循按功能聚合的原则，避免跨层依赖。
