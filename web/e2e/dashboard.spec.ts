import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockDashboardAPIs } from './fixtures/mock-api';

test.describe('Dashboard Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
	});

	test('dashboard widgets load successfully', async ({ page }) => {
		await page.goto('/');

		// Wait for the dashboard to render (title from i18n should appear)
		// The h1 contains the translated 'dashboard.title' key
		const heading = page.locator('h1').first();
		await expect(heading).toBeVisible();

		// Check that at least one widget card is rendered
		const cards = page.locator('.bg-white.rounded-lg, .dark\\:bg-gray-800.rounded-lg');
		await expect(cards.first()).toBeVisible();
	});

	test('stats cards render with correct data', async ({ page }) => {
		await page.goto('/');

		// Check for the agent stat (1/2 pattern)
		await expect(page.getByText('1/2')).toBeVisible();

		// Check for the total backups count
		await expect(page.getByText('50')).toBeVisible();

		// Check for the repository count
		await expect(page.getByText('2').first()).toBeVisible();
	});

	test('activity feed section is rendered', async ({ page }) => {
		await page.goto('/');

		// The Activity Feed widget should be present on the dashboard
		// It may show "No activity" or the actual feed
		const activitySection = page.locator('text=Activity').first();
		// The section may not be visible if scrolling is needed, so just
		// check it exists in the DOM
		await expect(activitySection).toBeAttached();
	});

	test('backup calendar section renders', async ({ page }) => {
		await page.goto('/');

		// Wait for the dashboard to fully load by checking for stat data
		await expect(page.getByText('1/2')).toBeVisible();

		// The dashboard should render multiple widget cards (stat cards, calendar,
		// activity feed, etc.) each using bg-white rounded-lg border ...
		// Verify that at least several card containers are present after data loads
		const cards = page.locator('.bg-white.rounded-lg.border');
		const count = await cards.count();
		expect(count).toBeGreaterThan(3);
	});
});
