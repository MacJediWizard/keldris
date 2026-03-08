import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockOrgSettingsAPIs } from './fixtures/mock-api';

test.describe('Organization Settings Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockOrgSettingsAPIs(page);
	});

	test('page loads and displays organization name', async ({ page }) => {
		await page.goto('/organization/settings');

		await expect(page.getByRole('heading', { name: 'Organization Settings' })).toBeVisible();
		await expect(page.getByText('Manage settings for Test Organization')).toBeVisible();
	});

	test('general settings section shows org details', async ({ page }) => {
		await page.goto('/organization/settings');

		// Section heading
		await expect(page.getByText('General Settings')).toBeVisible();

		// Organization name field
		await expect(page.getByText('Organization Name')).toBeVisible();
		await expect(page.getByText('Test Organization').first()).toBeVisible();

		// URL Slug field
		await expect(page.getByText('URL Slug')).toBeVisible();
		await expect(page.getByText('test-organization')).toBeVisible();

		// Role badge
		await expect(page.getByText('Your Role')).toBeVisible();
		await expect(page.getByText('admin').first()).toBeVisible();

		// Created date
		await expect(page.getByText('Created')).toBeVisible();
	});

	test('edit button is visible for admin users', async ({ page }) => {
		await page.goto('/organization/settings');

		// The "Edit" link in General Settings section
		const editButtons = page.locator('button', { hasText: 'Edit' });
		await expect(editButtons.first()).toBeVisible();
	});

	test('clicking edit shows the editing form', async ({ page }) => {
		await page.goto('/organization/settings');

		// Click the first "Edit" button (General Settings section)
		const editButtons = page.locator('button', { hasText: 'Edit' });
		await editButtons.first().click();

		// Form fields should now be visible
		await expect(page.locator('input#name')).toBeVisible();
		await expect(page.locator('input#slug')).toBeVisible();

		// Form fields should be pre-populated
		await expect(page.locator('input#name')).toHaveValue('Test Organization');
		await expect(page.locator('input#slug')).toHaveValue('test-organization');

		// Save and Cancel buttons
		await expect(page.getByRole('button', { name: 'Save Changes' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
	});

	test('backup concurrency section displays correctly', async ({ page }) => {
		await page.goto('/organization/settings');

		await expect(page.getByText('Backup Concurrency Limits')).toBeVisible();
		await expect(page.getByText('Maximum Concurrent Backups')).toBeVisible();

		// The mock concurrency has max_concurrent_backups = 5
		await expect(page.getByText('5')).toBeVisible();

		// Currently Running and Queued labels (scope to dt elements to avoid matching notification text)
		await expect(page.getByText('Currently Running')).toBeVisible();
		await expect(page.locator('dt', { hasText: 'Queued' })).toBeVisible();
	});

	test('support section shows download bundle button', async ({ page }) => {
		await page.goto('/organization/settings');

		await expect(page.getByRole('heading', { name: 'Support', exact: true })).toBeVisible();
		await expect(page.getByText('Generate Support Bundle')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Download Bundle' })).toBeVisible();
	});

	test('danger zone is not visible for non-owner admin', async ({ page }) => {
		await page.goto('/organization/settings');

		// The current mock user is 'admin' not 'owner', so Danger Zone should not be shown
		await expect(page.getByText('Danger Zone')).not.toBeVisible();
	});
});
