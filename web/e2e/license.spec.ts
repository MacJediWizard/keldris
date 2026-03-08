import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import {
	mockLicenseAPIs,
	mockLicenseInfo,
	mockLicenseInfoPro,
} from './fixtures/mock-api';

test.describe('License Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
	});

	test('license page loads with free tier', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// Page heading
		await expect(page.getByRole('heading', { name: 'License', exact: true })).toBeVisible();
	});

	test('license details section is displayed', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// License Details card
		await expect(page.getByText('License Details')).toBeVisible();

		// Tier should show "free"
		await expect(page.getByText('Tier')).toBeVisible();
		await expect(page.getByText('free').first()).toBeVisible();

		// Key Source should show "Not configured"
		await expect(page.getByText('Key Source')).toBeVisible();
		await expect(page.getByText('Not configured')).toBeVisible();
	});

	test('license key activation form exists for free tier', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// Activate License section
		await expect(page.getByText('Activate License')).toBeVisible();
		await expect(
			page.getByText('Enter your license key to unlock Pro or Enterprise features.'),
		).toBeVisible();

		// License key input
		await expect(page.getByPlaceholder('Enter your license key...')).toBeVisible();

		// Activate button
		await expect(page.getByRole('button', { name: 'Activate' })).toBeVisible();
	});

	test('resource limits are displayed', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// Resource Limits card (scope to main to avoid sidebar nav matches)
		const main = page.locator('main');
		await expect(main.getByText('Resource Limits')).toBeVisible();

		// Individual limits: Agents=3, Servers=1, Users=5, Organizations=1, Storage=Unlimited
		await expect(main.getByText('Agents', { exact: true }).first()).toBeVisible();
		await expect(main.getByText('Servers', { exact: true }).first()).toBeVisible();
		await expect(main.getByText('Users', { exact: true }).first()).toBeVisible();
		await expect(main.getByText('Organizations', { exact: true }).first()).toBeVisible();
	});

	test('feature list shows included features for free tier', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// Included Features section
		await expect(page.getByText('Included Features')).toBeVisible();

		// Free tier has no features, should show empty message
		await expect(page.getByText('No features included in the current plan.')).toBeVisible();
	});

	test('pro license shows features and details', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfoPro);

		await page.goto('/organization/license');

		// Page heading
		await expect(page.getByRole('heading', { name: 'License', exact: true })).toBeVisible();

		// Tier should show "pro"
		await expect(page.getByText('pro').first()).toBeVisible();

		// Customer name
		await expect(page.getByText('Acme Corp').first()).toBeVisible();

		// Pro features should be listed as checkmarks (oidc, api_access, etc.)
		await expect(page.getByText('Included Features')).toBeVisible();
		// With features=['oidc', 'api_access', 'custom_reports', 'white_label', 'priority_support']
		// Features are rendered with underscores replaced by spaces and CSS capitalize
		await expect(page.getByText('oidc', { exact: true })).toBeVisible();
		await expect(page.getByText('api access', { exact: true })).toBeVisible();
	});

	test('free trial section shows for free tier', async ({ page }) => {
		await mockLicenseAPIs(page, mockLicenseInfo);

		await page.goto('/organization/license');

		// Start Free Trial section
		await expect(page.getByText('Start Free 14-Day Trial')).toBeVisible();
		await expect(page.getByPlaceholder('Enter your email...')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Start 14-Day Trial' })).toBeVisible();
	});
});
