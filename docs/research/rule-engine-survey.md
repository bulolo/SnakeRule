# 开源规则引擎技术调研

> 调研时间：2026-10
> 调研目的：为 SnakeRule 项目技术选型提供依据
> 调研人：szplanner 团队

---

## 一、Jev 模型介绍

**Jev** 是由前 OpenAI 研究员推出的 AI 决策模型，定位为 **"System One decision-maker"（系统一快速决策者）**：

| 特性 | 说明 |
|------|------|
| **响应速度** | 毫秒级，专为实时决策设计 |
| **输出形式** | 概率分布（0-1），而非单一答案 |
| **适用场景** | 风控、内容审核、业务判断 |
| **集成方式** | REST API，OpenAI 兼容接口 |

**关键论文**：[Are AI Decision Models Biased? A Case Study of Jev](https://arxiv.org/abs/2610.19619)（arXiv 2026-10）

**与规则引擎的结合思路**：
- Jev 可作为规则引擎的**动态条件评估节点**
- 在规则 Condition 阶段调用 Jev API 获取概率评分
- 由规则引擎根据阈值做最终 Action 决策
- 实现「传统规则 + AI 概率」混合决策架构

---

## 二、主流开源规则引擎对比

### 1. Apache Drools / KIE
- **仓库**：https://github.com/apache/incubator-kie
- **Stars**：6.3k+  **语言**：Java
- **核心算法**：RETE / PHREAK
- **优点**：最成熟企业级方案，支持 DMN/BPMN，生态完整
- **缺点**：重型，JVM 依赖，学习曲线陡峭
- **Jev 集成**：✅ 可在 DRL 中调用 Jev REST API 作为自定义条件函数

### 2. LiteFlow
- **仓库**：https://github.com/dromara/liteflow
- **Stars**：3.9k+  **语言**：Java（Spring Boot）
- **核心特性**：组件化、可热部署、DSL 编排、支持 AI Agent 节点
- **优点**：轻量，AI 组件化原生支持，国内生态活跃
- **缺点**：Java 生态，与我们 Go 技术栈不匹配
- **Jev 集成**：✅✅ 非常好，可将 Jev 封装为 Condition 组件

### 3. json-rules-engine ⭐ 核心参考
- **仓库**：https://github.com/CacheControl/json-rules-engine
- **Stars**：3.1k+  **语言**：JavaScript
- **核心特性**：JSON DSL 描述规则，嵌套条件，自定义操作符
- **优点**：设计简洁，JSON 规则天然可视化/配置化，前后端通用
- **缺点**：JS 实现，需用 Go 重新实现
- **Jev 集成**：✅✅ 自定义 Operator 直接调用 Jev API
- **结论**：**SnakeRule 核心 DSL 设计参考此方案**

### 4. Grule Rule Engine
- **仓库**：https://github.com/hyperjumptech/grule-rule-engine
- **Stars**：2.5k+  **语言**：Go ✅
- **核心特性**：Go 原生，类 Drools 的 GRL 语法，RETE 算法
- **优点**：Go 技术栈，性能好
- **缺点**：GRL 语法复杂，不如 JSON DSL 对业务人员友好
- **Jev 集成**：✅ 可在规则函数中调用 Jev HTTP API
- **结论**：**可参考其 RETE 实现，但 DSL 层用 JSON 方案替代**

### 5. expr-lang/expr
- **仓库**：https://github.com/expr-lang/expr
- **Stars**：8k+  **语言**：Go ✅
- **核心特性**：类型安全表达式引擎，字节码编译，极快
- **优点**：Stars 最高，Go 生态，执行性能优秀
- **缺点**：只是表达式引擎，不是完整规则引擎框架
- **结论**：**作为 SnakeRule 条件表达式求值的底层引擎**

### 6. Easy Rules
- **仓库**：https://github.com/j-easy/easy-rules
- **Stars**：5.3k+  **语言**：Java
- **特点**：极简 API，注解驱动
- **结论**：Java 生态，参考其简洁 API 设计理念

### 7. eKuiper (LF Edge)
- **仓库**：https://github.com/lf-edge/ekuiper
- **Stars**：1.7k+  **语言**：Go ✅
- **特点**：边缘端流处理 + 规则引擎，SQL 语法
- **结论**：IoT 特化场景，SnakeRule 不适用

---

## 三、选型决策

### SnakeRule 技术路线

```
用户界面（可视化规则构建器）
         ↓
  JSON DSL 规则描述（参考 json-rules-engine）
         ↓
  Go 规则引擎 Core
  ┌────────────────────────────────┐
  │  条件求值层                     │
  │  ┌─────────────┐ ┌──────────┐  │
  │  │ 传统条件求值 │ │ Jev AI节点│  │
  │  │(expr-lang)  │ │(HTTP调用) │  │
  │  └─────────────┘ └──────────┘  │
  │                                │
  │  Action 执行层                  │
  │  Webhook / Tag / Log / Event   │
  └────────────────────────────────┘
         ↓
  决策结果 + 命中日志
```

### 核心依赖选型

| 层次 | 选型 | 理由 |
|------|------|------|
| 规则描述 DSL | JSON（自定义 Schema） | 参考 json-rules-engine，可视化友好 |
| 表达式求值 | expr-lang/expr | Go 原生，性能好，类型安全 |
| AI 决策 | Jev API（HTTP） | 概率化输出，适合混合决策 |
| 规则存储 | PostgreSQL JSONB | 天然支持 JSON 查询与版本管理 |
| 规则缓存 | Redis | 毫秒级热加载 |
| 架构参考 | Grule（RETE） | Go 实现参考 |

---

## 四、Jev 集成方案

### 规则中的 Jev 节点

```json
{
  "fact": "jev.riskScore",
  "operator": "greaterThan",
  "value": 0.75,
  "jevConfig": {
    "model": "jev-v1",
    "prompt": "evaluate order risk",
    "timeout": 200
  }
}
```

### 执行流程

```
1. 规则引擎识别 jev.* 前缀的 fact
2. 调用 Jev API，传入当前 facts 上下文
3. Jev 返回概率评分（0-1）
4. 规则引擎用评分与 value 进行 operator 比较
5. 决策结果写入日志（含 Jev 评分详情）
```

### 降级策略

- Jev API 超时（默认 200ms）→ 使用兜底规则
- Jev 服务不可用 → fallback 到纯传统规则模式
- 支持 Jev 评分结果缓存（TTL 可配）

---

## 五、竞品参考

### shaiwz.com 规则引擎产品

> 待补充（页面为 SPA，需要登录后调研）

---

## 六、结论

SnakeRule 采用**自研 Go 规则引擎**方案：
- **DSL 层**：参考 json-rules-engine 的 JSON Schema 设计
- **执行层**：基于 expr-lang/expr 实现高性能表达式求值
- **AI 层**：集成 Jev API 作为智能条件节点
- **架构层**：参考 Grule 的 Go 实现模式

这样既保持了 Go 技术栈的一致性，又获得了 JSON DSL 的业务友好性，以及 Jev AI 的智能决策能力。

---

## 参考资料

- [json-rules-engine](https://github.com/CacheControl/json-rules-engine)
- [grule-rule-engine](https://github.com/hyperjumptech/grule-rule-engine)
- [expr-lang/expr](https://github.com/expr-lang/expr)
- [LiteFlow](https://github.com/dromara/liteflow)
- [Apache KIE/Drools](https://github.com/apache/incubator-kie)
- [Jev 论文 arXiv:2610.19619](https://arxiv.org/abs/2610.19619)
