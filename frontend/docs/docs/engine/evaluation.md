# 决策试运行

管理端规则集设计页面提供模拟运行面板。输入与规则条件对应的 facts JSON，执行后查看是否命中、命中的规则和返回的声明动作。

## 求值已保存规则集

开发环境先通过登录接口获得 Cookie，再调用 `POST /api/v1/engine/evaluate`：

```bash
curl -c /tmp/snakerule.cookies http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"<AUTH_EMAIL>","password":"<AUTH_PASSWORD>"}'

curl -b /tmp/snakerule.cookies http://localhost:8081/api/v1/engine/evaluate \
  -H 'Content-Type: application/json' \
  -d '{
    "rulesetId": "order-risk",
    "facts": { "order": { "amount": 12000 }, "customer": { "isNew": true } }
  }'
```

将占位符替换为 `backend/.env` 中配置的管理员邮箱和密码。规则管理与决策接口需要有效登录会话，浏览器工作台会自动管理 Cookie。

## 试运行编辑草稿

可在同一请求中增加 `ruleset` 对象，服务端会校验并直接对该对象求值。`ruleset.id` 必须与 `rulesetId` 相同。草稿不会写入规则仓储，也不会覆盖已保存规则。

每次求值按规则优先级从高到低处理启用规则，响应包括 `matched`、`hitRules` 和命中规则携带的 `actions`。动作目前只是声明数据，不会执行其业务副作用。设计器模拟命中时，配置的站内通知渠道会生成一条带“模拟”标记的通知；模拟不会发送外部 Webhook。每次求值会记录调用日志。

## 如何判断规则是否触发

HTTP `200` 表示求值请求执行成功。响应中的 `matched` 表示是否有启用规则条件匹配，`hitRules` 列出命中的规则：

- `matched: true` 且 `hitRules` 非空：调用成功，并至少命中一条规则。
- `matched: false` 且 `hitRules` 为空：调用成功，但没有规则条件匹配。检查事实字段路径、值类型和规则启用状态。
- HTTP 非 `2xx`：请求、鉴权或规则集校验失败，查看响应错误信息。

以电商演示规则为例，`order.amount` 大于 `10000` 且 `customer.isNew` 为 `true` 时，应命中“大额新客订单审核”。成功响应示例：

```json
{
  "rulesetId": "order-risk",
  "version": "1.0.0",
  "matched": true,
  "hitRules": [
    {
      "ruleId": "large-new-order",
      "name": "大额新客订单审核",
      "priority": 200,
      "actions": [{ "type": "tag", "config": { "label": "MANUAL_REVIEW" } }]
    }
  ]
}
```

项目详情页的“调用日志”按页展示最近求值结果，包括命中、未命中、HTTP 状态码、凭据类型、来源和耗时。设计器模拟标记为“设计器模拟”，项目 API Key/JWT 标记为“真实 API 调用”。日志不保存请求中的 `facts` 或鉴权凭据。列表支持 `page`、`size` 和 `is_pager`，响应使用 `list` 与 `pagination` 字段。
