import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockDashboardAPIs, mockRepositoriesAPIs } from './fixtures/mock-api';

test.describe('Repositories Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockDashboardAPIs(page);
		await mockRepositoriesAPIs(page);

		// Repositories page also calls organizations and agents
		await page.route('**/api/v1/agents', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: [] }) }),
		);
	});

	test('page loads successfully', async ({ page }) => {
		await page.goto('/repositories');

		// The page should render without errors
		// Check for repository names from mock data
		await expect(page.getByText('local-backups')).toBeVisible();
		await expect(page.getByText('s3-archive')).toBeVisible();
	});

	test('repository cards or table rows render', async ({ page }) => {
		await page.goto('/repositories');

		// Check that we have at least 2 items (matching our mock data)
		const repoNames = page.getByText('local-backups');
		await expect(repoNames).toBeVisible();
		const repoNames2 = page.getByText('s3-archive');
		await expect(repoNames2).toBeVisible();
	});

	test('Add Repository button is present', async ({ page }) => {
		await page.goto('/repositories');

		// Look for a button to add/create a new repository
		const addButton = page.getByRole('button', { name: /add repository|create|new/i });
		await expect(addButton).toBeVisible();
	});

	test('repository type badges display', async ({ page }) => {
		await page.goto('/repositories');

		// The mock repositories have types 'local' and 's3'.
		// Type badges are rendered as <span> elements with rounded-full class inside
		// repository cards.  Labels are capitalized: "Local", "S3".
		// Avoid matching hidden <option> elements in the type filter <select>.
		const localBadge = page.locator('span.rounded-full', { hasText: 'Local' }).first();
		await expect(localBadge).toBeVisible();

		const s3Badge = page.locator('span.rounded-full', { hasText: 'S3' }).first();
		await expect(s3Badge).toBeVisible();
	});
});
