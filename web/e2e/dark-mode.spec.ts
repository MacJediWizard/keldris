import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockAgentsAPIs, mockDashboardAPIs } from './fixtures/mock-api';

test.describe('Dark Mode / Theme Toggle', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockAgentsAPIs(page);
	});

	test('toggle dark mode activates dark theme', async ({ page }) => {
		await page.goto('/');

		// The theme toggle button is in the sidebar footer, titled "Theme: ..."
		const themeButton = page.locator('button[title*="Theme"]');
		await expect(themeButton).toBeVisible();

		// Click to cycle through themes: system -> light -> dark -> system ...
		// The initial state depends on localStorage; keep clicking until dark is active
		let isDark = await page.locator('html').evaluate((el) => el.classList.contains('dark'));

		// Click at most 3 times to cycle to dark mode
		for (let i = 0; i < 3 && !isDark; i++) {
			await themeButton.click();
			// Small wait for theme to apply
			await page.waitForTimeout(100);
			isDark = await page.locator('html').evaluate((el) => el.classList.contains('dark'));
		}

		expect(isDark).toBe(true);
	});

	test('dark mode applies dark background', async ({ page }) => {
		await page.goto('/');

		// Force dark mode via localStorage, then reload so the theme hook reads it on mount
		await page.evaluate(() => {
			localStorage.setItem('keldris-theme', 'dark');
		});
		await page.reload();

		// Wait for the React app to mount and apply the theme
		await expect(page.locator('aside')).toBeVisible();

		// Verify the html element has the 'dark' class
		const htmlClass = await page.locator('html').getAttribute('class');
		expect(htmlClass).toContain('dark');

		// Verify the main background has dark styling
		const bgColor = await page.locator('.min-h-screen').evaluate((el) => {
			return window.getComputedStyle(el).backgroundColor;
		});
		// Dark mode bg-gray-900 is a very dark color
		// Just check it is NOT white (rgb(255, 255, 255))
		expect(bgColor).not.toBe('rgb(255, 255, 255)');
	});

	test('toggle back to light mode restores light theme', async ({ page }) => {
		// Navigate first so localStorage is accessible
		await page.goto('/');

		// Set dark mode via localStorage and reload
		await page.evaluate(() => {
			localStorage.setItem('keldris-theme', 'dark');
		});
		await page.reload();

		// Wait for app to mount
		const themeButton = page.locator('button[title*="Theme"]');
		await expect(themeButton).toBeVisible();

		// Click to cycle from dark -> system -> light
		// We need to check after each click
		let isLight = false;
		for (let i = 0; i < 3 && !isLight; i++) {
			await themeButton.click();
			await page.waitForTimeout(100);
			const stored = await page.evaluate(() => localStorage.getItem('keldris-theme'));
			if (stored === 'light') {
				isLight = true;
			}
		}

		expect(isLight).toBe(true);

		// Verify dark class is removed
		const htmlClass = await page.locator('html').getAttribute('class');
		expect(htmlClass ?? '').not.toContain('dark');
	});

	test('theme persists across page navigation', async ({ page }) => {
		// Navigate first so localStorage is accessible
		await page.goto('/');

		// Set dark mode and reload
		await page.evaluate(() => {
			localStorage.setItem('keldris-theme', 'dark');
		});
		await page.reload();

		// Wait for app to mount
		await expect(page.locator('aside')).toBeVisible();

		// Verify dark mode is active
		let htmlClass = await page.locator('html').getAttribute('class');
		expect(htmlClass).toContain('dark');

		// Navigate to agents page
		await page.locator('aside a[href="/agents"]').click();
		await expect(page).toHaveURL(/\/agents/);

		// Dark mode should still be active
		htmlClass = await page.locator('html').getAttribute('class');
		expect(htmlClass).toContain('dark');

		// Navigate to repositories
		await page.locator('aside a[href="/repositories"]').click();
		await expect(page).toHaveURL(/\/repositories/);

		// Dark mode should persist
		htmlClass = await page.locator('html').getAttribute('class');
		expect(htmlClass).toContain('dark');
	});
});
