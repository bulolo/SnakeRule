# 规则集与条件

规则集包含基本信息和有序规则列表。规则可设置优先级、启用状态、条件和动作。求值按优先级从高到低执行，并返回匹配规则及其声明动作。

## 条件组合

条件支持 `all`（全部满足）、`any`（至少一个满足）和 `not`（反转子条件）嵌套组合。叶子条件通过事实路径、操作符和值进行比较：

```json
{
  "all": [
    { "fact": "order.amount", "operator": "greaterThan", "value": 10000 },
    { "fact": "customer.isNew", "operator": "equal", "value": true }
  ]
}
```

事实以 JSON 对象传入，点分路径用于读取嵌套字段。支持相等比较、数值比较、集合包含、字符串包含和空值检查等操作符。具体 API 模型以仓库中的 [OpenAPI 契约](https://github.com/bulolo/SnakeRule/blob/main/backend/server/docs/openapi.json)为准。

## 当前范围

规则集保存在内存中。动作作为决策结果返回，目前不会调用 Webhook 或执行外部操作。更多求值示例见[决策试运行](/engine/evaluation)。
