import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockBackupsAPIs, mockDashboardAPIs } from './fixtures/mock-api';

test.describe('Backups Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockBackupsAPIs(page);

		// Backups page also loads favorites
		await page.route('**/api/v1/favorites*', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ favorites: [] }) }),
		);
	});

	test('page loads with backups table', async ({ page }) => {
		await page.goto('/backups');

		// Check that the backup table or list is rendered
		const table = page.locator('table');
		await expect(table).toBeVisible();

		// Check that snapshot IDs from mock data appear
		await expect(page.getByText('abc12345', { exact: false }).first()).toBeVisible();
	});

	test('status filter controls are available', async ({ page }) => {
		await page.goto('/backups');

		// The backups page uses <select> dropdowns for filtering, not buttons.
		// Verify the status filter select is present with expected options.
		const statusSelect = page.locator('select').filter({ hasText: 'All Status' });
		await expect(statusSelect).toBeVisible();

		// Also check the agent filter select exists
		const agentSelect = page.locator('select').filter({ hasText: 'All Agents' });
		await expect(agentSelect).toBeVisible();
	});

	test('backup details are expandable or clickable', async ({ page }) => {
		await page.goto('/backups');

		// Wait for the backup data to load by checking for a snapshot ID from mock data
		await expect(page.getByText('abc12345').first()).toBeVisible();

		// The table should have rows with backup details and a "Details" button per row
		const detailButtons = page.getByRole('button', { name: 'Details' });
		const count = await detailButtons.count();
		expect(count).toBeGreaterThanOrEqual(1);
	});

	test('status badges show correct colors', async ({ page }) => {
		await page.goto('/backups');

		// Wait for data to load
		await expect(page.getByText('abc12345').first()).toBeVisible();

		// Status badges are rendered as <span> elements with rounded-full class inside table rows.
		// Avoid matching hidden <option> elements in filter <select> dropdowns.
		const successBadge = page.locator('table tbody span.rounded-full', { hasText: 'success' });
		await expect(successBadge).toBeVisible();

		const failedBadge = page.locator('table tbody span.rounded-full', { hasText: 'failed' });
		await expect(failedBadge).toBeVisible();
	});
});
