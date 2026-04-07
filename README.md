# dds-billing

自建支付服务，为 Sub2API 平台提供充值功能。采用可插拔支付渠道架构，首期对接蓝兔支付，后续可通过配置切换到其他渠道。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go / Gin / GORM |
| 前端 | React + TypeScript + Vite + TailwindCSS |
| 数据库 | MySQL 8.0+ |
| 支付渠道 | 可插拔，首期：蓝兔支付 |
| 对接平台 | Sub2API 管理 API |

## 架构链路

```
用户点击充值 → Sub2API iframe 嵌入充值页
  → 选择金额和支付方式 → 创建订单 → 获取支付二维码
  → 用户扫码付款 → 支付渠道回调 → 验签 → 更新订单
  → 调用 Sub2API 充值 → 余额到账
```

## 项目结构

```
dds-billing/
├── cmd/server/main.go                 # 入口
├── internal/
│   ├── config/config.go               # 配置加载
│   ├── handler/
│   │   ├── router.go                  # 路由注册
│   │   ├── order.go                   # POST /api/orders
│   │   ├── query.go                   # GET /api/orders/:order_no
│   │   ├── notify.go                  # POST /api/notify/:provider（通用回调）
│   │   └── health.go                  # GET /health
│   ├── logic/
│   │   ├── order.go                   # 订单创建逻辑
│   │   └── recharge.go               # Sub2API 充值逻辑
│   ├── payment/
│   │   ├── provider.go                # PaymentProvider 接口
│   │   ├── registry.go                # 渠道注册表
│   │   └── ltzf/                      # 蓝兔支付实现
│   ├── sub2api/client.go              # Sub2API 对接
│   ├── model/order.go                 # 订单模型
│   ├── repo/order.go                  # 数据库操作
│   └── middleware/cors.go             # CORS
├── configs/
│   ├── config.example.yaml            # 配置模板（提交到仓库）
│   └── config.yaml                    # 实际配置（gitignore）
└── web/                               # React 前端
    └── src/
        ├── pages/{Pay,Result,Orders}.tsx
        ├── components/QRCode.tsx
        └── api/index.ts
```

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- MySQL 8.0+

### 1. 创建数据库

```sql
CREATE DATABASE IF NOT EXISTS dds_billing DEFAULT CHARSET utf8mb4;
```

### 2. 配置

```bash
cp configs/config.example.yaml configs/config.yaml
```

编辑 `configs/config.yaml`，填入：
- 数据库连接信息
- 蓝兔支付商户号和密钥
- Sub2API 地址和 Admin API Key

### 3. 启动后端

```bash
go mod download
go run cmd/server/main.go
```

后端默认监听 `:3000`。

### 4. 启动前端

```bash
cd web
npm install
npm run dev
```

前端默认监听 `:5173`，API 请求自动代理到后端。

### 5. 配置 Sub2API

在 Sub2API 管理后台，将充值页面 URL 设置为：

```
http://localhost:5173/pay
```

Sub2API 会自动在 URL 后拼接 `user_id`、`token`、`theme` 等参数。

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/orders` | 创建充值订单 |
| GET | `/api/orders/:order_no` | 查询订单状态（前端轮询） |
| GET | `/api/orders` | 订单列表 |
| POST | `/api/notify/:provider` | 支付渠道回调（通用） |
| GET | `/api/config` | 获取支付配置（前端用） |
| GET | `/health` | 健康检查 |

## 扩展支付渠道

新增渠道只需 3 步：

1. 创建 `internal/payment/xxx/provider.go`，实现 `PaymentProvider` 接口
2. 在 `cmd/server/main.go` 中注册：`payment.Register("xxx", xxx.NewProvider(cfg.Xxx))`
3. 在 `config.yaml` 中配置渠道参数，将 `payment.provider` 改为 `"xxx"`

订单逻辑、充值逻辑、前端页面无需修改。

## 切换支付渠道

只需修改 `config.yaml` 一行配置，重启服务：

```yaml
payment:
  provider: "easypay"  # 从 ltzf 切换到 easypay
```
