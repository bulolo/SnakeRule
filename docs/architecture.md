# SnakeRule 架构说明

> 本文描述仓库当前实现。规划中的持久化、动作执行和分布式计算不会作为现有能力描述。

## 系统边界

```text
浏览器
  └── Nginx
      ├── /             → React + Vite 管理端
      ├── /docs/        → VitePress 文档站点
      ├── /api/         → Go HTTP API
      └── /healthz      → Go 健康检查

Go API
  ├── transport/http   路由、认证中间件、请求和响应
  ├── service          规则集校验、工作空间管理、决策编排
  ├── engine/evaluator 纯条件校验和求值
  ├── repo             仓储接口和 JSON 文件持久化实现
  ├── domain           工作空间、项目、项目 Key、告警策略/渠道、规则和决策模型
  └── licensing        Pulse Agent 授权文件校验和加密存储
```

开发和生产环境均使用 Compose 编排 API、管理端、文档站点和 Nginx。Nginx 是浏览器统一入口；当前没有独立 gateway 服务。PostgreSQL 与 Redis 只作为可选开发容器启动，API 不依赖它们。

## 请求与领域流程

### 规则集管理

管理端调用由 OpenAPI 契约生成的 TypeScript 客户端。受保护的 HTTP handler 将工作空间与项目上下文交给 repository。规则集以 `workspaceID/projectID` 隔离，在保存前校验 ID、名称、条件结构和操作符，并按优先级稳定排序。系统会初始化 `default` 工作空间，但工作空间内不自动创建项目；管理员可创建空项目或电商订单演示项目。

### 规则求值

`POST /api/v1/engine/evaluate` 接受 `rulesetId` 和 `facts`。管理会话按当前工作空间与项目范围求值；项目 API Key 通过 `X-API-Key`、JWT 通过 Bearer 认证，二者只可求值其绑定项目中的已保存规则集。管理会话还可携带 `ruleset` 求值草稿，不写入仓储。求值按优先级从高到低遍历启用规则，返回所有命中规则及其声明动作。

条件树包含且仅包含一个分支：`all`、`any`、`not` 或事实叶子。事实通过点分路径从 JSON 对象读取。当前支持相等/不等、数值比较、`in`、`notIn`、`contains`、`isNull` 和 `isNotNull`。

### 工作空间和会话

登录由配置的管理员邮箱和密码完成，成功后签发 HttpOnly、SameSite Lax Cookie。会话及工作空间/项目选择保存在数据目录下的 JSON 文件，TTL 默认 24 小时；切换工作空间会清除项目选择。

### 系统告警渠道

系统设置提供企业微信、钉钉、飞书机器人和自定义 Webhook 渠道的增删改查和测试发送。钉钉/飞书请求按机器人协议生成签名，自定义 Webhook 可使用 Bearer 密钥；密钥不会被读取接口返回。项目可引用已启用渠道并设置规则命中筛选；已保存规则命中后异步发送通知。当前没有投递队列、重试或 UI 投递记录，失败会写入服务日志。

项目 API Key 以 `sr_live_` 开头，JWT 有效期为 30 天；创建时明文只返回一次，仓储只保存 SHA-256 摘要，并绑定到工作空间与项目。凭据可撤销；当前没有 API Key 有效期、轮换和最后使用时间。

授权有效时，服务启动后延迟 10 秒首次通过 Pulse Agent 上报心跳，之后默认每 4 分钟上报一次。心跳包括版本、环境、实例 ID、运行时、健康状态及工作空间/项目/规则集数量；上报失败不会影响本地规则服务。上报地址可通过 `PULSE_HEARTBEAT_URL` 覆盖。

## 代码边界

| 目录 | 职责 |
|------|------|
| `backend/server/cmd/main` | 进程入口与退出处理 |
| `backend/server/internal/bootstrap` | 配置、持久化仓储、服务和 HTTP 服务装配；建立默认工作空间与站内通知渠道 |
| `backend/server/internal/config` | `.env` 与环境变量配置加载 |
| `backend/server/internal/domain` | API 与业务共享的数据模型 |
| `backend/server/internal/engine/evaluator` | 递归条件树校验和确定性求值 |
| `backend/server/internal/repo` | JSON 文件持久化的规则集、工作空间、项目、凭据、告警渠道和日志仓储 |
| `backend/server/internal/service` | 业务校验、规则整理、工作空间 slug 唯一性和决策流程 |
| `backend/server/internal/transport/http` | API 路由、会话认证、错误映射及内置轻量控制台资源 |
| `backend/server/internal/licensing` | Pulse 授权校验、服务端加密存储 |
| `frontend/admin/src/components` | 登录、选择器和平台/系统设置组件 |
| `frontend/admin/src/workbench.tsx` | 管理工作台、规则列表、画布和 JSON 编辑 |
| `frontend/admin/src/lib/sdk/generated.ts` | OpenAPI 生成的管理端 TypeScript 客户端 |
| `frontend/docs/docs` | VitePress 产品和开发文档 |

## 数据生命周期与限制

- 规则集、工作空间、项目、API Key/JWT、告警渠道、通知、调用日志、会话和许可证状态保存在数据卷的 JSON 文件中；开发和生产 Compose 均需保留数据卷。
- 空工作空间不会自动创建项目；电商演示规则集只在创建演示项目时生成。
- 规则动作只作为响应数据返回；没有 Webhook、变量写入或其他副作用执行器。
- 规则求值会写入调用日志并标记设计器模拟、真实项目 API 或管理会话来源；不保存 facts 内容。
- 登录只有一个配置的超级管理员身份；没有用户管理、RBAC 或租户账号体系。
- 管理 API 使用会话 Cookie；项目支持 API Key 与短期 JWT 调用已保存规则，不支持 OAuth/OIDC。
- 项目告警已支持规则命中后的异步渠道发送，但尚无投递队列、重试和历史记录；系统其他运行参数仍为只读信息。
- 授权密文通过服务端路径保存，并非保存在规则仓储中；当前规则功能没有按授权状态门禁。

## 部署入口

开发环境：`make dev-init` 创建本地 `backend/.env`，`make dev-up` 构建并运行开发栈。管理端和文档源码使用挂载卷，Go API 来自容器镜像。

生产环境：`make prod-init` 创建忽略的 `deploy/.env.prod` 和随机凭据；`make prod-up` 构建并启动生产 Compose。生产 Compose 只向宿主机发布 Nginx 端口，授权文件写入数据卷。正式承载业务规则前必须先实现持久化规则/工作空间仓储，并规划备份、会话失效策略和 TLS。

## 演进顺序

下一阶段先完善持久化和 API 安全边界，再引入发布版本/审计与动作执行器。分离规则管理和计算集群、Redis 热更新、AI 决策和外部生态集成属于后续评估项，不是当前部署拓扑。任务状态见[开发路线图](./todo.md)。
