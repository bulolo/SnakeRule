# PostgreSQL 备份与恢复

本文适用于生产 Docker Compose。Kubernetes 部署可在 PostgreSQL Pod 内运行 `pg_dump` 并将归档流出到受控存储；不要把普通 PVC 快照默认视为 PostgreSQL 一致性备份。定期执行恢复演练；仅有备份文件而未验证恢复，不能视为可恢复。

## 备份

在仓库根目录执行。归档文件包含 schema 与数据，采用 PostgreSQL 自定义格式：

```bash
mkdir -p backups
docker compose -p snakerule-prod --project-directory . \
  --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml \
  exec -T database pg_dump -U snakerule -d snakerule -Fc \
  > "backups/snakerule-$(date +%Y%m%d-%H%M%S).dump"
```

确认命令成功，并检查归档目录可读：

```bash
pg_restore --list backups/snakerule-YYYYMMDD-HHMMSS.dump >/dev/null
```

备份应复制到与运行主机分离的受控存储，并按组织要求设置加密、访问权限和保留周期。不要把备份文件或 `deploy/.env.prod` 提交到 Git。

Kubernetes 环境可从仓库外执行以下命令，将 `<namespace>` 和 `<postgres-pod>` 替换为实际值：

```bash
mkdir -p backups
kubectl exec -n <namespace> <postgres-pod> -- \
  pg_dump -U snakerule -d snakerule -Fc \
  > "backups/snakerule-$(date +%Y%m%d-%H%M%S).dump"
```

## 恢复演练

优先恢复到隔离的临时 PostgreSQL 数据库或临时环境。目标数据库必须为空：

```bash
createdb snakerule_restore_check
pg_restore --no-owner --no-privileges \
  --dbname snakerule_restore_check \
  backups/snakerule-YYYYMMDD-HHMMSS.dump
```

启动与备份版本对应的 SnakeRule，检查 `/readyz`，并抽查组织、项目、规则集、成员、凭据记录、通知和调用日志。确认后删除演练数据库：

```bash
dropdb snakerule_restore_check
```

## 生产恢复

恢复会替换数据库内容。先安排维护窗口、停止 API 写入，并额外备份当前数据库；确认目标归档、版本和目标环境后再执行。

```bash
docker compose -p snakerule-prod --project-directory . \
  --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml stop backend

docker compose -p snakerule-prod --project-directory . \
  --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml \
  exec -T database dropdb -U snakerule --if-exists snakerule

docker compose -p snakerule-prod --project-directory . \
  --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml \
  exec -T database createdb -U snakerule -O snakerule snakerule

cat backups/snakerule-YYYYMMDD-HHMMSS.dump | \
  docker compose -p snakerule-prod --project-directory . \
    --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml \
    exec -T database pg_restore -U snakerule -d snakerule \
      --no-owner --no-privileges

docker compose -p snakerule-prod --project-directory . \
  --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml up -d backend
```

检查 API 健康状态和关键业务数据后，再恢复对外流量。若新版本迁移已改变 schema，使用兼容该归档的应用版本完成恢复，再按升级流程迁移；不要把 Goose `Down` 当作生产回滚方案。

## 发布前数据库检查

- 所有 schema 变更都新增顺序迁移，不修改已应用的迁移文件。
- CI 对空 PostgreSQL 执行迁移，并重复启动迁移入口验证其可重复运行。
- 生产升级前确认最近备份成功、归档可读，并记录恢复所需的应用版本和授权加密密钥。
- 迁移失败时保持旧应用与备份可用；评估并修复原因后再继续升级，不要在生产库手工删除迁移版本记录。
- 初始版本的自动迁移在服务启动时运行。生产升级仍使用单 API 副本，直到仓储并发改造完成。
