<div align="center">

# 🐍 SnakeRule

**企业级可视化规则引擎平台**

链式精准，智能决策 —— 让业务规则像蛇形链路一样灵活流转，支持传统规则与 AI 决策模型混合驱动。

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-15+-000000?logo=next.js)](https://nextjs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](./LICENSE)

简体中文 | [English](./README_EN.md)

[官网](https://szplanner.com) | [文档中心](./frontend/docs/docs/index.md) | [在线演示](#)

</div>

---

## 🌐 产品矩阵

SnakeRule 是 **szplanner.com** 平台产品矩阵的第四个成员：

| 产品 | 定位 | 核心能力 |
|------|------|---------|
| 🐱 **CatWiki** | 知识管理平台 | 结构化知识沉淀、团队协作 |
| 🦉 **OwlApi** | API 智能网关 | SQL → RESTful API，混合云接入 |
| 🐟 **Rayflow** | 流程编排引擎 | 可视化工作流、节点编排 |
| 🐍 **SnakeRule** | 规则引擎平台 | 业务规则管理、AI 决策集成 |

---

## 💡 产品定位

SnakeRule 是一款**企业级可视化规则引擎平台**，将传统规则引擎与 AI 决策模型（如 Jev）深度融合，帮助企业在不修改代码的情况下，灵活配置、热更新业务规则。

### 核心价值

- 🎯 **精准决策** — 支持复杂嵌套条件（AND/OR/NOT），毫秒级规则匹配
- 🤖 **AI 增强** — 内置 Jev AI 决策节点，传统规则 + AI 概率评分混合驱动
- 🔥 **热更新** — 规则无需重启服务实时生效，零停机变更
- 🧩 **可视化** — 拖拽式规则配置界面，业务人员可独立操作
- 🔌 **易集成** — RESTful API / SDK / Webhook 多种接入方式
- 📊 **可追溯** — 完整规则命中日志与决策链路可视化

---

## 🚀 使用场景

| 场景 | 描述 | 示例 |
|------|------|------|
| **风控审核** | 多维度条件判断触发审核流程 | 金额 > 10000 且 新用户 → 人工审核 |
| **营销策略** | 用户分群动态发放权益 | VIP + 近7日活跃 → 发放优惠券 |
| **内容审核** | AI + 规则双层过滤 | Jev 评分 > 0.8 + 关键词命中 → 拦截 |
| **费率计算** | 动态定价与折扣规则 | 季节 + 库存 + 用户等级 → 最终价格 |
| **告警路由** | 监控指标触发差异化通知 | CPU > 90% 且持续 5min → 紧急告警 |
| **合规检查** | 数据字段合规性校验 | 身份证格式 + 黑名单 → 合规放行 |

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                      SnakeRule 平台                      │
│                                                         │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │  管理控制台   │    │   官网/文档   │                  │
│  │ (React/Vite) │    │ (VitePress)  │                  │
│  └──────┬───────┘    └──────────────┘                  │
│         │                                               │
│  ┌──────▼──────────────────────────────────────┐       │
│  │            API Server (Go/Gin)               │       │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐ │       │
│  │  │ 规则管理  │  │ 决策执行  │  │  日志追踪  │ │       │
│  │  └──────────┘  └────┬─────┘  └───────────┘ │       │
│  │                     │                       │       │
│  │  ┌──────────────────▼─────────────────────┐ │       │
│  │  │         SnakeRule Engine Core           │ │       │
│  │  │  ┌─────────────┐  ┌─────────────────┐  │ │       │
│  │  │  │ 传统规则求值  │  │  Jev AI 决策节点 │  │ │       │
│  │  │  │(条件/动作链) │  │  (概率化评分)    │  │ │       │
│  │  │  └─────────────┘  └─────────────────┘  │ │       │
│  │  └────────────────────────────────────────┘ │       │
│  └──────────────────────┬──────────────────────┘       │
│                         │                              │
│  ┌──────────┐  ┌────────▼──────┐  ┌─────────────┐    │
│  │PostgreSQL│  │     Redis     │  │  Jev API 服务│    │
│  │(规则存储) │  │  (缓存/队列)  │  │ (AI决策调用) │    │
│  └──────────┘  └───────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────┘
```

---

## 🧩 核心概念

SnakeRule 当前围绕规则集、规则、条件、动作和决策结果组织。规则集以 JSON DSL 表达，条件支持 `all / any / not` 组合；求值结果返回命中的规则及其动作。当前版本提供规则集编辑和决策试运行，变量、函数及外部动作执行会在后续迭代中逐步实现。

### 三种规则类型

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| **GeneralRule（普通规则）** | 单条/多条规则顺序执行，支持 first-match / all-match | 风控、告警路由 |
| **RuleSet（规则集）** | 一组规则按优先级排列，支持子规则集嵌套引用 | 营销策略、费率计算 |
| **DecisionTable（决策表）** | 矩阵结构，列=条件、行=规则，命中策略 `first / all / unique / collect` | 定价表、合规矩阵 |

---

## 🛠️ 技术栈

### 后端

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| **语言** | Go 1.24+ | 高性能，与 OwlApi 保持一致 |
| **Web 框架** | Go `net/http`（MVP） | 标准库路由；当前阶段少依赖、单进程启动 |
| **规则引擎核心** | 自研（参考 json-rules-engine） | JSON DSL 描述规则，支持嵌套条件 |
| **表达式引擎** | JSON 条件求值（MVP） | expr-lang 是后续复杂表达式能力的候选 |
| **数据库** | 进程内存（MVP） | PostgreSQL 作为下一阶段持久化适配器 |
| **缓存** | 暂无（MVP） | Redis 规则热缓存后续接入 |
| **消息队列** | 暂无（MVP） | 异步决策后续接入 |
| **AI 集成** | Jev API（规划） | 作为后续决策节点 |
| **配置管理** | Viper | 多环境配置 |
| **ORM** | sqlx + 手写 SQL | 性能优先，与 OwlApi 一致 |

### 前端与文档

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| **管理端框架** | React + Vite | 规则集工作台与 API 客户端 |
| **文档站点** | VitePress | 使用指南、规则说明与 API 概览 |
| **规则编辑器** | 自研拖拽式规则构建器 | 可视化条件/动作配置 |
| **流程可视化** | ReactFlow | 规则链路可视化 |
| **状态管理** | Zustand | 轻量状态管理 |
| **代码编辑器** | Monaco Editor | JSON 规则直接编辑 |
| **图表** | Recharts | 决策统计图表 |

### 基础设施

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| **容器化** | Docker + Docker Compose | 本地开发环境 |
| **编排** | Kubernetes + Helm | 生产部署 |
| **反向代理** | Nginx | 静态资源 + API 代理 |
| **CI/CD** | GitHub Actions | 自动化构建发布 |
| **监控** | OpenTelemetry + Prometheus | 可观测性 |

---

## 📁 项目结构

```
SnakeRule/
├── backend/
│   ├── server/                    # 主服务
│   │   ├── cmd/main/              # 程序入口
│   │   ├── internal/
│   │   │   ├── bootstrap/         # 服务启动初始化
│   │   │   ├── config/            # 配置读取与校验
│   │   │   ├── domain/            # Rule / Ruleset / Decision 模型
│   │   │   ├── engine/evaluator/  # 规则条件求值
│   │   │   ├── repo/              # 当前为内存存储接口
│   │   │   ├── service/           # 业务逻辑层
│   │   │   ├── transport/http/    # REST API 与嵌入式规则工作台
│   │   │   └── version/           # 服务版本
│   │   ├── docs/openapi.json      # OpenAPI 契约
│   │   └── go.mod
│   └── .env.example
│
├── frontend/admin/                # React + Vite 管理端与 OpenAPI SDK
├── frontend/docs/                 # VitePress 产品文档站点
│
├── docs/
│   ├── research/                  # 技术调研
│   ├── architecture.md            # 架构设计
│   └── todo.md                    # 开发计划
│
├── scripts/
│   ├── generate_sdk.py            # 生成 TypeScript SDK
│   └── set_version.py             # 更新项目版本
│
├── VERSION
├── Makefile
├── docker-compose.dev.yml
└── README.md
```

---

## ⚡ 快速开始

### 前置条件

- Go 1.24+
- Make

### 本地开发

```bash
make dev-init
make dev-up
```

`make dev-up` 会启动 Nginx、管理端、文档站点和 Go API。浏览器打开 http://localhost:8080 使用规则工作台，http://localhost:8080/docs/ 阅读文档；管理端直连地址是 http://localhost:3000，文档开发服务器为 http://localhost:8003/docs/，API 默认地址为 http://localhost:8081。也可以执行 `make dev-open` 或 `make docs-open` 打开站点。运行日志显示在终端中；另开终端执行 `make dev-down` 可停止服务。

文档位于 `frontend/docs`，可通过 `make docs-dev` 单独启动预览、`make docs-build` 构建静态站点。文档的基础维护、项目结构、规则模型和 API 说明均按 SnakeRule 当前实现编写。

管理端现已接入超级管理员邮箱登录，当前本地开发账号配置在被 Git 忽略的 `backend/.env` 中。可通过 `AUTH_EMAIL`、`AUTH_NAME`、`AUTH_PASSWORD` 和 `AUTH_SESSION_TTL` 设置登录身份；部署环境必须使用独立密码。登录后可进入规则集列表、拖拽设计画布和模拟决策，退出或会话过期后会返回登录页。

右上角提供「平台设置」与「系统设置」入口：平台页展示版本、授权版本、服务状态及规则服务配置；系统页展示工作空间、规则引擎和账号安全信息。当前为只读信息面板，后续再接持久化配置项。

规则存储目前使用进程内存；PostgreSQL 和 Redis 容器可通过 `make infra-up` 启动，当前 API 暂不依赖它们。

`make gen-sdk` 根据服务的 OpenAPI 契约生成管理端 TypeScript API SDK；`make set-version v=0.1.1` 同步更新 `VERSION`、服务版本、OpenAPI 文档和管理端版本。

---

## 🔌 规则示例

### 普通规则（GeneralRule）— JSON DSL

```json
{
  "name": "高风险订单拦截",
  "type": "general",
  "version": "1.0.0",
  "conditions": {
    "all": [
      {
        "fact": "order.amount",
        "operator": "greaterThan",
        "value": 10000
      },
      {
        "any": [
          {
            "fact": "user.isNew",
            "operator": "equal",
            "value": true
          },
          {
            "fact": "jev.riskScore",
            "operator": "greaterThan",
            "value": 0.75
          }
        ]
      }
    ]
  },
  "actions": [
    {
      "type": "webhook",
      "config": { "url": "https://review.example.com/trigger", "method": "POST" }
    },
    {
      "type": "tag",
      "config": { "label": "HIGH_RISK", "ttl": 86400 }
    }
  ]
}
```

### 决策表（DecisionTable）— JSON DSL

```json
{
  "name": "会员等级折扣表",
  "type": "decision_table",
  "hitPolicy": "first",
  "inputs": [
    { "name": "user.level",   "type": "string" },
    { "name": "order.amount", "type": "number" }
  ],
  "outputs": [
    { "name": "discount", "type": "number" },
    { "name": "freeShipping", "type": "boolean" }
  ],
  "rows": [
    { "conditions": ["== 'GOLD'",  "> 500"],  "outputs": [0.85, true]  },
    { "conditions": ["== 'GOLD'",  "<= 500"], "outputs": [0.90, false] },
    { "conditions": ["== 'SILVER'","> 300"],  "outputs": [0.92, true]  },
    { "conditions": ["== 'SILVER'","<= 300"], "outputs": [0.95, false] },
    { "conditions": ["-",          "-"],       "outputs": [1.00, false] }
  ]
}
```

### API 调用示例

```bash
curl -X POST http://localhost:8081/api/v1/engine/evaluate \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: your-api-key" \
  -d '{
    "rulesetId": "high-risk-order",
    "facts": {
      "order": { "amount": 15000, "id": "ORD-001" },
      "user":  { "id": "USR-123", "isNew": true }
    }
  }'
```

**响应示例：**

```json
{
  "decisionId": "d-7f3a2c",
  "matched": true,
  "hitRules": [
    { "ruleId": "rule-001", "name": "高风险订单拦截", "priority": 1 }
  ],
  "actions": [
    { "type": "webhook", "status": "ok", "latencyMs": 32 },
    { "type": "tag",     "status": "ok", "label": "HIGH_RISK" }
  ],
  "jevScores": { "riskScore": 0.87 },
  "latencyMs": 8,
  "traceUrl": "/api/v1/decisions/d-7f3a2c"
}
```

---

## 🗺️ 开发路线图

> 详细任务清单见 [docs/todo.md](./docs/todo.md)

### Phase 1 — 核心引擎（MVP）
- [ ] 基础组件：InputParameter 类型系统、Variable、Function 函数库、Condition 操作符
- [ ] 三种规则类型：GeneralRule / RuleSet / DecisionTable（含命中策略）
- [ ] 条件求值器（AND/OR/NOT 嵌套，短路优化）
- [ ] Action 执行器（webhook / tag / log / set_variable / abort）
- [ ] 规则 CRUD + 发布 / 克隆 / 版本快照 API
- [ ] 决策追踪日志（命中路径 + Jev AI 评分）
- [ ] Redis 热缓存（二级缓存 + pub/sub 失效广播）
- [ ] Next.js 前端基础（列表 / Monaco 编辑 / 历史查询）

### Phase 2 — 可视化编辑器
- [ ] 拖拽式规则构建器（AND/OR 分组可视化）
- [ ] 决策表 Excel-like 可视化编辑器
- [ ] 规则 Dry-Run 面板（实时预览命中结果）
- [ ] 规则链路可视化（ReactFlow 求值树）
- [ ] 决策统计图表（命中率 / P99 延迟 / TOP 规则）

### Phase 3 — 权限管理与多租户
- [ ] 工作空间（Workspace）隔离与切换
- [ ] RBAC 角色权限（SuperAdmin / Admin / Editor / Viewer）
- [ ] JWT 认证 + API Key 管理

### Phase 4 — AI 增强
- [ ] Jev API 客户端（重试 / 超时 / 降级）
- [ ] `jev_fact_resolve` Action：AI 补充缺失 Fact
- [ ] Jev 推理结果缓存与可观测性
- [ ] AI + 传统规则混合驱动 / A/B 测试

### Phase 5 — 生态集成
- [ ] OwlApi Webhook → SnakeRule 事件触发
- [ ] Rayflow 工作流节点内嵌规则决策
- [ ] CatWiki 知识变更联动规则重评
- [ ] Go / TypeScript SDK
- [ ] Helm Chart + HPA 弹性扩缩

---

## 📄 License

[Apache License 2.0](./LICENSE)

---

<div align="center">

Made with ❤️ by the **szplanner** team

🐱 CatWiki · 🦉 OwlApi · 🐟 Rayflow · 🐍 **SnakeRule**

</div>
