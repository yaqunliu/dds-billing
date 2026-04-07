# 易支付 (EasyPay) 接入方案

## 概述

通过 Sub2ApiPay 接入易支付聚合平台，实现用户自助充值。易支付无需企业资质，注册即用，适合快速上线和测试验证。

## 前置条件

- Sub2API v0.1.103 已部署并正常运行
- Docker + Docker Compose 环境就绪
- 一个易支付平台账号（如彩虹易支付等）
- 域名 + HTTPS 证书（支付回调必须 HTTPS）

## 第一步：注册易支付平台账号

1. 选择一个易支付服务商注册账号
2. 登录后台获取以下凭证：
   - **商户ID (PID)**
   - **商户密钥 (PKey)**
   - **API地址** — 通常是 `https://易支付域名/`
3. 记录这三个值，后续配置要用

## 第二步：部署 Sub2ApiPay

```bash
# 创建工作目录
mkdir -p /opt/sub2apipay && cd /opt/sub2apipay

# 下载 docker-compose 和环境变量模板
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/docker-compose.hub.yml
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/.env.example

# 创建配置文件
cp .env.example .env
```

## 第三步：配置环境变量

编辑 `.env` 文件，填写以下内容：

### 核心配置

```env
# Sub2API 连接（本地 Docker 用 host.docker.internal，生产用实际域名）
SUB2API_BASE_URL=http://host.docker.internal:9988
SUB2API_ADMIN_API_KEY=从Sub2API管理后台获取

# Sub2ApiPay 自身配置
ADMIN_TOKEN=自定义一个强密码
NEXT_PUBLIC_APP_URL=https://pay.yourdomain.com
```

### 易支付配置

```env
# 启用易支付
PAYMENT_PROVIDERS=easypay

# 易支付凭证
EASY_PAY_PID=你的商户ID
EASY_PAY_PKEY=你的商户密钥
EASY_PAY_API_BASE=https://易支付平台域名/

# 回调地址（自动拼接，一般不需要手动改）
EASY_PAY_NOTIFY_URL=${NEXT_PUBLIC_APP_URL}/api/easy-pay/notify
```

### 业务参数（按需调整）

```env
MIN_RECHARGE_AMOUNT=1        # 最低充值金额（元）
MAX_RECHARGE_AMOUNT=1000     # 单笔最高（元）
MAX_DAILY_RECHARGE_AMOUNT=10000  # 每日限额（元）
ORDER_TIMEOUT_MINUTES=5      # 订单超时时间（分钟）
```

## 第四步：启动服务

```bash
docker compose -f docker-compose.hub.yml up -d
```

启动后 Sub2ApiPay 监听 3001 端口。Prisma 会自动执行数据库迁移。

检查是否正常运行：
```bash
docker compose -f docker-compose.hub.yml logs -f
```

## 第五步：配置反向代理 (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name pay.yourdomain.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:3001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

重载 Nginx：
```bash
nginx -t && nginx -s reload
```

## 第六步：在 Sub2API 中嵌入支付页面

登录 Sub2API 管理后台 → 系统设置 → 配置充值相关 URL：

- **充值页面**：`https://pay.yourdomain.com/pay`
- **订单查询**：`https://pay.yourdomain.com/pay/orders`

Sub2API 会自动通过 iframe 嵌入，并传递 `user_id`、`token`、`theme`、`lang` 参数。

## 第七步：测试验证

1. **管理后台测试**：访问 `https://pay.yourdomain.com/admin?token=你的ADMIN_TOKEN`，确认后台可正常访问
2. **支付页面测试**：用普通用户登录 Sub2API，点击充值，确认支付页面正常加载
3. **支付流程测试**：完成一笔小额充值（如 1 元），观察：
   - 易支付跳转是否正常
   - 支付完成后订单状态变化：PENDING → PAID → COMPLETED
   - Sub2API 中用户余额是否自动增加
4. **回调测试**：在 Sub2ApiPay 管理后台查看订单详情，确认回调正常接收

## 常见问题

| 问题 | 排查方向 |
|------|---------|
| 支付后余额未到账 | 检查 `SUB2API_ADMIN_API_KEY` 是否正确；查看 Sub2ApiPay 日志中是否有调用 Sub2API 接口的报错 |
| 回调失败 | 确认 HTTPS 配置正确；确认 `NEXT_PUBLIC_APP_URL` 与实际访问地址一致 |
| 订单一直 PENDING | 检查易支付平台回调地址配置；查看 Nginx 日志是否收到回调请求 |
| iframe 加载空白 | 检查浏览器控制台是否有跨域错误；确认 `X-Frame-Options` 头未阻止嵌入 |

## 配置变量速查

| 变量 | 必填 | 说明 |
|------|------|------|
| `SUB2API_BASE_URL` | 是 | Sub2API 服务地址 |
| `SUB2API_ADMIN_API_KEY` | 是 | Sub2API 管理 API 密钥 |
| `ADMIN_TOKEN` | 是 | 支付后台管理密码 |
| `NEXT_PUBLIC_APP_URL` | 是 | 支付服务公网地址 |
| `PAYMENT_PROVIDERS` | 是 | 设为 `easypay` |
| `EASY_PAY_PID` | 是 | 易支付商户 ID |
| `EASY_PAY_PKEY` | 是 | 易支付商户密钥 |
| `EASY_PAY_API_BASE` | 是 | 易支付 API 地址 |
