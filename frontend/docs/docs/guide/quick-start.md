# 快速开始

## 环境要求

- Docker Desktop（包含 Docker Compose）
- Make

## 启动开发环境

在项目根目录执行：

```bash
make dev-init
make dev-up
```

首次启动会构建 Go API、React 管理端、文档站点和 Nginx。服务在前台运行，按 `Ctrl+C` 停止；也可以另开终端执行 `make dev-down`。

| 服务 | 地址 | 用途 |
| --- | --- | --- |
| 管理端 | `http://localhost:8080` | 登录并管理规则集 |
| 文档站点 | `http://localhost:8080/docs/` | 阅读使用与开发文档 |
| API | `http://localhost:8081` | 直接访问开发 API |
| API 健康检查 | `http://localhost:8080/healthz` | 检查服务状态 |

也可以用 `make dev-open` 打开管理端，用 `make docs-open` 打开文档。`make dev-status` 查看容器状态。

登录账号通过 `backend/.env` 中的 `AUTH_EMAIL`、`AUTH_NAME` 和 `AUTH_PASSWORD` 配置。`make dev-init` 会从示例文件创建本地配置；请勿将真实密码提交到代码仓库。

## 常用命令

```bash
make dev-up       # 启动开发环境
make dev-down     # 停止开发环境
make dev-build    # 构建开发镜像
make dev-logs     # 查看所有服务日志
make docs-build   # 构建静态文档站点
make check-backend # 检查并构建 Go 服务
make gen-sdk      # 根据 OpenAPI 更新 TypeScript SDK
```

## 数据说明

当前开发版本将规则集和登录会话保存在 API 进程内存中，重启后数据会被清空。PostgreSQL 与 Redis 是可选基础设施，目前不是规则存储的后端。
