import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser, mockLicense } from './fixtures/auth';
import { mockAuditLogs, mockAuditLogsAPIs } from './fixtures/mock-api';

test.describe('Audit Logs Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockAuditLogsAPIs(page);

		// Override the license mock to grant audit_logs feature
		await page.route('**/api/v1/license', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					...mockLicense,
					tier: 'professional',
					features: ['audit_logs'],
				}),
			}),
		);

		// Override feature check to allow audit_logs
		await page.route('**/api/v1/license/features/*/check', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ result: { allowed: true, tier_required: 'professional' } }),
			}),
		);
	});

	test('page loads with audit log heading', async ({ page }) => {
		await page.goto('/audit-logs');

		await expect(page.locator('h1').getByText('Audit Logs')).toBeVisible();
		await expect(page.getByText('Track all user and system actions for compliance')).toBeVisible();
	});

	test('filter controls exist', async ({ page }) => {
		await page.goto('/audit-logs');

		// Search input
		await expect(page.getByPlaceholder('Search logs...')).toBeVisible();

		// Action filter select
		const actionSelect = page.locator('select').filter({ hasText: 'All Actions' });
		await expect(actionSelect).toBeVisible();

		// Resource filter select
		const resourceSelect = page.locator('select').filter({ hasText: 'All Resources' });
		await expect(resourceSelect).toBeVisible();

		// Result filter select
		const resultSelect = page.locator('select').filter({ hasText: 'All Results' });
		await expect(resultSelect).toBeVisible();

		// Search button
		await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
	});

	test('audit log table renders with correct column headers', async ({ page }) => {
		await page.goto('/audit-logs');

		// Wait for actual audit log data to load (confirms the table rendered with data, not just loading skeleton)
		await expect(page.getByText('192.168.1.100').first()).toBeVisible();

		const table = page.locator('table');
		await expect(table).toBeVisible();

		// Check column headers (scope to th to avoid matching hidden select options)
		const th = page.locator('table th');
		await expect(th.getByText('Timestamp')).toBeVisible();
		await expect(th.getByText('Action')).toBeVisible();
		await expect(th.getByText('Resource')).toBeVisible();
		await expect(th.getByText('IP Address')).toBeVisible();
		await expect(th.getByText('Result')).toBeVisible();
		await expect(th.getByText('Details')).toBeVisible();
	});

	test('log entries render with data from mock', async ({ page }) => {
		await page.goto('/audit-logs');

		// IP addresses should be visible
		await expect(page.getByText('192.168.1.100').first()).toBeVisible();
		await expect(page.getByText('10.0.0.50')).toBeVisible();

		// Details text should appear
		await expect(page.getByText('Password login')).toBeVisible();
		await expect(page.getByText('Created agent prod-db-1')).toBeVisible();
		await expect(page.getByText('Insufficient permissions')).toBeVisible();
	});

	test('export CSV button exists', async ({ page }) => {
		await page.goto('/audit-logs');

		await expect(page.getByRole('button', { name: /Export CSV/ })).toBeVisible();
	});

	test('export JSON button exists', async ({ page }) => {
		await page.goto('/audit-logs');

		await expect(page.getByRole('button', { name: /Export JSON/ })).toBeVisible();
	});

	test('pagination shows result count', async ({ page }) => {
		await page.goto('/audit-logs');

		// Pagination text: "Showing 1 to 3 of 3 results"
		await expect(page.getByText(/Showing 1 to 3 of 3 results/)).toBeVisible();
	});
});

test.describe('Audit Logs Page - Community Tier', () => {
	test('shows upgrade prompt when feature is not available', async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockAuditLogsAPIs(page);

		// Keep the default community license (no audit_logs)

		await page.goto('/audit-logs');

		// Use first() since both the page h1 and the UpgradePromptCard h3 contain "Audit Logs"
		await expect(page.getByRole('heading', { name: 'Audit Logs' }).first()).toBeVisible();
		// Should show upgrade prompt instead of the table
		await expect(page.getByText('Upgrade')).toBeVisible();
	});
});
