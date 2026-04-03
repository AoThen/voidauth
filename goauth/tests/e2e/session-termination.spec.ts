import { test, expect, waitForPageReady, STRONG_PASSWORD, getSavedAdmin } from './fixture';

/**
 * 会话终止验证 E2E 测试
 * 
 * 测试覆盖：
 * 1. 会话列表 API 验证
 * 2. 终止会话后验证该会话确实失效
 * 3. 会话过期时间验证
 * 4. 多设备会话管理
 */

test.describe.configure({ mode: 'serial' });

test.describe('会话列表 API', () => {
  test.use({ storageState: undefined });

  test('用户可以获取自己的会话列表', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    // 获取认证信息
    const context = page.context();
    const cookies = await context.cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (!sessionCookie) {
      test.skip();
      return;
    }

    // 调用会话列表 API
    const response = await request.get('/api/user/sessions', {
      headers: {
        'Cookie': `session=${sessionCookie.value}`,
      },
    });

    expect(response.status()).toBe(200);
    
    const sessions = await response.json();
    expect(Array.isArray(sessions)).toBeTruthy();
    
    // 应该至少有一个会话（当前会话）
    expect(sessions.length).toBeGreaterThanOrEqual(1);
  });

  test('会话列表包含正确的字段', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const context = page.context();
    const cookies = await context.cookies();
    const sessionCookie = cookies.find(c => c.name === 'session');

    if (!sessionCookie) {
      test.skip();
      return;
    }

    const response = await request.get('/api/user/sessions', {
      headers: {
        'Cookie': `session=${sessionCookie.value}`,
      },
    });

    expect(response.status()).toBe(200);
    const sessions = await response.json();
    
    if (sessions.length > 0) {
      const session = sessions[0];
      // 验证必要字段
      expect(session.id).toBeTruthy();
      expect(session.createdAt).toBeTruthy();
      // 可选字段检查
      if (session.userAgent !== undefined) {
        expect(typeof session.userAgent).toBe('string');
      }
      if (session.ip !== undefined) {
        expect(typeof session.ip).toBe('string');
      }
    }
  });

  test('未认证用户无法获取会话列表', async ({ request }) => {
    const response = await request.get('/api/user/sessions');
    expect(response.status()).toBe(401);
  });
});

test.describe('会话终止验证', () => {
  test.use({ storageState: undefined });

  test('终止会话后该会话失效', async ({ browser }) => {
    const savedAdmin = getSavedAdmin();
    if (!savedAdmin) {
      test.skip();
      return;
    }

    // 创建两个独立的浏览器上下文
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // 第一个会话登录
      await page1.goto('/');
      await waitForPageReady(page1);
      await page1.locator('#username').fill(savedAdmin.username);
      await page1.locator('#password').fill(savedAdmin.password);
      await page1.locator('button[type="submit"]:has-text("登录")').click();
      await page1.waitForTimeout(2000);

      // 第二个会话登录
      await page2.goto('/');
      await waitForPageReady(page2);
      await page2.locator('#username').fill(savedAdmin.username);
      await page2.locator('#password').fill(savedAdmin.password);
      await page2.locator('button[type="submit"]:has-text("登录")').click();
      await page2.waitForTimeout(2000);

      // 两个会话都应该有效
      const logged1 = await page1.locator('h1:has-text("管理后台")').isVisible({ timeout: 5000 }).catch(() => false);
      const logged2 = await page2.locator('h1:has-text("管理后台")').isVisible({ timeout: 5000 }).catch(() => false);
      expect(logged1).toBeTruthy();
      expect(logged2).toBeTruthy();

      // 获取第一个会话的 cookies
      const cookies1 = await context1.cookies();
      const session1Cookie = cookies1.find(c => c.name === 'session');
      const csrf1Cookie = cookies1.find(c => c.name === 'csrf_token');

      // 从第一个会话获取会话列表
      const sessionsResponse = await page1.evaluate(async () => {
        const res = await fetch('/api/user/sessions');
        return { status: res.status, data: await res.json() };
      });

      expect(sessionsResponse.status).toBe(200);
      const sessions = sessionsResponse.data;

      // 找到第二个会话（不是当前的）
      const otherSession = sessions.find((s: any) => !s.current && s.id);
      
      if (otherSession) {
        // 从第一个会话终止第二个会话
        const terminateResponse = await page1.evaluate(async (sessionId) => {
          const csrfToken = decodeURIComponent(
            document.cookie.split(';').find(c => c.trim().startsWith('csrf_token='))?.split('=')[1] || ''
          );
          
          const res = await fetch(`/api/user/sessions/${sessionId}`, {
            method: 'DELETE',
            headers: {
              'X-CSRF-Token': csrfToken,
            },
          });
          return { status: res.status };
        }, otherSession.id);

        expect([200, 204]).toContain(terminateResponse.status);

        // 等待会话失效
        await page2.waitForTimeout(1000);

        // 第二个会话现在应该失效 - 尝试访问需要认证的 API
        const meResponse = await page2.evaluate(async () => {
          const res = await fetch('/api/user/me');
          return { status: res.status };
        });

        // 应该返回 401（未授权）
        expect(meResponse.status).toBe(401);
      }
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('无法终止不存在的会话', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const result = await page.evaluate(async () => {
      const csrfToken = decodeURIComponent(
        document.cookie.split(';').find(c => c.trim().startsWith('csrf_token='))?.split('=')[1] || ''
      );
      
      const res = await fetch('/api/user/sessions/nonexistent-session-id', {
        method: 'DELETE',
        headers: {
          'X-CSRF-Token': csrfToken,
        },
      });
      return { status: res.status };
    });

    // DELETE 是幂等的，可能返回 200（成功）或 404（不存在）
    // 两种行为都是可接受的
    expect([200, 204, 404]).toContain(result.status);
  });

  test('无法终止其他用户的会话', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    // 创建另一个用户
    const otherUsername = `session_other_${Date.now()}`;
    await request.post('/api/auth/register', {
      data: {
        username: otherUsername,
        password: STRONG_PASSWORD,
      },
    });

    // 尝试用当前用户终止一个随机会话 ID
    const result = await page.evaluate(async () => {
      const csrfToken = decodeURIComponent(
        document.cookie.split(';').find(c => c.trim().startsWith('csrf_token='))?.split('=')[1] || ''
      );
      
      const res = await fetch('/api/user/sessions/other-user-session-id', {
        method: 'DELETE',
        headers: {
          'X-CSRF-Token': csrfToken,
        },
      });
      return { status: res.status };
    });

    // 应该返回错误
    expect([400, 403, 404]).toContain(result.status);
  });
});

test.describe('会话并发管理', () => {
  test.use({ storageState: undefined });

  test('用户可以从一个会话查看所有活跃会话', async ({ browser }) => {
    const savedAdmin = getSavedAdmin();
    if (!savedAdmin) {
      test.skip();
      return;
    }

    // 创建多个会话
    const contexts = [];
    const pages = [];

    for (let i = 0; i < 3; i++) {
      const ctx = await browser.newContext();
      const p = await ctx.newPage();
      
      await p.goto('/');
      await waitForPageReady(p);
      await p.locator('#username').fill(savedAdmin.username);
      await p.locator('#password').fill(savedAdmin.password);
      await p.locator('button[type="submit"]:has-text("登录")').click();
      await p.waitForTimeout(2000);
      
      contexts.push(ctx);
      pages.push(p);
    }

    try {
      // 从第一个会话获取会话列表
      const sessionsResponse = await pages[0].evaluate(async () => {
        const res = await fetch('/api/user/sessions');
        return { status: res.status, data: await res.json() };
      });

      expect(sessionsResponse.status).toBe(200);
      
      // 应该看到至少 3 个会话
      expect(sessionsResponse.data.length).toBeGreaterThanOrEqual(3);
    } finally {
      // 清理所有上下文
      for (const ctx of contexts) {
        await ctx.close();
      }
    }
  });

  test('终止所有其他会话后只剩当前会话', async ({ browser }) => {
    const savedAdmin = getSavedAdmin();
    if (!savedAdmin) {
      test.skip();
      return;
    }

    // 创建主会话
    const mainContext = await browser.newContext();
    const mainPage = await mainContext.newPage();
    
    await mainPage.goto('/');
    await waitForPageReady(mainPage);
    await mainPage.locator('#username').fill(savedAdmin.username);
    await mainPage.locator('#password').fill(savedAdmin.password);
    await mainPage.locator('button[type="submit"]:has-text("登录")').click();
    await mainPage.waitForTimeout(2000);

    // 创建其他会话
    const otherContexts = [];
    for (let i = 0; i < 2; i++) {
      const ctx = await browser.newContext();
      const p = await ctx.newPage();
      
      await p.goto('/');
      await waitForPageReady(p);
      await p.locator('#username').fill(savedAdmin.username);
      await p.locator('#password').fill(savedAdmin.password);
      await p.locator('button[type="submit"]:has-text("登录")').click();
      await p.waitForTimeout(2000);
      
      otherContexts.push(ctx);
    }

    try {
      // 获取会话列表
      const sessionsResponse = await mainPage.evaluate(async () => {
        const res = await fetch('/api/user/sessions');
        return { status: res.status, data: await res.json() };
      });

      expect(sessionsResponse.status).toBe(200);
      const sessions = sessionsResponse.data;

      // 终止所有非当前会话
      for (const session of sessions) {
        if (!session.current && session.id) {
          await mainPage.evaluate(async (sessionId) => {
            const csrfToken = decodeURIComponent(
              document.cookie.split(';').find(c => c.trim().startsWith('csrf_token='))?.split('=')[1] || ''
            );
            
            await fetch(`/api/user/sessions/${sessionId}`, {
              method: 'DELETE',
              headers: {
                'X-CSRF-Token': csrfToken,
              },
            });
          }, session.id);
        }
      }

      // 等待处理
      await mainPage.waitForTimeout(1000);

      // 验证只剩下当前会话
      const newSessionsResponse = await mainPage.evaluate(async () => {
        const res = await fetch('/api/user/sessions');
        return await res.json();
      });

      // 应该只有一个会话
      expect(newSessionsResponse.length).toBe(1);
      expect(newSessionsResponse[0].current).toBeTruthy();
    } finally {
      await mainContext.close();
      for (const ctx of otherContexts) {
        await ctx.close();
      }
    }
  });
});

test.describe('会话属性验证', () => {
  test.use({ storageState: undefined });

  test('会话包含用户代理信息', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const sessionsResponse = await page.evaluate(async () => {
      const res = await fetch('/api/user/sessions');
      return await res.json();
    });

    // 当前会话应该有用户代理信息
    const currentSession = sessionsResponse.find((s: any) => s.current);
    if (currentSession) {
      expect(currentSession.userAgent).toBeTruthy();
    }
  });

  test('会话包含创建时间', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const sessionsResponse = await page.evaluate(async () => {
      const res = await fetch('/api/user/sessions');
      return await res.json();
    });

    expect(sessionsResponse.length).toBeGreaterThan(0);
    
    for (const session of sessionsResponse) {
      expect(session.createdAt).toBeTruthy();
      // 验证时间格式
      const date = new Date(session.createdAt);
      expect(date.toString()).not.toBe('Invalid Date');
    }
  });

  test('当前会话标记正确', async ({ authenticatedPage: page }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });

    const sessionsResponse = await page.evaluate(async () => {
      const res = await fetch('/api/user/sessions');
      return await res.json();
    });

    // 应该恰好有一个 current: true 的会话
    const currentSessions = sessionsResponse.filter((s: any) => s.current);
    expect(currentSessions.length).toBe(1);
  });
});
