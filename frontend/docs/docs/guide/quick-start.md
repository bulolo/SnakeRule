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

首次初始化使用示例配置中的账号和占位密码。只在本机试用时可直接登录；共享环境前请先修改 `AUTH_PASSWORD` 并重建后端容器。登录后请先查看[管理工作台栏目说明](./admin-console.md)。

## 平台授权

登录后可在「平台设置 → 系统信息」查看版本、实例 ID 和授权状态，并上传、替换或清除授权文件。`make dev-init` 会在被 Git 忽略的 `backend/.env` 中生成本地加密密钥；重启或重建容器时该密钥会保留。

需要使用产品授权时，请向授权签发方获取 SnakeRule 产品公钥并配置 `PULSE_PRODUCT_PUBLIC_KEY`。未配置授权时，仍可使用社区版规则能力。

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

开发与生产 Compose 均会启动 PostgreSQL 保存业务数据。生产部署请定期备份数据库。Redis 为可选基础设施。

## 生产 Compose

生产镜像与开发热更新环境分开构建。先在项目根目录生成本地生产配置，再编辑管理员邮箱和 Pulse 产品公钥：

```bash
make prod-init
# 编辑 deploy/.env.prod，设置 AUTH_EMAIL 与 Pulse 签发的 PULSE_PRODUCT_PUBLIC_KEY
make prod-up
```

Nginx 反向代理默认监听 `http://localhost:8080`，管理端和文档分别位于 `/` 与 `/docs/`。生产 Compose 只公开 Nginx 端口，业务数据写入 PostgreSQL 持久化卷；公网部署前请配置 TLS 和数据库备份。当前 API 仍按单副本部署。
