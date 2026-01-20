![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/voidauth/voidauth/release.yml)
![GitHub License](https://img.shields.io/github/license/voidauth/voidauth)
![GitHub Release](https://img.shields.io/github/v/release/voidauth/voidauth?logo=github)
![GitHub Repo stars](https://img.shields.io/github/stars/voidauth/voidauth?style=flat&logo=github)


<br>
<p align="center">
  <a href='https://voidauth.app'>
    <img src="https://raw.githubusercontent.com/voidauth/voidauth/refs/heads/main/docs/logo_full_text.svg" width="180" title="VoidAuth" alt="VoidAuth logo"/>
  </a>
</p>

<p align="center">
  <strong>
    单点登录 - 您的自托管应用守护者
  </strong>
</p>

<br>

<div align="center">
  <a href="https://voidauth.app">官网</a> |
  <a href="https://github.com/voidauth/voidauth">源代码</a> |
  <a href="https://github.com/voidauth/voidauth/issues">问题反馈</a>
</div>

<br>

<p align="center">
  <img src="https://raw.githubusercontent.com/voidauth/voidauth/refs/heads/main/docs/public/screenshots/login_page.png" title="登录门户" alt="登录门户" width="280">
</p>

## 什么是 VoidAuth

VoidAuth 是一个开源的单点登录(SSO)认证和用户管理提供商，守护在您的自托管应用之前。它对管理员和终端用户都易于使用，支持密钥、用户邀请、自注册、邮件支持等实用功能！

## 主要特性

- 🌐 OpenID Connect (OIDC) 提供商
- 🔄 Proxy ForwardAuth 代理支持
- 👤 用户和组管理
- 📧 用户自注册和邀请
- 🎨 可定制化 (Logo、标题、主题颜色、邮件模板)
- 🔑 多因素认证、密钥和无密码登录支持
- 📧 安全的密码重置与邮件验证
- 🔒 Postgres 或 SQLite 数据库加密存储
- 🇨🇳 **中文界面支持** (i18n)
- 🛡️ **登录防暴力破解保护**

## 快速开始

### 使用 Docker Compose (推荐)

将以下内容保存为 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  voidauth:
    image: voidauth/voidauth:latest
    restart: unless-stopped
    ports:
      - "3000:3000"
    volumes:
      - ./voidauth/config:/app/config
      - ./voidauth/theme:/app/theme
    environment:
      APP_URL: https://auth.example.com
      STORAGE_KEY: ${STORAGE_KEY:-your-secure-random-key-min-32-chars}
      DB_HOST: voidauth-db
      DB_PASSWORD: ${DB_PASSWORD:-your-secure-db-password}
      ADMIN_INITIAL_PASSWORD: ${ADMIN_INITIAL_PASSWORD:-}
      LOGIN_MAX_ATTEMPTS: ${LOGIN_MAX_ATTEMPTS:-10}
      LOGIN_BLOCK_DURATION: ${LOGIN_BLOCK_DURATION:-15}
    depends_on:
      voidauth-db:
        condition: service_healthy

  voidauth-db:
    image: postgres:18
    restart: unless-stopped
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD:-your-secure-db-password}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -h localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

### 环境变量配置

创建 `.env` 文件：

```bash
# 必需的环境变量
APP_URL=https://auth.example.com
STORAGE_KEY=your-secure-random-key-min-32-chars
DB_HOST=voidauth-db
DB_PASSWORD=your-secure-db-password

# 可选: 初始管理员密码 (最少8个字符)
# 如果未设置，将自动生成32位随机密码
ADMIN_INITIAL_PASSWORD=your-admin-password

# 登录防暴力破解配置
LOGIN_MAX_ATTEMPTS=10      # 最大失败登录尝试次数
LOGIN_BLOCK_DURATION=15    # 锁定时间(分钟)
```

### 启动服务

```bash
# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f voidauth

# 停止服务
docker compose down

# 停止并删除数据卷
docker compose down -v
```

### 首次登录

> [!IMPORTANT]
> VoidAuth 首次启动时会自动创建初始管理员账户。
>
> - **用户名**: `auth_admin`
> - **密码**: 
>   - 如果设置了 `ADMIN_INITIAL_PASSWORD` 环境变量，则使用该密码
>   - 否则自动生成32位随机密码并记录在服务器日志中
> - **推荐**: 在部署前设置 `ADMIN_INITIAL_PASSWORD` 环境变量
> - **安全提示**: 登录后请立即从管理面板更改密码
>
> 查看日志获取初始密码:
> ```bash
> docker compose logs voidauth | grep -i password
> ```

## 功能说明

### 🇨🇳 多语言支持 (i18n)

VoidAuth 现在支持中文界面！

- 默认根据浏览器语言自动检测
- 可以在页面顶部的语言切换按钮手动切换
- 支持语言: English (en), 中文 (zh)
- 语言偏好会保存在本地存储中

**配置默认语言** (可选):
```bash
APP_LANGUAGE=zh  # 设置默认语言为中文
```

### 🛡️ 登录防暴力破解保护

系统内置登录保护功能，防止暴力破解攻击：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `LOGIN_MAX_ATTEMPTS` | 10 | 最大失败登录尝试次数 |
| `LOGIN_BLOCK_DURATION` | 15 | 锁定时间(分钟) |

**锁定策略: 用户名 + IP 组合**

系统采用 **"用户名:IP"** 组合作为锁定标识，而非单独锁定 IP 或账号。这种设计有其安全考量：

| 攻击场景 | 锁定表现 |
|----------|----------|
| 攻击者尝试破解 `admin` 账号 | `admin:192.168.1.100` 被锁定，不影响其他用户 |
| 同一 IP 尝试多个账号 | 每个 "用户名:IP" 独立计数，只锁定具体账号 |
| 同一账号来自不同 IP | 每个 IP 独立计数，不会误封正常用户 |

**工作原理**:
1. 每次失败登录记录 `{用户名}:{IP}` 组合的尝试次数
2. 达到最大尝试次数后，该组合被锁定
3. 锁定期间返回 HTTP 429 状态码
4. 成功登录后计数器重置
5. 锁定时间到期后自动解除 (内存存储，重启Docker后失效)

**安全优势**:
- ✅ 防止针对单一账号的暴力破解
- ✅ 防止分布式攻击 (同一IP试不同账号)
- ✅ 不误封正常用户 (不同IP的同一账号不受影响)
- ✅ 攻击者无法通过切换账号绕过限制

**锁定示例**:
```
攻击者: 192.168.1.100
尝试 admin 失败 10 次 → admin:192.168.1.100 被锁定 15 分钟
切换尝试 user 失败 10 次 → user:192.168.1.100 被锁定 15 分钟
切换 IP 为 192.168.1.101 → 可继续尝试 (新IP未达上限)
```

**示例响应**:
```json
// 登录失败 (401)
{
  "message": "Invalid username or password.",
  "remainingAttempts": 3
}

// 被锁定 (429)
{
  "message": "Too many failed login attempts. Please try again in 15 minutes.",
  "remainingMinutes": 15
}
```

### 初始管理员配置

**方式1: 使用环境变量 (推荐)**
```bash
ADMIN_INITIAL_PASSWORD=YourSecurePassword123
```

**方式2: 查看日志**
```bash
docker compose logs voidauth
```

**重要安全提示**:
- 密码长度至少8个字符
- 首次登录后请立即更改密码
- 建议使用强密码

## 管理面板

管理员可以通过侧边栏菜单访问管理面板，进行用户和设置管理。

<p align="center">
  <img src="https://raw.githubusercontent.com/voidauth/voidauth/refs/heads/main/docs/public/screenshots/admin_panel.png" title="管理面板" alt="管理面板" width="600">
</p>

## 配置选项

### 环境变量参考

| 变量名 | 必需 | 默认值 | 说明 |
|--------|------|--------|------|
| `APP_URL` | 是 | - | 应用URL，如 https://auth.example.com |
| `STORAGE_KEY` | 是 | - | 32字符以上的随机密钥 |
| `DB_HOST` | 是 | - | 数据库主机名 |
| `DB_PASSWORD` | 是 | - | 数据库密码 |
| `DB_PORT` | 否 | 5432 | 数据库端口 |
| `DB_NAME` | 否 | voidauth | 数据库名称 |
| `DB_USER` | 否 | postgres | 数据库用户 |
| `ADMIN_INITIAL_PASSWORD` | 否 | 自动生成 | 初始管理员密码 |
| `LOGIN_MAX_ATTEMPTS` | 否 | 10 | 最大登录尝试次数 |
| `LOGIN_BLOCK_DURATION` | 否 | 15 | 锁定时间(分钟) |
| `APP_LANGUAGE` | 否 | auto | 默认语言 (en/zh/auto) |
| `REDIS_URL` | 否 | - | Redis连接字符串 |
| `SMTP_HOST` | 否 | - | SMTP服务器地址 |
| `SMTP_PORT` | 否 | 587 | SMTP端口 |
| `SMTP_USER` | 否 | - | SMTP用户名 |
| `SMTP_PASS` | 否 | - | SMTP密码 |
| `SMTP_FROM` | 否 | - | 发件人地址 |

### 持久化存储

| 路径 | 说明 |
|------|------|
| `/app/config` | 配置文件目录 |
| `/app/theme` | 主题文件目录 |
| `/app/db` | SQLite数据库文件 (如使用SQLite) |

## 邮件配置

要启用邮件功能 (密码重置、邀请等)，需要配置SMTP:

```yaml
environment:
  SMTP_HOST: smtp.example.com
  SMTP_PORT: 587
  SMTP_USER: your-email@example.com
  SMTP_PASS: your-email-password
  SMTP_FROM: VoidAuth <noreply@example.com>
```

## 常见问题

### Q: 如何创建新用户?
A: 管理员可以通过"邀请"页面创建新邀请，然后发送邀请链接给用户。

### Q: 如何更改管理员密码?
A: 登录后点击右上角头像，选择"个人设置" -> "更改密码"。

### Q: 忘记了管理员密码怎么办?
A:
1. 查看Docker日志: `docker compose logs voidauth`
2. 或设置新的 `ADMIN_INITIAL_PASSWORD` 环境变量并重启容器

### Q: 支持哪些数据库?
A: 支持 PostgreSQL (推荐) 和 SQLite。

### Q: 如何升级 VoidAuth?
A:
```bash
docker compose pull voidauth/voidauth:latest
docker compose up -d
```

### Q: 中文界面不生效?
A:
1. 确保浏览器语言设置为中文
2. 或使用语言切换按钮
3. 或设置 `APP_LANGUAGE=zh` 环境变量

### Q: 登录保护会在Docker重启后重置吗?
A: 是的，由于使用内存存储，重启Docker后会重置所有计数器。

## 技术栈

- [node-oidc-provider](https://github.com/panva/node-oidc-provider) - OIDC 提供商功能
- [SimpleWebAuthn](https://github.com/MasterKale/SimpleWebAuthn) - 密钥/无密码登录支持
- [Angular](https://angular.dev) - 前端框架
- [Knex](https://knexjs.org/) - 数据库连接和查询构建器
- [zxcvbn-ts](https://zxcvbn-ts.github.io/zxcvbn/) - 密码强度计算

## 赞助商

[![](https://github.com/GitTimeraider.png?size=50)](https://github.com/GitTimeraider)

## 致谢

本项目的实现离不开以下优秀工作的支持:

- 感谢所有开源贡献者
- 感谢用户的问题反馈和建议

## 免责声明

VoidAuth 尚未经过安全审计，且使用了第三方包实现大部分功能，请自行承担使用风险。
