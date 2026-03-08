import { expect, test } from '@playwright/test';
import { mockUnauthenticatedUser } from './fixtures/auth';
import {
	mockSetupAPIs,
	mockSetupStatusComplete,
	mockSetupStatusIncomplete,
	mockSetupStatusSuperuserStep,
} from './fixtures/mock-api';

test.describe('Setup Page', () => {
	test('setup page loads when setup not completed', async ({ page }) => {
		await mockUnauthenticatedUser(page);
		await mockSetupAPIs(page, mockSetupStatusIncomplete);

		await page.goto('/setup');

		// Page header should be visible
		await expect(page.getByText('Keldris Server Setup')).toBeVisible();
		await expect(page.getByText('Configure your Keldris backup server')).toBeVisible();

		// Stepper should show Database and Superuser steps
		await expect(page.locator('nav[aria-label="Progress"]').getByText('Database', { exact: true })).toBeVisible();
		await expect(page.locator('nav[aria-label="Progress"]').getByText('Superuser', { exact: true })).toBeVisible();
	});

	test('redirects to login when setup is completed', async ({ page }) => {
		await mockUnauthenticatedUser(page);
		await mockSetupAPIs(page, mockSetupStatusComplete);

		await page.goto('/setup');

		// Should redirect to /login
		await page.waitForURL('**/login');
	});

	test('database step renders with test connection button', async ({ page }) => {
		await mockUnauthenticatedUser(page);
		await mockSetupAPIs(page, mockSetupStatusIncomplete);

		await page.goto('/setup');

		// Database step content
		await expect(page.getByRole('heading', { name: 'Database Connection' })).toBeVisible();
		await expect(
			page.getByText('Verifying the database connection to ensure the server can store data.'),
		).toBeVisible();

		// Auto-test fires on mount and succeeds, so we should see the success message
		await expect(page.getByText('Database connection successful')).toBeVisible();
	});

	test('superuser step has name, email, and password fields', async ({ page }) => {
		await mockUnauthenticatedUser(page);
		await mockSetupAPIs(page, mockSetupStatusSuperuserStep);

		await page.goto('/setup');

		// Superuser step content
		await expect(page.getByText('Create Superuser Account')).toBeVisible();
		await expect(
			page.getByText('Create the administrator account that will have full access to manage the server.'),
		).toBeVisible();

		// Form fields
		await expect(page.locator('#name')).toBeVisible();
		await expect(page.locator('#email')).toBeVisible();
		await expect(page.locator('#password')).toBeVisible();
		await expect(page.locator('#confirmPassword')).toBeVisible();

		// Submit button
		await expect(page.getByRole('button', { name: 'Create Account' })).toBeVisible();
	});
});
