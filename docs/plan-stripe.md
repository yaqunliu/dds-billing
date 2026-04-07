# 接入 Stripe 支付方案

## Context

团队基于 Sub2API v0.1.103 搭建了 AI API 中转平台，需要接入支付实现用户自助充值。通过香港/美国身份注册 Stripe 账户，利用 Stripe 内置的支付宝/微信通道服务国内用户。使用 Sub2ApiPay 作为充值网关，原生支持 Stripe。

## 架构链路

```
用户点击充值 → Sub2API → iframe 嵌入 Sub2ApiPay 充值页
    → Sub2ApiPay 创建 Stripe Checkout Session
    → 跳转 Stripe 收银台（支持支付宝/微信/信用卡）
    → 用户选择支付方式完成付款
    → Stripe Webhook 回调 Sub2ApiPay
    → Sub2ApiPay 调用 Sub2API 管理接口完成充值
```

## 费率


| 支付方式    | 费率（美国账户）       | 费率（香港账户）         |
| ------- | -------------- | ---------------- |
| 信用卡/借记卡 | 2.9% + $0.30/笔 | 3.4% + HK$2.35/笔 |
| 支付宝     | 2.2% + $0.35/笔 | 3.4% + HK$2.35/笔 |
| 微信支付    | 2.2% + $0.35/笔 | 3.4% + HK$2.35/笔 |


---

## Step 1: 注册 Stripe 账户

1. 访问 [stripe.com](https://stripe.com) 注册账户
2. 使用香港或美国身份完成企业/个人验证：
  - 香港：需要香港身份证或护照 + 香港银行账户
  - 美国：需要 SSN/EIN + 美国银行账户
3. 提交审核，通常 1-2 个工作日通过

## Step 2: 开启支付宝和微信支付

1. 登录 Stripe Dashboard
2. 进入 **Settings → Payment methods**
3. 找到 **Alipay** 和 **WeChat Pay**，点击开启
4. 同意相关条款
5. 等待审核通过（约 7 天）

> 审核期间可以先用信用卡通道测试完整流程。

## Step 3: 获取 Stripe API 密钥

在 Stripe Dashboard → **Developers → API keys** 获取：

- **Publishable key**：`pk_live_...`（前端用）
- **Secret key**：`sk_live_...`（后端用，Sub2ApiPay 需要）

同时配置 Webhook：

1. 进入 **Developers → Webhooks**
2. 添加端点：`https://pay.yourdomain.com/api/stripe/webhook`
3. 选择监听事件：`checkout.session.completed`、`payment_intent.succeeded`
4. 获取 **Webhook Secret**：`whsec_...`

## Step 4: 部署 Sub2API v0.1.103（如尚未部署）

```bash
mkdir -p /opt/sub2api && cd /opt/sub2api
# Docker Compose 部署，镜像版本锁定 v0.1.103
# 依赖：PostgreSQL 15+、Redis 7+
docker compose up -d
```

完成初始化向导，从管理后台获取 **Admin API Key**。

## Step 5: 部署 Sub2ApiPay

```bash
mkdir -p /opt/sub2apipay && cd /opt/sub2apipay
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/docker-compose.hub.yml
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/.env.example
cp .env.example .env
```

## Step 6: 配置 .env

```env
# ===== 核心配置 =====
SUB2API_BASE_URL=http://host.docker.internal:9988   # 本地用这个，生产用实际域名
SUB2API_ADMIN_API_KEY=<从 Sub2API 管理后台获取>
ADMIN_TOKEN=<自定义强密码>
NEXT_PUBLIC_APP_URL=https://pay.yourdomain.com       # 支付服务公网地址（必须 HTTPS）

# ===== Stripe 配置 =====
PAYMENT_PROVIDERS=stripe
STRIPE_SECRET_KEY=sk_live_xxxxxxxxxxxxxxxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxxxxxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxxxxx

# ===== 业务参数 =====
MIN_RECHARGE_AMOUNT=1
MAX_RECHARGE_AMOUNT=1000
MAX_DAILY_RECHARGE_AMOUNT=10000
ORDER_TIMEOUT_MINUTES=5
```

### 如果想同时接入 Stripe + 启象支付（双渠道）

```env
PAYMENT_PROVIDERS=stripe,easypay

# Stripe 配置（同上）
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# 启象支付配置
EASY_PAY_PID=<启象支付商户ID>
EASY_PAY_PKEY=<启象支付商户密钥>
EASY_PAY_API_BASE=https://api.payqixiang.cn/
```

## Step 7: 启动并验证服务

```bash
docker compose -f docker-compose.hub.yml up -d
docker compose -f docker-compose.hub.yml logs -f  # 检查启动日志
```

默认端口 3001，Prisma 自动迁移数据库。

## Step 8: 配置反向代理（HTTPS）

Stripe Webhook 回调必须 HTTPS。

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

无域名替代方案：


| 方案                | 命令                                               | 说明          |
| ----------------- | ------------------------------------------------ | ----------- |
| Cloudflare Tunnel | `cloudflared tunnel --url http://localhost:3001` | 免费，自动 HTTPS |
| ngrok             | `ngrok http 3001`                                | 免费版有限制      |


## Step 9: 在 Sub2API 中嵌入支付页

Sub2API 管理后台 → 系统设置 → 配置：

- 充值页面：`https://pay.yourdomain.com/pay`
- 订单查询：`https://pay.yourdomain.com/pay/orders`

## Step 10: 测试验证

### 使用 Stripe 测试模式

Stripe 提供完整的测试环境，**无需真实付款**：

1. 在 .env 中使用测试密钥（`sk_test_...` / `pk_test_...`）
2. Stripe 测试卡号：
  - 成功：`4242 4242 4242 4242`
  - 失败：`4000 0000 0000 0002`
  - 支付宝测试：在测试模式下选择 Alipay 会跳转到模拟页面
3. 验证完整流程：
  - 用户点击充值 → 跳转 Stripe 收银台
  - 选择支付方式 → 完成支付
  - Webhook 回调成功 → 订单 PENDING → PAID → COMPLETED
  - Sub2API 用户余额自动增加
4. 测试通过后，将密钥切换为正式密钥（`sk_live_...` / `pk_live_...`）

### 支付宝/微信通道测试

支付宝/微信通道审核通过后（约 7 天），用真实手机支付宝/微信扫码进行小额测试。

---

## Stripe vs 启象支付对比


| 维度     | Stripe       | 启象支付      |
| ------ | ------------ | --------- |
| 费率     | 2.9% + $0.30 | 1%-2%     |
| 资金安全   | 上市公司，全球持牌    | 第三方小平台    |
| 提现     | 直接到港/美银行账户   | 平台余额再提现   |
| 支付宝/微信 | 内置，后台开启      | 内置，渠道商已签约 |
| 注册门槛   | 需香港/美国身份     | 无门槛       |
| 测试环境   | 完整沙箱，测试卡号    | 无         |
| 适合场景   | 正式运营，资金量大    | 快速上线，小规模  |


## 重要注意事项

- **Stripe 支付宝/微信只支持单次付款**，不支持订阅扣款。充值场景不受影响。
- **货币设置**：如果面向国内用户，建议设置 CNY 为支付货币，避免汇率转换。
- **Webhook 可靠性**：Stripe 会自动重试失败的 Webhook（最多 3 天），比易支付回调更可靠。
- **退款**：Stripe 支持在 Dashboard 直接操作退款，支付宝/微信通道也支持退款。

## 常见问题速查


| 问题          | 排查                                                                        |
| ----------- | ------------------------------------------------------------------------- |
| Webhook 收不到 | 检查 HTTPS 配置；确认 Webhook URL 和 Secret 正确；Stripe Dashboard → Webhooks 查看投递日志 |
| 支付宝/微信不可用   | 确认已在 Dashboard 开启且审核通过；检查是否在测试模式                                          |
| 余额未到账       | 检查 `SUB2API_ADMIN_API_KEY`；查看 Sub2ApiPay 日志                               |
| 货币不匹配       | 确认 Stripe 和 Sub2ApiPay 的货币设置一致                                            |


