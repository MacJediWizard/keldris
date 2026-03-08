import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser, mockLicense } from './fixtures/auth';
import { mockSSOSettingsAPIs, mockSSOGroupMappings } from './fixtures/mock-api';

test.describe('SSO Settings Page', () => {
	test.describe('with SSO feature enabled', () => {
		test.beforeEach(async ({ page }) => {
			await mockAuthenticatedUser(page);
			await mockSSOSettingsAPIs(page);

			// Override license to include SSO feature (must be after mockAuthenticatedUser
			// so this route takes priority — Playwright matches last-registered first)
			await page.route('**/api/v1/license', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						...mockLicense,
						tier: 'professional',
						features: ['oidc', 'sso_sync'],
					}),
				}),
			);
		});

		test('page loads with heading and description', async ({ page }) => {
			await page.goto('/organization/sso');

			await expect(page.getByRole('heading', { name: 'SSO Group Sync Settings' })).toBeVisible();
			await expect(page.getByText('Map OIDC groups from your identity provider to Keldris organization roles')).toBeVisible();
		});

		test('default settings section displays correctly', async ({ page }) => {
			await page.goto('/organization/sso');

			await expect(page.getByText('Default Settings')).toBeVisible();

			// Default role shows "No default" since mockSSOSettings.default_role is null
			await expect(page.getByText('Default Role for Unmapped Groups')).toBeVisible();
			await expect(page.getByText('No default (require explicit mapping)')).toBeVisible();

			// Auto-create orgs is disabled
			await expect(page.getByText('Auto-create Organizations')).toBeVisible();
			await expect(page.getByText('Disabled')).toBeVisible();
		});

		test('edit button visible in default settings for admin', async ({ page }) => {
			await page.goto('/organization/sso');

			// The "Edit" button in the Default Settings card header
			const editButton = page.locator('button', { hasText: 'Edit' }).first();
			await expect(editButton).toBeVisible();
		});

		test('group mappings table shows existing mappings', async ({ page }) => {
			await page.goto('/organization/sso');

			await expect(page.getByText('Group Mappings')).toBeVisible();

			// Table headers
			await expect(page.getByText('OIDC Group Name')).toBeVisible();

			// Mapping rows
			for (const mapping of mockSSOGroupMappings) {
				await expect(page.getByText(mapping.oidc_group_name)).toBeVisible();
			}

			// Role badges
			const adminBadge = page.locator('span.rounded-full', { hasText: 'admin' }).first();
			await expect(adminBadge).toBeVisible();
			const memberBadge = page.locator('span.rounded-full', { hasText: 'member' }).first();
			await expect(memberBadge).toBeVisible();
		});

		test('add mapping button is visible for admin', async ({ page }) => {
			await page.goto('/organization/sso');

			await expect(page.getByRole('button', { name: 'Add Mapping' })).toBeVisible();
		});

		test('add mapping modal opens on button click', async ({ page }) => {
			await page.goto('/organization/sso');

			await page.getByRole('button', { name: 'Add Mapping' }).first().click();

			// Modal should appear with form
			await expect(page.getByRole('heading', { name: 'Add Group Mapping' })).toBeVisible();
			await expect(page.locator('input#groupName')).toBeVisible();
			await expect(page.locator('select#role')).toBeVisible();
			await expect(page.locator('input#autoCreate')).toBeVisible();
			await expect(page.locator('form').getByRole('button', { name: 'Add Mapping' })).toBeVisible();
			await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
		});

		test('edit and delete actions visible for each mapping', async ({ page }) => {
			await page.goto('/organization/sso');

			// Wait for the group mappings table to render
			await expect(page.getByText('engineering')).toBeVisible();

			// Each mapping row should have Edit and Delete buttons
			const editLinks = page.locator('table button', { hasText: 'Edit' });
			expect(await editLinks.count()).toBe(mockSSOGroupMappings.length);

			const deleteLinks = page.locator('table button', { hasText: 'Delete' });
			expect(await deleteLinks.count()).toBe(mockSSOGroupMappings.length);
		});

		test('auto create org column shows Yes/No correctly', async ({ page }) => {
			await page.goto('/organization/sso');

			// 'engineering' mapping has auto_create_org = false (shows "No")
			// 'support' mapping has auto_create_org = true (shows "Yes")
			await expect(page.getByText('Yes')).toBeVisible();
			await expect(page.getByText('No').first()).toBeVisible();
		});
	});

	test.describe('without SSO feature', () => {
		test.beforeEach(async ({ page }) => {
			await mockAuthenticatedUser(page);
			await mockSSOSettingsAPIs(page);

			// Use default community license (no SSO)
		});

		test('shows upgrade prompt when SSO is not available', async ({ page }) => {
			await page.goto('/organization/sso');

			// The heading should still show
			await expect(page.getByRole('heading', { name: 'SSO Group Sync Settings' })).toBeVisible();

			// But instead of the settings, an upgrade prompt should appear
			// The UpgradePrompt component should render
			await expect(page.getByText('Map OIDC groups from your identity provider to Keldris organization roles')).toBeVisible();

			// The group mappings table should NOT be visible
			await expect(page.getByText('Group Mappings')).not.toBeVisible();
		});
	});
});
