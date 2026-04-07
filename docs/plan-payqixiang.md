# 接入启象支付 (PayQiXiang) 方案

## Context

团队基于 Sub2API v0.1.103 搭建了 AI API 中转平台，需要接入支付实现用户自助充值。因无企业资质和 ICP 备案，选择通过启象支付（易支付渠道商）接入，借用其资质走支付宝/微信通道。使用 Sub2ApiPay 作为充值网关，原生支持易支付协议。

## 架构链路

```
用户点击充值 → Sub2API → iframe 嵌入 Sub2ApiPay 充值页
    → Sub2ApiPay 生成订单 → 跳转启象支付 (api.payqixiang.cn)
    → 用户扫码付款（支付宝/微信）
    → 启象支付回调 Sub2ApiPay
    → Sub2ApiPay 调用 Sub2API 管理接口完成充值
```

---

## Step 1: 注册启象支付商户

1. 访问启象支付平台注册商户账号
2. 获取三个关键凭证：
  - **PID**（商户 ID）
  - **PKey**（商户密钥）
  - **API 地址**：`https://api.payqixiang.cn/`
3. 确认支持的支付方式（支付宝/微信）已开通

## Step 2: 部署 Sub2API v0.1.103（如尚未部署）

```bash
mkdir -p /opt/sub2api && cd /opt/sub2api
# 使用 Docker Compose 部署，镜像版本锁定 v0.1.103
# 依赖：PostgreSQL 15+、Redis 7+

docker compose up -d
```

完成初始化向导，从管理后台获取 **Admin API Key**（Sub2ApiPay 需要用它来调接口充值）。

## Step 3: 部署 Sub2ApiPay

```bash
mkdir -p /opt/sub2apipay && cd /opt/sub2apipay
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/docker-compose.hub.yml
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/.env.example
cp .env.example .env
```

## Step 4: 配置 .env

```env
# ===== 核心配置 =====
SUB2API_BASE_URL=http://host.docker.internal:9988   # 本地用这个，生产用实际域名
SUB2API_ADMIN_API_KEY=<从 Sub2API 管理后台获取>
ADMIN_TOKEN=<自定义强密码>
NEXT_PUBLIC_APP_URL=https://pay.yourdomain.com       # 支付服务公网地址

# ===== 启象支付配置 =====
PAYMENT_PROVIDERS=easypay
EASY_PAY_PID=<启象支付商户ID>
EASY_PAY_PKEY=<启象支付商户密钥>
EASY_PAY_API_BASE=https://api.payqixiang.cn/

# ===== 业务参数 =====
MIN_RECHARGE_AMOUNT=1
MAX_RECHARGE_AMOUNT=1000
MAX_DAILY_RECHARGE_AMOUNT=10000
ORDER_TIMEOUT_MINUTES=5
```

## Step 5: 启动并验证服务

```bash
docker compose -f docker-compose.hub.yml up -d
docker compose -f docker-compose.hub.yml logs -f  # 检查启动日志
```

默认端口 3001，Prisma 自动迁移数据库。

## Step 6: 配置反向代理（HTTPS）

回调必须 HTTPS。如果没有域名，可用 Cloudflare Tunnel 或 ngrok 替代。

Nginx 示例：

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

## Step 7: 在 Sub2API 中嵌入支付页

Sub2API 管理后台 → 系统设置 → 配置：

- 充值页面：`https://pay.yourdomain.com/pay`
- 订单查询：`https://pay.yourdomain.com/pay/orders`

Sub2API 自动通过 iframe 嵌入并传递 `user_id`、`token`、`theme`、`lang` 参数。

## Step 8: 端到端测试

1. 访问 `https://pay.yourdomain.com/admin?token=<ADMIN_TOKEN>`，确认管理后台正常
2. 用普通用户登录 Sub2API，点击充值，确认支付页面加载
3. 完成 1 元小额支付，验证：
  - 跳转到启象支付收银台正常
  - 支付完成回调成功（订单 PENDING → PAID → COMPLETED）
  - Sub2API 用户余额自动增加
4. 在管理后台查看订单详情和回调日志

---

## 无域名替代方案

如果没有域名和 HTTPS 证书，可用以下方式暴露本地服务：


| 方案                | 命令                                               | 说明          |
| ----------------- | ------------------------------------------------ | ----------- |
| Cloudflare Tunnel | `cloudflared tunnel --url http://localhost:3001` | 免费，自动 HTTPS |
| ngrok             | `ngrok http 3001`                                | 免费版有限制      |


拿到的 HTTPS 地址填入 `NEXT_PUBLIC_APP_URL` 和 Sub2API 配置中即可。

## 常见问题速查


| 问题           | 排查                                              |
| ------------ | ----------------------------------------------- |
| 余额未到账        | 检查 `SUB2API_ADMIN_API_KEY` 是否正确；看 Sub2ApiPay 日志 |
| 回调失败         | HTTPS 是否正常；`NEXT_PUBLIC_APP_URL` 是否与实际地址一致      |
| 订单卡在 PENDING | 启象支付后台检查回调地址；Nginx 日志是否收到回调                     |
| iframe 空白    | 浏览器控制台查跨域错误；检查 `X-Frame-Options` 头              |


