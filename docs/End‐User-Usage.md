# 用户体验 User Experience

## 登录 Login

如果用户尚未登录 Goauth，并且直接访问 **APP_SERVER_APPURL** 或通过 **OIDC 应用**或 **Proxy Auth** 流程重定向到登录门户，则会进入登录页面。Goauth 支持密码登录和 TOTP 双因素认证。"记住我"复选框将会话延长至 30 天。

<p align=center>
<img src="/public/screenshots/login_page.png" width="375" />
</p>

## 多因素认证 (MFA)

如果用户因全局策略、组成员身份或所访问的 OIDC 应用或 ProxyAuth 域名的安全策略需要 MFA，则会被引导至多因素认证页面。如果用户账户上没有可用的 MFA 方式但需要 MFA，他们将有机会在此页面上设置一个。

<p align=center>
<img src="/public/screenshots/mfa_required.png" width="375" />
</p>

<p align=center>
<img src="/public/screenshots/mfa_required_register.png" width="375" />
</p>

## 注册 Sign Up

如果 **APP_UI_SIGNUPENABLED** 环境变量设置为 `true`，[登录](#登录-login)页面上将显示注册选项。用户名是必填项。密码强度要求由 **APP_SECURITY_PASSWORDMINSCORE** 环境变量设置。密码强度使用 [zxcvbn](https://github.com/nbutton23/zxcvbn-go) 计算。

<p align=center>
<img src="/public/screenshots/sign_up.png" width="375" />
</p>

## 接受邀请 Accept Invitation

当用户访问有效的邀请链接时显示此页面，该页面与[注册](#注册-sign-up)表单字段相同，但管理员已填写的字段会预先填充。

## 个人设置 Profile Settings

用户直接访问 Goauth 时的默认页面，用户可以在此更改个人设置、密码或管理 TOTP 双因素认证。

<p align=center>
<img src="/public/screenshots/091a0122-75d7-44d0-9c97-e395c945cf4f.png" width="375" />
</p>
