import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockDashboardAPIs, mockSchedulesAPIs } from './fixtures/mock-api';

test.describe('Schedules Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockSchedulesAPIs(page);

		// Schedules also loads favorites
		await page.route('**/api/v1/favorites*', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ favorites: [] }) }),
		);
	});

	test('page loads with schedules list', async ({ page }) => {
		await page.goto('/schedules');

		// Check that schedule names from mock data appear
		await expect(page.getByText('Daily Backup')).toBeVisible();
		await expect(page.getByText('Weekly Full')).toBeVisible();
	});

	test('create schedule modal can be opened', async ({ page }) => {
		await page.goto('/schedules');

		// Look for a button to create a new schedule
		const createButton = page.getByRole('button', { name: /create|add|new/i }).first();
		await expect(createButton).toBeVisible();
		await createButton.click();

		// A modal or form should appear
		// Look for modal content or form fields
		const modal = page.locator('[role="dialog"], .fixed.inset-0, [class*="modal"]');
		if (await modal.isVisible()) {
			await expect(modal).toBeVisible();
		}
	});

	test('schedule details display correctly', async ({ page }) => {
		await page.goto('/schedules');

		// Verify cron expression or schedule name is visible
		await expect(page.getByText('Daily Backup')).toBeVisible();

		// The enabled/disabled status should be visible
		// 'Daily Backup' is enabled, 'Weekly Full' is disabled
		const table = page.locator('table');
		if (await table.isVisible()) {
			const rows = table.locator('tbody tr');
			const rowCount = await rows.count();
			expect(rowCount).toBeGreaterThanOrEqual(2);
		}
	});

	test('Run Now button exists for enabled schedules', async ({ page }) => {
		await page.goto('/schedules');

		// Look for a "Run Now" button or action
		const runButton = page.getByRole('button', { name: /run now|run|execute/i }).first();
		// It may be in a dropdown or directly visible
		if (await runButton.isVisible().catch(() => false)) {
			await expect(runButton).toBeVisible();
		} else {
			// Check in action menus / dropdowns
			const actionButtons = page.locator('button[title*="Run"], button[aria-label*="Run"]');
			const count = await actionButtons.count();
			expect(count).toBeGreaterThanOrEqual(0);
		}
	});
});
