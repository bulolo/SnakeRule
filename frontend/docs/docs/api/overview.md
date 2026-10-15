# API 概览

开发环境 API 默认地址为 `http://localhost:8081`，也可以通过 Nginx 统一入口 `http://localhost:8080/api/` 访问。健康检查地址是 `/healthz`。

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
| `GET` | `/api/v1/rulesets` | 列出规则集 |
| `POST` | `/api/v1/rulesets` | 创建或保存规则集 |
| `GET` | `/api/v1/rulesets/{id}` | 查看规则集 |
| `PUT` | `/api/v1/rulesets/{id}` | 更新规则集 |
| `DELETE` | `/api/v1/rulesets/{id}` | 删除规则集 |
| `POST` | `/api/v1/engine/evaluate` | 求值已保存规则集，或携带 `ruleset` 预览编辑草稿 |
| `GET` | `/api/v1/platform/license` | 查看授权状态 |
| `POST` | `/api/v1/platform/license` | 上传并校验授权文件 |
| `DELETE` | `/api/v1/platform/license` | 清除授权 |

规则管理、工作空间和求值请求要求有效登录会话。规则集按当前会话选择的工作空间隔离；删除工作空间会同时删除其中的规则集。默认工作空间的 slug 固定为 `default`，默认空间和当前空间不能删除。求值请求携带 `ruleset` 时会按请求内容校验并运行草稿，不写入规则存储；不携带时按 `rulesetId` 读取已保存规则集。请求与响应字段请以仓库中的 `backend/server/docs/openapi.json` 为准；管理端 TypeScript 客户端可通过 `make gen-sdk` 重新生成。
