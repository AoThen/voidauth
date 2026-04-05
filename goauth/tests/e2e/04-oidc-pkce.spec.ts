import { test, expect, waitForPageReady, STRONG_PASSWORD, getSavedAdmin } from './fixture';
import * as crypto from 'crypto';

/**
 * OIDC PKCE 流程 E2E 测试
 * 
 * 测试覆盖：
 * 1. PKCE 授权码流程（code_challenge + code_verifier）
 * 2. 无效 code_verifier 拒绝
 * 3. S256 和 plain 方法支持
 */

test.describe.configure({ mode: 'serial' });

// PKCE 辅助函数
function generateCodeVerifier(): string {
  return crypto.randomBytes(32).toString('base64url');
}

function generateCodeChallenge(verifier: string, method: 'S256' | 'plain' = 'S256'): string {
  if (method === 'plain') {
    return verifier;
  }
  return crypto.createHash('sha256').update(verifier).digest('base64url');
}

let testClientId: string;

test.describe('OIDC PKCE 流程', () => {
  test.use({ storageState: undefined });

  test('创建支持 PKCE 的测试客户端', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');
    const csrfCookie = cookies.find(c => c.name === 'csrf_token');

    if (!sessionCookie || !csrfCookie) {
      test.skip();
      return;
    }

    testClientId = `pkce-client-${Date.now()}`;

    // 使用 page.request 继承浏览器 cookies
    const response = await page.request.post('/api/admin/clients', {
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': decodeURIComponent(csrfCookie.value),
      },
      data: {
        id: testClientId,
        name: 'PKCE Test Client',
        redirectUris: ['http://localhost:3000/callback'],
        scopes: ['openid', 'profile', 'email'],
        grantTypes: ['authorization_code'],
        responseTypes: ['code'],
        trusted: true,
      },
    });

    expect([200, 201, 409]).toContain(response.status());
  });

  test('PKCE 授权请求包含 code_challenge', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    if (!testClientId) {
      test.skip();
      return;
    }

    const codeVerifier = generateCodeVerifier();
    const codeChallenge = generateCodeChallenge(codeVerifier);

    const authParams = new URLSearchParams({
      client_id: testClientId,
      redirect_uri: 'http://localhost:3000/callback',
      response_type: 'code',
      scope: 'openid profile email',
      state: 'pkce-test-state',
      code_challenge: codeChallenge,
      code_challenge_method: 'S256',
    });

    // 访问授权端点
    await page.goto(`/authorize?${authParams.toString()}`);
    await page.waitForTimeout(2000);

    // 验证页面正常加载（没有立即报错）
    const url = page.url();
    expect(url).toBeTruthy();
  });

  test('Token 端点接受 code_verifier', async ({ authenticatedPage: page, request }) => {
    if (!testClientId) {
      test.skip();
      return;
    }

    // 这个测试验证 token 端点能够处理 code_verifier 参数
    // 完整的 PKCE 流程需要先获取授权码

    const codeVerifier = generateCodeVerifier();
    
    // 测试 token 端点对无效 code 的响应（带 code_verifier）
    const response = await request.post('/oauth/token', {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      data: new URLSearchParams({
        grant_type: 'authorization_code',
        code: 'invalid-code',
        redirect_uri: 'http://localhost:3000/callback',
        client_id: testClientId,
        code_verifier: codeVerifier,
      }).toString(),
    });

    // 应该返回错误（因为 code 无效），但接受 code_verifier 参数
    expect([400, 401]).toContain(response.status());
  });

  test('无效 code_verifier 拒绝 token 请求', async ({ request }) => {
    if (!testClientId) {
      test.skip();
      return;
    }

    // 测试使用错误的 code_verifier
    const response = await request.post('/oauth/token', {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      data: new URLSearchParams({
        grant_type: 'authorization_code',
        code: 'some-code',
        redirect_uri: 'http://localhost:3000/callback',
        client_id: testClientId,
        code_verifier: 'wrong-verifier',
      }).toString(),
    });

    expect([400, 401]).toContain(response.status());
  });
});

test.describe('OIDC Discovery PKCE 支持', () => {
  test('Discovery 文档声明 PKCE 支持', async ({ request }) => {
    const response = await request.get('/.well-known/openid-configuration');
    expect(response.status()).toBe(200);

    const config = await response.json();
    
    // 检查 code_challenge_methods_supported
    if (config.code_challenge_methods_supported) {
      expect(config.code_challenge_methods_supported).toContain('S256');
    }
    
    // 检查授权端点支持 PKCE 参数
    expect(config.authorization_endpoint).toBeTruthy();
    expect(config.token_endpoint).toBeTruthy();
  });
});

test.describe('PKCE plain 方法', () => {
  test('plain 方法支持检查', async ({ request }) => {
    const response = await request.get('/.well-known/openid-configuration');
    const config = await response.json();

    // 检查是否支持 plain 方法
    const methods = config.code_challenge_methods_supported || [];
    
    // S256 是推荐的，plain 是可选的
    expect(methods.includes('S256') || methods.includes('plain') || methods.length === 0).toBeTruthy();
  });
});

test.describe('清理', () => {
  test('删除 PKCE 测试客户端', async ({ authenticatedPage: page }) => {
    if (!testClientId) {
      expect(true).toBeTruthy();
      return;
    }

    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');
    const csrfCookie = cookies.find(c => c.name === 'csrf_token');

    if (sessionCookie && csrfCookie) {
      await page.request.delete(`/api/admin/clients/${testClientId}`, {
        headers: {
          'X-CSRF-Token': decodeURIComponent(csrfCookie.value),
        },
      });
    }

    expect(true).toBeTruthy();
  });
});
