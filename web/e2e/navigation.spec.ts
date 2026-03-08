import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import {
	mockAgentsAPIs,
	mockBackupsAPIs,
	mockDashboardAPIs,
	mockRepositoriesAPIs,
	mockSchedulesAPIs,
} from './fixtures/mock-api';

test.describe('Sidebar Navigation & Routing', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockAgentsAPIs(page);
		await mockRepositoriesAPIs(page);
		await mockSchedulesAPIs(page);
		await mockBackupsAPIs(page);
	});

	test('sidebar renders all primary nav links', async ({ page }) => {
		await page.goto('/');

		const sidebar = page.locator('aside');
		await expect(sidebar).toBeVisible();

		// Check that the main nav items are present (using the i18n keys that resolve to English)
		// These are visible because i18n fallbacks to the key name or English translations
		const navLinks = sidebar.locator('nav a');
		const navCount = await navLinks.count();
		expect(navCount).toBeGreaterThanOrEqual(10);

		// Verify a few known nav items exist by their href
		await expect(sidebar.locator('a[href="/"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/agents"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/repositories"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/schedules"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/backups"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/restore"]')).toBeVisible();
		await expect(sidebar.locator('a[href="/alerts"]')).toBeVisible();
	});

	test('clicking Agents nav navigates to agents page', async ({ page }) => {
		await page.goto('/');

		await page.locator('aside a[href="/agents"]').click();
		await expect(page).toHaveURL(/\/agents$/);
	});

	test('clicking Repositories nav navigates to repositories page', async ({ page }) => {
		await page.goto('/');

		await page.locator('aside a[href="/repositories"]').click();
		await expect(page).toHaveURL(/\/repositories$/);
	});

	test('clicking Schedules nav navigates to schedules page', async ({ page }) => {
		await page.goto('/');

		await page.locator('aside a[href="/schedules"]').click();
		await expect(page).toHaveURL(/\/schedules$/);
	});

	test('active nav item is highlighted', async ({ page }) => {
		await page.goto('/agents');

		// The active link should have bg-indigo-600 class
		const agentsLink = page.locator('aside a[href="/agents"]');
		await expect(agentsLink).toHaveClass(/bg-indigo-600/);

		// Other links should NOT have bg-indigo-600
		const dashboardLink = page.locator('aside a[href="/"]');
		await expect(dashboardLink).not.toHaveClass(/bg-indigo-600/);
	});

	test('deep URL navigation works', async ({ page }) => {
		// Navigate directly to a deep URL
		await page.goto('/backups');

		// The page should load and the sidebar should show backups as active
		const backupsLink = page.locator('aside a[href="/backups"]');
		await expect(backupsLink).toHaveClass(/bg-indigo-600/);
	});

	test('navigation between pages preserves sidebar state', async ({ page }) => {
		await page.goto('/');

		// Navigate to agents
		await page.locator('aside a[href="/agents"]').click();
		await expect(page).toHaveURL(/\/agents$/);

		// Navigate to repositories
		await page.locator('aside a[href="/repositories"]').click();
		await expect(page).toHaveURL(/\/repositories$/);

		// Navigate back to dashboard
		await page.locator('aside a[href="/"]').click();
		await expect(page).toHaveURL(/\/$/);

		// Sidebar should still be visible
		await expect(page.locator('aside')).toBeVisible();
	});
});
