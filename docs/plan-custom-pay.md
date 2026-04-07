# 自建支付平台开发 Plan（可插拔支付渠道 + Sub2API）

## Context

团队基于 Sub2API v0.1.103 搭建了 AI API 中转平台。易支付渠道商费率过高（1.58-1.8%），且不安全，决定自建支付服务。采用可插拔的支付渠道架构，支付渠道通过配置文件切换，首期对接蓝兔支付（官方 ISV，费率约 0.6%+0.7%，个人可用，资金官方直清），后续可轻松扩展其他渠道。

项目代号：**dds-billing**（即当前项目目录）

## 设计原则

**支付渠道可插拔** — 所有支付渠道实现统一接口（PaymentProvider），通过配置文件选择使用哪个渠道，新增渠道只需：

1. 新增一个 `internal/payment/xxx/` 目录实现接口
2. 在 `registry.go` 注册
3. 在 `config.yaml` 中配置渠道参数

## 技术栈


| 组件   | 技术                                      |
| ---- | --------------------------------------- |
| 后端   | Go 1.25+ / Gin / GORM                   |
| 前端   | React + TypeScript + Vite + TailwindCSS |
| 数据库  | MySQL 8.0+                          |
| 支付渠道 | 可插拔，首期：蓝兔支付（ltzf.cn）                    |
| 对接平台 | Sub2API v0.1.103 管理 API                 |


## 架构链路

```
用户点击充值 → Sub2API → iframe 嵌入 dds-billing 充值页
    → 用户选择金额和支付方式 → 前端调后端创建订单
    → 后端调蓝兔支付 API 获取支付二维码
    → 前端展示二维码，用户扫码付款
    → 蓝兔支付回调后端 notify 接口
    → 后端验签 → 更新订单状态
    → 调用 Sub2API 管理 API 给用户充值
    → 前端轮询到成功 → 显示充值完成
```

---

## 蓝兔支付 API 摘要

### 创建支付（微信扫码）

```
POST https://api.ltzf.cn/api/wxpay/native

参数：
  mch_id        string  必填  商户号
  out_trade_no  string  必填  商户订单号
  total_fee     string  必填  支付金额（元）
  body          string  必填  商品描述
  timestamp     string  必填  当前时间戳（秒）
  notify_url    string  必填  回调通知地址
  sign          string  必填  MD5签名

返回：
  code: 0
  data.code_url:   微信支付链接
  data.QRcode_url: 二维码图片地址
```

### 创建支付（支付宝扫码）

```
POST https://api.ltzf.cn/api/alipay/native

参数：同微信扫码
返回：同微信扫码
```

### 支付回调通知

```
POST <notify_url>

参数：
  code          string  0=成功
  mch_id        string  商户号
  order_no      string  系统订单号
  out_trade_no  string  商户订单号
  pay_no        string  支付宝/微信订单号
  total_fee     string  支付金额
  pay_channel   string  alipay/wxpay
  trade_type    string  NATIVE/H5/APP
  success_time  string  支付完成时间
  attach        string  附加数据
  sign          string  签名

应答：成功返回 "SUCCESS"，失败返回 "FAIL"
```

### 签名算法

```
1. 筛选：只取必填且非空参数（不含 sign 本身）
2. 排序：按参数名 ASCII 字典序升序
3. 拼接：key1=value1&key2=value2&...&key=商户密钥
4. 加密：MD5(拼接字符串).toUpperCase()
```

### 查询订单

```
POST https://api.ltzf.cn/api/wxpay/get_pay_order
参数：mch_id, out_trade_no, timestamp, sign
限频：1次/5秒
```

### 退款

```
POST https://api.ltzf.cn/api/wxpay/refund_order
参数：mch_id, out_trade_no, out_refund_no, timestamp, refund_fee, sign
```

---

## 支付渠道接口设计（核心）

### PaymentProvider 接口

```go
// internal/payment/provider.go

type PaymentType string

const (
    PaymentTypeWxpay  PaymentType = "wxpay"
    PaymentTypeAlipay PaymentType = "alipay"
)

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
    OrderNo     string      // 商户订单号
    Amount      string      // 支付金额（元）
    Subject     string      // 商品描述
    NotifyURL   string      // 回调通知地址
    PaymentType PaymentType // wxpay / alipay
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
    TradeNo   string // 渠道系统订单号
    PayURL    string // 支付链接
    QRCodeURL string // 二维码图片地址
}

// PaymentNotification 回调通知数据
type PaymentNotification struct {
    OrderNo     string // 商户订单号
    TradeNo     string // 渠道系统订单号
    PayNo       string // 支付宝/微信订单号
    Amount      string // 支付金额
    PaymentType PaymentType
    PaidAt      string // 支付时间
}

// PaymentProvider 支付渠道统一接口
type PaymentProvider interface {
    // Name 渠道名称（用于日志和订单记录）
    Name() string

    // SupportedTypes 支持的支付方式
    SupportedTypes() []PaymentType

    // CreatePayment 创建支付订单，返回二维码/支付链接
    CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)

    // VerifyNotification 验证回调签名，解析通知数据
    VerifyNotification(ctx context.Context, body []byte, params map[string]string) (*PaymentNotification, error)

    // QueryOrder 主动查询订单状态（可选，用于补偿）
    QueryOrder(ctx context.Context, orderNo string) (*PaymentNotification, error)
}
```

### 渠道注册表

```go
// internal/payment/registry.go

var providers = map[string]PaymentProvider{}

func Register(name string, p PaymentProvider) {
    providers[name] = p
}

func Get(name string) (PaymentProvider, bool) {
    p, ok := providers[name]
    return p, ok
}

// GetActive 根据配置返回当前激活的渠道
func GetActive(cfg *config.Config) PaymentProvider {
    p, ok := Get(cfg.Payment.Provider)
    if !ok {
        panic("unknown payment provider: " + cfg.Payment.Provider)
    }
    return p
}
```

### 新增渠道只需 3 步

```
1. 创建 internal/payment/xxx/provider.go，实现 PaymentProvider 接口
2. 在 init() 或 main() 中调用 payment.Register("xxx", NewXxxProvider(cfg))
3. 配置文件中设置 payment.provider: "xxx"
```

### 回调路由自动注册

```go
// internal/handler/notify.go

// 回调路由按渠道名注册：POST /api/notify/:provider
// 例如：POST /api/notify/ltzf、POST /api/notify/zpay、POST /api/notify/easypay
func (h *NotifyHandler) Handle(c *gin.Context) {
    providerName := c.Param("provider")
    provider, ok := payment.Get(providerName)
    // ... 通用回调处理逻辑
}
```

---

## 项目结构

```
dds-billing/
├── cmd/
│   └── server/
│       └── main.go              # 入口，启动 Gin server，注册支付渠道
├── internal/
│   ├── config/
│   │   └── config.go            # 配置加载（YAML + 环境变量）
│   ├── handler/
│   │   ├── order.go             # POST /api/orders（创建订单）
│   │   ├── notify.go            # POST /api/notify/:provider（通用回调）
│   │   ├── query.go             # GET /api/orders/:id（查询订单状态）
│   │   └── health.go            # GET /health
│   ├── logic/
│   │   ├── order.go             # 订单业务逻辑（创建、确认、充值）
│   │   └── recharge.go          # 调用 Sub2API 充值
│   ├── payment/
│   │   ├── provider.go          # PaymentProvider 接口定义
│   │   ├── registry.go          # 渠道注册表（Register/Get/GetActive）
│   │   ├── ltzf/                # 蓝兔支付实现
│   │   │   ├── provider.go      # 实现 PaymentProvider 接口
│   │   │   ├── client.go        # 蓝兔 HTTP API 调用
│   │   │   └── sign.go          # MD5 签名/验签
│   │   ├── easypay/             # 易支付实现（预留，后续可加）
│   │   │   ├── provider.go
│   │   │   ├── client.go
│   │   │   └── sign.go
│   │   ├── zpay/                # ZPAY 实现（预留）
│   │   └── stripe/              # Stripe 实现（预留）
│   ├── sub2api/
│   │   └── client.go            # Sub2API 管理 API 调用
│   ├── model/
│   │   └── order.go             # 订单模型 + GORM 定义
│   ├── repo/
│   │   └── order.go             # 订单数据库操作
│   └── middleware/
│       └── cors.go              # CORS 中间件
├── configs/
│   └── config.yaml              # 配置文件（含支付渠道选择）
├── web/                         # React 前端
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Pay.tsx          # 充值页面
│   │   │   ├── Result.tsx       # 支付结果页
│   │   │   └── Orders.tsx       # 订单列表页
│   │   ├── components/
│   │   │   ├── PaymentForm.tsx  # 金额 + 支付方式选择
│   │   │   └── QRCode.tsx       # 二维码展示 + 状态轮询
│   │   ├── api/
│   │   │   └── index.ts         # Axios API 封装
│   │   └── App.tsx
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── docs/
├── go.mod
├── go.sum
└── Makefile
```

---

## 数据库设计

### orders 表

```sql
CREATE TABLE orders (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_no        VARCHAR(64) NOT NULL UNIQUE,     -- 商户订单号
    user_id         BIGINT NOT NULL,                 -- Sub2API 用户ID
    user_email      VARCHAR(255),                    -- 用户邮箱（审计用）
    amount          DECIMAL(10,2) NOT NULL,          -- 充值金额（元）
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                    -- pending / paid / recharging / completed / failed / expired
    payment_type    VARCHAR(20) NOT NULL,            -- wxpay / alipay
    provider        VARCHAR(32) NOT NULL,            -- 支付渠道：ltzf / easypay / zpay / stripe
    trade_no        VARCHAR(128),                    -- 渠道系统订单号
    pay_no          VARCHAR(128),                    -- 支付宝/微信订单号
    qr_code_url     TEXT,                            -- 二维码图片地址
    notify_data     TEXT,                            -- 回调原始数据（备查）
    recharge_code   VARCHAR(64) UNIQUE,              -- Sub2API 充值码
    failed_reason   VARCHAR(512),                    -- 失败原因
    expires_at      DATETIME NOT NULL,               -- 订单过期时间
    paid_at         DATETIME,
    completed_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_orders_user_id (user_id),
    INDEX idx_orders_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## API 接口设计

### 1. 创建订单

```
POST /api/orders

Request:
{
  "token": "sub2api用户token",
  "amount": 50.00,
  "payment_type": "wxpay"   // wxpay | alipay
}

Response:
{
  "code": 0,
  "data": {
    "order_no": "ORD20260407xxxx",
    "amount": 50.00,
    "status": "pending",
    "qr_code_url": "https://api.ltzf.cn/qr/xxxxx",
    "expires_at": "2026-04-07T10:05:00Z"
  }
}
```

**逻辑：**

1. 用 token 调 Sub2API `GET /api/v1/auth/me` 验证用户身份
2. 校验金额范围
3. 生成订单号，写入数据库（status=pending）
4. 调蓝兔支付 API（wxpay/native 或 alipay/native）
5. 返回二维码地址给前端

### 2. 查询订单状态

```
GET /api/orders/:order_no

Response:
{
  "code": 0,
  "data": {
    "order_no": "ORD20260407xxxx",
    "status": "completed",
    "amount": 50.00,
    "paid_at": "2026-04-07T10:02:30Z"
  }
}
```

### 3. 蓝兔支付回调

```
POST /api/notify/ltzf

Request: 蓝兔支付 POST 表单参数（见上方 API 摘要）

Response: "SUCCESS" 或 "FAIL"
```

**逻辑：**

1. 解析表单参数
2. MD5 验签
3. 查找订单（out_trade_no）
4. 校验金额一致
5. 更新订单状态 → paid
6. 调用 Sub2API 充值：`POST /api/v1/admin/redeem-codes/create-and-redeem`
7. 充值成功 → 订单状态改为 completed
8. 充值失败 → 订单状态改为 failed，记录原因
9. 返回 "SUCCESS"

### 4. 健康检查

```
GET /health
Response: { "status": "ok" }
```

---

## Sub2API 对接

唯一需要调用的接口：

```
POST {SUB2API_BASE_URL}/api/v1/admin/redeem-codes/create-and-redeem

Headers:
  x-api-key: {SUB2API_ADMIN_API_KEY}
  Idempotency-Key: dds-billing:recharge:{order_no}

Body:
{
  "code": "{order_no}",
  "type": "balance",
  "value": 50.00,
  "user_id": 123,
  "notes": "dds-billing recharge order:{order_no}"
}
```

幂等键确保重复调用不会重复充值。

---

## 前端页面

### 充值页 `/pay`

URL 参数（Sub2API 自动传递）：

- `user_id` — 用户 ID
- `token` — 用户登录 token
- `theme` — light/dark
- `lang` — zh/en
- `ui_mode` — standalone/embedded

页面内容：

1. 快捷金额按钮：10 / 20 / 50 / 100 / 200 / 500
2. 自定义金额输入框
3. 支付方式选择：微信支付 / 支付宝
4. 点击「支付」→ 调 POST /api/orders → 弹出二维码
5. 二维码弹窗 + 倒计时（5 分钟）+ 每 2 秒轮询订单状态
6. 支付成功 → 显示成功页面

### 订单列表页 `/pay/orders`

- 展示当前用户的充值记录
- 状态标签：待支付 / 已支付 / 已完成 / 已过期 / 失败

---

## 配置文件

```yaml
# configs/config.yaml
server:
  port: 3000
  mode: debug   # debug / release

database:
  dsn: "root:password@tcp(localhost:3306)/dds_billing?charset=utf8mb4&parseTime=True&loc=Local"

# ========== 支付渠道配置（核心） ==========
payment:
  # 当前使用的渠道（切换渠道只需改这一行）
  provider: "ltzf"            # ltzf / easypay / zpay / stripe

  # 前端展示的支付方式（渠道需支持）
  enabled_types:
    - wxpay
    - alipay

# 蓝兔支付（provider: ltzf）
ltzf:
  mch_id: "你的商户号"
  secret_key: "你的商户密钥"
  notify_url: "https://xxx.ngrok.io/api/notify/ltzf"

# 易支付（provider: easypay）— 预留
easypay:
  pid: ""
  pkey: ""
  api_base: ""
  notify_url: ""

# ZPAY（provider: zpay）— 预留
zpay:
  mch_id: ""
  secret_key: ""
  notify_url: ""

# Stripe（provider: stripe）— 预留
stripe:
  secret_key: ""
  publishable_key: ""
  webhook_secret: ""

# ========== Sub2API ==========
sub2api:
  base_url: "http://localhost:8080"
  admin_api_key: "从Sub2API管理后台获取"

# ========== 业务规则 ==========
billing:
  min_amount: 1.00
  max_amount: 1000.00
  order_timeout_minutes: 5
```

### 切换支付渠道示例

只需改一行配置，重启服务即可：

```yaml
# 从蓝兔切换到易支付
payment:
  provider: "easypay"    # 改这里

# 填上对应渠道的配置
easypay:
  pid: "2099"
  pkey: "xxxxx"
  api_base: "https://api.payqixiang.cn/"
  notify_url: "https://pay.example.com/api/notify/easypay"
```

---

## 开发步骤

### Phase 1：项目脚手架 + 支付接口抽象（0.5 天）

1. 初始化 Go 模块 `go mod init dds-billing`
2. 搭建 Gin 项目结构（cmd/internal/configs）
3. 配置 GORM + MySQL 连接
4. 创建 orders 表（自动迁移，含 provider 字段）
5. **定义 PaymentProvider 接口 + 渠道注册表**
6. 初始化 React + Vite 前端项目（web/）

### Phase 2：蓝兔支付渠道实现（1 天）

1. 实现 MD5 签名/验签（`internal/payment/ltzf/sign.go`）
2. 实现蓝兔 provider（`internal/payment/ltzf/provider.go`）
  - `CreatePayment()` → 调 wxpay/native 或 alipay/native
  - `VerifyNotification()` → 验签 + 解析回调参数
  - `QueryOrder()` → 主动查询订单
3. 在 main.go 中根据配置注册蓝兔 provider
4. 实现**通用回调路由** `POST /api/notify/:provider`
  - 根据 URL 中的 provider 名找到对应实现
  - 调用 `VerifyNotification()` 验签
  - 后续充值逻辑与渠道无关

### Phase 3：订单逻辑 + Sub2API 对接（1 天）

1. 创建订单接口 — 验证用户 token → 生成订单 → **调 GetActive() 获取当前渠道** → 返回二维码
2. 查询订单接口 — 前端轮询用
3. Sub2API client — 调 `create-and-redeem` 接口给用户充值
4. 回调处理完整链路 — 通用：验签 → 更新订单 → 充值 → 更新状态

### Phase 4：前端页面（1 天）

1. 充值页面 — 金额选择 + 支付方式（**从后端 /api/config 获取可用支付方式**） + 提交
2. 二维码弹窗 — 展示二维码 + 倒计时 + 轮询状态
3. 支付结果页
4. 订单列表页
5. 支持 dark/light 主题 + iframe 嵌入模式

### Phase 5：联调测试（0.5 天）

1. 本地启动 Sub2API + dds-billing
2. ngrok 暴露本地服务（蓝兔回调需要公网地址）
3. 在 Sub2API 配置充值页 URL
4. 完成小额支付测试，验证全链路：
  - 创建订单 → 二维码 → 扫码付款 → 回调 → 验签 → 充值 → 余额到账
5. **验证渠道切换**：改配置为 easypay（如果已实现），确认流程正常

---

## 验证清单

- 蓝兔支付签名生成与验证正确
- 微信扫码支付创建成功，二维码可展示
- 支付宝扫码支付创建成功
- 回调通知正常接收并验签通过
- 订单状态流转正确：pending → paid → recharging → completed
- Sub2API 用户余额正确增加
- 重复回调不会重复充值（幂等性）
- 过期订单自动标记为 expired
- 前端轮询正常，支付成功后正确跳转
- iframe 嵌入模式在 Sub2API 中正常显示

## 关键文件路径


| 功能            | 文件                                                            |
| ------------- | ------------------------------------------------------------- |
| 入口            | `cmd/server/main.go`                                          |
| 路由注册          | `internal/handler/router.go`                                  |
| 创建订单          | `internal/handler/order.go` + `internal/logic/order.go`       |
| **通用回调处理**    | `internal/handler/notify.go`（`POST /api/notify/:provider`）    |
| **支付渠道接口**    | `internal/payment/provider.go`                                |
| **渠道注册表**     | `internal/payment/registry.go`                                |
| 蓝兔支付实现        | `internal/payment/ltzf/provider.go` + `client.go` + `sign.go` |
| Sub2API 充值    | `internal/sub2api/client.go`                                  |
| 订单模型          | `internal/model/order.go`                                     |
| **配置（含渠道选择）** | `internal/config/config.go` + `configs/config.yaml`           |
| 充值页面          | `web/src/pages/Pay.tsx`                                       |
| 二维码组件         | `web/src/components/QRCode.tsx`                               |


## 扩展新渠道示例（以 ZPAY 为例）

```
1. 创建 internal/payment/zpay/provider.go
   - 实现 PaymentProvider 接口的 4 个方法
   - ZPAY 的签名方式、API 地址、参数格式写在这里

2. main.go 中注册：
   payment.Register("zpay", zpay.NewProvider(cfg.Zpay))

3. config.yaml 中配置：
   payment:
     provider: "zpay"
   zpay:
     mch_id: "xxx"
     secret_key: "xxx"
     notify_url: "https://pay.example.com/api/notify/zpay"

4. 完成。订单逻辑、充值逻辑、前端页面全部不需要改。
```

