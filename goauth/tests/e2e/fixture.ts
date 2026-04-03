import { test as base, Page, expect, BrowserContext } from '@playwright/test';
import { execSync } from 'child_process';
import { resolve } from 'path';
import { existsSync, writeFileSync, readFileSync } from 'fs';

// 强密码，满足 zxcvbn 3+ 分数要求
const STRONG_PASSWORD = 'Correct-Horse-Battery-Staple-2024!';

// 用户信息接口
interface TestUser {
  username: string;
  password: string;
  email?: string;
}

// 扩展的测试 fixture
type E2EFixtures = {
  authenticatedPage: Page;
  adminPage: Page;
  testUser: TestUser;
};

// 扩展 expect
export { expect };

// 状态文件路径 - 使用固定的绝对路径存储管理员凭据
// 这样可以确保所有测试文件使用同一个状态文件
const STATE_FILE = '/tmp/goauth-e2e-admin-state.json';

// 生成随机用户名
function generateUsername(): string {
  return `testuser_${Date.now()}_${Math.random().toString(36).substring(2, 8)}`;
}

// 生成随机邮箱
function generateEmail(username: string): string {
  return `${username}@test.example.com`;
}

// 获取已保存的管理员凭据
export function getSavedAdmin(): { username: string; password: string; email: string } | null {
  if (!existsSync(STATE_FILE)) return null;
  try {
    const data = readFileSync(STATE_FILE, 'utf-8');
    return JSON.parse(data);
  } catch {
    return null;
  }
}

// 保存管理员凭据
export function saveAdmin(username: string, password: string, email: string): void {
  writeFileSync(STATE_FILE, JSON.stringify({ username, password, email, timestamp: Date.now() }));
}

/**
 * 等待页面完全加载（包括 Alpine.js 初始化和渲染）
 */
export async function waitForPageReady(page: Page): Promise<void> {
  await page.waitForLoadState('networkidle');
  
  // 等待 Alpine.js 加载并初始化完成
  await page.waitForFunction(() => {
    const alpine = (window as any).Alpine;
    if (!alpine) return false;
    // 检查 Alpine 是否已完成初始化
    return true;
  }, { timeout: 10000 }).catch(() => {
    // Alpine.js 可能从 CDN 加载较慢，继续执行
  });
  
  // 等待 Alpine.js 完成渲染 - 确保页面内容稳定
  // 检查 h1 元素是否存在且内容不为空
  await page.waitForFunction(() => {
    const h1 = document.querySelector('h1');
    return h1 && h1.textContent && h1.textContent.trim().length > 0;
  }, { timeout: 5000 }).catch(() => {
    // 如果没有 h1，继续执行
  });
  
  // 额外等待确保 DOM 更新和动画完成
  await page.waitForTimeout(300);
}

/**
 * 等待元素可见（正确的异步等待方式）
 * 注意：isVisible() 不支持 timeout 参数，必须使用 waitFor
 * 包含重试机制以处理 Alpine.js 渲染延迟
 */
export async function waitForVisible(page: Page, selector: string, timeout: number = 5000): Promise<boolean> {
  const startTime = Date.now();
  const retryInterval = 500;
  
  while (Date.now() - startTime < timeout) {
    try {
      const locator = page.locator(selector).first();
      await locator.waitFor({ state: 'visible', timeout: retryInterval });
      return true;
    } catch {
      // 继续重试
      await page.waitForTimeout(100);
    }
  }
  
  // 最终检查
  try {
    const count = await page.locator(selector).count();
    if (count > 0) {
      const isVisible = await page.locator(selector).first().isVisible();
      return isVisible;
    }
  } catch {
    // 忽略错误
  }
  
  return false;
}

/**
 * 注册用户并登录
 * 注意：goauth 注册后需要手动登录
 */
async function registerAndLogin(page: Page, username: string, password: string, email: string): Promise<void> {
  // 导航到首页
  await page.goto('/');
  await waitForPageReady(page);
  
  // 点击注册链接
  const registerLink = page.locator('a:has-text("注册")');
  if (await waitForVisible(page, 'a:has-text("注册")', 3000)) {
    await registerLink.click();
    await page.waitForTimeout(1000);
  } else {
    await page.goto('/#register');
    await page.waitForTimeout(1000);
  }
  
  // 确认在注册页面
  await page.locator('h1:has-text("注册")').waitFor({ state: 'visible', timeout: 5000 });
  await page.waitForTimeout(500);
  
  // 填写注册表单
  await page.locator('#reg-username').click();
  await page.locator('#reg-username').fill(username);
  await page.locator('#reg-email').click();
  await page.locator('#reg-email').fill(email);
  await page.locator('#reg-password').click();
  await page.locator('#reg-password').fill(password);
  await page.locator('#reg-confirm').click();
  await page.locator('#reg-confirm').fill(password);
  await page.waitForTimeout(500);
  
  // 提交注册
  await page.locator('button:has-text("注册")').click();
  
  // 等待注册结果
  await page.waitForTimeout(2000);
  
  // 检查是否注册成功（跳转到登录页）
  const onLoginPage = await waitForVisible(page, 'h1:has-text("登录")', 5000);
  
  if (!onLoginPage) {
    // 检查是否有错误消息
    const errorMsg = await page.locator('.error').textContent({ timeout: 1000 }).catch(() => '');
    console.log(`Registration result: not on login page, error: ${errorMsg}`);
    
    // 导航到登录页
    await page.goto('/#login');
    await waitForPageReady(page);
  }
  
  // 等待登录表单出现
  await page.locator('#username').waitFor({ state: 'visible', timeout: 10000 });
  await page.waitForTimeout(500);
  
  // 填写登录表单
  await page.locator('#username').click();
  await page.locator('#username').fill(username);
  await page.locator('#password').click();
  await page.locator('#password').fill(password);
  await page.waitForTimeout(500);
  
  // 提交登录
  await page.locator('button[type="submit"]:has-text("登录")').click();
  
  // 等待登录结果
  await page.waitForTimeout(3000);
  
  // 检查登录是否成功
  const loginSuccess = await waitForVisible(page, 'h1:has-text("管理后台"), h1:has-text("用户设置")', 5000);
  
  if (!loginSuccess) {
    // 检查错误消息
    const errorMsg = await page.locator('.error').textContent({ timeout: 1000 }).catch(() => '');
    console.log(`Login failed for user ${username}: ${errorMsg}`);
    
    throw new Error(`Login failed after registration: ${errorMsg}`);
  }
}

/**
 * 尝试批准待审批用户（通过查找已存在的管理员）
 */
async function approvePendingUser(page: Page, username: string): Promise<boolean> {
  try {
    // 尝试通过 API 批准用户（需要管理员权限）
    // 这里我们使用页面上下文来调用 API
    const result = await page.evaluate(async (user) => {
      try {
        // 首先获取用户列表找到用户 ID
        const listRes = await fetch('/api/admin/users');
        if (!listRes.ok) return { success: false, error: 'Failed to get user list' };
        
        const users = await listRes.json();
        const targetUser = users.find((u: any) => u.username === user);
        if (!targetUser) return { success: false, error: 'User not found' };
        
        // 批准用户
        const approveRes = await fetch(`/api/admin/users/${targetUser.id}/approve`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        });
        
        return { success: approveRes.ok, status: approveRes.status };
      } catch (e: any) {
        return { success: false, error: e.message };
      }
    }, username);
    
    return result.success;
  } catch {
    return false;
  }
}

/**
 * 获取数据库中的第一个用户（管理员）
 */
async function getFirstAdminFromDB(page: Page): Promise<{ id: string; username: string } | null> {
  try {
    const result = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/admin/users');
        if (!res.ok) return null;
        const users = await res.json();
        // 第一个用户通常是管理员
        const admin = users.find((u: any) => u.isAdmin === true);
        if (admin) {
          return { id: admin.id, username: admin.username };
        }
        return null;
      } catch {
        return null;
      }
    });
    return result;
  } catch {
    return null;
  }
}

/**
 * Goauth 测试基础 Fixture
 * 
 * 提供以下功能：
 * - authenticatedPage: 已认证的页面（自动注册并登录）
 * - adminPage: 管理员页面（第一个用户自动成为管理员）
 * - testUser: 测试用户数据生成
 */
export const test = base.extend<E2EFixtures>({
  // 创建一个已认证的页面
  authenticatedPage: async ({ page, context }, use) => {
    // 测试间延迟，避免触发限流
    await new Promise(resolve => setTimeout(resolve, 500));

    // 导航到首页
    await page.goto('/');
    await waitForPageReady(page);

    // 检查是否已登录 - 使用管理后台标题判断
    let alreadyLoggedIn = await waitForVisible(page, 'h1:has-text("管理后台"), h1:has-text("用户设置")', 2000);

    // 如果已登录，直接使用
    if (alreadyLoggedIn) {
      await use(page);
      return;
    }

    // 检查是否已有保存的管理员凭据
    let savedAdmin = getSavedAdmin();

    // 如果有保存的管理员凭据，尝试登录
    if (savedAdmin) {
      // 清除所有 cookies 确保干净状态
      await context.clearCookies();
      
      // 刷新页面确保未认证状态
      await page.goto('/');
      await waitForPageReady(page);
      
      // 导航到登录页面
      await page.goto('/#login');
      await waitForPageReady(page);
      
      // 等待登录表单可见（确保页面正确渲染）
      try {
        await page.locator('#username').waitFor({ state: 'visible', timeout: 5000 });
      } catch {
        // 如果登录表单不可见，可能是页面状态问题，创建新用户
        console.log('Login form not visible, creating new user');
        savedAdmin = null;
      }
      
      if (savedAdmin) {
        // 填写登录表单
        await page.locator('#username').fill(savedAdmin.username);
        await page.locator('#password').fill(savedAdmin.password);
        await page.locator('button[type="submit"]:has-text("登录")').click();

        // 等待页面完全加载
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(1000);
        
        // 检查登录是否成功
        alreadyLoggedIn = await waitForVisible(page, 'h1:has-text("管理后台"), h1:has-text("用户设置")', 10000);
        
        // 如果登录成功，直接使用
        if (alreadyLoggedIn) {
          await use(page);
          return;
        }
        
        // 登录失败，继续创建新用户
        const errorMsg = await page.locator('.error').textContent({ timeout: 1000 }).catch(() => '');
        console.log(`Login failed for ${savedAdmin.username}: ${errorMsg}`);
      }
    }

    // 创建新的管理员用户（清空数据库后的第一个用户自动成为管理员）
    const username = generateUsername();
    const password = STRONG_PASSWORD;
    const email = generateEmail(username);
    
    // 清除所有 cookies 确保未认证状态
    const browserContext = page.context();
    await browserContext.clearCookies();
    
    // 导航到注册页面
    await page.goto('/#register');
    await waitForPageReady(page);
    
    // 等待注册表单
    await page.locator('#reg-username').waitFor({ state: 'visible', timeout: 10000 });
    
    await page.locator('#reg-username').fill(username);
    await page.locator('#reg-email').fill(email);
    await page.locator('#reg-password').fill(password);
    await page.locator('#reg-confirm').fill(password);
    await page.locator('button:has-text("注册")').click();
    
    // 等待注册完成
    await page.waitForTimeout(2000);
    
    // 导航到登录页
    await page.goto('/#login');
    await waitForPageReady(page);
    
    // 等待登录表单
    await page.locator('#username').waitFor({ state: 'visible', timeout: 10000 });
    
    // 登录
    await page.locator('#username').fill(username);
    await page.locator('#password').fill(password);
    await page.locator('button[type="submit"]:has-text("登录")').click();
    
    // 等待页面完全加载
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    // 检查登录是否成功 - 使用更长的超时
    alreadyLoggedIn = await waitForVisible(page, 'h1:has-text("管理后台"), h1:has-text("用户设置")', 10000);
    
    if (!alreadyLoggedIn) {
      // 如果还是失败，抛出错误
      const errorMsg = await page.locator('.error').textContent({ timeout: 1000 }).catch(() => '');
      throw new Error(`Failed to authenticate user: ${errorMsg}`);
    }
    
    // 保存管理员凭据
    saveAdmin(username, password, email);

    // 使用已认证的页面
    await use(page);
  },

  // 创建管理员页面（等同于 authenticatedPage，语义化）
  adminPage: async ({ authenticatedPage }, use) => {
    // authenticatedPage 已经是管理员
    await use(authenticatedPage);
  },

  // 创建测试用户信息
  testUser: async ({}, use) => {
    const username = generateUsername();
    const password = STRONG_PASSWORD;
    const email = generateEmail(username);

    await use({ username, password, email });
  },
});

// ==================== 辅助函数 ====================

/**
 * 注册用户
 */
export async function registerUser(page: Page, user: TestUser): Promise<void> {
  await page.goto('/');
  await waitForPageReady(page);
  
  const registerLink = page.locator('a:has-text("注册")');
  if (await waitForVisible(page, 'a:has-text("注册")', 3000)) {
    await registerLink.click();
    await page.waitForTimeout(500);
  }
  
  await page.locator('#reg-username').fill(user.username);
  if (user.email) {
    await page.locator('#reg-email').fill(user.email);
  }
  await page.locator('#reg-password').fill(user.password);
  await page.locator('#reg-confirm').fill(user.password);

  await page.locator('button:has-text("注册")').click();
  
  // 等待回到登录页
  await page.locator('h1:has-text("登录")').waitFor({ state: 'visible', timeout: 10000 });
  await waitForPageReady(page);
}

/**
 * 登录用户
 */
export async function loginUser(page: Page, username: string, password: string): Promise<void> {
  // 导航到登录页面
  await page.goto('/#login');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(500);

  // 检查是否已经在登录状态
  const loggedIn = await waitForVisible(page, 'button:has-text("退出登录")', 2000);
  if (loggedIn) {
    return; // 已经登录
  }

  // 等待登录表单可见
  await page.locator('#username').waitFor({ state: 'visible', timeout: 10000 });

  await page.locator('#username').fill(username);
  await page.locator('#password').fill(password);
  await page.locator('button[type="submit"]:has-text("登录")').click();

  await page.waitForTimeout(2000);
  await waitForPageReady(page);
}

/**
 * 登出用户
 */
export async function logoutUser(page: Page): Promise<void> {
  // 使用页面上下文调用登出 API（这样会携带 session cookie）
  await page.evaluate(async () => {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } catch {
      // 忽略错误
    }
  });
  
  // 清除所有 cookies
  const context = page.context();
  await context.clearCookies();
  
  // 导航到登录页面并等待完全加载
  await page.goto('/#login');
  await page.waitForLoadState('networkidle');
  
  // 等待 Alpine.js 初始化完成
  await page.waitForFunction(() => {
    const alpine = (window as any).Alpine;
    return alpine !== undefined;
  }, { timeout: 10000 }).catch(() => {});
  
  // 额外等待让页面渲染完成
  await page.waitForTimeout(1000);
}

/**
 * 生成测试用户数据
 */
export function generateTestUser(): TestUser & { name: string } {
  const username = generateUsername();
  return {
    username,
    password: STRONG_PASSWORD,
    email: generateEmail(username),
    name: `Test User ${Date.now()}`
  };
}

/**
 * 切换管理后台标签页
 */
export async function switchAdminTab(
  page: Page, 
  tab: 'users' | 'groups' | 'clients' | 'invitations' | 'proxyauth'
): Promise<void> {
  const tabNames: Record<string, string> = {
    users: '用户',
    groups: '分组',
    clients: '客户端',
    invitations: '邀请',
    proxyauth: '代理认证'
  };
  
  const tabButton = page.locator(`button:has-text("${tabNames[tab]}")`);
  await tabButton.click();
  await page.waitForTimeout(500);
}

/**
 * 检查是否显示错误消息
 */
export async function hasErrorMessage(page: Page): Promise<boolean> {
  return await waitForVisible(page, '.error', 3000);
}

/**
 * 获取错误消息文本
 */
export async function getErrorMessage(page: Page): Promise<string> {
  const errorElement = page.locator('.error');
  return await errorElement.textContent() || '';
}

// 导出常量
export { STRONG_PASSWORD };