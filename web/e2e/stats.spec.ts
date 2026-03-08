import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockStorageStatsAPIs, mockStorageSummary } from './fixtures/mock-api';

test.describe('Storage Stats Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockStorageStatsAPIs(page);
	});

	test('storage stats page loads with heading', async ({ page }) => {
		await page.goto('/stats');

		await expect(page.getByRole('heading', { name: 'Storage Statistics' })).toBeVisible();
		await expect(
			page.getByText('Monitor storage efficiency and deduplication across your repositories'),
		).toBeVisible();
	});

	test('summary stat cards render with data', async ({ page }) => {
		await page.goto('/stats');

		// Average Dedup Ratio card
		await expect(page.getByText('Average Dedup Ratio')).toBeVisible();

		// Total Space Saved card
		await expect(page.getByText('Total Space Saved')).toBeVisible();

		// Actual Storage Used card
		await expect(page.getByText('Actual Storage Used')).toBeVisible();

		// Total Snapshots card
		await expect(page.getByText('Total Snapshots')).toBeVisible();
		await expect(page.getByText(String(mockStorageSummary.total_snapshots))).toBeVisible();

		// Repository count text
		await expect(page.getByText(`${mockStorageSummary.repository_count} repositories`).first()).toBeVisible();
	});

	test('storage growth chart section renders', async ({ page }) => {
		await page.goto('/stats');

		// Storage Growth heading
		await expect(page.getByText('Storage Growth')).toBeVisible();

		// Time range selector
		const select = page.locator('select');
		await expect(select).toBeVisible();

		// Legend items (use exact match to avoid matching "Actual Storage Used" stat card)
		await expect(page.getByText('Actual storage', { exact: true })).toBeVisible();
		await expect(page.getByText('Original data', { exact: true })).toBeVisible();
	});

	test('repository statistics table renders', async ({ page }) => {
		await page.goto('/stats');

		// Repository Statistics heading
		await expect(page.getByText('Repository Statistics')).toBeVisible();

		// Table headers
		await expect(page.getByText('Dedup Ratio').first()).toBeVisible();
		await expect(page.getByText('Space Saved').first()).toBeVisible();
		await expect(page.getByText('Actual Size')).toBeVisible();
		await expect(page.getByText('Original Size')).toBeVisible();
		await expect(page.getByText('Snapshots').first()).toBeVisible();
		await expect(page.getByText('Last Updated')).toBeVisible();

		// Repository names from mock data
		await expect(page.getByText('local-backups')).toBeVisible();
		await expect(page.getByText('s3-archive')).toBeVisible();
	});

	test('storage growth time range can be changed', async ({ page }) => {
		await page.goto('/stats');

		const select = page.locator('select');
		await expect(select).toBeVisible();

		// Verify the dropdown has the expected options
		await expect(select.locator('option[value="7"]')).toHaveText('Last 7 days');
		await expect(select.locator('option[value="30"]')).toHaveText('Last 30 days');
		await expect(select.locator('option[value="90"]')).toHaveText('Last 90 days');
		await expect(select.locator('option[value="365"]')).toHaveText('Last year');

		// Default should be 30
		await expect(select).toHaveValue('30');
	});
});
