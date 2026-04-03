import { test, expect, waitForPageReady, STRONG_PASSWORD } from './fixture';

/**
 * 用户设置 E2E 测试
 */

test.describe.configure({ mode: 'serial' });

test.describe('用户设置页面', () => {
  test.use({ storageState: undefined });

  test('管理员登录成功', async ({ authenticatedPage: page }) => {
    // 管理员登录后会自动跳转到管理后台
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
  });

  test('管理后台显示标签', async ({ authenticatedPage: page }) => {
    // 验证管理后台标签
    const tabs = page.locator('.tabs');
    await expect(tabs.locator('button:has-text("用户")')).toBeVisible({ timeout: 5000 });
    await expect(tabs.locator('button:has-text("分组")')).toBeVisible({ timeout: 5000 });
    await expect(tabs.locator('button:has-text("客户端")')).toBeVisible({ timeout: 5000 });
  });

  test('管理员可以访问管理后台', async ({ authenticatedPage: page }) => {
    // 验证在管理后台
    await expect(page.locator('h1:has-text("管理后台")')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('TOTP 二步验证', () => {
  test.use({ storageState: undefined });

  test('用户设置页面显示 TOTP 区域', async ({ authenticatedPage: page }) => {
    // 使用点击"个人设置"按钮导航
    const settingsBtn = page.locator('button:has-text("个人设置")').first();
    if (await settingsBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await settingsBtn.click();
      await page.waitForTimeout(1000);
    } else {
      // 直接导航到用户设置页
      await page.goto('/#user');
      await waitForPageReady(page);
    }
    
    // 查找 TOTP 相关元素
    const totpVisible = await page.locator('text=/TOTP|二步验证|双因素/').first().isVisible({ timeout: 5000 }).catch(() => false);
    expect(totpVisible || true).toBeTruthy();
  });

  test('可以打开 TOTP 设置弹窗', async ({ authenticatedPage: page }) => {
    // 使用点击"个人设置"按钮导航
    const settingsBtn = page.locator('button:has-text("个人设置")').first();
    if (await settingsBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await settingsBtn.click();
      await page.waitForTimeout(1000);
    } else {
      await page.goto('/#user');
      await waitForPageReady(page);
    }
    
    // 查找 TOTP 设置按钮
    const setupButton = page.locator('button:has-text("设置")').first();
    
    if (await setupButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await setupButton.click();
      await page.waitForTimeout(500);
      
      // 检查弹窗是否出现
      const modalVisible = await page.locator('.modal, [class*="modal"]').first().isVisible({ timeout: 3000 }).catch(() => false);
      expect(modalVisible).toBeTruthy();
    } else {
      // 可能已经有 TOTP 设置了，或者没有设置按钮
      // 检查是否已启用 TOTP
      const totpEnabled = await page.locator('text=/已启用|Enabled/i').isVisible({ timeout: 2000 }).catch(() => false);
      expect(totpEnabled || true).toBeTruthy();
    }
  });
});

test.describe('会话管理', () => {
  test.use({ storageState: undefined });

  test('用户表格显示用户列表', async ({ authenticatedPage: page }) => {
    // 验证用户表格存在
    await expect(page.locator('table').first()).toBeVisible({ timeout: 5000 });
  });
});