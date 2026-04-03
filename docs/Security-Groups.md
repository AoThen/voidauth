# 安全组 Security Groups

安全组在管理员安全组页面创建，用户可以从安全组更新或用户更新页面添加到安全组。

> [!IMPORTANT]
> 属于 `auth_admins` 组的 Goauth 用户将成为管理员。这是在使用初始管理员账户创建/邀请自己后提升权限的方式。

> [!TIP]
> 拥有 `auth_admins` 组的用户不会被拒绝访问任何资源，无论其他安全组限制如何。

<p align=center>
<img width="336" alt="image" src="/public/screenshots/91429974-7e2c-4c3a-80a4-ad25e5ea6416.png" />
</p>

安全组用于 OIDC 应用和 ProxyAuth 域名。

### ProxyAuth
安全组在 ProxyAuth 域名中用于：
* ProxyAuth 域名授权
* 可信头 SSO，用户的组会添加到 'Remote-Groups' 头中

有关 ProxyAuth 设置的信息，请访问 [ProxyAuth 设置指南](ProxyAuth-and-Trusted-Header-SSO-Setup.md)。

### OIDC
安全组在 OIDC 应用中用于：
* OIDC 应用授权
* 当 OIDC 应用请求 'groups' 作用域时随令牌发送。OIDC 应用可以使用组进行自己的授权，例如 [Jellyfin SSO Plugin](https://github.com/9p4/jellyfin-plugin-sso)

有关 OIDC 设置的信息，请访问 [OIDC 设置指南](OIDC-Setup.md)。