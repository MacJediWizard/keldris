import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockRestoreAPIs, mockSnapshots } from './fixtures/mock-api';

test.describe('Restore Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockRestoreAPIs(page);
	});

	test('page loads with restore heading and snapshots tab', async ({ page }) => {
		await page.goto('/restore');

		await expect(page.getByRole('heading', { name: 'Restore' })).toBeVisible();
		await expect(page.getByText('Browse snapshots and restore files')).toBeVisible();
	});

	test('snapshots tab is active by default and shows table', async ({ page }) => {
		await page.goto('/restore');

		// Snapshots tab should be active
		const snapshotsTab = page.getByRole('button', { name: 'Snapshots' });
		await expect(snapshotsTab).toBeVisible();

		// Table should display snapshot rows
		const table = page.locator('table');
		await expect(table).toBeVisible();

		// Snapshot short IDs should appear
		await expect(page.getByText(mockSnapshots[0].short_id)).toBeVisible();
		await expect(page.getByText(mockSnapshots[1].short_id)).toBeVisible();
	});

	test('agent filter select is available', async ({ page }) => {
		await page.goto('/restore');

		// The agent filter dropdown should have "All Agents" as the default option
		const agentSelect = page.locator('select').filter({ hasText: 'All Agents' });
		await expect(agentSelect).toBeVisible();

		// Should also contain agent hostnames as options
		await expect(agentSelect.locator('option', { hasText: 'prod-server-1' })).toBeAttached();
		await expect(agentSelect.locator('option', { hasText: 'staging-server' })).toBeAttached();
	});

	test('repository filter select is available on snapshots tab', async ({ page }) => {
		await page.goto('/restore');

		// The repository filter dropdown should have "All Repositories"
		const repoSelect = page.locator('select').filter({ hasText: 'All Repositories' });
		await expect(repoSelect).toBeVisible();

		// Should contain repository names
		await expect(repoSelect.locator('option', { hasText: 'local-backups' })).toBeAttached();
		await expect(repoSelect.locator('option', { hasText: 's3-archive' })).toBeAttached();
	});

	test('snapshot row has Restore action button', async ({ page }) => {
		await page.goto('/restore');

		// Each snapshot row should have a "Restore" button
		const restoreButtons = page.locator('table tbody').getByText('Restore');
		await expect(restoreButtons.first()).toBeVisible();
	});

	test('compare button exists but is disabled with no selection', async ({ page }) => {
		await page.goto('/restore');

		const compareButton = page.getByRole('button', { name: /Compare/ });
		await expect(compareButton).toBeVisible();
		await expect(compareButton).toBeDisabled();
	});

	test('restore jobs tab renders when clicked', async ({ page }) => {
		await page.goto('/restore');

		// Click on Restore Jobs tab
		await page.getByRole('button', { name: 'Restore Jobs' }).click();

		// Should show restore job data (snapshot ID substring)
		await expect(page.getByText('abc12345')).toBeVisible();
	});

	test('file history link is visible', async ({ page }) => {
		await page.goto('/restore');

		// Scope to main content area to avoid matching the sidebar nav link
		const fileHistoryLink = page.getByRole('main').getByRole('link', { name: /File History/ });
		await expect(fileHistoryLink).toBeVisible();
	});
});
