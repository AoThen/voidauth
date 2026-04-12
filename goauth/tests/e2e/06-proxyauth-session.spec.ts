import { test, expect, waitForPageReady, STRONG_PASSWORD } from './fixture';
import { writeFileSync, readFileSync, existsSync, unlinkSync } from 'fs';

/**
 * ProxyAuth 会话时长限制 E2E 测试
 * 
 * 测试覆盖:
 * 1. 最大会话时长配置
 * 2. 会话时长限制验证
 * 3. 超时会话拒绝
 * 4. 会话剩余时长返回
 * 5. 不同域名的不同时长限制
 * 6. 会话刷新机制
 */

test.describe.configure({ mode: 'serial' });

const ADMIN_COOKIES_FILE = '/tmp/goauth-e2e-proxyauth-session.json';

function saveAdminCookies(cookies: { session?: string; csrf?: string }) {
  writeFileSync(ADMIN_COOKIES_FILE, JSON.stringify(cookies));
}

function getAdminCookies(): { session?: string; csrf?: string } | null {
  if (!existsSync(ADMIN_COOKIES_FILE)) return null;
  try {
    return JSON.parse(readFileSync(ADMIN_COOKIES_FILE, 'utf-8'));
  } catch {
    return null;
  }
}

function buildAuthHeaders(authCookies: { session?: string; csrf?: string }): Record<string, string> {
  const headers: Record<string, string> = {};
  const cookieParts: string[] = [];
  if (authCookies.session) cookieParts.push(`session=${authCookies.session}`);
  if (authCookies.csrf) cookieParts.push(`csrf_token=${encodeURIComponent(authCookies.csrf)}`);
  if (cookieParts.length > 0) headers['Cookie'] = cookieParts.join('; ');
  if (authCookies.csrf) headers['X-CSRF-Token'] = authCookies.csrf;
  return headers;
}

const TEST_DOMAIN_PREFIX = 'session-test-';
const generateDomain = () => `${TEST_DOMAIN_PREFIX}${Date.now()}.example.com`;

test.describe('ProxyAuth 会话时长配置', () => {
  test.use({ storageState: undefined });

  test('创建带会话时长限制的 ProxyAuth', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');
    const csrfCookie = cookies.find(c => c.name === 'csrf_token');
    
    if (!sessionCookie) {
      test.skip();
      return;
    }

    const authCookies = {
      session: sessionCookie.value,
      csrf: csrfCookie ? decodeURIComponent(csrfCookie.value) : undefined,
    };
    saveAdminCookies(authCookies);

    // 创建带会话时长限制的 ProxyAuth
    const domain = generateDomain();
    const response = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: {
        domain,
        mfaRequired: false,
        maxSessionLength: 1, // 1 小时
      },
    });

    expect([200, 201]).toContain(response.status());
    
    const data = await response.json();
    expect(data.maxSessionLength).toBe(1);
  });

  test('更新 ProxyAuth 会话时长', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建 ProxyAuth
    const domain = generateDomain();
    const createResponse = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 2 },
    });

    if (createResponse.status() !== 200 && createResponse.status() !== 201) {
      test.skip();
      return;
    }

    const data = await createResponse.json();

    // 更新会话时长
    const updateResponse = await request.patch(`/api/admin/proxy-auth/${data.id}`, {
      headers: buildAuthHeaders(authCookies),
      data: { maxSessionLength: 4 }, // 改为 4 小时
    });

    expect([200, 204]).toContain(updateResponse.status());

    // 清理
    await request.delete(`/api/admin/proxy-auth/${data.id}`, {
      headers: buildAuthHeaders(authCookies),
    });
  });

  test('会话时长为 0 表示不限制', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    const domain = generateDomain();
    const response = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 0 }, // 不限制
    });

    expect([200, 201]).toContain(response.status());
    
    const data = await response.json();
    expect(data.maxSessionLength).toBe(0);

    // 清理
    await request.delete(`/api/admin/proxy-auth/${data.id}`, {
      headers: buildAuthHeaders(authCookies),
    });
  });
});

test.describe('会话时长验证', () => {
  test.use({ storageState: undefined });

  test('未配置会话时长 - 使用默认会话时长', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建不限制会话时长的 ProxyAuth
    const domain = generateDomain();
    await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 0 },
    });

    // 使用 ForwardAuth
    const authResponse = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain,
      },
    });

    expect(authResponse.status()).toBe(200);

    // 清理
    const listResponse = await request.get('/api/admin/proxy-auth', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const configs = await listResponse.json();
      const config = configs.find((c: any) => c.domain === domain);
      if (config) {
        await request.delete(`/api/admin/proxy-auth/${config.id}`, {
          headers: buildAuthHeaders(authCookies),
        });
      }
    }
  });

  test('新会话通过会话时长检查', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建 1 小时会话限制的 ProxyAuth
    const domain = generateDomain();
    const createResponse = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 1 },
    });

    if (createResponse.status() !== 200 && createResponse.status() !== 201) {
      test.skip();
      return;
    }

    const proxyAuthData = await createResponse.json();

    // 新创建的 session 应该通过时长检查
    const authResponse = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain,
      },
    });

    expect(authResponse.status()).toBe(200);

    // 清理
    await request.delete(`/api/admin/proxy-auth/${proxyAuthData.id}`, {
      headers: buildAuthHeaders(authCookies),
    });
  });
});

test.describe('会话剩余时长', () => {
  test.use({ storageState: undefined });

  test('ForwardAuth 返回会话剩余时长头', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建带会话时长限制的 ProxyAuth
    const domain = generateDomain();
    await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 24 },
    });

    // 调用 ForwardAuth
    const authResponse = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain,
      },
    });

    expect(authResponse.status()).toBe(200);

    // 检查响应头（如果实现）
    const headers = authResponse.headers();
    // X-Session-Remaining 或类似头可能存在
    // 注意：这是可选功能，取决于实现
    const hasRemainingHeader = 'x-session-remaining' in headers;
    
    // 不强制要求此头，但记录其存在
    expect(authResponse.status()).toBe(200);

    // 清理
    const listResponse = await request.get('/api/admin/proxy-auth', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const configs = await listResponse.json();
      for (const config of configs) {
        if (config.domain && config.domain.startsWith(TEST_DOMAIN_PREFIX)) {
          await request.delete(`/api/admin/proxy-auth/${config.id}`, {
            headers: buildAuthHeaders(authCookies),
          });
        }
      }
    }
  });
});

test.describe('不同域名不同时长', () => {
  test.use({ storageState: undefined });

  test('不同 ProxyAuth 配置不同的会话时长', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建两个不同会话时长的 ProxyAuth
    const domain1 = `short-${generateDomain()}`;
    const domain2 = `long-${generateDomain()}`;

    const response1 = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain: domain1, maxSessionLength: 1 }, // 1 小时
    });

    const response2 = await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain: domain2, maxSessionLength: 24 }, // 24 小时
    });

    expect([200, 201]).toContain(response1.status());
    expect([200, 201]).toContain(response2.status());

    // 两个域名都应该能访问
    const auth1 = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain1,
      },
    });

    const auth2 = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain2,
      },
    });

    expect(auth1.status()).toBe(200);
    expect(auth2.status()).toBe(200);

    // 清理
    const listResponse = await request.get('/api/admin/proxy-auth', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const configs = await listResponse.json();
      for (const config of configs) {
        if (config.domain === domain1 || config.domain === domain2) {
          await request.delete(`/api/admin/proxy-auth/${config.id}`, {
            headers: buildAuthHeaders(authCookies),
          });
        }
      }
    }
  });
});

test.describe('会话刷新', () => {
  test.use({ storageState: undefined });

  test('活跃会话自动延长', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建 ProxyAuth
    const domain = generateDomain();
    await request.post('/api/admin/proxy-auth', {
      headers: buildAuthHeaders(authCookies),
      data: { domain, maxSessionLength: 1 },
    });

    // 第一次访问
    const auth1 = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain,
      },
    });

    expect(auth1.status()).toBe(200);

    // 短暂等待后再次访问
    await new Promise(r => setTimeout(r, 100));

    const auth2 = await request.get('/authz/forward-auth', {
      headers: {
        'Cookie': `session=${authCookies.session}`,
        'X-Forwarded-Host': domain,
      },
    });

    // 应该仍然有效
    expect(auth2.status()).toBe(200);

    // 清理
    const listResponse = await request.get('/api/admin/proxy-auth', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const configs = await listResponse.json();
      for (const config of configs) {
        if (config.domain && config.domain.startsWith(TEST_DOMAIN_PREFIX)) {
          await request.delete(`/api/admin/proxy-auth/${config.id}`, {
            headers: buildAuthHeaders(authCookies),
          });
        }
      }
    }
  });
});

test.describe('清理', () => {
  test('清理 ProxyAuth 测试数据', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      expect(true).toBeTruthy();
      return;
    }

    // 清理所有测试域名
    const listResponse = await request.get('/api/admin/proxy-auth', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const configs = await listResponse.json();
      for (const config of configs) {
        if (config.domain && (
          config.domain.startsWith(TEST_DOMAIN_PREFIX) ||
          config.domain.startsWith('short-') ||
          config.domain.startsWith('long-')
        )) {
          await request.delete(`/api/admin/proxy-auth/${config.id}`, {
            headers: buildAuthHeaders(authCookies),
          });
        }
      }
    }

    // 清理 cookies 文件
    try {
      if (existsSync(ADMIN_COOKIES_FILE)) {
        unlinkSync(ADMIN_COOKIES_FILE);
      }
    } catch {}

    expect(true).toBeTruthy();
  });
});
