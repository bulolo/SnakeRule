# 项目结构

```text
SnakeRule/
├── backend/
│   ├── server/           # Go API、规则求值器与 OpenAPI 契约
│   └── pulse-agent/      # 后端镜像打包的 Pulse Agent 及校验和
├── frontend/
│   ├── admin/            # React + Vite 管理工作台与生成 SDK
│   └── docs/             # VitePress 文档站点
├── deploy/               # 生产 Compose、Helm Chart 和 Nginx 配置
│   ├── docker-compose.prod.yml
│   └── nginx/
├── docs/                 # 当前架构、路线图和技术选型记录
├── scripts/              # 环境初始化、SDK 和版本维护脚本
├── docker-compose.dev.yml
└── Makefile
```

API 路由、业务服务、规则领域对象和求值器位于 `backend/server/internal`。管理端通过 OpenAPI 生成的 TypeScript SDK 调用 API。开发和生产 Compose 均由 Nginx 统一入口：`/api/v1/` 转发到后端，`/docs/` 转发到文档站点，根路径转发到管理端。Kubernetes Chart 位于 `deploy/helm`，仅部署 API、管理端和 Nginx，不创建独立 Gateway 服务。

阅读仓库中的[架构说明](https://github.com/bulolo/SnakeRule/blob/0.0.2/docs/architecture.md)了解当前模块边界、数据生命周期和后续演进方向。
