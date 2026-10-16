<div align="center">

# 🐍 SnakeRule

**可视化规则管理与确定性决策引擎**

通过规则集描述业务判断，在管理工作台中编辑、验证并试运行决策。

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-149ECA?logo=react)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite)](https://vite.dev/)
[![Docker Compose](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)
[![License](https://img.shields.io/badge/License-SnakeRule%20Community-orange)](./LICENSE)

简体中文

[GitHub](https://github.com/bulolo/SnakeRule) | [文档中心](./frontend/docs/docs/index.md) | [快速开始](./frontend/docs/docs/guide/quick-start.md)

</div>

---

## 🚀 最近更新

### v0.1.0 · 管理工作台与规则引擎基础版

- 🧭 **工作台概览**：查看当前工作空间与项目概况，并从项目入口继续管理规则。
- 🧩 **规则集管理**：创建、编辑、筛选和删除规则集；支持拖拽式规则画布与 JSON 编辑。
- 🧪 **决策试运行**：使用输入 facts 预览已保存规则或编辑中的草稿，草稿预览不会写入规则存储。
- 🗂️ **工作空间**：支持切换，以及创建、编辑和删除非默认、非当前空间。
- 🧱 **项目管理**：工作空间内可创建项目；电商演示项目内置订单风控规则，可在项目内独立管理规则集。
- 🔑 **项目 API Key**：为项目签发和撤销服务端求值密钥；密钥创建时显示一次，后端只保存 SHA-256 摘要。
- 🔔 **项目告警策略**：在项目内选择全局告警渠道和触发规则；已保存规则命中时发送通知。
- 🔐 **管理登录**：超级管理员邮箱密码登录，使用 HttpOnly 会话 Cookie，并支持退出和会话过期。
- ⚙️ **平台与系统设置**：维护工作空间、授权信息和全局告警渠道；支持企业微信、钉钉、飞书机器人与自定义 Webhook，并可发送测试通知。
- 📦 **开发与生产 Compose**：Nginx 统一入口、管理端、文档站点和 Go API 可通过 Make 命令启动。

---

## 💡 使用场景

| 场景 | 描述 |
|------|------|
| 🛡️ **风控预判** | 根据订单金额、客户属性等 facts 组合条件，返回命中的审核规则。 |
| 🎟️ **营销规则** | 按用户等级、订单信息等条件匹配促销规则，并将声明动作包含在决策结果中。 |
| ✅ **资格校验** | 将业务资格条件集中在规则集中管理，并在发布到应用前试运行样例数据。 |
| 🧮 **策略试算** | 调整条件和优先级，通过画布或 JSON 编辑并即时检查决策结果。 |

这些场景说明规则引擎的使用方式，不代表 SnakeRule 已内置业务系统集成或外部动作执行器。

---

## 🎯 项目亮点

- ✅ **一条命令启动开发环境**：`make dev-up` 启动 API、管理端、文档站点和 Nginx，统一从 `http://localhost:8080` 访问。
- ✅ **画布和 JSON 双模式**：日常规则可通过画布调整，也可在 JSON 模式检查完整规则定义。
- ✅ **确定性求值**：支持嵌套 `all`、`any`、`not` 条件、优先级排序和多条规则命中结果。
- ✅ **编辑草稿可预览**：试运行请求可携带尚未保存的规则集，不必先覆盖已保存版本。
- ✅ **API 契约和 SDK 同步**：OpenAPI 契约位于 `backend/server/docs/openapi.json`，可用 `make gen-sdk` 更新管理端 TypeScript 客户端。
- ✅ **当前能力边界清晰**：规则集、工作空间和会话使用进程内存；动作仅作为结果返回，不会触发外部副作用。

---

## ✨ 核心特性

### 🎛️ 管理工作台

- **概览**：展示当前工作空间与项目概况，并提供进入项目列表的快捷入口。
- **项目内规则管理**：项目详情展示具体规则和规则集，并可进入完整列表按名称或 ID 搜索、按启用状态筛选。
- **可视化编辑**：在画布中维护规则、条件和声明动作；支持规则排序以及 JSON 高级编辑。
- **试运行**：输入 facts 查看命中状态、命中规则和声明动作；可以评估编辑草稿。
- **工作空间切换**：从顶栏切换当前会话的空间；平台设置提供工作空间增删改查。
- **项目**：从项目卡片进入项目详情并查看具体规则；在项目页创建空项目或电商演示项目。鉴权和告警通过详情右上角入口弹窗配置，工作空间切换保留在顶栏。
- **平台设置**：查看产品版本和运行信息、授权状态、当前工作空间及只读配置。
- **系统设置**：管理企业微信、钉钉、飞书机器人和自定义 Webhook 渠道，支持添加、编辑、启停、删除与测试通知；同时查看通用运行和登录安全信息。
- **账号菜单**：查看当前管理员并从右上角退出登录。

### ⚙️ 规则引擎

- **规则集**：包含 ID、名称、版本、说明、启用状态和有序规则列表。
- **规则**：包含 ID、名称、优先级、启用状态、条件树和声明动作。
- **条件组合**：支持 `all`（全部满足）、`any`（至少一个满足）与 `not`（取反）递归组合。
- **事实路径**：通过 `order.amount` 等点分路径读取传入 JSON facts 中的嵌套字段。
- **操作符**：支持相等、不等、数值比较、`in`、`notIn`、`contains`、`isNull` 和 `isNotNull`；具体名称见下方 API 文档与 OpenAPI 契约。
- **优先级与命中**：规则按优先级由高到低评估，返回所有匹配且启用的规则。
- **动作**：`tag`、`log` 等动作当前作为声明数据随命中结果返回，不执行 Webhook、写变量或其他外部操作。

### 🔌 HTTP API

- 管理员登录、会话查询与退出。
- 管理接口使用会话 Cookie；规则求值接口支持项目 API Key（`X-API-Key`）。尚未提供 JWT 或 OAuth/OIDC。
- 项目和工作空间增删改查、项目 API Key 签发/撤销、项目告警策略配置。
- 工作空间列表、创建、更新、删除和当前空间切换。
- 规则集的创建、读取、更新和删除。
- 对已保存规则集或请求中的草稿执行同步求值。
- Pulse Agent 授权状态查询、授权文件上传与清除。
- `/healthz`、`/readyz` 和 `/api/v1/meta` 健康及版本信息。

完整请求字段和响应模型以 [OpenAPI 契约](./backend/server/docs/openapi.json) 为准。

---

## 🔌 规则示例

以下条件在订单金额大于 10,000 且客户是新客户时命中：

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

开发环境先从 `backend/.env` 取得管理员邮箱和密码，再登录并试运行默认演示规则：

```bash
curl -c /tmp/snakerule.cookies http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"<AUTH_EMAIL>","password":"<AUTH_PASSWORD>"}'

curl -b /tmp/snakerule.cookies http://localhost:8080/api/v1/engine/evaluate \
  -H 'Content-Type: application/json' \
  -d '{"rulesetId":"order-risk","facts":{"order":{"amount":12000},"customer":{"isNew":true}}}'
```

试运行结果返回 `matched`、`hitRules` 和规则中声明的 `actions`。这些动作不会被服务执行；请求也不会创建决策历史记录。交互式流程见[决策试运行文档](./frontend/docs/docs/engine/evaluation.md)。

---

## 📸 应用截图

仓库目前未包含管理台截图；可按[快速开始](./frontend/docs/docs/guide/quick-start.md)启动本地环境查看。

---

## 🏗️ 技术架构

```text
浏览器
  │
  ▼
Nginx :8080 ── /docs/ ──► VitePress 文档站点
  │          └─ /api/  ──► Go API :8081
  └─ /      ────────────► React + Vite 管理端
                              │
                              ├─ 登录与工作空间 API
                              ├─ 规则集 CRUD API
                              └─ 规则求值 API
                                     │
                                     ▼
                         服务层 + 条件求值器
                                     │
                                     ▼
                         进程内存规则/空间仓储
```

### 后端技术栈

- **语言**：Go 1.24+
- **HTTP 服务**：Go 标准库 `net/http` 和 `ServeMux`
- **规则模型**：Go 领域类型与 JSON 条件 DSL
- **服务组织**：bootstrap、HTTP transport、service、repository、domain 分层
- **当前仓储**：并发安全的进程内存实现；API 重启后规则集、工作空间和会话会重置。
- **平台授权**：通过 Pulse Agent 校验授权；授权密文单独保存在服务端配置的数据路径。

### 前端与文档

- **管理端**：React 18、TypeScript、Vite
- **API 客户端**：从仓库 OpenAPI 契约生成的 TypeScript SDK
- **文档站点**：VitePress
- **页面入口**：概览、项目、平台设置、系统设置；规则集从具体项目内进入。具体功能边界见[工作台指南](./frontend/docs/docs/guide/admin-console.md)。

### 部署

- **开发环境**：Docker Compose 启动 API、管理端、文档站点和 Nginx；前端源码挂载用于热更新。
- **生产环境**：独立构建 API、管理端、文档站点镜像，Nginx 作为对外入口。
- **可选基础设施**：Compose 提供 PostgreSQL 和 Redis 开发容器，但当前 API 不使用它们作为规则存储或缓存。

---

## 📁 项目结构

```text
SnakeRule/
├── backend/
│   ├── server/
│   │   ├── cmd/main/                 # API 服务入口
│   │   ├── internal/
│   │   │   ├── bootstrap/            # 依赖初始化和默认规则
│   │   │   ├── config/               # 环境配置
│   │   │   ├── domain/               # 项目、告警、规则、条件和决策模型
│   │   │   ├── engine/evaluator/     # 条件校验和求值
│   │   │   ├── licensing/            # Pulse 授权管理
│   │   │   ├── repo/                 # 仓储接口与内存实现
│   │   │   ├── service/              # 规则、工作空间业务逻辑
│   │   │   └── transport/http/       # REST API 与内置轻量工作台资源
│   │   └── docs/openapi.json         # API 契约
│   └── pulse-agent/                  # 随后端镜像打包的 Pulse Agent
├── frontend/
│   ├── admin/                        # React 管理工作台与生成 SDK
│   └── docs/                         # VitePress 文档站点
├── deploy/                            # 生产 Compose、环境示例和 Nginx 配置
│   ├── docker-compose.prod.yml
│   └── nginx/
├── docs/                             # 架构、路线图和技术选型记录
├── scripts/                          # 环境初始化、SDK 和版本脚本
├── docker-compose.dev.yml
├── Makefile
└── VERSION
```

---

## 🚀 安装与配置

### 前置要求

- Docker Desktop 或 Docker Engine，包含 Docker Compose V2
- Make
- Git
- Go 1.24+（仅在本机直接编译或运行后端时需要）
- Node.js 22+（仅在本机单独构建管理端或文档时需要）

### 配置说明

开发环境配置由 `make dev-init` 从 `backend/.env.example` 创建到 Git 忽略的 `backend/.env`。首次启动后，使用其中的 `AUTH_EMAIL` 和 `AUTH_PASSWORD` 登录；共享或部署环境请设置自己的强密码。

授权上传需要 Pulse 授权签发方提供的产品公钥。授权未配置时，社区规则管理和求值仍可运行；当前没有规则功能点依赖商业授权状态。

---

## ⚡ 快速开始

```bash
git clone https://github.com/bulolo/SnakeRule.git
cd SnakeRule
make dev-init
make dev-up
```

服务启动后：

| 服务 | 地址 | 说明 |
|------|------|------|
| 管理工作台 | [http://localhost:8080](http://localhost:8080) | Nginx 统一入口，登录后管理规则 |
| 文档站点 | [http://localhost:8080/docs/](http://localhost:8080/docs/) | 产品和 API 使用文档 |
| Go API | [http://localhost:8081](http://localhost:8081) | 开发环境直连地址 |
| API 健康检查 | [http://localhost:8080/healthz](http://localhost:8080/healthz) | 经 Nginx 检查 API 服务 |

`make dev-up` 前台运行 Compose 并输出日志；按 `Ctrl+C` 停止，或在另一终端执行 `make dev-down`。更多入口见[快速开始指南](./frontend/docs/docs/guide/quick-start.md)。

---

## 🏗️ 项目管理（Makefile）

| 命令 | 说明 |
|------|------|
| `make dev-init` | 从示例生成本地开发环境配置，并补齐本地授权加密密钥。 |
| `make dev-up` | 构建并以前台方式启动开发 Compose。 |
| `make dev-rebuild` | 重建并后台启动开发服务。 |
| `make dev-status` | 查看容器状态及管理端、文档和 API 健康状态。 |
| `make dev-logs` | 跟踪开发服务日志；也可用 `make dev-logs-backend` 等按服务查看。 |
| `make dev-down` | 停止并移除开发容器；不删除命名数据卷。 |
| `make prod-init` | 生成本地生产配置和随机管理员密码、授权加密密钥。 |
| `make prod-build` | 构建生产镜像。 |
| `make prod-up` / `make prod-down` | 启动或停止生产 Compose。 |
| `make check-backend` | 检查 Go 格式、运行 `go vet` 并编译后端。 |
| `make gen-sdk` | 从 OpenAPI 契约生成管理端 SDK。 |
| `make set-version v=0.1.1` | 更新项目版本号及关联的版本元数据。 |
| `make docs-build` | 构建 VitePress 文档。 |
| `make publish-codeup` | 推送当前分支到配置的 Codeup `origin` 远端。 |
| `make publish-github` | 推送当前分支到 README 中的 GitHub 仓库地址。 |

使用 `make help` 查看完整命令列表。可用 `DRY_RUN=1` 预览发布命令。

---

## ❓ 常见问题

**Q：开发登录账号在哪里配置？**

A：查看 `backend/.env` 中的 `AUTH_EMAIL` 与 `AUTH_PASSWORD`。该文件由 `make dev-init` 创建并被 Git 忽略。

**Q：重启后规则集或工作空间还在吗？**

A：当前保存在 API 进程内存中，重启后恢复为默认工作空间和演示规则。PostgreSQL 容器目前不是规则仓储。

**Q：规则里的 Webhook 或 tag 会自动执行吗？**

A：不会。动作目前是声明数据，会随决策结果返回；外部副作用执行器尚未实现。

**Q：授权未配置会影响规则求值吗？**

A：不会。授权页面提供授权状态查看和文件管理；目前没有需要授权解锁的规则功能。

**Q：业务后端能用 API Key 或 JWT 调用求值 API 吗？**

A：支持项目级 API Key。进入「项目」创建密钥，创建时复制一次，然后通过 `X-API-Key` 请求头调用 `POST /api/v1/engine/evaluate`。密钥只允许求值、可撤销，服务端只保存摘要；当前尚无过期时间与最后使用时间。尚未提供 JWT/OAuth。

**Q：为什么生产启动后数据会丢失？**

A：生产 Compose 当前持久化授权文件，不持久化规则集、工作空间和会话。正式承载业务数据前需接入数据库并配置 TLS。

---

## 📚 相关文档

- [文档中心首页](./frontend/docs/docs/index.md)
- [快速开始与开发/生产环境](./frontend/docs/docs/guide/quick-start.md)
- [管理工作台栏目说明](./frontend/docs/docs/guide/admin-console.md)
- [项目结构](./frontend/docs/docs/guide/project-structure.md)
- [规则集与条件 DSL](./frontend/docs/docs/engine/rulesets.md)
- [决策试运行](./frontend/docs/docs/engine/evaluation.md)
- [API 概览](./frontend/docs/docs/api/overview.md)
- [当前架构](./docs/architecture.md)
- [开发路线图](./docs/todo.md)
- [规则引擎选型记录](./docs/research/rule-engine-survey.md)
- [AI 决策扩展预研](./docs/research/jev-integration.md)

---

## 📄 许可证

SnakeRule 使用[社区许可证](./LICENSE)：以 Apache License 2.0 为基础，并附加 SnakeRule 品牌保留和未经授权不得提供商业多租户 SaaS 服务的限制。

因此该文件不是未经修改的标准 Apache-2.0 许可证，不能仅用 SPDX 标识 `Apache-2.0` 描述。个人使用、内部部署和单租户集成的具体条件，以及商业多租户服务的授权要求，以 [LICENSE](./LICENSE) 全文为准。商业授权咨询请通过 [GitHub Issues](https://github.com/bulolo/SnakeRule/issues) 联系维护者。

---

## 🌱 生态项目

- [OwlApi](https://github.com/bulolo/owlapi) — SQL to API 智能网关。
- [RayFlow](https://github.com/bulolo/RayFlow) — 流程编排项目。
- [CatWiki](https://github.com/bulolo/CatWiki) — 知识管理项目。

<div align="center">

Made with ❤️ by the **szplanner** team

</div>
