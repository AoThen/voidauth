# ProxyAuth 代理认证

ProxyAuth 域名通过反向代理与 Goauth 协作来保护应用。当用户访问受保护的域名时，反向代理会向 Goauth 验证用户是否已登录且有权访问。

你可以在 Goauth 管理员的 ProxyAuth 域名页面设置 ProxyAuth 保护的域名。

<p align=center>
<img align=center src="/public/screenshots/proxy_auth_domain.png" width="300" />
</p>

## 识别用户 Identifying Users

ProxyAuth 可以通过以下三种方法识别尝试访问受保护域名的用户，按顺序尝试：

1. 检查 `x-goauth-session` 中的有效用户会话 Cookie。这是用户通过浏览器发起请求时最常见的识别方式。
2. 检查 `Proxy-Authorization` 头是否使用 Basic Auth，如果是，则根据头中的用户名和密码识别用户。如果设置了该头但未找到用户，则返回 407 状态码并设置 Proxy-Authenticate 响应头。
3. 检查 `Authorization` 头是否使用 Basic Auth，如果是，则根据头中的用户名和密码识别用户。如果设置了该头但未找到用户，则返回 401 状态码并设置 WWW-Authenticate 响应头。

## 授权 Authorization

> [!CAUTION]
> 如果 ProxyAuth 域名未分配任何安全组，则**任何已登录用户**都可以访问该域名。

当用户访问受保护的域名时，系统将从**最具体**到**最不具体**匹配第一个 ProxyAuth 域名来检查访问权限。在下面的示例中，只有 **[users]** 组的用户无法访问 `app.example.com/admin/user_accounts`，但可以访问 `app.example.com/home`。同样，他们也无法访问 `secret.example.com`，该域名会被 `*.example.com/*` 匹配，只允许 `admin` 组的用户访问。

<p align=center>
<img align=center src="/public/screenshots/3f0b0afc-5bcf-436c-8def-f45e68adb019.png" width="800" />
</p>

创建 ProxyAuth 域名时，尾部的 `/` 和 `.` 等分隔符**会被检查**。访问 `*.example.com` 的权限不包括 `example.com`，它们需要分别添加。

> [!IMPORTANT]
> 你可以设置通配符 ProxyAuth 域名 `*/*`，它将覆盖 ProxyAuth 域名设置中其他条目未匹配的任何域名。

## 响应 Responses

当用户被识别并确认有权访问请求的资源时，将发送 `200` 成功响应。

在所有拒绝的请求中，响应会添加一个 `Location` 头，指向 Goauth 登录页面的位置。

* 当用户未被识别时，根据使用的授权端点和识别方法，将发送 `401`、`407` 或 `302` 响应码。
* 当用户被找到但**拒绝**访问时，将发送 `403` Forbidden 响应码。

## Trusted Header SSO 可信头单点登录

如果请求被允许，响应中将设置以下头信息。这可在某些自托管应用上启用可信头 SSO：
* `Remote-User` = `username`
* `Remote-Email` = `email`
* `Remote-Name` = `name`
* `Remote-Groups` = `groups`，以逗号分隔的列表，例如 `users,admins,owners`

## 反向代理 ProxyAuth 设置 Reverse-Proxy ProxyAuth Setup

Goauth 暴露两个代理认证端点，使用哪个取决于你的反向代理。

| 端点                    | 反向代理      |
| ------------------------| ------------- |
| /api/authz/forward-auth | [Caddy](https://caddyserver.com/docs/caddyfile/directives/forward_auth), [Traefik](https://doc.traefik.io/traefik/middlewares/http/forwardauth/)  |
| /api/authz/auth-request | [NGINX](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html) |

这些端点基本相同，但 NGINX 的限制使得需要单独的端点。

> [!WARNING]
> 你必须**同时**正确设置反向代理和 Goauth 才能保护域名！这通常涉及修改反向代理配置将域名"置于"Goauth 之后，然后在 Goauth ProxyAuth 域名列表中添加该域名并设置访问组。

### Caddy

使用 [Caddy](https://caddyserver.com) 作为反向代理，可以通过以下 CaddyFile 设置 ProxyAuth，保护域名 **app.example.com**，Goauth 托管在 **auth.example.com**：

``` Caddy
# 托管 Goauth
auth.example.com {
  reverse_proxy goauth:3000
}

# # 或在 example.com 的子目录托管 Goauth
# example.com {
#   # 不要重写或剥离路径
#   # 使用 'handle' 而不是 'handle_path'
#   handle /auth/* {
#     reverse_proxy goauth:3000
#   }
# }

# 托管受保护的应用
app.example.com {
  forward_auth goauth:3000 {
    uri /api/authz/forward-auth
    copy_headers Remote-User Remote-Groups Remote-Name Remote-Email
  }

  reverse_proxy app:8080
}
```

### NGINX Snippets 配置片段

要使用 NGINX 或 NGINX Proxy Manager，需要将 `snippets/` 目录提供给配置使用。在所有示例中，该目录挂载/位于 `/config/nginx/snippets/`。

`proxy.conf`
``` conf
proxy_set_header Host $host;
proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Host $http_host;
proxy_set_header X-Forwarded-URI $request_uri;
proxy_set_header X-Forwarded-For $remote_addr;
```

`websockets.conf`
``` conf
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

`auth-location.conf`
``` conf
location /api/authz/auth-request {
  internal;

  include /config/nginx/snippets/proxy.conf;

  # 额外设置，不要将整个请求体传递给 auth_request
  proxy_set_header Content-Length "";
  proxy_set_header Connection "";
  proxy_pass_request_body off;

  # 发送 auth_request 的 URL。应为 ${APP_URL}/api/authz/auth-request
  proxy_pass http://goauth:3000/api/authz/auth-request;
}
```

`proxy-auth.conf`
``` conf
auth_request /api/authz/auth-request;

proxy_set_header Remote-User $upstream_http_remote_user;
proxy_set_header Remote-Groups $upstream_http_remote_groups;
proxy_set_header Remote-Email $upstream_http_remote_email;
proxy_set_header Remote-Name $upstream_http_remote_name;

# 如果响应为 401 或 407 状态码，尝试像 302 一样重定向到 Location 头
# NGINX auth_request 只能处理 2xx 和 4xx 状态码，这是变通方案
auth_request_set $redirection_url $upstream_http_location;
error_page 401 =302 $redirection_url;
error_page 407 =302 $redirection_url;
```

### NGINX

NGINX 通过挂载在 `/etc/nginx/nginx.conf` 的 `nginx.conf` 文件进行配置。在此示例中，我们还将配置片段挂载到 `/config/nginx/snippets/`，供配置使用。

> [!WARNING]
> 虽然超出了本指南范围，但你**必须**为面向浏览器的反向代理设置 `https://` 终止，并使用浏览器信任的证书，否则 Goauth 将**无法**正常工作。裸 NGINX 反向代理**不会**自动完成此操作。

`nginx.conf`
``` conf
events {
  worker_connections 1024;
}

http {

  # 托管 Goauth
  server {
    listen 443 ssl http2;
    server_name auth.example.com;

    # SSL (https) 配置...

    location / {
      include /config/nginx/snippets/proxy.conf;
      proxy_pass http://goauth:3000;
    }
  }

  # # 或在 example.com 的子目录托管 Goauth
  # server {
  #   listen 443 ssl http2;
  #   server_name example.com;

  #   # 子目录托管
  #   location /goauth/ {
  #     include /config/nginx/snippets/proxy.conf;
  #     proxy_pass http://goauth:3000;
  #   }
  # }

  # 托管受保护的应用
  server {
    listen 443 ssl http2;
    server_name app.example.com;

    # SSL (https) 配置...

    # 配置片段使 Goauth 可用于 auth_request
    include /config/nginx/snippets/auth-location.conf;

    location / {
      include /config/nginx/snippets/proxy.conf;
      # 如果受保护的应用使用 websockets，取消下面这行的注释
      # include /config/nginx/snippets/websockets.conf;

      # 配置片段在访问受保护应用前向 Goauth 发送 auth_request
      include /config/nginx/snippets/proxy-auth.conf;
      proxy_pass http://app:8080;
    }
  }
}
```

### NGINX Proxy Manager

作为基于 NGINX 的反向代理，NGINX Proxy Manager 也需要相同的 [NGINX Snippets](#nginx-snippets-配置片段)，在所有示例中假设挂载在 `/config/nginx/snippets/`。以下示例假设你在 example.com 的子域名上托管 Goauth 和受保护的应用，如果在子目录托管则需要调整。

#### 托管 Goauth：

1. 访问 `Proxy Hosts` 页面并 `Add a Proxy Host`
2. 填写 `Details` 标签页
    * <img src="/public/screenshots/NPM_goauth_details.png" width="500" />
3. 在 `SSL` 标签页填写 SSL 配置
4. 在 `Advanced` 标签页填写；这是为了包含 `proxy.conf` 配置片段，之后添加任何受保护应用也需要这样做
    ``` conf
    location / {
      include /config/nginx/snippets/proxy.conf;
      proxy_pass $forward_scheme://$server:$port;
    }
    ```
    * <img src="/public/screenshots/NPM_goauth_advanced.png" width="500" />

#### 托管受保护的应用：

1. 访问 `Proxy Hosts` 页面并 `Add a Proxy Host`
2. 填写 `Details` 标签页
    * <img src="/public/screenshots/NPM_app_details.png" width="500" />
3. 在 `SSL` 标签页填写 SSL 配置
4. 在 `Advanced` 标签页填写；此配置将包含指示 NGINX 使用 Goauth 进行 ProxyAuth 的配置片段
    ``` conf
    include /config/nginx/snippets/auth-location.conf;

    location / {
      include /config/nginx/snippets/proxy.conf;
      # 如果受保护的应用使用 websockets，取消下面这行的注释
      # include /config/nginx/snippets/websockets.conf;

      include /config/nginx/snippets/proxy-auth.conf;
      proxy_pass $forward_scheme://$server:$port;
    }
    ```
    * <img src="/public/screenshots/NPM_app_advanced.png" width="500" />

> [!WARNING]
> 如果受保护的应用需要 websockets，请确保在 `Advanced` 标签页中取消包含 `websockets.conf` 配置片段的那行注释。

### Traefik

以下是一个 `compose.yml` 文件示例，使用 Traefik 配置 Goauth 作为 ProxyAuth 中间件来保护 whoami 应用。此示例不包含证书和 `https://` 设置，相关文档可参考 [Traefik TLS Overview](https://doc.traefik.io/traefik/reference/routing-configuration/http/tls/overview/)，但这是**必需的**。

``` yaml
volumes:
  traefik-config:
  db:

services:
  traefik:
    container_name: traefik
    image: traefik:v3.5
    command:
      # EntryPoints
      - "--entrypoints.web.address=:80"
      - "--entrypoints.web.http.redirections.entrypoint.to=websecure"
      - "--entrypoints.web.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.web.http.redirections.entrypoint.permanent=true"
      - "--entrypoints.websecure.address=:443"
      - "--entrypoints.websecure.http.tls=true"

      # Providers 
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--providers.docker.network=proxy"

      # API & Dashboard 
      - '--api=true'
      - "--api.dashboard=true"
      - "--api.insecure=false"

      # Observability 
      - '--log=true'
      - "--log.level=INFO"
      - "--accesslog=true"
      - "--metrics.prometheus=true"
      - '--global.sendAnonymousUsage=false'
    ports:
      - '80:80'
      - '443:443'
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - traefik-config:/config
    labels:
      traefik.enable: 'true'
      traefik.http.routers.api.rule: 'Host(`traefik.example.com`)'
      traefik.http.routers.api.entryPoints: 'websecure'
      traefik.http.routers.api.tls: 'true'
      traefik.http.routers.api.service: 'api@internal'
      traefik.http.routers.api.middlewares: 'goauth@docker'
    depends_on:
      - goauth

  goauth: 
    image: goauth/goauth:latest
    volumes:
      - ./goauth/config:/app/config
    environment:
      # 必需的环境变量
      APP_URL: # 必需，例如 https://auth.example.com
      STORAGE_KEY: # 必需
      DB_PASSWORD: # 必需
      DB_HOST: goauth-db
    labels:
      traefik.enable: 'true'
      traefik.http.routers.goauth.rule: 'Host(`auth.example.com`)'
      traefik.http.routers.goauth.entryPoints: 'websecure'
      traefik.http.routers.goauth.tls: 'true'
      traefik.http.middlewares.goauth.forwardAuth.address: 'http://goauth:3000/api/authz/forward-auth'
      traefik.http.middlewares.goauth.forwardAuth.trustForwardHeader: 'true'
      traefik.http.middlewares.goauth.forwardAuth.authResponseHeaders: 'Remote-User,Remote-Name,Remote-Email,Remote-Groups'
    depends_on:
      - goauth-db

  goauth-db:
    image: postgres:18
    environment:
      POSTGRES_PASSWORD: # 必需
    volumes:
      - db:/var/lib/postgresql/18/docker
    
  whoami:
    container_name: whoami
    image: traefik/whoami
    labels:
      traefik.enable: 'true'
      traefik.http.routers.whoami.rule: 'Host(`whoami.example.com`)'
      traefik.http.routers.whoami.entryPoints: 'websecure'
      traefik.http.routers.whoami.tls: 'true'
      traefik.http.routers.whoami.middlewares: 'goauth@docker'
```