import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser, mockLicense } from './fixtures/auth';
import { mockReportsAPIs, mockReportSchedules } from './fixtures/mock-api';

test.describe('Reports Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockReportsAPIs(page);

		// Override the license mock to grant advanced_reporting feature
		await page.route('**/api/v1/license', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					...mockLicense,
					tier: 'professional',
					features: ['custom_reports', 'advanced_reporting'],
				}),
			}),
		);

		// Override feature check to allow advanced_reporting
		await page.route('**/api/v1/license/features/*/check', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ result: { allowed: true, tier_required: 'professional' } }),
			}),
		);
	});

	test('page loads with heading', async ({ page }) => {
		await page.goto('/reports');

		await expect(page.getByRole('heading', { name: 'Email Reports' })).toBeVisible();
		await expect(page.getByText('Schedule automated backup summary reports')).toBeVisible();
	});

	test('schedule and history tabs render', async ({ page }) => {
		await page.goto('/reports');

		const schedulesTab = page.getByRole('button', { name: 'Schedules' });
		await expect(schedulesTab).toBeVisible();

		const historyTab = page.getByRole('button', { name: 'History' });
		await expect(historyTab).toBeVisible();
	});

	test('create schedule button exists', async ({ page }) => {
		await page.goto('/reports');

		const createButton = page.getByRole('button', { name: 'Create Schedule' });
		await expect(createButton).toBeVisible();
	});

	test('schedules table displays mock schedule data', async ({ page }) => {
		await page.goto('/reports');

		// Schedule name should appear in the table
		await expect(page.getByText(mockReportSchedules[0].name)).toBeVisible();

		// Frequency badge should appear in the table (scope to table to avoid matching the schedule name "Weekly Summary")
		await expect(page.locator('table tbody span').getByText('weekly')).toBeVisible();
	});

	test('history tab shows report history when clicked', async ({ page }) => {
		await page.goto('/reports');

		// Switch to history tab
		await page.getByRole('button', { name: 'History' }).click();

		// History table should be visible with column headers
		await expect(page.getByText('Type').first()).toBeVisible();
		await expect(page.getByText('Period').first()).toBeVisible();
		await expect(page.getByText('Sent At').first()).toBeVisible();
	});

	test('create schedule button opens modal', async ({ page }) => {
		await page.goto('/reports');

		await page.getByRole('button', { name: 'Create Schedule' }).click();

		// Modal should appear with form fields
		await expect(page.getByText('Create Report Schedule')).toBeVisible();
		await expect(page.getByLabel('Schedule Name')).toBeVisible();
		await expect(page.getByLabel('Frequency')).toBeVisible();
		await expect(page.getByLabel(/Recipients/)).toBeVisible();
	});
});

test.describe('Reports Page - Community Tier', () => {
	test('shows upgrade prompt when feature is not available', async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockReportsAPIs(page);

		// Keep the default community license (no advanced_reporting)

		await page.goto('/reports');

		await expect(page.getByRole('heading', { name: 'Email Reports' })).toBeVisible();
		// Should show upgrade prompt instead of the schedules table
		await expect(page.getByText('Upgrade')).toBeVisible();
	});
});
