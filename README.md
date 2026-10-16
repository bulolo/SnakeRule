<div align="center">

# 🐍 SnakeRule

**规则与 AI 协同决策平台**

用可视化规则管理承载清晰、可验证的业务判断，并为 AI 决策能力提供协同扩展空间。

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-149ECA?logo=react)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite)](https://vite.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql)](https://www.postgresql.org/)
[![Docker Compose](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)
[![License](https://img.shields.io/badge/License-SnakeRule%20Community-orange)](./LICENSE)

[GitHub](https://github.com/bulolo/SnakeRule) · [文档中心](./frontend/docs/docs/index.md) · [快速开始](./frontend/docs/docs/guide/quick-start.md)

</div>

---

## 🚀 当前版本

### v0.0.2 · 规则管理与决策求值基础版

- 🧩 **可视化规则集管理**：通过拖拽画布或 JSON 编辑规则，查看并调整规则条件和动作。
- 🧪 **决策试运行**：使用样例 facts 检查已保存规则集，也可以直接预览尚未保存的草稿。
- 🏢 **组织与项目**：按组织隔离项目，规则集归属具体项目；可从电商订单风控演示项目开始体验。
- 🔐 **项目鉴权**：支持 API Key、平台签发 JWT 和项目自签 HS256 JWT。
- 🔔 **通知与调用记录**：按规则集配置命中通知，查看调用来源、请求与响应详情。
- 🛠️ **开发和部署工具**：提供 Make 命令、Docker Compose、Helm Chart、OpenAPI 契约和 TypeScript SDK。

> 产品定位包含规则与 AI 协同决策；当前版本提供可视化规则管理和确定性规则求值，尚未在求值流程中调用 AI 模型服务。AI 集成边界见[预研说明](./docs/research/jev-integration.md)。

---

## 💡 使用场景

| 场景 | 说明 |
| --- | --- |
| 🛡️ 订单风控 | 根据订单金额、客户属性和风险标签返回审核策略。 |
| 🎟️ 营销决策 | 按用户等级、订单信息等条件匹配促销规则。 |
| ✅ 资格校验 | 将业务资格条件集中维护，并在接入应用前使用样例验证。 |
| 🧮 策略试算 | 调整条件与优先级，比较输入事实对应的决策结果。 |

这些场景说明规则能力的使用方式。当前规则动作随结果返回，不会自动调用外部服务或执行业务副作用。

---

## 🎯 项目亮点

- **完整开发环境**：`make dev-up` 启动 PostgreSQL、Go API、管理端、文档站和 Nginx，可从 `http://localhost:8080` 访问。
- **画布与 JSON 双模式**：基础条件可视化编辑，复杂规则可在 JSON 视图维护。
- **确定性求值**：支持嵌套 `all`、`any`、`not` 条件、规则优先级和多条规则命中。
- **草稿预览**：无需覆盖已保存规则，即可试运行未保存的规则集定义。
- **项目级调用凭据**：API Key 和 JWT 绑定具体项目，可撤销并用于调用规则求值 API。
- **OpenAPI 驱动客户端**：管理端 SDK 根据 `backend/server/docs/openapi.json` 生成，可用 `make gen-sdk` 更新。
- **PostgreSQL 持久化**：业务数据保存在数据库；当前 API 仓储仍使用进程内缓存，因此部署为单 API 副本。

---

## ✨ 核心功能

### 🎛️ 管理工作台

- 概览展示近期求值调用、命中和失败情况，并查看全局调用日志。
- 在项目中管理规则集、项目成员、鉴权凭据与命中通知策略。
- 平台设置管理系统信息、授权、组织和品牌；组织设置管理组织用户和告警渠道。
- 管理端登录使用 HttpOnly 会话 Cookie；角色包括超级管理员、组织管理员、项目管理员和项目成员。

### ⚙️ 规则引擎

- 规则集包含名称、版本、启用状态和规则列表；规则包含优先级、条件树和声明动作。
- 条件支持 `all`（全部满足）、`any`（至少一个满足）与 `not`（取反）递归组合。
- 使用 `order.amount` 等点分路径读取 JSON facts 中的嵌套字段。
- 操作符包含相等、不等、数值比较、`in`、`notIn`、`contains`、`isNull` 和 `isNotNull`。
- 规则按优先级由高到低求值，返回所有匹配的启用规则及其声明动作。
- 动作是求值结果中的数据；当前不会执行 Webhook、变量写入或其他外部副作用。

### 🔌 HTTP API

- 管理 API 提供登录、组织、项目、成员、规则集、通知、调用日志和授权等接口。
- 规则求值 API 支持管理会话、项目 API Key（`X-API-Key`）和项目 JWT（Bearer）。
- 求值请求可引用已保存规则集；管理会话也可提交规则草稿进行预览。
- 响应统一使用 `{ "code": 0, "msg": "success", "data": ... }`；分页列表使用 `list` 与 `pagination`，并支持 `is_pager=0`。
- 完整路径、字段、响应模型和鉴权要求以 [OpenAPI 契约](./backend/server/docs/openapi.json) 为准。

---

## 🔎 规则示例

当订单金额大于 10,000 且客户为新客时，命中“大额新客订单”规则：

```json
{
  "id": "order-risk",
  "name": "订单风险规则",
  "version": "1.0.0",
  "enabled": true,
  "rules": [
    {
      "id": "large-new-order",
      "name": "大额新客订单",
      "priority": 100,
      "enabled": true,
      "conditions": {
        "all": [
          { "fact": "order.amount", "operator": "greaterThan", "value": 10000 },
          { "fact": "customer.isNew", "operator": "equal", "value": true }
        ]
      },
      "actions": [{ "type": "tag", "config": { "label": "MANUAL_REVIEW" } }]
    }
  ]
}
```

`tag` 在当前版本只是结果数据，不会自动触发外部操作。完整试运行步骤见[决策试运行指南](./frontend/docs/docs/engine/evaluation.md)。

---

## 🏗️ 技术架构

```text
浏览器
  │
  ▼
Nginx :8080 ── /       ──► React + Vite 管理端
            ├─ /docs/  ──► VitePress 文档站
            └─ /api/v1/──► Go HTTP API :8081
                               │
                         服务层与规则求值器
                               │
                               ▼
                         PostgreSQL 数据库
```

| 模块 | 技术与职责 |
| --- | --- |
| API 服务 | Go 1.24+、标准库 `net/http`，负责认证、管理 API 和规则求值。 |
| 规则引擎 | Go 领域模型与 JSON 条件 DSL，提供条件校验和确定性求值。 |
| 数据存储 | PostgreSQL；Goose 在服务启动时应用数据库迁移。 |
| 管理端 | React 18、TypeScript、Vite，通过生成的 SDK 调用 API。 |
| 文档站 | VitePress，开发环境由 Nginx 在 `/docs/` 提供访问。 |
| 部署 | Docker Compose 用于开发和生产；Helm Chart 用于 Kubernetes。 |

当前仓储在进程内保留数据缓存，API 需运行单副本。当前部署不包含独立 Gateway 服务；Nginx 仅负责统一入口和反向代理。

---

## 📁 项目结构

```text
SnakeRule/
├── backend/
│   ├── server/
│   │   ├── cmd/main/                 # API 服务入口
│   │   ├── internal/                 # 领域、求值、服务、仓储和 HTTP API
│   │   └── docs/openapi.json         # API 契约
│   └── pulse-agent/                  # 随后端镜像打包的授权代理
├── frontend/
│   ├── admin/                        # React 管理工作台和生成 SDK
│   └── docs/                         # VitePress 文档站
├── deploy/
│   ├── docker-compose.prod.yml       # 生产 Compose
│   ├── helm/                         # Kubernetes Chart
│   └── nginx/                        # 反向代理配置
├── docs/                             # 架构、路线图和技术预研
├── scripts/                          # 环境、版本和 SDK 脚本
├── docker-compose.dev.yml
├── Makefile
└── VERSION
```

---

## 🚀 安装与配置

### 环境要求

- Docker Desktop 或 Docker Engine，包含 Docker Compose V2
- Make
- Git
- Go 1.24+（仅本机直接构建或运行后端时需要）
- Node.js 22+（仅单独构建管理端或文档站时需要）

### 启动开发环境

```bash
git clone https://github.com/bulolo/SnakeRule.git
cd SnakeRule
make dev-init
make dev-up
```

首次启动会创建 `backend/.env`，使用其中的 `AUTH_EMAIL` 和 `AUTH_PASSWORD` 登录。示例密码只适合本机试用；共享环境请先更换。

| 服务 | 地址 |
| --- | --- |
| 管理工作台 | [http://localhost:8080](http://localhost:8080) |
| 文档站 | [http://localhost:8080/docs/](http://localhost:8080/docs/) |
| Go API | [http://localhost:8081](http://localhost:8081) |
| API 健康检查 | [http://localhost:8080/healthz](http://localhost:8080/healthz) |

`make dev-up` 在前台运行并显示服务日志；按 `Ctrl+C` 停止，或另开终端执行 `make dev-down`。开发数据保存在 PostgreSQL 卷中。

---

## 🧰 项目管理（Makefile）

项目根目录的 Makefile 提供常用开发、部署和发布命令：

| 命令 | 用途 |
| --- | --- |
| `make dev-init` | 创建本地开发配置。 |
| `make dev-up` | 构建并以前台方式启动开发环境。 |
| `make dev-rebuild` | 重建并后台启动开发服务。 |
| `make dev-status` / `make dev-logs` | 查看服务状态或跟踪日志。 |
| `make dev-down` | 停止开发服务，保留数据卷。 |
| `make dev-clean` | 删除开发容器、镜像和数据卷，清空开发数据。 |
| `make prod-init` / `make prod-up` | 初始化并启动生产 Compose。 |
| `make helm-lint` / `make helm-install` | 检查或安装 Kubernetes Chart。 |
| `make check-backend` | 检查 Go 格式、运行 `go vet` 并编译。 |
| `make gen-sdk` | 从 OpenAPI 契约生成管理端 TypeScript SDK。 |
| `make set-version v=0.1.0` | 更新项目版本号。 |
| `make docs-build` | 构建 VitePress 文档站。 |
| `make publish-codeup` / `make publish-github` | 推送当前分支到对应远端。 |
| `make publish-all` | 推送当前分支到 Codeup 与 GitHub。 |

完整命令见 `make help`。发布命令支持 `DRY_RUN=1` 预览推送操作。

---

## ❓ 常见问题

**Q：开发环境的登录信息在哪里？**

A：查看 `backend/.env` 中的 `AUTH_EMAIL` 与 `AUTH_PASSWORD`。该文件由 `make dev-init` 创建并被 Git 忽略。

**Q：重启服务后项目和规则集还在吗？**

A：会保留在 PostgreSQL 数据卷中。运行 `make dev-clean` 会删除数据卷并清空开发数据。

**Q：规则中的动作会直接执行吗？**

A：不会。动作目前作为求值结果返回；Webhook 等外部副作用执行器尚未实现。

**Q：求值 API 支持哪些鉴权方式？**

A：支持项目 API Key 和 JWT。JWT 可由平台签发，也可使用项目 HS256 密钥自行签发；凭据可撤销。API Key 轮换和最后使用时间尚未实现。

**Q：可以部署多个 API 副本吗？**

A：当前不支持。仓储使用进程内缓存，请按单 API 副本部署；数据库备份和公网 TLS 需根据部署环境配置。

---

## 📚 相关文档

- [文档中心首页](./frontend/docs/docs/index.md)
- [快速开始](./frontend/docs/docs/guide/quick-start.md)
- [管理工作台](./frontend/docs/docs/guide/admin-console.md)
- [规则集与条件 DSL](./frontend/docs/docs/engine/rulesets.md)
- [决策试运行](./frontend/docs/docs/engine/evaluation.md)
- [API 概览](./frontend/docs/docs/api/overview.md)
- [项目结构](./frontend/docs/docs/guide/project-structure.md)
- [系统架构](./docs/architecture.md)
- [数据库备份与恢复](./docs/operations/database-backup-restore.md)
- [开发路线图](./docs/todo.md)
- [AI 决策扩展预研](./docs/research/jev-integration.md)

---

## 📄 许可证与商业使用

SnakeRule 使用基于 Apache License 2.0 并附加 SnakeRule 条款的[社区许可证](./LICENSE)：

- **个人与内部使用**：可按许可证用于个人用途、企业内部部署和单租户业务集成。
- **品牌**：使用和分发时须保留 SnakeRule 品牌标识。
- **商业 SaaS**：未经书面授权，不得基于本项目提供商业多租户规则引擎服务。

商业授权请通过 [GitHub Issues](https://github.com/bulolo/SnakeRule/issues) 联系维护者。完整条款见 [LICENSE](./LICENSE)。

---

## 🌱 生态项目

- [OwlApi](https://github.com/bulolo/owlapi)：SQL to API 智能网关平台。
- [RayFlow](https://github.com/bulolo/RayFlow)：流式湖仓控制与数据工作流平台。
- [CatWiki](https://github.com/bulolo/CatWiki)：AI 知识库与智能问答平台。

---

<div align="center">

Made with ❤️ by the **szplanner** team

</div>
