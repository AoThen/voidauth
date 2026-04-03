# Goauth

精简的 OIDC Provider，使用 Go + Alpine.js，SQLite 数据库，适合小规模部署。

## 特性

- **OIDC Provider** - 完整的 OpenID Connect 支持
- **密码认证** - 支持 TOTP 二步验证
- **管理后台** - 用户/分组/客户端/邀请/代理认证管理
- **邀请注册** - 生成邀请链接，自动关联分组
- **ProxyAuth** - 支持 Traefik ForwardAuth 和 Nginx auth_request
- **单二进制** - 无外部依赖，SQLite 嵌入式数据库

## 快速开始

### Docker

```bash
docker run -d \
  -p 3000:3000 \
  -v goauth-data:/app/data \
  -e APP_OIDC_ISSUER=http://localhost:3000 \
  ghcr.io/aoten/goauth:latest
```

### Docker Compose

```yaml
services:
  goauth:
    image: ghcr.io/aoten/goauth:latest
    ports:
      - "3000:3000"
    volumes:
      - goauth-data:/app/data
    environment:
      - APP_OIDC_ISSUER=http://localhost:3000
      - APP_UI_APPNAME=Goauth

volumes:
  goauth-data:
```

### 从源码构建

```bash
git clone https://github.com/aoten/goauth.git
cd goauth
make build
./bin/goauth serve
```

## 配置

### 环境变量

所有配置使用 `APP_` 前缀：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `APP_SERVER_PORT` | 服务端口 | `3000` |
| `APP_SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `APP_SERVER_ENVIRONMENT` | 环境 (development/production) | `development` |
| `APP_SERVER_APPURL` | 应用 URL | - |
| `APP_DATABASE_PATH` | SQLite 数据库路径 | `./data/goauth.db` |
| `APP_OIDC_ISSUER` | OIDC Issuer URL | `http://localhost:3000` |
| `APP_UI_APPNAME` | 应用名称 | `Goauth` |
| `APP_UI_APPCOLOR` | 主题颜色 | `#906bc7` |
| `APP_UI_SIGNUPENABLED` | 开放注册 | `true` |
| `APP_SECURITY_PASSWORDMIN` | 密码最小长度 | `8` |
| `APP_SECURITY_PASSWORDMINSCORE` | 密码强度分数 (0-4) | `3` |
| `APP_SECURITY_LOGINMAXATTEMPTS` | 登录失败次数限制 | `10` |
| `APP_SECURITY_LOGINBLOCKDURATION` | 封禁时长 (分钟) | `30` |
| `APP_LOGGING_LEVEL` | 日志级别 | `info` |

## CLI 命令

```bash
# 启动服务器（默认命令）
goauth serve [--config config.yaml]

# 运行数据库迁移
goauth migrate [--config config.yaml]

# 重置用户密码
goauth reset-password -u <username>

# 创建管理员
goauth create-admin -u <username> -p <password>

# 查看版本
goauth --version
```

## ProxyAuth 配置

### Traefik ForwardAuth

```yaml
services:
  app:
    image: myapp
    labels:
      - "traefik.http.middlewares.goauth.forwardauth.address=http://goauth:3000/authz/forward-auth"
      - "traefik.http.middlewares.goauth.forwardauth.authResponseHeaders=X-Auth-User,X-Auth-Email"
      - "traefik.http.routers.app.middlewares=goauth@docker"
```

### Nginx auth_request

```nginx
server {
    location / {
        auth_request /authz;
        auth_request_set $user $upstream_http_x_auth_user;
        proxy_set_header X-Auth-User $user;
        proxy_pass http://app;
    }

    location = /authz {
        internal;
        proxy_pass http://goauth:3000/authz/auth-request;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header X-Original-URI $request_uri;
    }
}
```

## 与完整版区别

| 功能 | Goauth | Goauth (Full) |
|------|--------|----------|
| 数据库 | SQLite | PostgreSQL/SQLite |
| 前端 | Alpine.js | Angular |
| 邮件 | ❌ | ✅ |
| Passkey | ❌ | ✅ |
| 国际化 | 仅中文 | 多语言 |
| Consent 页面 | 自动授权 | 用户确认 |
| 二进制大小 | ~25MB | ~100MB |

## 安全特性

- **密码哈希** - bcrypt (cost=10)
- **密码强度** - zxcvbn 检测
- **暴力破解防护** - 失败次数限制 + IP 封禁
- **会话管理** - 安全 Cookie，支持"记住我"
- **审计日志** - 关键操作记录

## 项目结构

```
goauth/
├── cmd/server/main.go        # 入口
├── internal/
│   ├── config/               # 配置管理
│   ├── db/                   # 数据库
│   ├── handler/              # HTTP 处理器
│   ├── middleware/           # 中间件
│   ├── model/                # 数据模型
│   ├── oidc/                 # OIDC Provider
│   ├── repo/                 # 数据访问层
│   ├── service/              # 业务逻辑层
│   └── util/                 # 工具函数
├── migrations/               # 数据库迁移
├── web/                      # 前端文件
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 构建

```bash
# 本地构建
make build

# Docker 构建
make docker

# 运行测试
make test-integration
```

## License

MIT