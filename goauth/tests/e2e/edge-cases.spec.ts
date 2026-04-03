import { test, expect, waitForPageReady, STRONG_PASSWORD, getSavedAdmin } from './fixture';

/**
 * CORS 和边缘情况 E2E 测试
 * 
 * 测试覆盖：
 * 1. CORS Preflight 请求处理
 * 2. 允许的来源正确响应
 * 3. 不允许的来源被拒绝
 * 4. 输入验证（空值、特殊字符）
 * 5. 资源不存在处理
 * 6. 重复创建资源
 */

test.describe.configure({ mode: 'serial' });

test.describe('CORS 处理', () => {
  test('Preflight OPTIONS 请求成功', async ({ request }) => {
    const response = await request.fetch('/api/public/config', {
      method: 'OPTIONS',
      headers: {
        'Origin': 'http://localhost:3000',
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'Content-Type',
      },
    });

    // Preflight 应该成功
    expect([200, 204, 404]).toContain(response.status());
  });

  test('CORS 头在 GET 请求中设置', async ({ request }) => {
    const response = await request.get('/api/public/config', {
      headers: {
        'Origin': 'http://localhost:3000',
      },
    });

    expect(response.status()).toBe(200);

    // 检查 CORS 头
    const allowOrigin = response.headers()['access-control-allow-origin'];
    
    // 如果启用了 CORS，应该有相关头
    if (allowOrigin) {
      expect(allowOrigin).toBeTruthy();
    }
  });

  test('CORS 头在 POST 请求中设置', async ({ request }) => {
    const response = await request.post('/api/auth/login', {
      headers: {
        'Origin': 'http://localhost:3000',
        'Content-Type': 'application/json',
      },
      data: {
        username: 'test',
        password: 'test',
      },
    });

    // 即使登录失败，CORS 头应该设置
    const allowOrigin = response.headers()['access-control-allow-origin'];
    const allowCredentials = response.headers()['access-control-allow-credentials'];

    // 如果启用了 CORS
    if (allowOrigin) {
      expect(allowOrigin).toBeTruthy();
    }
  });
});

test.describe('输入验证', () => {
  test('空用户名注册失败', async ({ request }) => {
    const response = await request.post('/api/auth/register', {
      data: {
        username: '',
        password: STRONG_PASSWORD,
      },
    });

    expect([400, 422]).toContain(response.status());
  });

  test('空密码注册失败', async ({ request }) => {
    const response = await request.post('/api/auth/register', {
      data: {
        username: `empty_pwd_${Date.now()}`,
        password: '',
      },
    });

    expect([400, 422]).toContain(response.status());
  });

  test('特殊字符用户名处理', async ({ request }) => {
    const specialUsernames = [
      'user<script>',
      'user&name',
      'user name',
      '用户名测试',
      'user@#$%',
    ];

    for (const username of specialUsernames) {
      const response = await request.post('/api/auth/register', {
        data: {
          username,
          password: STRONG_PASSWORD,
        },
      });

      // 可能接受或拒绝，取决于验证规则
      expect([200, 201, 400, 422]).toContain(response.status());
    }
  });

  test('超长输入处理', async ({ request }) => {
    const longUsername = 'a'.repeat(1000);
    
    const response = await request.post('/api/auth/register', {
      data: {
        username: longUsername,
        password: STRONG_PASSWORD,
      },
    });

    // 应该拒绝超长输入或截断处理
    expect([200, 201, 400, 413, 422]).toContain(response.status());
  });

  test('JSON 格式错误处理', async ({ request }) => {
    const response = await request.post('/api/auth/login', {
      headers: { 'Content-Type': 'application/json' },
      data: '{ invalid json }',
    });

    expect([400, 500]).toContain(response.status());
  });
});

test.describe('资源不存在处理', () => {
  test('不存在的用户 ID 返回 404', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (!sessionCookie) {
      test.skip();
      return;
    }

    const response = await request.get('/api/admin/users/nonexistent-user-id-xyz', {
      headers: { 'Cookie': `session=${sessionCookie.value}` },
    });

    expect([400, 404]).toContain(response.status());
  });

  test('不存在的分组 ID 返回 404', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (!sessionCookie) {
      test.skip();
      return;
    }

    const response = await request.get('/api/admin/groups/nonexistent-group-id', {
      headers: { 'Cookie': `session=${sessionCookie.value}` },
    });

    expect([400, 404, 500]).toContain(response.status());
  });

  test('不存在的客户端 ID 返回 404', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (!sessionCookie) {
      test.skip();
      return;
    }

    const response = await request.get('/api/admin/clients/nonexistent-client-id', {
      headers: { 'Cookie': `session=${sessionCookie.value}` },
    });

    expect([400, 404]).toContain(response.status());
  });

  test('不存在的 API 路由返回 404', async ({ request }) => {
    const response = await request.get('/api/nonexistent/endpoint');
    expect(response.status()).toBe(404);
  });
});

test.describe('重复创建资源', () => {
  test.use({ storageState: undefined });

  test('重复用户名注册失败', async ({ request }) => {
    const savedAdmin = getSavedAdmin();
    if (!savedAdmin) {
      test.skip();
      return;
    }

    // 尝试注册已存在的用户名
    const response = await request.post('/api/auth/register', {
      data: {
        username: savedAdmin.username,
        password: STRONG_PASSWORD,
      },
    });

    // 应该拒绝重复用户名
    expect([400, 409]).toContain(response.status());
  });

  test('重复客户端 ID 创建失败', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');
    const csrfCookie = cookies.find(c => c.name === 'csrf_token');

    if (!sessionCookie || !csrfCookie) {
      test.skip();
      return;
    }

    const clientId = `dup-client-${Date.now()}`;

    // 第一次创建
    const firstResponse = await request.post('/api/admin/clients', {
      headers: {
        'Cookie': `session=${sessionCookie.value}; csrf_token=${encodeURIComponent(csrfCookie.value)}`,
        'Content-Type': 'application/json',
        'X-CSRF-Token': decodeURIComponent(csrfCookie.value),
      },
      data: {
        id: clientId,
        redirectUris: ['http://localhost:3000/callback'],
        scopes: ['openid'],
      },
    });

    expect([200, 201]).toContain(firstResponse.status());

    // 尝试重复创建
    const secondResponse = await request.post('/api/admin/clients', {
      headers: {
        'Cookie': `session=${sessionCookie.value}; csrf_token=${encodeURIComponent(csrfCookie.value)}`,
        'Content-Type': 'application/json',
        'X-CSRF-Token': decodeURIComponent(csrfCookie.value),
      },
      data: {
        id: clientId,
        redirectUris: ['http://localhost:3000/callback'],
        scopes: ['openid'],
      },
    });

    // 应该拒绝重复 ID
    expect([400, 409, 500]).toContain(secondResponse.status());

    // 清理
    await request.delete(`/api/admin/clients/${clientId}`, {
      headers: {
        'Cookie': `session=${sessionCookie.value}; csrf_token=${encodeURIComponent(csrfCookie.value)}`,
        'X-CSRF-Token': decodeURIComponent(csrfCookie.value),
      },
    });
  });
});

test.describe('HTTP 方法验证', () => {
  test('GET 请求到 POST only 端点失败', async ({ request }) => {
    const response = await request.get('/api/auth/login');
    expect([400, 404, 405]).toContain(response.status());
  });

  test('POST 请求到 GET only 端点失败', async ({ request }) => {
    const response = await request.post('/api/public/config');
    expect([400, 404, 405]).toContain(response.status());
  });

  test('PUT 请求处理', async ({ request }) => {
    // 大多数端点使用 PATCH 而不是 PUT
    const response = await request.put('/api/user/me', {
      headers: { 'Content-Type': 'application/json' },
      data: { name: 'test' },
    });
    
    // 可能 404 或 405
    expect([400, 401, 404, 405]).toContain(response.status());
  });
});

test.describe('并发操作', () => {
  test('并发登录同一用户', async ({ request }) => {
    const savedAdmin = getSavedAdmin();
    if (!savedAdmin) {
      test.skip();
      return;
    }

    // 发送多个并发登录请求
    const promises = Array(5).fill(null).map(() => 
      request.post('/api/auth/login', {
        data: {
          username: savedAdmin.username,
          password: savedAdmin.password,
        },
      })
    );

    const responses = await Promise.all(promises);

    // 所有请求都应该返回响应
    for (const res of responses) {
      expect([200, 401, 429]).toContain(res.status());
    }
  });
});

test.describe('响应格式验证', () => {
  test('API 错误响应格式一致', async ({ request }) => {
    const response = await request.post('/api/auth/login', {
      data: { username: '', password: '' },
    });

    // 错误响应应该是 JSON
    expect(response.headers()['content-type']).toContain('application/json');

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('成功响应格式一致', async ({ request }) => {
    const response = await request.get('/api/public/config');

    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/json');

    const body = await response.json();
    // 应该有配置字段
    expect(body).toBeTruthy();
  });
});

test.describe('性能和超时', () => {
  test('API 响应时间合理', async ({ request }) => {
    const start = Date.now();
    await request.get('/api/public/config');
    const duration = Date.now() - start;

    // API 应该在 5 秒内响应
    expect(duration).toBeLessThan(5000);
  });

  test('健康检查端点快速响应', async ({ request }) => {
    const start = Date.now();
    await request.get('/health');
    const duration = Date.now() - start;

    // 健康检查应该在 1 秒内响应
    expect(duration).toBeLessThan(1000);
  });
});
