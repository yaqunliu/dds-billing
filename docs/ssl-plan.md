# SSL 证书自动化 + 完整部署流程

## Context

项目使用 Let's Encrypt 免费证书，需要自动化证书申请和续签。当前 nginx 配置已支持 SSL，但证书需要手动获取和复制。需要脚本化整个流程，并更新部署文档使步骤清晰完整。

## 方案

使用 certbot Docker 容器申请和续签证书，不需要在宿主机安装 certbot。

### 证书申请流程

1. 首次申请时 HTTPS 还不可用，需要 Nginx 先以 HTTP 模式运行（serve certbot challenge）
2. certbot 通过 webroot 方式验证域名（写文件到 `/.well-known/acme-challenge/`）
3. 证书申请成功后，Nginx 切换到 HTTPS 模式

### 需要创建的文件


| 文件                      | 说明                                         |
| ----------------------- | ------------------------------------------ |
| `scripts/init-ssl.sh`   | 首次申请证书脚本（启动临时 nginx → certbot 申请 → 重启正式服务） |
| `scripts/renew-ssl.sh`  | 证书续签脚本（certbot renew → reload nginx）       |
| `nginx/nginx.init.conf` | 初始化用的 Nginx 配置（仅 HTTP，用于 certbot 验证）       |


### 需要修改的文件


| 文件                   | 说明                                                |
| -------------------- | ------------------------------------------------- |
| `nginx/nginx.conf`   | 添加 `/.well-known/acme-challenge/` location，供续签时使用 |
| `docker-compose.yml` | 添加 certbot 数据卷映射，nginx 挂载 certbot webroot         |
| `docs/deploy.md`     | 重写部署步骤，整合证书申请流程                                   |
| `.gitignore`         | 添加 certbot 数据目录                                   |


### 实现细节

#### scripts/init-ssl.sh

1. 接收域名参数和邮箱参数
2. 用 docker run 启动临时 nginx 容器（nginx.init.conf，仅 80 端口）
3. 用 docker run certbot/certbot 执行 certonly --webroot
4. 证书输出到 `./ssl/letsencrypt/`
5. 停掉临时 nginx
6. 提示用户执行 `docker compose up -d` 启动正式服务

#### scripts/renew-ssl.sh

1. 用 docker run certbot/certbot renew
2. docker compose exec nginx nginx -s reload

#### nginx.conf 改动

- 在 HTTPS server 块中添加 `location /.well-known/acme-challenge/`，指向 certbot webroot
- SSL 证书路径改为 `/etc/nginx/ssl/letsencrypt/live/{domain}/`

#### docker-compose.yml 改动

- nginx volumes 添加 certbot webroot 和证书目录的映射
- 证书目录从 `./ssl` 改为 `./ssl/letsencrypt`

### 完整部署步骤（deploy.md 更新）

1. 服务器环境准备（Docker、Docker Compose、Git）
2. clone 代码
3. 创建 config.yaml 和 .env
4. 首次申请 SSL 证书：`./scripts/init-ssl.sh your-domain.com your@email.com`
5. 构建并启动：`docker compose up -d`
6. 验证服务
7. 配置证书自动续签 crontab：`./scripts/renew-ssl.sh`
8. 配置 Sub2API 充值页 URL

## 验证

1. `./scripts/init-ssl.sh` 执行成功，证书文件存在于 `ssl/letsencrypt/live/domain/`
2. `docker compose up -d` 三个容器正常运行
3. `curl -I https://your-domain.com` 返回 200 且证书有效
4. `./scripts/renew-ssl.sh` 执行不报错



深度呼吸，什么叫深度呼吸，我们呼吸的方式可能是错的，你的呼吸模式一直停留在一种”最低配置“的状态，远远没有发挥出它应有的修复功能，我们的日常呼吸是浅呼吸，气只到了胸腔没有腹部，要想修复，你要深呼吸，深度呼吸是能主动触发”修复模式“的最简单方式，具体做法，坐下来或者躺下来，闭上眼睛，用鼻子慢慢吸气，细四秒，

用鼻子慢慢吸气，吸4秒，不是往胸腔吸，是往腹部吸，想象你的肚子是一个气球，你在把它慢慢吹大。吸满之后，屏住，停两秒，然后用嘴巴慢慢呼气，呼出6秒，做十次这种呼吸，大约2分钟