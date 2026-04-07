# 支付宝官方接入方案

## 概述

通过 Sub2ApiPay 接入支付宝官方支付，支持 PC 扫码和手机端支付。需要企业资质，费率 0.6%，适合正式运营环境。

## 前置条件

- Sub2API v0.1.103 已部署并正常运行
- Docker + Docker Compose 环境就绪
- **企业营业执照**（个人无法申请）
- 域名 + HTTPS 证书
- 支付宝开放平台企业开发者账号

## 第一步：支付宝开放平台准备工作

### 1.1 注册并认证

1. 访问 [支付宝开放平台](https://open.alipay.com)
2. 用企业支付宝账号登录
3. 完成企业实名认证（需营业执照照片、法人信息等）
4. 审核通过后进入下一步（通常 1-3 个工作日）

### 1.2 创建应用

1. 进入「控制台」→「我的应用」→「创建应用」→ 选择「网页/移动应用」
2. 填写应用名称、图标等基本信息
3. 提交审核（通常 1 个工作日）

### 1.3 配置密钥

支付宝使用 RSA2 (SHA256WithRSA) 签名方式，需要生成密钥对：

```bash
# 生成 RSA2 私钥（2048位）
openssl genrsa -out app_private_key.pem 2048

# 从私钥导出公钥
openssl rsa -in app_private_key.pem -pubout -out app_public_key.pem

# 提取公钥内容（去掉头尾标记，用于上传到支付宝）
grep -v "^-" app_public_key.pem | tr -d '\n'
```

在支付宝开放平台配置：
1. 进入应用详情 →「开发设置」→「接口加签方式」
2. 选择「公钥」模式，将上面生成的公钥内容粘贴上传
3. 上传后，支付宝会返回一个**支付宝公钥**，复制保存

最终你需要三个值：
- **APP_ID**：应用详情页顶部
- **应用私钥**：你本地生成的 `app_private_key.pem` 内容
- **支付宝公钥**：支付宝平台返回的公钥

### 1.4 签约支付能力

在应用详情 →「产品绑定」中签约：
- **电脑网站支付**（用于 PC 扫码付）
- **手机网站支付**（用于移动端 H5 付，可选）

签约需要提供网站地址、ICP 备案号等信息。

### 1.5 配置回调地址

在应用的「开发设置」中配置：
- **授权回调地址**：`https://pay.yourdomain.com/api/alipay/notify`

## 第二步：部署 Sub2ApiPay

（如果已按易支付方案部署过，跳到第三步直接改配置）

```bash
mkdir -p /opt/sub2apipay && cd /opt/sub2apipay

curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/docker-compose.hub.yml
curl -O https://raw.githubusercontent.com/touwaeriol/sub2apipay/main/.env.example
cp .env.example .env
```

## 第三步：配置环境变量

编辑 `.env` 文件：

### 核心配置

```env
SUB2API_BASE_URL=http://host.docker.internal:9988
SUB2API_ADMIN_API_KEY=从Sub2API管理后台获取

ADMIN_TOKEN=自定义强密码
NEXT_PUBLIC_APP_URL=https://pay.yourdomain.com
```

### 支付宝配置

```env
# 启用支付宝（可与其他渠道并存，逗号分隔）
PAYMENT_PROVIDERS=alipay

# 支付宝凭证
ALIPAY_APP_ID=你的应用APPID
ALIPAY_PRIVATE_KEY=你的应用私钥内容（一整行，不含头尾标记）
ALIPAY_PUBLIC_KEY=支付宝返回的公钥内容（一整行，不含头尾标记）
```

> **注意**：私钥和公钥都是一整行字符串，不要包含 `-----BEGIN...-----` 头尾，不要有换行。

### 如果需要同时启用易支付 + 支付宝

```env
PAYMENT_PROVIDERS=easypay,alipay
# 然后同时配置两个渠道的变量
```

### 业务参数

```env
MIN_RECHARGE_AMOUNT=1
MAX_RECHARGE_AMOUNT=1000
MAX_DAILY_RECHARGE_AMOUNT=10000
ORDER_TIMEOUT_MINUTES=5
```

## 第四步：启动/重启服务

```bash
# 首次启动
docker compose -f docker-compose.hub.yml up -d

# 如果是修改配置后重启
docker compose -f docker-compose.hub.yml down
docker compose -f docker-compose.hub.yml up -d
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

## 第六步：在 Sub2API 中嵌入

同易支付方案，在 Sub2API 管理后台配置：
- 充值页面：`https://pay.yourdomain.com/pay`
- 订单查询：`https://pay.yourdomain.com/pay/orders`

## 第七步：测试验证

### 沙箱环境测试（推荐先用沙箱）

支付宝提供沙箱环境用于测试：
1. 访问 [沙箱环境](https://open.alipay.com/develop/sandbox/app)
2. 使用沙箱 APP_ID 和沙箱密钥配置
3. 用沙箱版支付宝 App 扫码测试
4. 验证完整流程后再切换到正式环境

### 正式环境测试

1. 用普通用户登录 Sub2API，点击充值
2. 选择支付宝支付，完成小额支付（如 0.01 元，如果平台支持）
3. 确认订单状态流转：PENDING → PAID → COMPLETED
4. 确认 Sub2API 余额自动到账
5. 在管理后台查看订单详情和回调记录

## 常见问题

| 问题 | 排查方向 |
|------|---------|
| 签名验证失败 | 检查私钥/公钥是否正确，是否去掉了头尾标记和换行；确认使用的是 RSA2 而非 RSA |
| 回调未收到 | 确认 HTTPS 正常；确认支付宝开放平台回调地址配置正确 |
| TRADE_NOT_ALLOWED | 检查应用是否已签约对应支付能力；确认应用审核已通过 |
| 余额未到账 | 检查 `SUB2API_ADMIN_API_KEY`；查看 Sub2ApiPay 日志 |
| 手机端无法支付 | 确认已签约「手机网站支付」能力 |

## 配置变量速查

| 变量 | 必填 | 说明 |
|------|------|------|
| `SUB2API_BASE_URL` | 是 | Sub2API 服务地址 |
| `SUB2API_ADMIN_API_KEY` | 是 | Sub2API 管理 API 密钥 |
| `ADMIN_TOKEN` | 是 | 支付后台管理密码 |
| `NEXT_PUBLIC_APP_URL` | 是 | 支付服务公网地址（必须 HTTPS） |
| `PAYMENT_PROVIDERS` | 是 | 包含 `alipay` |
| `ALIPAY_APP_ID` | 是 | 支付宝应用 APP_ID |
| `ALIPAY_PRIVATE_KEY` | 是 | 应用私钥（一整行，无头尾） |
| `ALIPAY_PUBLIC_KEY` | 是 | 支付宝公钥（一整行，无头尾） |

## 支付宝接入 vs 易支付对比

| 维度 | 支付宝官方 | 易支付 |
|------|-----------|--------|
| 准备时间 | 3-5 个工作日（认证+审核+签约） | 几分钟 |
| 密钥配置 | 需要 OpenSSL 生成 RSA2 密钥对 | 平台直接给 PID/PKey |
| 费率 | 0.6% | 1%-3%（取决于平台） |
| 稳定性 | 官方服务，高可用 | 取决于第三方平台 |
| 适用场景 | 正式运营 | 快速测试/个人项目 |
