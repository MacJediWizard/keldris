import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockAgentsAPIs, mockDashboardAPIs } from './fixtures/mock-api';

test.describe('Agents Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockAgentsAPIs(page);
	});

	test('page loads with agents table', async ({ page }) => {
		await page.goto('/agents');

		// Should display the agents table with rows
		const table = page.locator('table');
		await expect(table).toBeVisible();

		// Both agent hostnames should appear
		await expect(page.getByText('prod-server-1')).toBeVisible();
		await expect(page.getByText('staging-server')).toBeVisible();
	});

	test('Add Agent button is visible', async ({ page }) => {
		await page.goto('/agents');

		// Look for a button that creates/adds a new agent
		const addButton = page.getByRole('button', { name: /add agent|register|generate/i });
		await expect(addButton).toBeVisible();
	});

	test('agent row displays hostname and OS info', async ({ page }) => {
		await page.goto('/agents');

		// The agent row should display hostname along with OS details
		await expect(page.getByText('prod-server-1')).toBeVisible();

		// OS info should be shown below hostname (e.g. "linux amd64")
		await expect(page.getByText('linux amd64')).toBeVisible();
	});

	test('status badges display correctly', async ({ page }) => {
		await page.goto('/agents');

		// Status badges are rendered as <span> elements with rounded-full class inside table rows.
		// Avoid matching hidden <option> elements in the status filter <select>.
		const activeBadge = page.locator('table tbody span.rounded-full', { hasText: 'active' });
		await expect(activeBadge).toBeVisible();

		const offlineBadge = page.locator('table tbody span.rounded-full', { hasText: 'offline' });
		await expect(offlineBadge).toBeVisible();
	});
});
