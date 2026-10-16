# API 概览

开发环境 API 默认地址为 `http://localhost:8081`，也可以通过 Nginx 统一入口 `http://localhost:8080/api/v1/` 访问。健康检查地址是 `/healthz`，就绪检查地址是 `/readyz`，版本信息地址是 `/api/v1/meta`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 邮箱和密码登录，设置会话 Cookie |
| `GET` | `/api/v1/auth/session` | 查询当前会话 |
| `POST` | `/api/v1/auth/logout` | 退出登录并清除会话 |
| `GET` | `/api/v1/organizations` | 列出组织并返回当前选择 |
| `POST` | `/api/v1/organizations` | 创建组织，可指定初始管理员 `admin_user_id` |
| `PUT` | `/api/v1/organizations/active` | 切换当前会话的组织 |
| `PUT` | `/api/v1/organizations/{id}` | 更新组织 |
| `DELETE` | `/api/v1/organizations/{id}` | 删除非默认且非当前组织 |
| `GET` | `/api/v1/projects` | 列出当前组织项目和当前选择 |
| `POST` | `/api/v1/projects` | 创建空项目或电商演示项目（`template: ecommerce`） |
| `PUT` | `/api/v1/projects/active` | 切换当前项目 |
| `PUT` / `DELETE` | `/api/v1/projects/{id}` | 更新或删除项目 |
| `GET` / `POST` | `/api/v1/projects/{id}/credentials` | 列出或创建项目 API Key/JWT；新凭据仅创建时返回 |
| `DELETE` | `/api/v1/projects/{id}/credentials/{keyID}` | 撤销项目 API Key |
| `GET` | `/api/v1/rulesets` | 列出规则集 |
| `POST` | `/api/v1/rulesets` | 创建或保存规则集 |
| `GET` | `/api/v1/rulesets/{id}` | 查看规则集 |
| `PUT` | `/api/v1/rulesets/{id}` | 更新规则集 |
| `DELETE` | `/api/v1/rulesets/{id}` | 删除规则集 |
| `POST` | `/api/v1/engine/evaluate` | 求值已保存规则集，或携带 `ruleset` 预览编辑草稿 |
| `GET` | `/api/v1/platform/license` | 查看授权状态 |
| `POST` | `/api/v1/platform/license` | 上传并校验授权文件 |
| `DELETE` | `/api/v1/platform/license` | 清除授权 |
| `GET` | `/api/v1/organizations/{organizationID}/users` | 列出当前组织用户（组织管理员） |
| `POST` | `/api/v1/organizations/{organizationID}/users` | 创建当前组织用户 |
| `PUT` / `DELETE` | `/api/v1/organizations/{organizationID}/users/{id}` | 更新或移出当前组织用户 |
| `GET` / `POST` | `/api/v1/organizations/{organizationID}/alert-channels` | 列出或创建当前组织告警渠道 |
| `PUT` / `DELETE` | `/api/v1/organizations/{organizationID}/alert-channels/{id}` | 更新或删除当前组织告警渠道 |
| `POST` | `/api/v1/organizations/{organizationID}/alert-channels/{id}/test` | 发送测试通知 |
| `GET` / `POST` | `/api/v1/projects/{id}/members` | 查看或添加项目成员 |
| `PUT` / `DELETE` | `/api/v1/projects/{id}/members/{userID}` | 修改项目角色或移除成员 |
| `GET` | `/api/v1/projects/{id}/member-candidates` | 列出可加入该项目的组织用户 |

授权管理接口要求超级管理员会话；组织用户和告警渠道由当前组织管理员管理。组织管理员可访问所属组织的全部项目；项目管理员可管理自己所属项目的成员与规则；项目成员可以访问获授权项目。规则集按当前组织和项目隔离。求值接口可使用会话或项目 API Key；Key 请求必须携带 `X-API-Key`，且只能读取该 Key 所属项目的已保存规则集。携带 `ruleset` 的草稿求值仍要求管理台会话，且不会触发项目告警。项目可选择当前组织设置配置的告警渠道；已保存规则命中后异步通知。请求与响应字段请以 `backend/server/docs/openapi.json` 为准。

授权管理接口要求有效管理员会话，授权凭证由服务端处理。项目 API Key 和 JWT 当前仅有求值权限；API Key 服务端只保存摘要且支持撤销。JWT 支持平台签发（默认 30 天，可传 `expires_at`）与项目自签（`signing_key: true`，JWT Header 使用 `kid`，Payload 必须包含 `exp`）；签名密钥加密保存且创建时只返回一次。系统未接入 OAuth/OIDC。决策动作仍只作为结果返回，不会执行。
