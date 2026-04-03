import { test, expect } from '@playwright/test';

/**
 * OIDC 端点 E2E 测试
 * 
 * 测试 OIDC Provider 的标准端点是否正常工作。
 * 注意：JWKS 端点可能需要额外配置才能工作。
 */

test.describe('OIDC 发现端点', () => {
  test('OIDC 发现端点返回正确配置', async ({ request }) => {
    const response = await request.get('/.well-known/openid-configuration');
    expect(response.status()).toBe(200);
    
    const config = await response.json();
    
    // 验证必要的 OIDC 配置字段
    expect(config.issuer).toBeTruthy();
    expect(config.authorization_endpoint).toBeTruthy();
    expect(config.token_endpoint).toBeTruthy();
    expect(config.jwks_uri).toBeTruthy();
    expect(config.response_types_supported).toContain('code');
  });
});
