# API 概览

开发环境 API 默认地址为 `http://localhost:8081`，也可以通过 Nginx 统一入口 `http://localhost:8080/api/` 访问。健康检查地址是 `/healthz`，就绪检查地址是 `/readyz`，版本信息地址是 `/api/v1/meta`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 邮箱和密码登录，设置会话 Cookie |
| `GET` | `/api/v1/auth/session` | 查询当前会话 |
| `POST` | `/api/v1/auth/logout` | 退出登录并清除会话 |
| `GET` | `/api/v1/workspaces` | 列出工作空间并返回当前选择 |
| `POST` | `/api/v1/workspaces` | 创建工作空间 |
| `PUT` | `/api/v1/workspaces/active` | 切换当前会话的工作空间 |
| `PUT` | `/api/v1/workspaces/{id}` | 更新工作空间 |
| `DELETE` | `/api/v1/workspaces/{id}` | 删除非默认且非当前工作空间 |
| `GET` | `/api/v1/projects` | 列出当前工作空间项目和当前选择 |
| `POST` | `/api/v1/projects` | 创建空项目或电商演示项目（`template: ecommerce`） |
| `PUT` | `/api/v1/projects/active` | 切换当前项目 |
| `PUT` / `DELETE` | `/api/v1/projects/{id}` | 更新或删除项目 |
| `GET` / `POST` | `/api/v1/projects/{id}/api-keys` | 列出或创建项目 API Key；新密钥仅创建时返回 |
| `DELETE` | `/api/v1/projects/{id}/api-keys/{keyID}` | 撤销项目 API Key |
| `PUT` | `/api/v1/projects/{id}/alerts` | 更新项目告警策略 |
| `GET` | `/api/v1/rulesets` | 列出规则集 |
| `POST` | `/api/v1/rulesets` | 创建或保存规则集 |
| `GET` | `/api/v1/rulesets/{id}` | 查看规则集 |
| `PUT` | `/api/v1/rulesets/{id}` | 更新规则集 |
| `DELETE` | `/api/v1/rulesets/{id}` | 删除规则集 |
| `POST` | `/api/v1/engine/evaluate` | 求值已保存规则集，或携带 `ruleset` 预览编辑草稿 |
| `GET` | `/api/v1/platform/license` | 查看授权状态 |
| `POST` | `/api/v1/platform/license` | 上传并校验授权文件 |
| `DELETE` | `/api/v1/platform/license` | 清除授权 |
| `GET` / `POST` | `/api/v1/system/alert-channels` | 列出或创建平台告警渠道 |
| `PUT` / `DELETE` | `/api/v1/system/alert-channels/{id}` | 更新或删除告警渠道 |
| `POST` | `/api/v1/system/alert-channels/{id}/test` | 发送测试通知 |

管理接口要求有效会话。规则集按当前工作空间和项目隔离。求值接口可使用会话或项目 API Key；Key 请求必须携带 `X-API-Key`，且只能读取该 Key 所属项目的已保存规则集。携带 `ruleset` 的草稿求值仍要求管理台会话，且不会触发项目告警。项目可选择系统设置配置的告警渠道；已保存规则命中后异步通知。请求与响应字段请以 `backend/server/docs/openapi.json` 为准。

授权管理接口要求有效管理员会话，授权凭证由服务端处理。项目 API Key 当前仅有求值权限，服务端只保存摘要且支持撤销；还没有过期和最后使用时间。系统没有 JWT 或 OAuth/OIDC。项目、规则集、密钥、告警渠道和策略均使用进程内存仓储，服务重启后会重置；决策动作仍只作为结果返回，不会执行。
