# 国际化 (i18n) 设置指南

本项目已添加中英文国际化支持。

## 安装依赖

在项目根目录运行以下命令：

```bash
npm install
```

或者在 frontend 目录下运行：

```bash
cd frontend && npm install
```

## 功能说明

1. **语言切换**：在页面顶部导航栏中点击语言图标（地球图标）可在中文和英文之间切换
2. **自动语言检测**：首次访问时会根据浏览器语言自动选择合适的语言
3. **持久化存储**：语言选择会保存在 localStorage 中，下次访问时会记住用户的选择

## 支持的语言

- 英语 (en) - 默认语言
- 中文 (zh) - 简体中文

## 添加新的翻译

1. 编辑翻译文件：
   - 英文翻译：`frontend/src/assets/i18n/en.json`
   - 中文翻译：`frontend/src/assets/i18n/zh.json`

2. 在组件中使用翻译：
   ```html
   <span>{{ 'your.translation.key' | translate }}</span>
   ```

3. 在 TypeScript 中使用翻译：
   ```typescript
   constructor(private translate: TranslateService) {
     const translatedText = this.translate.instant('your.translation.key');
   }
   ```

## 已翻译的页面

- 登录页面
- 注册页面
- 忘记密码页面
- 重置密码页面
- 双因素认证 (MFA) 页面
- 邮箱验证页面
- 授权同意页面
- 需要审批页面
- 顶部导航栏

## 技术实现

使用了 `@ngx-translate` 库来实现国际化功能：

- `@ngx-translate/core`: 核心翻译功能
- `@ngx-translate/http-loader`: 从 JSON 文件加载翻译

## 故障排除

如果翻译不显示：

1. 确保 `@ngx-translate` 包已正确安装
2. 检查浏览器控制台是否有加载翻译文件的错误
3. 确保 JSON 翻译文件的格式正确
4. 尝试清除浏览器缓存并重新加载页面

## 扩展语言支持

要添加新语言支持：

1. 在 `frontend/src/assets/i18n/` 目录下创建新的 JSON 文件（例如 `fr.json` 用于法语）
2. 在 `i18n.service.ts` 中添加新语言代码：

```typescript
this.translate.addLangs(['en', 'zh', 'fr'])
```

3. 翻译现有的所有文本到新语言
4. 根据需要更新语言切换逻辑
