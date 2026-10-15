<div align="center">

# 🐍 SnakeRule

**企业级可视化规则引擎平台**

链式精准，智能决策 —— 让业务规则像蛇形链路一样灵活流转，支持传统规则与 AI 决策模型混合驱动。

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-15+-000000?logo=next.js)](https://nextjs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](./LICENSE)

简体中文 | [English](./README_EN.md)

[官网](https://szplanner.com) | [文档中心](#) | [在线演示](#)

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
│  │ (Next.js)    │    │  (Next.js)   │                  │
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

> 参照 [Boot Rule](http://ruleengine.cn/doc/quickstart/Quickstart.html) 功能体系，SnakeRule 由以下六大基础组件构成：

| 概念 | 说明 |
|------|------|
| **InputParameter（输入参数）** | 决策时传入的上下文数据，支持 `string / number / boolean / date / array / object` 类型，可来自 Request Body、Header 或常量 |
| **Variable（变量）** | 规则运行时可定义中间变量，支持链式引用与作用域隔离（规则级 / 规则集级） |
| **Function（函数）** | 内置字符串、数学、日期函数库，支持自定义函数插件扩展（如 `dateDiff / regex / sum / age`） |
| **Condition（条件）** | 支持 `AND / OR / NOT` 无限嵌套，内置 `eq / ne / gt / gte / lt / lte / in / notIn / contains / isNull` 等操作符 |
| **Rule（规则）** | 单条规则，包含条件（Condition）和动作（Action），动作类型：`set_variable / webhook / tag / log / abort` |
| **Decision（决策）** | 一次规则执行的完整记录，含输入 Facts、命中规则、执行 Actions、耗时、Jev AI 评分 |

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
| **Web 框架** | Gin | 轻量高性能 HTTP 框架 |
| **规则引擎核心** | 自研（参考 json-rules-engine） | JSON DSL 描述规则，支持嵌套条件 |
| **表达式引擎** | expr-lang/expr | 类型安全表达式求值，字节码编译 |
| **数据库** | PostgreSQL 16+ | 规则、日志、租户数据持久化 |
| **缓存** | Redis | 规则热缓存、决策结果缓存 |
| **消息队列** | Redis Stream / NATS | 异步规则执行、日志投递 |
| **AI 集成** | Jev API / OpenAI Compatible | AI 决策节点调用 |
| **配置管理** | Viper | 多环境配置 |
| **ORM** | sqlx + 手写 SQL | 性能优先，与 OwlApi 一致 |

### 前端

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| **框架** | Next.js 15+ | 与 OwlApi 保持一致 |
| **UI 组件库** | shadcn/ui + Tailwind CSS | 现代化设计系统 |
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
│   │   │   ├── domain/            # 领域模型（Rule/Decision/Log）
│   │   │   ├── repo/              # 数据库访问层
│   │   │   ├── service/           # 业务逻辑层
│   │   │   ├── transport/         # HTTP 路由与 Handler
│   │   │   ├── engine/            # 规则引擎核心
│   │   │   │   ├── evaluator/     # 条件求值器
│   │   │   │   ├── executor/      # 动作执行器
│   │   │   │   ├── jev/           # Jev AI 决策节点适配
│   │   │   │   └── cache/         # 规则热缓存
│   │   │   └── pkg/               # 工具包
│   │   ├── docs/                  # Swagger API 文档
│   │   └── go.mod
│   ├── shared/                    # 共享类型定义
│   └── .env.example
│
├── frontend/
│   ├── src/                       # 管理控制台主应用
│   │   ├── app/                   # Next.js App Router
│   │   ├── components/
│   │   │   ├── rule-builder/      # 可视化规则构建器
│   │   │   └── rule-visualizer/   # 规则链路可视化
│   │   ├── hooks/
│   │   └── store/                 # Zustand 状态
│   ├── admin/                     # 超管后台
│   └── docs/                      # 文档站
│
├── deploy/
│   ├── docker-compose.yml         # 生产 Compose
│   ├── docker-compose.dev.yml     # 开发 Compose
│   ├── helm/                      # Helm Charts
│   ├── nginx/                     # Nginx 配置
│   └── ingress/                   # K8s Ingress
│
├── docs/
│   ├── research/                  # 技术调研文档
│   │   ├── rule-engine-survey.md  # 开源规则引擎调研
│   │   └── jev-integration.md     # Jev 集成方案
│   ├── architecture.md            # 架构设计
│   └── todo.md                    # 开发计划
│
├── scripts/
│   ├── seed_db.sh                 # 数据库初始化
│   ├── smoke_test.py              # 冒烟测试
│   └── version.py                 # 版本管理
│
├── Makefile
├── docker-compose.dev.yml
├── .gitignore
├── LICENSE
└── README.md
```

---

## ⚡ 快速开始

### 前置条件

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16+
- Redis 7+

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/szplanner/snakerule.git
cd SnakeRule

# 2. 启动基础设施
docker compose -f docker-compose.dev.yml up -d postgres redis

# 3. 启动后端
cd backend
cp .env.example .env
go run server/cmd/main/main.go

# 4. 启动前端
cd frontend/src
pnpm install
pnpm dev
```

### 使用 Docker Compose 一键启动

```bash
docker compose up -d
```

访问：
- 管理控制台：http://localhost:3000
- API 文档：http://localhost:8080/swagger

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
curl -X POST http://localhost:8080/api/v1/engine/evaluate \
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
