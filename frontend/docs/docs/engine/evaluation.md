# 决策试运行

管理端规则集页面提供模拟运行面板。输入与规则条件对应的事实 JSON，执行后可以查看是否命中、命中的规则和返回动作。

API 客户端可调用 `POST /api/v1/engine/evaluate`：

```bash
curl -X POST http://localhost:8081/api/v1/engine/evaluate \
  -H 'Content-Type: application/json' \
  -H 'Cookie: snakerule_session=<登录后获得的会话 Cookie>' \
  -d '{
    "rulesetId": "order-risk",
    "facts": { "order": { "amount": 12000 }, "customer": { "isNew": true } }
  }'
```

规则管理与决策接口需要有效登录会话。可先通过 `POST /api/v1/auth/login` 登录；浏览器工作台会自动管理会话 Cookie。演练不会写入执行日志，返回的动作仅描述匹配结果。
