# dds-billing Docker 部署计划

## Context

项目需要部署到服务器，通过跳板机 git clone 代码后以 Docker 方式运行。前端需要 HTTPS 域名和 SSL 证书。

## 部署架构

```
                    ┌─────────────────────────────────┐
  用户浏览器 ──HTTPS──▶│  Nginx 容器 (:443/:80)          │
                    │  ├─ SSL 证书终结                  │
                    │  ├─ /api/* → proxy Go:3000       │
                    │  └─ /* → serve 前端静态文件        │
                    └──────────┬──────────────────────┘
                               │
                    ┌──────────▼──────────────────────┐
                    │  Go 后端容器 (:3000)              │
                    │  ├─ API 服务                     │
                    │  └─ 连接 MySQL                   │
                    └──────────┬──────────────────────┘
                               │
                    ┌──────────▼──────────────────────┐
                    │  MySQL 容器 (:3306)              │
                    │  └─ 数据持久化到 volume           │
                    └─────────────────────────────────┘
```

3 个容器：Nginx + Go 后端 + MySQL，通过 docker-compose 编排。

## 需要创建的文件


| 文件                   | 说明                             |
| -------------------- | ------------------------------ |
| `Dockerfile`         | 多阶段构建：编译 Go 二进制                |
| `docker-compose.yml` | 编排 3 个容器 + 网络 + 数据卷            |
| `nginx/nginx.conf`   | Nginx 配置：SSL、反代 API、serve 静态文件 |
| `nginx/Dockerfile`   | Nginx 镜像：构建前端 + 复制 dist + 配置文件 |
| `.dockerignore`      | 排除不需要的文件                       |
| `docs/deploy.md`     | 部署操作文档（完整步骤）                   |


## 需要修改的文件


| 文件                   | 变更                         |
| -------------------- | -------------------------- |
| `cmd/server/main.go` | 支持通过环境变量或命令行参数指定 config 路径 |
| `.gitignore`         | 添加 SSL 证书目录                |


## 实现细节

### 1. Dockerfile（Go 后端，多阶段构建）

- **Stage 1**: golang 镜像编译 Go 二进制
- **Stage 2**: 最小运行镜像（alpine），只包含二进制

### 2. nginx/Dockerfile

- **Stage 1**: node 镜像构建前端 `npm run build`
- **Stage 2**: nginx 镜像，复制 dist + nginx.conf

### 3. docker-compose.yml

三个 service：

- `mysql`: MySQL 8.0，数据卷持久化，自动创建数据库
- `backend`: Go 后端，依赖 mysql，挂载 config
- `nginx`: Nginx，依赖 backend，映射 80/443，挂载 SSL 证书

配置文件通过 volume 挂载（不打进镜像），证书目录也通过 volume 挂载。

### 4. nginx.conf

- 监听 80 → 301 跳转 HTTPS
- 监听 443 → SSL 配置
- `location /api/` → `proxy_pass http://backend:3000`
- `location /health` → `proxy_pass http://backend:3000`
- `location /` → 静态文件，`try_files $uri /index.html`（SPA）

### 5. 部署操作步骤

1. 服务器环境要求（Docker、Docker Compose、Git）
2. 通过跳板机 clone 代码
3. 创建配置文件（cp config.example.yaml config.yaml）
4. SSL 证书配置（申请 + 放置路径）
5. `docker compose up -d` 启动
6. 验证服务
7. Sub2API 管理后台配置充值页 URL
8. 常用运维命令（日志、重启、更新部署）

## 验证清单

1. `docker compose build` 构建成功
2. `docker compose up -d` 三个容器正常运行
3. `curl http://localhost/health` 返回 ok
4. 浏览器访问 HTTPS 域名能打开充值页
5. 创建订单 → 支付 → 回调 → 充值全链路正常

