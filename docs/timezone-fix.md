# 时区问题修复说明

## 问题描述

订单在过期时间前被错误标记为 `expired`，导致支付回调到达时订单已过期，用户付款成功但无法充值。

## 根本原因

MySQL 使用 `SYSTEM` 时区（CST +08:00），而 Go 后端使用 UTC，导致：
- Go 写入 `expires_at = 2026-04-17 07:22:09` (UTC)
- MySQL `NOW()` 返回 `2026-04-17 16:22:09` (CST)
- 条件 `expires_at <= NOW()` 立即成立，订单刚创建就被判定过期

## 修复内容

### 1. 统一时区为 UTC

**docker-compose.yml**:
- MySQL: 添加 `TZ: UTC` 和 `--default-time-zone='+00:00'`
- Backend: 添加 `TZ: UTC`

### 2. 时间判断改用 Go 端

**internal/repo/order.go**:
- `ExpireTimedOutOrders`: `expires_at <= NOW()` → `expires_at <= time.Now().Add(-3*time.Hour)`
  - 添加 3 小时缓冲期，避免极限情况下用户最后一秒付款但订单已过期
- `ListPendingOrders`: 移除时间判断，只查 `status = pending`
  - 因为过期订单已被 `ExpireTimedOutOrders` 标记为 `expired`，无需重复判断

### 3. 支持过期订单对账

**internal/handler/notify.go**:
- 允许 `expired` 状态的订单接收支付回调并更新为 `paid`
- 记录对账日志便于追踪

## 部署步骤

1. 重启服务使时区生效:
   ```bash
   docker compose down
   docker compose up -d
   ```

2. 验证时区已统一:
   ```sql
   SELECT @@global.time_zone, @@session.time_zone, NOW(), UTC_TIMESTAMP();
   -- 应该都显示 UTC，NOW() 和 UTC_TIMESTAMP() 相同
   ```

3. 手动修复已受影响的订单（如有）:
   ```sql
   UPDATE orders 
   SET status = 'paid', paid_at = NOW() 
   WHERE order_no = 'ORD202604170717096780' AND status = 'expired';
   ```
   然后手动触发充值或等待下次 ticker 自动处理。

## 验证

创建测试订单，观察：
- `expires_at` 应为 UTC 时间
- 订单不会在过期时间前被标记为 `expired`
- 过期 3 小时后才会被清理
