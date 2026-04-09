# dds-billing 部署指南

## 服务器环境要求

- Docker 24+
- Docker Compose v2+
- Git
- 80 和 443 端口可用
- 域名已解析到服务器 IP

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
cp .env.example .env
```

编辑 `.env`，设置 MySQL 密码：

```
MYSQL_ROOT_PASSWORD=your_secure_password
```

编辑 `configs/config.yaml`：

```yaml
database:
  # host 写 mysql（docker-compose service 名）
  dsn: "root:your_secure_password@tcp(mysql:3306)/dds_billing?charset=utf8mb4&parseTime=True&loc=Local"

payment:
  provider: "easypay"

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

### 3. 修改 Nginx 域名

编辑 `nginx/nginx.conf`，将 `your-domain.com` 替换为你的实际域名：

```bash
sed -i 's/your-domain.com/pay.example.com/g' nginx/nginx.conf
```

### 4. 申请 SSL 证书

**首次建议用 `--staging` 申请测试证书**，验证流程无误后再申请正式证书：

```bash
# 第一步：测试证书（不消耗额度） ， 邮箱用来收取万一自动续签失败发送的证书过期提醒
./scripts/init-ssl.sh your-domain.com your@email.com --staging

# 确认无误后，删掉测试证书，申请正式证书
rm -rf certbot/
./scripts/init-ssl.sh your-domain.com your@email.com
```

脚本会自动：

1. 创建临时自签名证书 → 启动 Nginx
2. 通过 certbot webroot 方式验证域名
3. 获取 Let's Encrypt 证书
4. 重载 Nginx 使用正式证书

### 5. 验证服务

```bash
# 检查容器状态（应该有 4 个运行中）
docker compose ps

# 检查后端健康
curl https://your-domain.com/health

# 查看日志
docker compose logs -f backend
```

### 6. 配置 Sub2API

在 Sub2API 管理后台，将充值页面 URL 设置为：

```
https://your-domain.com/pay
```

### 7. 设置证书自动续签

Let's Encrypt 证书有效期 90 天。docker-compose 中的 certbot 容器会每 12 小时自动检查续签。

额外建议添加 crontab 作为备份：

```bash
crontab -e
```

添加：

```
0 3 1 * * cd /opt/dds-billing && ./scripts/renew-ssl.sh >> /var/log/dds-billing-ssl-renew.log 2>&1
```

## 常用运维命令

```bash
# 查看所有容器状态
docker compose ps

# 查看日志（实时）
docker compose logs -f backend
docker compose logs -f nginx

# 重启单个服务
docker compose restart backend

# 更新部署（拉取最新代码后）
git pull
docker compose build
docker compose up -d

# 停止所有服务
docker compose down

# 停止并删除数据（慎用！）
docker compose down -v

# 进入容器调试
docker compose exec backend sh
docker compose exec mysql mysql -u root -p dds_billing

# 手动续签证书
./scripts/renew-ssl.sh
```

## 服务器目录结构

```
/opt/dds-billing/
├── configs/config.yaml       # 实际配置
├── .env                      # MySQL 密码
├── certbot/
│   ├── conf/                 # Let's Encrypt 证书（自动生成）
│   └── www/                  # ACME challenge 文件（自动生成）
├── docker-compose.yml
├── Dockerfile
├── nginx/
│   ├── Dockerfile
│   └── nginx.conf            # 需要替换域名
├── scripts/
│   ├── init-ssl.sh           # 首次申请证书
│   └── renew-ssl.sh          # 手动续签证书
└── ...
```

