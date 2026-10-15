# 项目结构

```text
SnakeRule/
├── backend/server/       # Go API、规则求值器与 OpenAPI 契约
├── frontend/admin/       # React + Vite 管理工作台
├── frontend/docs/        # VitePress 文档站点
├── deploy/nginx/         # 开发环境统一入口配置
├── docs/                 # 架构说明与技术调研
├── scripts/              # SDK 与版本维护脚本
├── docker-compose.dev.yml
└── Makefile
```

API 路由、业务服务、规则领域对象和求值器位于 `backend/server/internal`。管理端通过 OpenAPI 生成的 TypeScript SDK 调用 API。开发环境由 Compose 启动，Nginx 将 `/api/` 转发到后端，将管理端和 `/docs/` 文档站点统一到 `localhost:8080`。

阅读[架构说明](https://github.com/bulolo/SnakeRule/blob/main/docs/architecture.md)了解当前边界与后续演进方向。
