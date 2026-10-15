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

## 平台授权

登录后可在「平台设置 → 系统信息」查看版本、实例 ID 和授权状态，并上传、替换或清除授权文件。`make dev-init` 会在被 Git 忽略的 `backend/.env` 中生成本地加密密钥；重启或重建容器时该密钥会保留，`LICENSE_STORE_PATH` 默认写入持久化数据卷。

实际校验授权前，需要在 `backend/.env` 配置 Pulse 为 SnakeRule 签发的 `PULSE_PRODUCT_PUBLIC_KEY`。公钥需由授权签发方提供；没有配置时平台仍可运行社区功能，但上传的授权不会被接受。授权凭证由 Pulse Agent 校验，通过后才会加密保存在服务端。当前规则引擎尚无需要商业授权解锁的功能点，因此页面只展示授权状态，不会改变现有规则执行能力。

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

当前开发版本将规则集、工作空间和登录会话保存在 API 进程内存中，重启后数据会被清空。授权密文单独保存在 Docker 持久化数据卷中。PostgreSQL 与 Redis 是可选基础设施，目前不是规则存储的后端。

## 生产 Compose

生产镜像与开发热更新环境分开构建。先在项目根目录生成本地生产配置，再编辑管理员邮箱和 Pulse 产品公钥：

```bash
make prod-init
# 编辑 deploy/.env.prod，设置 AUTH_EMAIL 与 Pulse 签发的 PULSE_PRODUCT_PUBLIC_KEY
make prod-up
```

网关默认监听 `http://localhost:8080`，管理端和文档分别位于 `/` 与 `/docs/`。生产 Compose 只公开 Nginx 网关端口，授权数据使用 Docker 持久化卷；规则集、工作空间和登录会话目前仍在 API 内存中，重启后会清空。当前 Compose 适合单实例试运行，正式承载业务数据前需将这三类数据接入持久化数据库，并在公网前配置 TLS 反向代理。
