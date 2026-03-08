import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockAlertsAPIs, mockAlerts } from './fixtures/mock-api';

test.describe('Alerts Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockAlertsAPIs(page);
	});

	test('page loads with alerts heading', async ({ page }) => {
		await page.goto('/alerts');

		await expect(page.getByRole('heading', { name: 'Alerts' })).toBeVisible();
		await expect(page.getByText('Monitor and manage system alerts')).toBeVisible();
	});

	test('status filter cards show Active, Acknowledged, Resolved counts', async ({ page }) => {
		await page.goto('/alerts');

		// Active count card (1 active alert in mock data)
		await expect(page.getByText('Active').first()).toBeVisible();

		// Acknowledged count card (1 acknowledged alert in mock data)
		await expect(page.getByText('Acknowledged').first()).toBeVisible();

		// Resolved count card (1 resolved alert in mock data)
		await expect(page.getByText('Resolved').first()).toBeVisible();
	});

	test('severity filter select exists with options', async ({ page }) => {
		await page.goto('/alerts');

		const severitySelect = page.locator('select').filter({ hasText: 'All Severity' });
		await expect(severitySelect).toBeVisible();

		// Check filter options exist
		await expect(severitySelect.locator('option', { hasText: 'Critical' })).toBeAttached();
		await expect(severitySelect.locator('option', { hasText: 'Warning' })).toBeAttached();
		await expect(severitySelect.locator('option', { hasText: 'Info' })).toBeAttached();
	});

	test('status filter select exists with options', async ({ page }) => {
		await page.goto('/alerts');

		const statusSelect = page.locator('select').filter({ hasText: 'All Status' });
		await expect(statusSelect).toBeVisible();

		await expect(statusSelect.locator('option', { hasText: 'Active' })).toBeAttached();
		await expect(statusSelect.locator('option', { hasText: 'Acknowledged' })).toBeAttached();
		await expect(statusSelect.locator('option', { hasText: 'Resolved' })).toBeAttached();
	});

	test('alert cards render with mock data', async ({ page }) => {
		await page.goto('/alerts');

		// Each alert title should appear
		for (const alert of mockAlerts) {
			await expect(page.getByText(alert.title)).toBeVisible();
		}

		// Alert messages should also be visible
		await expect(page.getByText(mockAlerts[0].message)).toBeVisible();
	});

	test('Acknowledge button visible for active alerts', async ({ page }) => {
		await page.goto('/alerts');

		// The active alert should have an Acknowledge button (exact match to avoid matching "Acknowledged" filter)
		const acknowledgeButton = page.getByRole('button', { name: 'Acknowledge', exact: true });
		await expect(acknowledgeButton).toBeVisible();
	});

	test('Resolve button visible for non-resolved alerts', async ({ page }) => {
		await page.goto('/alerts');

		// Both active and acknowledged alerts should have Resolve buttons (exact match to avoid matching "Resolved" filter)
		const resolveButtons = page.getByRole('button', { name: 'Resolve', exact: true });
		// active (1) + acknowledged (1) = 2 resolve buttons
		await expect(resolveButtons).toHaveCount(2);
	});
});
