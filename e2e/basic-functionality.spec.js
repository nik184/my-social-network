// @ts-check
const { test, expect } = require('@playwright/test');

test.describe('Basic Application Functionality', () => {
  test('should load Node 1 homepage', async ({ page }) => {
    await page.goto('http://localhost:6996/');

    // Should have navigation
    await expect(page.locator('.nav-link')).toHaveCount(2); // Profile and Friends
  });

  test('should load Node 2 homepage', async ({ page }) => {
    await page.goto('http://localhost:6997');
    
    // Should have navigation
    await expect(page.locator('.nav-link')).toHaveCount(2); // Profile and Friends
  });

  test('should navigate to friends page on both nodes', async ({ page }) => {
    // Test Node 1
    await page.goto('http://localhost:6996/friends');
    await expect(page.locator('h1')).toContainText('My Friends');
    await expect(page.locator('#connectionStringInput')).toBeVisible();
    
    // Test Node 2
    await page.goto('http://localhost:6997/friends');
    await expect(page.locator('h1')).toContainText('My Friends');
    await expect(page.locator('#connectionStringInput')).toBeVisible();
  });

  test('should have API endpoints responding', async ({ page }) => {
    // Test Node 1 API
    await page.goto('http://localhost:6996/api/info');
    const node1Response = await page.textContent('pre');
    const node1Data = JSON.parse(node1Response);
    expect(node1Data.node).toBeTruthy();
    expect(node1Data.node.id).toBeTruthy();
    
    // Test Node 2 API
    await page.goto('http://localhost:6997/api/info');
    const node2Response = await page.textContent('pre');
    const node2Data = JSON.parse(node2Response);
    expect(node2Data.node).toBeTruthy();
    expect(node2Data.node.id).toBeTruthy();
    
    // Ensure different peer IDs
    expect(node1Data.node.id).not.toBe(node2Data.node.id);
  });
});