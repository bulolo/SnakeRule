# API 概览

开发环境 API 默认地址为 `http://localhost:8081`，也可以通过 Nginx 统一入口 `http://localhost:8080/api/` 访问。健康检查地址是 `/healthz`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 邮箱和密码登录，设置会话 Cookie |
| `GET` | `/api/v1/auth/session` | 查询当前会话 |
| `POST` | `/api/v1/auth/logout` | 退出登录并清除会话 |
| `GET` | `/api/v1/rulesets` | 列出规则集 |
| `POST` | `/api/v1/rulesets` | 创建或保存规则集 |
| `GET` | `/api/v1/rulesets/{id}` | 查看规则集 |
| `PUT` | `/api/v1/rulesets/{id}` | 更新规则集 |
| `DELETE` | `/api/v1/rulesets/{id}` | 删除规则集 |
| `POST` | `/api/v1/engine/evaluate` | 对规则集执行求值 |

规则管理和求值请求要求有效登录会话。请求与响应字段请以仓库中的 `backend/server/docs/openapi.json` 为准；管理端 TypeScript 客户端可通过 `make gen-sdk` 重新生成。
