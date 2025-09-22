#  个人测试环境部署指导
## 前置条件

- Docker
- Docker Compose
- Go 1.22+

## 使用docker compose 部署

### 目录位置

- Compose 文件：`deploy/dev/docker-compose.yml`
- 环境变量：仓库根目录创建 `.env`
- 脚本：`scripts/`
- 
### 配置 `.env`

在仓库根目录新建 `.env`（可根据需要调整端口/账号）：

```env
APP_PORT=8080
APP_ENV=dev

MYSQL_ROOT_PASSWORD=root
MYSQL_DB=spike
MYSQL_USER=spike
MYSQL_PASSWORD=spike
MYSQL_PORT=3306

REDIS_PORT=6379

RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_AMQP_PORT=5672
RABBITMQ_MGMT_PORT=15672
```

说明：未在 `.env` 指定的变量，Compose 会使用 `deploy/dev/docker-compose.yml` 中的默认值。

### 启动/停止（使用脚本）

Linux/macOS（bash）：

```bash
# 启动依赖
bash scripts/linux/dev-up.sh

# 停止并清理（含数据卷）
bash scripts/linux/dev-down.sh
```


### 运行应用与验证

启动应用：

```bash
go run ./cmd/spike-api
```

健康检查：

```bash
curl -s http://localhost:8080/healthz
```

RabbitMQ 管理台：`http://localhost:15672`（默认账号密码 `guest/guest`）

可选连通性自检：

```bash
# MySQL（需要本地安装 mysql 客户端）
mysql -h127.0.0.1 -P3306 -uspike -pspike -e "select 1" spike

# Redis（需要本地安装 redis-cli）
redis-cli -p 6379 ping
```


### 常见问题

- 端口占用：修改 `.env` 中的端口或释放本机占用端口后重启。
- 数据重置：执行 `docker compose -f deploy/dev/docker-compose.yml down -v` 清理数据卷后再 `up`。