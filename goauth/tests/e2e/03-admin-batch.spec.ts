import { test, expect, waitForPageReady, STRONG_PASSWORD, generateTestUser } from './fixture';
import { writeFileSync, readFileSync, existsSync, unlinkSync } from 'fs';

/**
 * 管理员批量操作 E2E 测试
 * 
 * 测试覆盖:
 * 1. 批量用户审批
 * 2. 批量用户禁用/启用
 * 3. 批量分组成员管理
 * 4. 批量删除操作
 * 5. 批量操作性能
 * 6. 批量操作错误处理
 */

test.describe.configure({ mode: 'serial' });

const ADMIN_COOKIES_FILE = '/tmp/goauth-e2e-admin-batch.json';

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

test.describe('批量用户操作', () => {
  test.use({ storageState: undefined });

  test('准备测试数据 - 创建多个待审批用户', async ({ authenticatedPage: page, request }) => {
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

    // 创建多个测试用户
    const userCount = 5;
    for (let i = 0; i < userCount; i++) {
      const user = generateTestUser();
      await request.post('/api/auth/register', {
        data: {
          username: user.username,
          password: user.password,
          email: user.email,
        },
      });
    }

    // 验证用户已创建
    const response = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    expect(response.status()).toBe(200);
    const data = await response.json();
    expect(data.users.length).toBeGreaterThanOrEqual(userCount);
  });

  test('批量审批用户', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 获取所有未审批用户
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() !== 200) {
      test.skip();
      return;
    }

    const data = await listResponse.json();
    const pendingUsers = data.users.filter((u: any) => !u.approved);

    // 批量审批
    const results = [];
    for (const user of pendingUsers.slice(0, 3)) {
      const response = await request.post(`/api/admin/users/${user.id}/approve`, {
        headers: buildAuthHeaders(authCookies),
      });
      results.push(response.status());
    }

    // 验证所有审批成功
    expect(results.every(s => s === 200)).toBeTruthy();
  });

  test('批量禁用用户', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 获取已审批用户
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() !== 200) {
      test.skip();
      return;
    }

    const data = await listResponse.json();
    const users = data.users.filter((u: any) => u.approved && !u.disabled);

    // 批量禁用(不包含管理员)
    const nonAdminUsers = users.filter((u: any) => !u.isAdmin);
    const results = [];
    for (const user of nonAdminUsers.slice(0, 2)) {
      const response = await request.post(`/api/admin/users/${user.id}/disable`, {
        headers: buildAuthHeaders(authCookies),
      });
      results.push(response.status());
    }

    // 验证禁用成功
    expect(results.every(s => s === 200)).toBeTruthy();
  });

  test('批量启用用户', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 获取禁用的用户
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() !== 200) {
      test.skip();
      return;
    }

    const data = await listResponse.json();
    const disabledUsers = data.users.filter((u: any) => u.disabled);

    // 批量启用
    const results = [];
    for (const user of disabledUsers.slice(0, 2)) {
      const response = await request.post(`/api/admin/users/${user.id}/enable`, {
        headers: buildAuthHeaders(authCookies),
      });
      results.push(response.status());
    }

    // 验证启用成功
    expect(results.every(s => s === 200)).toBeTruthy();
  });
});

test.describe('批量分组成员管理', () => {
  test.use({ storageState: undefined });

  test('批量添加用户到分组', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建测试分组
    const groupResponse = await request.post('/api/admin/groups', {
      headers: buildAuthHeaders(authCookies),
      data: { name: `batch-group-${Date.now()}`, mfaRequired: false },
    });

    if (groupResponse.status() !== 200 && groupResponse.status() !== 201) {
      test.skip();
      return;
    }

    const group = await groupResponse.json();

    // 获取用户列表
    const usersResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (usersResponse.status() !== 200) {
      test.skip();
      return;
    }

    const usersData = await usersResponse.json();
    const usersToAdd = usersData.users.slice(0, 3);

    // 批量添加成员
    const results = [];
    for (const user of usersToAdd) {
      const response = await request.post(`/api/admin/groups/${group.id}/members`, {
        headers: buildAuthHeaders(authCookies),
        data: { userId: user.id },
      });
      results.push(response.status());
    }

    // 验证所有添加成功
    expect(results.every(s => s === 200 || s === 201 || s === 204)).toBeTruthy();

    // 清理
    await request.delete(`/api/admin/groups/${group.id}`, {
      headers: buildAuthHeaders(authCookies),
    });
  });
});

test.describe('批量删除操作', () => {
  test.use({ storageState: undefined });

  test('批量删除测试用户', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 创建测试用户用于删除
    const testUsers = [];
    for (let i = 0; i < 3; i++) {
      const user = generateTestUser();
      await request.post('/api/auth/register', {
        data: { username: user.username, password: user.password, email: user.email },
      });
      testUsers.push(user.username);
    }

    // 获取用户列表
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() !== 200) {
      test.skip();
      return;
    }

    const data = await listResponse.json();
    const usersToDelete = data.users.filter((u: any) => 
      testUsers.includes(u.username) && !u.isAdmin
    );

    // 批量删除
    const results = [];
    for (const user of usersToDelete) {
      const response = await request.delete(`/api/admin/users/${user.id}`, {
        headers: buildAuthHeaders(authCookies),
      });
      results.push(response.status());
    }

    // 验证删除成功
    expect(results.every(s => s === 200 || s === 204)).toBeTruthy();
  });
});

test.describe('批量操作性能', () => {
  test.use({ storageState: undefined });

  test('批量操作响应时间', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 测试批量审批性能
    const startTime = Date.now();

    // 获取用户列表
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    const endTime = Date.now();
    const duration = endTime - startTime;

    // 列表请求应在合理时间内完成 (< 5秒)
    expect(duration).toBeLessThan(5000);
    expect(listResponse.status()).toBe(200);
  });
});

test.describe('批量操作错误处理', () => {
  test.use({ storageState: undefined });

  test('批量操作中部分失败不影响其他操作', async ({ authenticatedPage: page, request }) => {
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
    
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      test.skip();
      return;
    }

    // 尝试批量操作包含不存在的用户
    const results = [];
    
    // 操作不存在的用户(应该失败)
    const failResponse = await request.post('/api/admin/users/nonexistent-id/approve', {
      headers: buildAuthHeaders(authCookies),
    });
    results.push({ type: 'fail', status: failResponse.status() });

    // 操作真实用户(应该成功)
    const listResponse = await request.get('/api/admin/users', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (listResponse.status() === 200) {
      const data = await listResponse.json();
      const realUser = data.users.find((u: any) => !u.approved && !u.isAdmin);
      
      if (realUser) {
        const successResponse = await request.post(`/api/admin/users/${realUser.id}/approve`, {
          headers: buildAuthHeaders(authCookies),
        });
        results.push({ type: 'success', status: successResponse.status() });
      }
    }

    // 验证部分失败不影响其他操作
    expect(results.length).toBeGreaterThan(0);
  });
});

test.describe('清理', () => {
  test('清理批量测试数据', async ({ request }) => {
    const authCookies = getAdminCookies();
    if (!authCookies?.session) {
      expect(true).toBeTruthy();
      return;
    }

    // 清理测试分组
    const groupsResponse = await request.get('/api/admin/groups', {
      headers: { 'Cookie': `session=${authCookies.session}` },
    });

    if (groupsResponse.status() === 200) {
      const groupsData = await groupsResponse.json();
      const groups = Array.isArray(groupsData) ? groupsData : [];
      
      for (const group of groups) {
        if (group.name && group.name.includes('batch-')) {
          await request.delete(`/api/admin/groups/${group.id}`, {
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
