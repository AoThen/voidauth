# 故障排除 Troubleshooting

一些常见问题及其原因。

### 无法创建会话 Could Not Create Session

无法设置 `x-goauth-session` 或 `x-goauth-interaction` Cookie。请确保 `APP_SERVER_APPURL` 环境变量设置为 Goauth 的公开 URL，并且你是从该 URL 访问应用。

这也可能是由无效的 `APP_SERVER_COOKIEDOMAIN` 环境变量值引起的。浏览器可能不允许在顶级域名（如 `com`、`co.uk`、`lan`）以及某些公共域名（如 `azurewebsites.net`、`cdn.cloudflare.net`）上设置 Cookie。你可以在 [Mozilla HTTP Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Cookies#define_where_cookies_are_sent) 文档中了解更多信息，并查看当前的 [Public Suffix List](https://publicsuffix.org/list/) 了解受限制的域名。

### 无效客户端 Invalid Client

请确保已创建 OIDC 应用，并且 Goauth 和"客户端"应用中的 `Client ID` 参数完全匹配。

### 无效重定向 URI Invalid Redirect Uri

"客户端"应用应指定正确的 `Redirect URL` 参数以输入 Goauth。这通常位于应用的 OIDC 文档中或其 OIDC 设置页面上（如果有）。你也可以在 [OIDC 应用指南](OIDC-Guides.md)页面找到该应用的示例。