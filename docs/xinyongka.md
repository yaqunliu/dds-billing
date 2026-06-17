# 国际信用卡支付接入开发 Plan（Stripe Card）

## 1. 背景与目标

当前 dds-billing 已通过 **Stripe** 渠道接入了**微信支付（wechat_pay）**和**支付宝（alipay）**，两者均面向国内用户、以扫码/跳转方式完成支付。

现需新增 **国际信用卡支付（Visa / Mastercard / Amex 等）**，面向**国外用户**：

- 支付页已有中英文双语（`web/src/utils/locale.ts`），本次复用，不重做。
- 在现有「微信 / 支付宝」之外，新增「信用卡」支付方式选项。
- **选择信用卡时，最低充值金额为 200 起充**（其余方式仍沿用 `billing.min_amount`）。

> 项目代号：**dds-billing**（当前目录）

---

## 2. 现状分析（与本次相关的代码）

### 2.1 支付抽象层

`internal/payment/provider.go` 定义统一接口 `PaymentProvider` 与 `PaymentType`：

```go
const (
    PaymentTypeWxpay  PaymentType = "wxpay"
    PaymentTypeAlipay PaymentType = "alipay"
)
```

### 2.2 Stripe Provider 现状

`internal/payment/stripe/provider.go` + `client.go`：

- `SupportedTypes()` 返回 `{wxpay, alipay}`。
- `stripeMethodType()` 把业务类型映射为 Stripe 的 `wechat_pay` / `alipay`。
- `CreatePayment()` 走 **PaymentIntent + Confirm(true)** 路径，从 `NextAction` 取二维码/跳转链接：
  - 微信：`NextAction.WeChatPayDisplayQRCode.Data`（`weixin://`，前端生成二维码）
  - 支付宝：`NextAction.AlipayHandleRedirect.URL`
- `TradeNo` = PaymentIntent ID（`pi_xxx`），存入 `orders.trade_no`。
- `VerifyNotification()` 只处理 webhook 事件 `payment_intent.succeeded`。
- `QueryOrder(tradeNo)` 用 `pi_xxx` 调 `RetrievePaymentIntent`，判断 `status == succeeded`。

### 2.3 订单流转

- `internal/logic/order.go`：
  - `CreateOrder()` 校验金额 `MinAmount ~ MaxAmount`、校验支付类型（仅 wxpay/alipay）、创建支付、落库。
  - 后台 `startPendingOrderChecker()` 每分钟扫 pending 订单，stripe 用 `order.TradeNo` 主动查单。
- `internal/handler/notify.go`：通用回调 `POST/GET /api/notify/:provider`，验签后置 paid 并异步 `ProcessRecharge`。
- `internal/handler/query.go`：`GET /api/orders/:order_no` 前端轮询，pending 时主动查单。

### 2.4 前端现状

- `web/src/pages/Pay.tsx`：金额选择 + 支付方式（`wxpay`/`alipay`）+ 创建订单，成功后弹 `QRCodeModal` 轮询。
- `web/src/utils/constant.ts`：`PAYMENT_TYPE_CONFIG`（图标/样式）、`QUICK_AMOUNTS`。
- `web/src/utils/locale.ts`：中英文文案、`PAYMENT_TYPE_CONFIG` 标签。
- `web/src/components/QRCode.tsx`：二维码弹窗 + 状态轮询 + 成功跳转。
- `web/src/api/index.ts`：`OrderCreateRequest.payment_type: 'wxpay' | 'alipay'`、`AppConfig`。

---

## 3. 关键设计决策

### 3.1 信用卡为什么不能复用现有 PaymentIntent + 二维码流程

微信/支付宝是「展示码 → 用户在自己 App 内确认」，服务端 `Confirm(true)` 后即可拿到跳转/二维码链接。

**信用卡需要采集卡号/CVC/3DS 验证**，出于 PCI 合规**不能由我们自己的页面采集**，必须交给 Stripe 托管页面或 Stripe.js 组件。因此信用卡走**独立分支**，不再生成二维码。

### 3.2 采集方式：Stripe Checkout（托管页，推荐） vs Payment Element（内嵌）

| 方案 | 说明 | 优点 | 缺点 |
|---|---|---|---|
| **A. Stripe Checkout Session（推荐）** | 后端创建 Session，返回 `session.url`，前端 `window.location` 跳转到 Stripe 托管收银台，付完跳回 `success_url` | 改动最小、PCI 合规零负担、**自带多语言**（`locale: 'auto'`）、自带 3DS、自带卡品牌识别 | 离开本站跳到 Stripe 页面 |
| B. Payment Element 内嵌 | 前端引入 Stripe.js + Elements，后端返回 `client_secret`，页面内确认 | 不跳转、体验统一 | 需引入 `@stripe/stripe-js`、`@stripe/react-stripe-js`，前端工作量大，需自己处理 3DS 回跳与 i18n |

> **决策：采用方案 A（Checkout Session）**。中英文由 Stripe Checkout 的 `locale` 参数承担（传入页面当前 `lang`），与「支付页已有中英文」目标一致。
> 若后续要求「不跳出站点」，再迭代为方案 B。

### 3.3 货币与「200 起充」的口径（⚠️ 需产品确认）

现有微信/支付宝以 **CNY（元）**计价，到账规则 `1 元 = 1 美元`（见 locale 文案），`ProcessRecharge` 按 `order.Amount` 数值 1:1 充值。

信用卡接入有两种口径：

- **方案 ①（推荐，改动最小、到账口径一致）**：信用卡**仍以 CNY 计价**，金额数值与微信/支付宝同义，Checkout 的 `currency='cny'`，持卡人银行自动按汇率换算扣款。`order.Amount` 含义不变，`ProcessRecharge` 无需改。最低 **200（CNY）**。
- **方案 ②**：信用卡以 **USD 计价**（`currency='usd'`），最低 **200（USD）**。需要确认到账如何换算（否则同样充 200，卡用户付出约 7 倍）；会引入「不同支付方式 amount 单位不一致」的复杂度，`ProcessRecharge`/对账/退款都要区分币种。

> **决策建议：方案 ①**。`card` 仍走 CNY，仅把「最低金额」从 `min_amount` 提到 **200**。
> 把币种做成**可配置**（`stripe.card_currency`，默认 `cny`），并把 `card_min_amount` 放入 `billing` 配置 + `/config` 接口，前后端共用同一来源，避免硬编码。
> **本决策需在动工前与产品 / 财务确认，确认结果回填本节。**

### 3.4 信用卡专属最低金额（200）落点

- 后端：新增 `billing.card_min_amount`（默认 200），`CreateOrder` 中按 `payment_type==card` 走该下限，并加 `MaxAmount` 上限校验。
- `/config` 接口下发 `card_min_amount`，前端据此校验与展示，**不在前端写死 200**。
- 前端：选择信用卡时，最低金额、快捷金额按钮、输入框校验、按钮可用态都切换到 `card_min_amount`。

### 3.5 回调与查单的双形态

信用卡走 Checkout，订单标识与事件与 PaymentIntent 不同：

- `TradeNo` = **Checkout Session ID（`cs_xxx`）**（落 `orders.trade_no`）。
- Webhook 事件：信用卡监听 **`checkout.session.completed`**（微信/支付宝仍是 `payment_intent.succeeded`）。
- 主动查单：`trade_no` 以 `cs_` 开头 → `RetrieveCheckoutSession` 判 `payment_status=='paid'`；以 `pi_` 开头 → 沿用 `RetrievePaymentIntent`。

> 这样 `order.go` 的 checker 与 `query.go` 的轮询逻辑**无需感知差异**，只调 `provider.QueryOrder(order.TradeNo)`，分支封装在 stripe provider 内部。

---

## 4. 后端改动清单

### 4.1 `internal/payment/provider.go`

新增支付类型常量：

```go
const (
    PaymentTypeWxpay  PaymentType = "wxpay"
    PaymentTypeAlipay PaymentType = "alipay"
    PaymentTypeCard   PaymentType = "card"   // 新增：国际信用卡
)
```

### 4.2 `internal/config/config.go`

- `StripeConfig` 增加：
  ```go
  CardCurrency string `yaml:"card_currency"` // 默认 cny
  ```
- `BillingConfig` 增加：
  ```go
  CardMinAmount float64 `yaml:"card_min_amount"` // 默认 200
  ```
- `Load()` 中补默认值：`CardCurrency` 空则 `"cny"`；`CardMinAmount` 为 0 则 `200`。

### 4.3 `internal/payment/stripe/client.go`

新增两个方法（与现有 `CreatePaymentIntent` 并列）：

```go
// CreateCheckoutSession 创建信用卡托管收银台 Session（card）。
// successURL 需包含 {CHECKOUT_SESSION_ID} 占位与 order_no；locale 传 "auto"/"zh"/"en"。
func (c *Client) CreateCheckoutSession(ctx context.Context, orderNo, subject string,
    amountCents int64, currency, successURL, cancelURL, locale string) (*gostripe.CheckoutSession, error) {
    params := &gostripe.CheckoutSessionCreateParams{
        Mode:               gostripe.String("payment"),
        PaymentMethodTypes: []*string{gostripe.String("card")},
        SuccessURL:         gostripe.String(successURL),
        CancelURL:          gostripe.String(cancelURL),
        ClientReferenceID:  gostripe.String(orderNo),
        Locale:             gostripe.String(locale), // auto / zh / en
        LineItems: []*gostripe.CheckoutSessionCreateLineItemParams{{
            Quantity: gostripe.Int64(1),
            PriceData: &gostripe.CheckoutSessionCreateLineItemPriceDataParams{
                Currency:   gostripe.String(currency),
                UnitAmount: gostripe.Int64(amountCents),
                ProductData: &gostripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
                    Name: gostripe.String(subject),
                },
            },
        }},
        PaymentIntentData: &gostripe.CheckoutSessionCreatePaymentIntentDataParams{
            Metadata: map[string]string{"order_no": orderNo, "project": "dds-billing"},
        },
        Metadata: map[string]string{"order_no": orderNo, "project": "dds-billing"},
    }
    return c.stripe.V1CheckoutSessions.Create(ctx, params)
}

// RetrieveCheckoutSession 查询 Session 状态
func (c *Client) RetrieveCheckoutSession(ctx context.Context, sessionID string) (*gostripe.CheckoutSession, error) {
    return c.stripe.V1CheckoutSessions.Retrieve(ctx, sessionID, nil)
}
```

> 注意：以 Stripe Go SDK v82 的实际类型名为准（`gostripe` 已在用），编译时核对 `CheckoutSession*` 结构体字段名。

### 4.4 `internal/payment/stripe/provider.go`

1. `SupportedTypes()` 增加 `PaymentTypeCard`。
2. `CreatePayment()` 增加 card 分支（其余不变）：
   ```go
   if req.PaymentType == payment.PaymentTypeCard {
       successURL := p.cfg.SuccessURL + "?order_no=" + req.OrderNo +
           "&status=success&session_id={CHECKOUT_SESSION_ID}"
       cancelURL := p.cfg.CancelURL
       currency := p.cfg.CardCurrency // 默认 cny
       cs, err := p.client.CreateCheckoutSession(ctx, req.OrderNo, req.Subject,
           amountCents, currency, successURL, cancelURL, req.Locale)
       if err != nil { return nil, fmt.Errorf("stripe create checkout session: %w", err) }
       return &payment.CreatePaymentResponse{
           TradeNo:   cs.ID,  // cs_xxx
           PayURL:    cs.URL, // 托管收银台 URL，前端跳转
           QRCodeURL: "",
       }, nil
   }
   ```
   > `req.Locale` 为新增字段，见 4.6；用于把页面语言透传给 Stripe Checkout。
3. `QueryOrder()` 按前缀分支：
   ```go
   if strings.HasPrefix(tradeNo, "cs_") {
       cs, err := p.client.RetrieveCheckoutSession(ctx, tradeNo)
       if err != nil { return nil, ... }
       if cs.PaymentStatus != "paid" {
           return nil, fmt.Errorf("checkout session not paid: %s", cs.PaymentStatus)
       }
       return checkoutSessionToNotification(cs), nil // 新增 helper：取 metadata.order_no、amount_total、PaymentIntent ID
   }
   // 否则沿用 RetrievePaymentIntent
   ```
4. `VerifyNotification()` 增加事件分支：
   ```go
   switch event.Type {
   case "payment_intent.succeeded":   // 微信/支付宝，逻辑不变
       ...
   case "checkout.session.completed": // 信用卡
       var cs gostripe.CheckoutSession
       json.Unmarshal(jsonData, &cs)
       if cs.PaymentStatus != "paid" {
           return nil, fmt.Errorf("%w: session not paid", payment.ErrEventIgnored)
       }
       n := checkoutSessionToNotification(&cs)
       n.PaidAt = fmt.Sprintf("%d", event.Created)
       return n, nil
   default:
       return nil, fmt.Errorf("%w: %s", payment.ErrEventIgnored, event.Type)
   }
   ```
5. 新增 helper `checkoutSessionToNotification(cs)`：`OrderNo` 取 `cs.Metadata["order_no"]`（或 `cs.ClientReferenceID`）、`Amount = amount_total/100`、`TradeNo/PayNo` 取 `cs.PaymentIntent.ID`（或 `cs.ID` 兜底）、`PaymentType = card`。

### 4.5 `internal/logic/order.go`

`CreateOrder()` 调整：

- 支付类型白名单加入 `card`：
  ```go
  if payType != payment.PaymentTypeWxpay &&
     payType != payment.PaymentTypeAlipay &&
     payType != payment.PaymentTypeCard {
      return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
  }
  ```
- 金额下限按类型区分：
  ```go
  minAmount := l.cfg.Billing.MinAmount
  if payType == payment.PaymentTypeCard {
      minAmount = l.cfg.Billing.CardMinAmount // 200
  }
  if req.Amount < minAmount || req.Amount > l.cfg.Billing.MaxAmount {
      return nil, fmt.Errorf("amount must be between %.2f and %.2f", minAmount, l.cfg.Billing.MaxAmount)
  }
  ```
- 透传 `Locale` 到 `CreatePaymentRequest`（见 4.6），来源为请求新增的 `lang` 字段。

> `CheckAndUpdateOrder` / checker：**无需改动**，仍用 `order.TradeNo` 调 `QueryOrder`，分支已封装在 provider。

### 4.6 `CreatePaymentRequest` 增加 Locale（`internal/payment/provider.go`）

```go
type CreatePaymentRequest struct {
    ...
    Locale string // auto / zh / en，仅信用卡 Checkout 使用
}
```

`CreateOrderRequest`（`logic/order.go`）增加 `Lang string json:"lang"`，在 `CreateOrder` 里映射为 `Locale`（`zh`/`en`，空则 `auto`）。

### 4.7 `internal/handler/router.go` — `/config` 接口

下发 `card_min_amount`，供前端使用：

```go
"data": gin.H{
    "enabled_types":   cfg.Payment.EnabledTypes,
    "min_amount":      cfg.Billing.MinAmount,
    "max_amount":      cfg.Billing.MaxAmount,
    "card_min_amount": cfg.Billing.CardMinAmount, // 新增
},
```

### 4.8 回调 handler（`internal/handler/notify.go`）

**无需改动**：已是通用 `VerifyNotification`，新事件类型在 provider 内消化；`ErrEventIgnored` 已正确返回 200。

---

## 5. 前端改动清单

### 5.1 `web/src/api/index.ts`

- `OrderCreateRequest.payment_type` 扩为 `'wxpay' | 'alipay' | 'card'`，新增可选 `lang?: 'zh' | 'en'`。
- `AppConfig` 增加 `card_min_amount: number`。

### 5.2 `web/src/utils/constant.ts`

- `PAYMENT_TYPE_CONFIG` 增加 `card` 项（图标、激活/悬停样式、`getLabel: (t) => t.creditCard`）。准备一张信用卡图标（`web/src/assets/card.png` 或 Visa/Master 组合图）。

### 5.3 `web/src/utils/locale.ts`

- `PayMessages` 类型与 `zh/en` 文案增加：
  - `creditCard`：`"信用卡" / "Credit Card"`
  - 信用卡专属最低金额提示，如 `cardMinTip(min)`：`"信用卡支付最低 ¥${min} 起充" / "Card payment requires a minimum of ¥${min}"`
  - 跳转提示 `redirectingToStripe`：`"正在跳转到安全支付页面..." / "Redirecting to secure checkout..."`
- `QRCODE_MESSAGES` 的 `paymentLabels` 不需要 card（信用卡不走二维码弹窗）。

### 5.4 `web/src/pages/Pay.tsx`（核心）

1. `paymentType` 类型扩为 `'wxpay' | 'alipay' | 'card'`。
2. 读取 `config.card_min_amount`，计算「当前生效最低金额」：
   ```ts
   const effectiveMin = paymentType === 'card'
       ? (config?.card_min_amount ?? 200)
       : (config?.min_amount ?? 10);
   ```
   - `isAmountValid`、输入框 placeholder、`amountRange` 提示、起充展示均改用 `effectiveMin`。
   - 切到 card 时，如当前金额 `< effectiveMin`，清空或给出 `cardMinTip` 提示并禁用按钮。
   - 快捷金额：card 下过滤掉小于 200 的按钮（或单独一套 `CARD_QUICK_AMOUNTS = [200, 300, 500, 1000, ...]`）。
3. 提交分支：
   ```ts
   const res = await createOrder({ token, amount: Number(amount), payment_type: paymentType, lang: appLang });
   if (res.data.code === 0) {
     if (paymentType === 'card') {
       // 信用卡：直接整页跳转到 Stripe Checkout
       window.location.href = res.data.data.pay_url;
       return;
     }
     // 微信/支付宝：保持原二维码弹窗
     setOrderNo(...); setQrCodeUrl(...); setShowQR(true);
   }
   ```
4. 付款成功后 Stripe 跳回 `success_url`（带 `order_no` & `status=success`）→ 进入现有 `Result.tsx`，其轮询 `getOrder` 拿到 paid/completed，与现状一致。

> `QRCode.tsx` **无需改动**（card 不经过它）。`Result.tsx` 复用现有逻辑，确认能从 URL `order_no` 正常拉单即可。

---

## 6. 配置改动（`configs/config.yaml` + `config.example.yaml`）

```yaml
payment:
  provider: "stripe"
  enabled_types:
    - alipay
    - wxpay
    - card        # 新增：信用卡

stripe:
  secret_key: "..."
  publishable_key: "..."
  webhook_secret: "..."
  notify_url: "https://your-domain.com/api/notify/stripe"
  success_url: "https://your-domain.com/pay/result"
  cancel_url: "https://your-domain.com/pay"
  card_currency: "cny"   # 新增：信用卡计价币种（决策 3.3，默认 cny）

billing:
  min_amount: 5.00
  max_amount: 3000.00
  card_min_amount: 200.00 # 新增：信用卡最低起充
  order_timeout_minutes: 5
```

> ⚠️ `enabled_types` 决定前端展示哪些支付方式；上线信用卡需把 `card` 加入。

---

## 7. Stripe 后台配置

1. **开通 Card 支付方式**：Stripe Dashboard → Settings → Payment methods 启用 Cards。
2. **Webhook 事件**：现有 webhook endpoint（`/api/notify/stripe`）需**新增订阅事件 `checkout.session.completed`**（保留原 `payment_intent.succeeded`）。`webhook_secret` 不变。
3. **测试卡号**：`4242 4242 4242 4242`（任意未来日期 + 任意 CVC）；3DS 测试卡 `4000 0027 6000 3184`。
4. 若用方案 ②（USD），确认账户支持结算币种与到账换算。

---

## 8. 测试计划

### 8.1 后端单测 / 手测
- `CreateOrder`：`card` 且金额 `< 200` → 报错；`>= 200` → 返回 `pay_url`（`cs_` 开头 trade_no）。
- `card` 且金额 `> max_amount` → 报错。
- 非 card 仍走 `min_amount=5`，回归不受影响。
- `QueryOrder`：`cs_` 与 `pi_` 两种 trade_no 均正确判定 paid。
- `VerifyNotification`：`checkout.session.completed`（paid）→ 置 paid 并触发 recharge；非 paid / 其他事件 → 返回 200 忽略。

### 8.2 端到端（Stripe 测试模式）
1. 前端选「信用卡」，金额 < 200 按钮禁用并提示；调到 ≥ 200 可提交。
2. 提交 → 跳转 Stripe Checkout（验证页面语言随 `lang` 切换 zh/en）。
3. 用测试卡支付成功 → 跳回 `Result` → 轮询显示充值完成 → Sub2API 到账。
4. 取消支付 → 跳回 `cancel_url`（支付页）。
5. Webhook 与「后台 1 分钟查单」双通道均能把订单置 paid（断网模拟 webhook 丢失时由查单兜底）。
6. 微信/支付宝回归：扫码、二维码弹窗、轮询、到账全部正常。

### 8.3 验收点
- 信用卡最低 200 在**前后端双重校验**（前端禁用 + 后端拒绝）。
- 中英文：支付页方式标签、提示、Stripe Checkout 页面语言一致。

---

## 9. 上线步骤与回滚

**上线顺序**：
1. 后端发布（含新配置项，`card` 暂不加入 `enabled_types`，灰度准备）。
2. Stripe 后台启用 Card + 订阅 `checkout.session.completed`。
3. 前端发布。
4. 配置 `enabled_types` 加入 `card` → 前端即出现信用卡选项（无需再发版）。

**回滚**：从 `enabled_types` 移除 `card` 即可立即下线入口；后端新增代码对既有 wxpay/alipay 无侵入，可安全保留。

---

## 10. 工作量与风险

| 模块 | 改动 | 预估 |
|---|---|---|
| 后端 provider/client（Checkout + 查单 + webhook 分支） | 中 | 1 ~ 1.5 天 |
| 后端 config/logic/router（币种、card_min、locale 透传） | 小 | 0.5 天 |
| 前端 Pay 页分支 + 校验 + 文案 + 图标 | 中 | 1 天 |
| 联调（Stripe 测试模式 + webhook） | 中 | 0.5 ~ 1 天 |

**风险点：**
- **货币/最低金额口径（3.3）必须先与产品/财务确认**，否则到账与对账会出问题 —— 阻塞项。
- Stripe Go SDK v82 的 `CheckoutSession*` 字段名需以实际版本为准，编译期核对。
- Checkout 为跳转式，需保证 `success_url`/`cancel_url` 为线上可达的 HTTPS 域名（本地联调用 ngrok/测试域名）。
- webhook 必须新增 `checkout.session.completed` 订阅，否则仅靠查单兜底（有延迟）。

---

## 11. 待确认清单（动工前回填）

- [ ] **3.3 决策**：信用卡计价 CNY（方案①，推荐）还是 USD（方案②）？到账如何换算？
- [ ] 信用卡最低 **200** 的币种单位（CNY / USD）与 `card_min_amount` 取值确认。
- [ ] 信用卡快捷金额按钮档位（如 200 / 300 / 500 / 1000）。
- [ ] 是否要求「不跳出站点」（若是则改用方案 B Payment Element，工作量上调）。
- [ ] 信用卡图标素材（单卡图 or Visa/Master/Amex 组合图）。

---

## 12. 后续演进：托管页 → 内嵌（Payment Element）的迁移成本

### 12.1 本质区别

「托管页」与「内嵌」绕不开一个架构差异：**后端返回什么 + 谁来确认支付**。

| | 托管 Checkout（首期方案） | 内嵌 Payment Element |
|---|---|---|
| 后端返回 | 跳转 `url`（`session.url`） | `client_secret` |
| 确认支付 | 在 Stripe 托管页完成 | 前端 Stripe.js `confirmPayment()` 完成 |
| 前端依赖 | 整页跳转即可 | 需 `publishable_key` 初始化 Stripe.js（`@stripe/stripe-js` + `@stripe/react-stripe-js`） |
| 多语言 | Stripe 托管页 `locale` 自动承担 | 由 Payment Element `locale` 选项承担 |

内嵌模式下前端必须拿到 **`client_secret`** 和 **`publishable_key`**，这两样目前后端都没对外暴露 —— 所以**切内嵌不是纯前端，后端仍有少量必改项**。

### 12.2 切内嵌时后端必改项

1. `CreatePaymentResponse` 增加 `ClientSecret` 字段，并透传到「创建订单」接口返回体（`logic/order.go` + `api/index.ts`）。
2. `CreatePayment` 的 card 分支改写：不再返回 `url`，改为返回 `client_secret`。
3. `/config` 接口暴露 `publishable_key`（前端初始化 Stripe.js 用）。

### 12.3 内嵌的两种变体（决定后端改动量）

- **变体 A：Embedded Checkout（`ui_mode: 'embedded'`）**
  仍是 CheckoutSession，`trade_no` 还是 `cs_`、webhook 还是 `checkout.session.completed`，**查单/回调分支完全复用**。后端仅改「返回 `client_secret` 而非 `url`」+ `success_url/cancel_url` 换成 `return_url`。改动最小。
- **变体 B：Payment Element + PaymentIntent**
  后端从「建 Session」改为「**直接建 PaymentIntent 且不 `Confirm(true)`**」，交前端确认。`trade_no` 变 `pi_`、webhook 变 `payment_intent.succeeded`。
  > 注意：`payment_intent.succeeded` 回调与 `RetrievePaymentIntent` 查单，**本项目为微信/支付宝已实现**，可直接复用 —— 反而是首期托管 Checkout 额外引入了 `cs_`/`checkout.session.completed` 分支。

### 12.4 降低未来迁移成本的前置动作（首期顺手做，低成本）

在做托管页时一并完成下面两件事，将来切内嵌就**几乎只剩前端**：

1. **`/config` 现在就返回 `publishable_key`**（一行，零风险）。
2. **`CreatePaymentResponse` 预留 `ClientSecret string` 字段**（首期留空）。

这样未来切内嵌时，后端实质改动只剩「card 分支改返回内容」一处。

---

## 13. Stripe 费率：托管页 vs 内嵌

**结论：托管 Checkout 不额外收费，与内嵌 Payment Element 的处理费率完全相同。**

- Stripe 的 **Checkout（托管页 + 内嵌）、Payment Links、Elements / Payment Element、Payment Intents API 都属于 Stripe Payments 核心能力，不因「用哪种集成方式/界面」而加收费用**。每笔成功收款按账户标准费率计费，与是否走托管页无关。
- 真正影响费率的是**卡种、币种、是否跨境**，而非集成方式，例如：
  - 国际卡 / 跨境交易通常比本地卡有附加费；
  - 涉及货币转换（如 3.3 选 USD 计价、或持卡人本币换算）会有币种转换费；
  - 这些对托管页和内嵌**一视同仁**。
- 会**额外收费的是可选增值功能**（默认不开通、与界面无关），例如 **Stripe Tax**（按笔加费）、**Adaptive Pricing**、**Billing/订阅** 等 —— 本项目用不到，无需开启。

> 具体百分比以**你方 Stripe 账户所属国家/地区的官方定价**为准（国内主体与海外主体费率不同），建议在 Dashboard → 该账户的 Pricing 页核对国际卡与跨境的实际数值后回填本节。

> 小结：在「托管 vs 内嵌」这个维度上，**费率不变**；选型只需从**开发成本**和**是否要求不跳出站点**两个角度权衡。
</content>
</invoke>
