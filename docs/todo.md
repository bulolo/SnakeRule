# SnakeRule 开发计划

> 架构原则：规则配置服务 / 规则计算服务集群**分离部署**，计算节点支持动态扩缩容。

---

## Phase 1 — MVP（核心引擎）

### 🗄️ 后端基础

- [ ] 项目初始化（Go Module, Gin, sqlx, Viper）
- [ ] 数据库 Schema 设计与迁移（rulesets / rules / decisions）
- [ ] 统一错误码与响应体规范
- [ ] 健康检查接口 `GET /healthz`
- [ ] Swagger / OpenAPI 文档生成

### 📐 规则输入与 DSL

- [ ] **输入参数（InputParameter）**
  - [ ] 参数类型系统：`string / number / boolean / date / array / object`
  - [ ] 参数来源：Request Body / Header / 上下文变量 / 常量
  - [ ] 参数校验（必填、类型检查、默认值）
- [ ] **变量（Variable）**
  - [ ] 中间变量定义与运行时赋值
  - [ ] 支持引用其他变量做链式计算
  - [ ] 变量作用域隔离（规则级 / 规则集级）
- [ ] **函数（Function）**
  - [ ] 内置函数库：`abs / round / len / now / dateDiff / split / trim / toUpper`
  - [ ] 字符串函数：`contains / startsWith / endsWith / regex`
  - [ ] 数学函数：`sum / avg / min / max / mod`
  - [ ] 日期函数：`dateAdd / dateFormat / isWeekend / age`
  - [ ] 自定义函数注册接口（插件扩展）
- [ ] **条件（Condition）**
  - [ ] 内置操作符：`eq / ne / gt / gte / lt / lte / in / notIn / contains / isNull / isNotNull`
  - [ ] 逻辑组合：`AND / OR / NOT` 无限嵌套
  - [ ] 条件组（Group）支持括号优先级
  - [ ] 正则表达式条件
  - [ ] 条件单元测试 / dry-run 接口

### ⚙️ 规则引擎 Core

- [ ] **JSON DSL 解析器**
  - [ ] Schema 定义（JSON Schema）
  - [ ] 语法校验与友好报错
- [ ] **条件求值器**
  - [ ] AND / OR / NOT 递归求值
  - [ ] 短路求值优化
  - [ ] 类型强转兼容（"18" == 18）
- [ ] **Action 执行器**
  - [ ] `set_variable`：设置返回变量值
  - [ ] `webhook`：触发外部 HTTP 回调
  - [ ] `tag`：打标签（用于分类 / 路由）
  - [ ] `log`：记录自定义日志
  - [ ] `abort`：中止后续规则执行

### 🚀 规则模型演进

- [ ] **普通规则（GeneralRule）**
  - [ ] 同步执行 `POST /api/v1/engine/evaluate`
  - [ ] 异步执行 `POST /api/v1/engine/evaluate/async`
  - [ ] 返回：命中规则列表 + 执行的 Actions + 耗时
- [ ] **规则集（RuleSet）**
  - [ ] 顺序执行模式（first-match / all-match）
  - [ ] 优先级权重排序
  - [ ] 规则集间依赖引用（子规则集）
- [ ] **决策表（DecisionTable）**
  - [ ] 列=条件、行=规则的矩阵结构
  - [ ] 命中策略：`first / all / unique / collect`
  - [ ] 输出列：支持多列结果输出
  - [ ] Excel / CSV 导入导出

### 📋 规则集 CRUD API

- [ ] `POST   /api/v1/rulesets`
- [ ] `GET    /api/v1/rulesets`
- [ ] `GET    /api/v1/rulesets/:id`
- [ ] `PUT    /api/v1/rulesets/:id`
- [ ] `DELETE /api/v1/rulesets/:id`
- [ ] `POST   /api/v1/rulesets/:id/publish`（发布 / 版本快照）
- [ ] `POST   /api/v1/rulesets/:id/clone`（克隆）

### 📝 决策日志与可观测性

- [ ] 决策日志入库（decisions 表）
- [ ] 记录：输入 Facts / 命中规则 / 执行动作 / 耗时 / 租户 ID
- [ ] 决策详情查询 `GET /api/v1/decisions/:id`
- [ ] 决策历史列表 `GET /api/v1/decisions`（分页 + 按规则集过滤）
- [ ] 决策追踪：展示每条规则的求值路径（命中 / 未命中原因）

### ⚡ 热缓存

- [ ] Redis 缓存已发布规则集（发布即刷新）
- [ ] 缓存 Key 策略：`ruleset:{id}:v{version}`
- [ ] 缓存 TTL + 主动失效（发布时广播 pub/sub）
- [ ] 本地内存二级缓存（sync.Map）减少 Redis 压力

### 🖥️ 前端基础

- [ ] Next.js 项目初始化（App Router + TypeScript）
- [ ] 规则集列表页（搜索 / 过滤 / 分页）
- [ ] 规则集创建 / 编辑（Monaco JSON 编辑器）
- [ ] 决策历史查询页

---

## Phase 2 — 可视化编辑器

- [ ] **拖拽式规则构建器**（无需手写 JSON）
  - [ ] 条件节点拖拽组合
  - [ ] AND / OR 分组可视化
  - [ ] Action 节点配置面板
- [ ] **决策表可视化编辑器**
  - [ ] Excel-like 网格编辑
  - [ ] 列类型配置（条件列 / 输出列）
  - [ ] 行增删改、批量导入
- [ ] **规则 Dry-Run 面板**：填入 Facts → 实时预览命中结果
- [ ] **规则链路可视化**（ReactFlow）：展示条件求值树
- [ ] 单次决策详情页：完整求值路径 + Actions 执行记录
- [ ] 决策统计图表（命中率 / P99 延迟 / 热门规则 TOP 10）

---

## Phase 3 — 权限管理与多租户

- [ ] **工作空间（Workspace）**
  - [ ] 工作空间 CRUD
  - [ ] 租户隔离：所有资源绑定 `tenant_id`
  - [ ] 工作空间切换（前端顶栏 Switcher）
- [ ] **RBAC 权限**
  - [ ] 角色定义：`SuperAdmin / Admin / Editor / Viewer`
  - [ ] 权限点：规则读 / 规则写 / 发布 / 决策执行 / 日志查看
  - [ ] 用户邀请与角色分配
  - [ ] JWT 认证 + Refresh Token
  - [ ] API Key 管理（用于服务端调用引擎）

---

## Phase 4 — AI 增强（Jev 集成）

- [ ] Jev API 客户端实现（HTTP + 重试 + 超时）
- [ ] `jev_fact_resolve` Action：调用 Jev 补充缺失 Fact
- [ ] Jev Fact Resolver 集成到条件求值前置步骤
- [ ] Jev 调用结果缓存（避免重复推理）
- [ ] Jev 调用降级策略（超时 → 跳过 / 使用默认值）
- [ ] AI 评分日志写入 `decisions.jev_scores`
- [ ] Jev 推理调用可观测性（调用量 / 耗时 / 成功率）

---

## Phase 5 — 生态集成

- [ ] **OwlApi Webhook → SnakeRule**：OwlApi 事件触发规则决策
- [ ] **Rayflow 节点**：Rayflow 工作流节点可内嵌调用规则引擎
- [ ] **CatWiki 知识条目变更**：触发相关规则重新评估
- [ ] Go SDK（规则引擎客户端库）
- [ ] TypeScript SDK（前端 / Node.js 调用）
- [ ] Helm Chart（Kubernetes 部署，支持 HPA 弹性扩缩）
- [ ] Open API 对外服务化 + Webhook 订阅 + API 限流配额

---

## 技术债 / 质量保障

- [ ] 单元测试覆盖率 ≥ 80%（条件求值器、DSL 解析器）
- [ ] 集成测试：决策执行全链路
- [ ] 性能基准测试（目标：万级规则 < 10ms 响应）
- [ ] CI/CD 流水线（GitHub Actions）
- [ ] Lint / Format 规范（golangci-lint, ESLint）
- [ ] 链路追踪（OpenTelemetry + Jaeger）
