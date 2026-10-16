# 规则集与条件

规则集包含 ID、名称、版本、说明、启用状态和有序规则列表。规则可设置 ID、名称、优先级、启用状态、条件和声明动作。保存时服务端校验规则集/规则必填字段、规则 ID 唯一性、条件结构和操作符，并按优先级从高到低稳定排序。

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

事实以 JSON 对象传入，点分路径用于读取嵌套字段。支持 `equal`/`eq`、`notEqual`/`ne`、`greaterThan`/`gt`、`greaterThanInclusive`/`gte`、`lessThan`/`lt`、`lessThanInclusive`/`lte`、`in`、`notIn`、`contains`、`isNull` 和 `isNotNull`。`in`/`notIn` 的比较值须为数组；数值比较要求实际值和比较值可作为数字读取。具体 API 模型以仓库中的 [OpenAPI 契约](https://github.com/bulolo/SnakeRule/blob/main/backend/server/docs/openapi.json)为准。

## 当前范围

规则集和工作空间保存在进程内存中，API 重启后会清空并重新加载默认工作空间和演示规则。动作作为决策结果返回，目前不会调用 Webhook 或执行外部操作。更多求值示例见[决策试运行](/engine/evaluation)。
