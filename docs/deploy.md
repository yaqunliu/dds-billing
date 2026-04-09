# dds-billing 部署指南

## 服务器环境要求

- Docker 24+
- Docker Compose v2+
- Git

## 部署步骤

### 1. 登录跳板机，clone 代码

```bash
ssh jumphost
ssh target-server

cd /opt
git clone <repo-url> dds-billing
cd dds-billing
```

### 2. 创建配置文件

```bash
cp configs/config.example.yaml configs/config.yaml
vim configs/config.yaml
```

需要填写的关键配置：

```yaml
database:
  # 容器内连接 MySQL，host 用 docker-compose 的 service 名
  dsn: "root:dds_billing_2026@tcp(mysql:3306)/dds_billing?charset=utf8mb4&parseTime=True&loc=Local"

payment:
  provider: "easypay"  # 或 stripe

# 按实际渠道填写密钥
easypay:
  pid: "xxx"
  pkey: "xxx"
  api_base: "https://api.payqixiang.cn"
  notify_url: "https://your-domain.com/api/notify/easypay"
  return_url: "https://your-domain.com/pay/result"

sub2api:
  base_url: "http://sub2api-host:port"
  admin_api_key: "xxx"
```

> **注意**：数据库 host 必须写 `mysql`（docker-compose service 名），不是 `localhost`。

### 3. 配置 SSL 证书

创建 ssl 目录，放入证书文件：

```bash
mkdir -p ssl
# 将证书文件放入
cp /path/to/fullchain.pem ssl/fullchain.pem
cp /path/to/privkey.pem ssl/privkey.pem
```

#### 使用 Let's Encrypt 免费证书

```bash
# 安装 certbot
apt install certbot

# 先临时停掉 nginx 容器（如果已在运行）
docker compose stop nginx

# 申请证书（standalone 模式，需要 80 端口空闲）
certbot certonly --standalone -d your-domain.com

# 复制证书
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem ssl/
cp /etc/letsencrypt/live/your-domain.com/privkey.pem ssl/
```

### 4. 修改 MySQL 密码（可选）

默认密码在 `docker-compose.yml` 中，可通过环境变量覆盖：

```bash
export MYSQL_ROOT_PASSWORD=your_secure_password
```

同时更新 `configs/config.yaml` 中的 DSN。

### 5. 构建并启动

```bash
docker compose build
docker compose up -d
```

首次启动会自动：
- 创建 MySQL 数据库 `dds_billing`
- Go 后端自动建表（GORM AutoMigrate）

### 6. 验证服务

```bash
# 检查容器状态
docker compose ps

# 检查后端健康
curl http://localhost/health
# 应返回 {"status":"ok"}

# 查看日志
docker compose logs -f backend
docker compose logs -f nginx
```

### 7. 配置 Sub2API

在 Sub2API 管理后台，将充值页面 URL 设置为：

```
https://your-domain.com/pay
```

## 常用运维命令

```bash
# 查看所有容器状态
docker compose ps

# 查看日志（实时）
docker compose logs -f backend
docker compose logs -f nginx
docker compose logs -f mysql

# 重启单个服务
docker compose restart backend

# 更新部署（拉取最新代码后）
git pull
docker compose build
docker compose up -d

# 停止所有服务
docker compose down

# 停止并删除数据（慎用！会丢失数据库数据）
docker compose down -v

# 进入容器调试
docker compose exec backend sh
docker compose exec mysql mysql -u root -p dds_billing
```

## 证书续期

Let's Encrypt 证书有效期 90 天，建议设置 crontab 自动续期：

```bash
# 编辑 crontab
crontab -e

# 添加（每月 1 号凌晨 3 点续期）
0 3 1 * * certbot renew --quiet && cp /etc/letsencrypt/live/your-domain.com/fullchain.pem /opt/dds-billing/ssl/ && cp /etc/letsencrypt/live/your-domain.com/privkey.pem /opt/dds-billing/ssl/ && docker compose -f /opt/dds-billing/docker-compose.yml restart nginx
```

## 目录结构（服务器上）

```
/opt/dds-billing/
├── configs/
│   └── config.yaml          # 实际配置（手动创建）
├── ssl/
│   ├── fullchain.pem        # SSL 证书
│   └── privkey.pem          # SSL 私钥
├── docker-compose.yml
├── Dockerfile
├── nginx/
│   ├── Dockerfile
│   └── nginx.conf
└── ...（其他代码文件）
```
