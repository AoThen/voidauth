# 密码重置 Password Resets

密码重置请求从管理员密码重置页面管理。在那里，你可以通过在顶部搜索栏搜索来为用户创建新的密码重置请求。你也可以查看现有的密码重置请求。对于每个请求，你可以复制重置链接或删除请求。

> [!NOTE]
> 由于 Goauth 不包含邮件功能，密码重置链接必须由管理员手动发送给用户。

<p align=center>
<img width="336" alt="image" src="/public/screenshots/admin-password-resets.png" />
</p>

## CLI 密码重置

你也可以使用 CLI 生成密码重置：

```bash
goauth reset-password -u <username>
```

这将向控制台输出一个新的临时密码。
