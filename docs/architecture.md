# SnakeRule 架构设计文档

> 版本：v0.1 | 状态：草稿

---

## 核心概念

| 概念 | 说明 |
|------|------|
| **Ruleset** | 规则集，一组相关规则的集合，支持版本管理 |
| **Rule** | 单条规则，包含 conditions（条件）和 actions（动作） |
| **Fact** | 决策时传入的上下文数据，key-value 结构 |
| **Condition** | 条件表达式，支持 AND/OR/NOT 嵌套 |
| **Action** | 条件满足时执行的动作（Webhook/Tag/Event 等） |
| **Decision** | 一次规则执行的完整记录，含输入、命中规则、输出 |

---

## 模块划分

```
internal/
├── bootstrap/     # 应用启动、依赖注入
├── config/        # 配置加载（Viper）
├── domain/        # 领域模型
│   ├── ruleset.go
│   ├── rule.go
│   ├── decision.go
│   └── action.go
├── repo/          # 数据访问
│   ├── ruleset_repo.go
│   └── decision_repo.go
├── service/       # 业务逻辑
│   ├── ruleset_svc.go
│   └── engine_svc.go
├── transport/     # HTTP 层
│   ├── router.go
│   └── handler/
├── engine/        # 规则引擎核心
│   ├── evaluator/ # 条件求值
│   ├── executor/  # 动作执行
│   ├── jev/       # Jev AI 节点
│   └── cache/     # 规则缓存
└── pkg/           # 工具库
```

## 当前基础框架（已落地）

服务端按 `cmd → bootstrap → transport → service → repo/domain` 组织，规则求值核心独立在 `internal/engine/evaluator`。HTTP 层使用 Go 标准库 `ServeMux`，首阶段不引入 Gin 中间件依赖；标准库已支持方法与路径匹配，足以承载当前少量 API。规则 DSL 按 JSON 递归条件树实现，规则只做确定性求值，Action 暂时作为决策结果返回，不触发外部副作用。

当前提供 `GET /healthz`、`GET /readyz`、规则集 CRUD 和 `POST /api/v1/engine/evaluate`。Go 后端容器通过 embed 提供轻量规则工作台；`make dev-up` 使用 Docker Compose 构建并以前台方式启动，访问 `http://localhost:8081` 即可编辑规则、提交 facts 并查看命中结果。规则仓储是并发安全的内存适配器，服务重启后数据会清空；PostgreSQL、Redis、异步决策、发布版本和多租户均留待后续阶段，避免在领域模型尚未稳定时锁定持久化结构。

目录落点：`backend/server/internal/domain`、`engine/evaluator`、`repo`、`service`、`transport/http`、`bootstrap`。每个 API 层通过服务接口访问仓储，后续可在保持 HTTP 与引擎契约稳定的前提下替换 PostgreSQL 实现。

---

## API 设计

### 规则管理

```
POST   /api/v1/rulesets          创建规则集
GET    /api/v1/rulesets          列表
GET    /api/v1/rulesets/:id      详情
PUT    /api/v1/rulesets/:id      更新
DELETE /api/v1/rulesets/:id      删除

POST   /api/v1/rulesets/:id/rules      新增规则
PUT    /api/v1/rulesets/:id/rules/:rid 更新规则
DELETE /api/v1/rulesets/:id/rules/:rid 删除规则
```

### 决策执行

```
POST /api/v1/engine/evaluate     执行规则决策（同步）
POST /api/v1/engine/evaluate/async  异步执行
GET  /api/v1/decisions/:id       查询决策结果
GET  /api/v1/decisions           决策历史列表
```

---

## 数据库 Schema

```sql
-- 规则集
CREATE TABLE rulesets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        VARCHAR(255) NOT NULL,
    version     VARCHAR(50) DEFAULT '1.0.0',
    description TEXT,
    enabled     BOOLEAN DEFAULT true,
    rules       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 决策日志
CREATE TABLE decisions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    ruleset_id  UUID NOT NULL,
    facts       JSONB NOT NULL,
    matched     JSONB,        -- 命中的规则
    actions     JSONB,        -- 执行的动作
    jev_scores  JSONB,        -- Jev 评分详情
    latency_ms  INTEGER,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```
