# 使用系统 MySQL 配置说明

## 变更内容

由于云主机已安装独立的 MySQL 服务（端口 3306），移除了 docker-compose.yml 中的 MySQL 容器，backend 改用 `network_mode: host` 直接连接宿主机 MySQL。

## 部署前准备

### 1. 配置系统 MySQL 时区为 UTC

```bash
sudo mysql -uroot -p
```

执行：
```sql
SET GLOBAL time_zone = '+00:00';
SET SESSION time_zone = '+00:00';
```

永久生效，编辑 `/etc/mysql/mysql.conf.d/mysqld.cnf`：
```ini
[mysqld]
default-time-zone = '+00:00'
```

重启 MySQL：
```bash
sudo systemctl restart mysql
```

验证：
```sql
SELECT @@global.time_zone, @@session.time_zone, NOW(), UTC_TIMESTAMP();
-- 应该都显示 +00:00，NOW() 和 UTC_TIMESTAMP() 相同
```

### 2. 创建数据库和用户（如果还没有）

```sql
CREATE DATABASE IF NOT EXISTS dds_billing CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 如果需要单独的用户（可选，也可以继续用 root）
CREATE USER IF NOT EXISTS 'dds_billing'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON dds_billing.* TO 'dds_billing'@'localhost';
FLUSH PRIVILEGES;
```

### 3. 更新 configs/config.yaml

确保 DSN 指向 localhost（因为 backend 用了 host 网络模式）：
```yaml
database:
  dsn: "root:your_password@tcp(localhost:3306)/dds_billing?charset=utf8mb4&parseTime=True&loc=UTC"
```

注意 `loc=UTC` 确保 Go 端也用 UTC 解析时间。

## 部署

```bash
sudo docker compose down
sudo docker compose up -d --build
```

## 验证

```bash
sudo docker compose logs -f backend
```

应该看到：
- `[checker] started, interval=60s`
- 数据库连接成功
- 没有时区相关错误
