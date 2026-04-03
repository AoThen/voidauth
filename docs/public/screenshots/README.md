# 截图说明

本目录存放文档所需的截图文件。

## 截图列表

### 用户指南 (user-guide.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| login.png | 登录页面 | 500x500 |
| signup.png | 注册页面 | 500x600 |
| mfa-required.png | MFA 验证页面 | 待补充 |
| mfa-register.png | MFA 注册页面 | 待补充 |
| profile.png | 个人设置页面 | 500x600 |

### 用户管理 (user-management.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| invitation.png | 邀请管理页面 | 1400x900 |
| user-edit.png | 用户管理页面 | 1400x900 |

### 安全组 (security-groups.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| groups.png | 安全组管理页面 | 1400x900 |

### 密码重置 (password-reset.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| password-resets.png | 密码重置管理页面 | 1400x900 |

### OIDC 应用设置 (oidc-setup.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| oidc-client.png | OIDC 客户端配置页面 | 1400x900 |
| oidc-endpoints.png | OIDC 端点信息面板 | 待补充 |

### 代理认证 (proxy-auth.md)

| 文件名 | 说明 | 尺寸 |
|--------|------|------|
| proxy-domain.png | ProxyAuth 域名设置页面 | 1400x900 |
| proxy-authz.png | ProxyAuth 授权规则示例 | 待补充 |

## 截图规范

- **格式**: PNG
- **主题**: 使用应用默认主题（紫色系）
- **敏感信息**: 已模糊处理用户名、邮箱等敏感信息
- **尺寸**: 
  - 登录/注册等小型表单: 500px 宽度
  - 管理后台页面: 1400x900 全屏

## 重新生成截图

```bash
cd goauth
make build
./bin/goauth serve &
cd tests/e2e
npx playwright test --config=screenshot.config.ts
```