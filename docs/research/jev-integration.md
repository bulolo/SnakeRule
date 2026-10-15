# Jev AI 决策模型集成方案

> 文档版本：v0.1
> 更新时间：2026-10

---

## 什么是 Jev？

Jev 是由前 OpenAI 研究员推出的 **System One 快速决策 AI 模型**，专为结构化、高频的实时决策场景设计。

**核心特点**：
- ⚡ **极低延迟**：针对快速决策优化，响应时间 < 100ms
- 📊 **概率输出**：返回置信度分布而非单一答案，便于阈值控制
- 🧠 **认知偏差研究**：[arXiv 2610.19619](https://arxiv.org/abs/2610.19619) 表明其在标准认知偏差测试中表现优于人类
- 🔌 **API 友好**：标准 REST 接口，OpenAI 兼容

---

## 集成架构

### 混合决策模型

```
传统规则引擎          Jev AI 节点
     │                    │
     ▼                    ▼
 确定性决策          概率性评分
（0 or 1）          （0.0 ~ 1.0）
     │                    │
     └─────────┬──────────┘
               ▼
         最终决策结果
```

### 在规则 DSL 中使用 Jev

```json
{
  "name": "内容风险评估",
  "conditions": {
    "all": [
      {
        "fact": "content.length",
        "operator": "greaterThan",
        "value": 10
      },
      {
        "fact": "jev.contentRisk",
        "operator": "greaterThan",
        "value": 0.8,
        "jevConfig": {
          "model": "jev-v1",
          "input": {
            "text": "{{content.text}}",
            "context": "content moderation"
          },
          "timeout_ms": 200,
          "fallback": 0.5
        }
      }
    ]
  },
  "actions": [
    { "type": "reject", "reason": "HIGH_RISK_CONTENT" }
  ]
}
```

---

## Go 实现方案

### Jev 节点接口定义

```go
// engine/jev/client.go
package jev

type Client interface {
    Evaluate(ctx context.Context, req *EvalRequest) (*EvalResponse, error)
}

type EvalRequest struct {
    Model   string                 `json:"model"`
    Input   map[string]interface{} `json:"input"`
    Context string                 `json:"context,omitempty"`
}

type EvalResponse struct {
    Score       float64            `json:"score"`       // 0.0 ~ 1.0
    Confidence  float64            `json:"confidence"`
    Reasoning   string             `json:"reasoning,omitempty"`
    Latency_ms  int64              `json:"latency_ms"`
}
```

### Fact 解析器集成

```go
// engine/evaluator/jev_resolver.go
func (r *JevResolver) ResolveFact(factName string, facts map[string]interface{}, cfg *JevConfig) (interface{}, error) {
    if !strings.HasPrefix(factName, "jev.") {
        return nil, ErrNotJevFact
    }

    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutMs)*time.Millisecond)
    defer cancel()

    resp, err := r.client.Evaluate(ctx, &EvalRequest{
        Model: cfg.Model,
        Input: r.buildInput(facts, cfg.Input),
    })
    if err != nil {
        return cfg.Fallback, nil // 降级返回兜底值
    }

    return resp.Score, nil
}
```

---

## 配置说明

```yaml
# .env / config.yaml
jev:
  api_url: "https://api.jev.so/v1"
  api_key: "${JEV_API_KEY}"
  default_model: "jev-v1"
  default_timeout_ms: 200
  cache:
    enabled: true
    ttl_seconds: 60
  fallback:
    enabled: true
    default_score: 0.5
```

---

## 注意事项

1. **延迟控制**：Jev 调用增加 50-200ms 延迟，需设置合理超时
2. **缓存策略**：相同输入的 Jev 结果可缓存 60s，降低 API 调用量
3. **降级处理**：Jev 不可用时使用 fallback 值，保证规则引擎可用性
4. **成本控制**：监控 Jev API 调用频率，必要时加入请求限流
5. **可观测性**：记录每次 Jev 调用的评分、延迟和降级事件
