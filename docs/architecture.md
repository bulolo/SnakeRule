# SnakeRule 架构说明

> 本文描述当前实现。业务状态写入 PostgreSQL；当前 API 仓储保留进程内缓存，因此部署仍使用单个 API 副本。

## 系统边界

```text
浏览器
  └── Nginx
      ├── /             → React + Vite 管理端
      ├── /docs/        → VitePress 文档站点
      ├── /api/v1/      → Go HTTP API
      └── /healthz      → Go 健康检查

Go API
  ├── transport/http   路由、认证中间件、请求和响应
  ├── service          规则集校验、组织管理、决策编排
  ├── engine/evaluator 纯条件校验和求值
  ├── repo             仓储接口和 PostgreSQL 实体表持久化实现
  ├── domain           组织、项目、项目 Key、告警策略/渠道、规则和决策模型
  └── licensing        Pulse Agent 授权文件校验和加密存储
```

开发和生产环境均使用 Compose 编排 API、管理端、文档站点、Nginx 和 PostgreSQL。Nginx 是浏览器统一入口；当前没有独立 gateway 服务。Redis 仅为可选基础设施，API 不依赖它。

## 请求与领域流程

### 规则集管理

管理端调用由 OpenAPI 契约生成的 TypeScript 客户端。API 成功响应统一为 `{ code, msg, data }`，错误响应包含稳定的 `error_code`；分页列表统一使用 `list` 与 `pagination`，`is_pager=0` 可请求全部数据。受保护的 HTTP handler 将组织与项目上下文交给 repository。规则集以 `organization_id/project_id` 隔离，在保存前校验 ID、名称、条件结构和操作符，并按优先级稳定排序。系统会初始化 `default` 组织，但组织内不自动创建项目；管理员可创建空项目或电商订单演示项目。

### 规则求值

`POST /api/v1/engine/evaluate` 接受 `ruleset_id` 和 `facts`。管理会话按当前组织与项目范围求值；项目 API Key 通过 `X-API-Key`、JWT 通过 Bearer 认证，二者只可求值其绑定项目中的已保存规则集。管理会话还可携带 `ruleset` 求值草稿，不写入仓储。求值按优先级从高到低遍历启用规则，返回所有命中规则及其声明动作。

条件树包含且仅包含一个分支：`all`、`any`、`not` 或事实叶子。事实通过点分路径从 JSON 对象读取。当前支持相等/不等、数值比较、`in`、`notIn`、`contains`、`isNull` 和 `isNotNull`。

### 组织和会话

登录通过用户账号完成，首次启动时从环境配置创建超级管理员。成功后签发 HttpOnly、SameSite Lax Cookie；会话及组织/项目选择存入 PostgreSQL，默认 24 小时过期。切换组织会清除项目选择。

### 组织告警渠道

组织设置提供企业微信、钉钉、飞书机器人和自定义 Webhook 渠道的增删改查和测试发送。钉钉/飞书请求按机器人协议生成签名，自定义 Webhook 可使用 Bearer 密钥；密钥不会被读取接口返回。项目可引用已启用渠道并设置规则命中筛选；已保存规则命中后异步发送通知。当前没有投递队列、重试或 UI 投递记录，失败会写入服务日志。

项目 API Key 创建时可设置过期时间；服务端仅保存其摘要，明文只返回一次。项目 JWT 支持平台签发访问 Token（默认 30 天，可指定过期时间）和项目自签 HS256 两种方式；自签密钥由服务端加密保存，创建时只返回一次。凭据绑定到组织与项目且可撤销。当前尚无凭据轮换流程和最后使用时间记录。

授权有效时，服务启动后延迟 10 秒首次通过 Pulse Agent 上报心跳，之后默认每 4 分钟上报一次。心跳包括版本、环境、实例 ID、运行时、健康状态及组织/项目/规则集数量；上报失败不会影响本地规则服务。上报地址可通过 `PULSE_HEARTBEAT_URL` 覆盖。

## 代码边界

| 目录 | 职责 |
|------|------|
| `backend/server/cmd/main` | 进程入口与退出处理 |
| `backend/server/internal/bootstrap` | 配置、持久化仓储、服务和 HTTP 服务装配；建立默认组织与站内通知渠道 |
| `backend/server/internal/config` | `.env` 与环境变量配置加载 |
| `backend/server/internal/domain` | API 与业务共享的数据模型 |
| `backend/server/internal/engine/evaluator` | 递归条件树校验和确定性求值 |
| `backend/server/internal/repo` | PostgreSQL 实体表仓储 |
| `backend/server/internal/service` | 业务校验、规则整理、组织 slug 唯一性和决策流程 |
| `backend/server/internal/transport/http` | API 路由、会话认证、错误映射及内置轻量控制台资源 |
| `backend/server/internal/licensing` | Pulse 授权校验、服务端加密存储 |
| `frontend/admin/src/components` | 登录、选择器和平台/组织设置组件 |
| `frontend/admin/src/workbench.tsx` | 管理工作台、规则列表、画布和 JSON 编辑 |
| `frontend/admin/src/lib/sdk/generated.ts` | OpenAPI 生成的管理端 TypeScript 客户端 |
| `frontend/docs/docs` | VitePress 产品和开发文档 |

## 数据生命周期与限制

- 组织、用户、项目、成员关系、规则集、项目凭据、告警渠道、通知、调用日志、会话、许可证状态和品牌配置分别写入对应 PostgreSQL 表。数据以 JSONB `document` 列保存领域文档，并通过显式关系列和索引支持列表与关联查询。
- 数据库结构升级脚本位于 `backend/server/internal/repo/migrations`，使用 Goose 按版本编号执行，记录在 `snakerule_goose_db_version`。服务启动时自动应用未执行的迁移，并通过 PostgreSQL advisory lock 避免并发执行。初始数据库结构由 `00001_initial_schema.sql` 创建。
- 仓储在进程内缓存数据，并将每个仓储快照写入数据库；当前保留单 API 副本部署。调用日志最多保留 2000 条。
- 空组织不会自动创建项目；电商演示规则集只在创建演示项目时生成。
- 规则动作只作为响应数据返回；没有 Webhook、变量写入或其他副作用执行器。
- 规则求值会写入调用日志并标记设计器模拟、真实项目 API 或管理会话来源；保存经敏感字段遮蔽且单条限长的请求/响应预览，不记录鉴权头。
- 用户角色包含超级管理员、组织管理员、项目管理员和项目成员；平台设置仅超级管理员可访问。
- 管理 API 使用会话 Cookie；项目 API Key 与 JWT 可调用绑定项目的已保存规则，不支持 OAuth/OIDC。
- 项目告警已支持规则命中后的异步渠道发送，但尚无投递队列、重试和历史记录；系统其他运行参数仍为只读信息。
- 授权密文通过服务端路径保存，并非保存在规则仓储中；当前规则功能没有按授权状态门禁。

## 部署入口

开发环境：`make dev-init` 创建本地 `backend/.env`，`make dev-up` 构建并运行开发栈。管理端和文档源码使用挂载卷，Go API 来自容器镜像。

生产环境：`make prod-init` 创建忽略的 `deploy/.env.prod` 和随机凭据；`make prod-up` 构建并启动生产 Compose。生产 Compose 只向宿主机发布 Nginx 端口，PostgreSQL 持久化业务数据。当前 API 仍保持单副本；公网部署需配置 TLS 与数据库备份。Helm Chart 位于 `deploy/helm`。

## 演进顺序

下一阶段优先完善数据库并发写入、安全审计和规则发布生命周期。AI 决策仍处于评估阶段：当前求值流程不调用模型服务。分离规则管理和计算集群、Redis 热更新及外部生态集成也属于后续评估项，不是当前部署拓扑。任务状态见[开发路线图](./todo.md)。
