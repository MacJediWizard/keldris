import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import { mockUserManagementAPIs, mockUsers } from './fixtures/mock-api';

test.describe('User Management Page', () => {
	test.beforeEach(async ({ page }) => {
		await mockAuthenticatedUser(page);
		await mockUserManagementAPIs(page);
	});

	test('page loads with heading and subtitle', async ({ page }) => {
		await page.goto('/organization/users');

		await expect(page.getByRole('heading', { name: 'User Management' })).toBeVisible();
		await expect(page.getByText('Manage users in Test Organization')).toBeVisible();
	});

	test('invite user button is visible for admin', async ({ page }) => {
		await page.goto('/organization/users');

		await expect(page.getByRole('button', { name: 'Invite User' })).toBeVisible();
	});

	test('users table shows all users', async ({ page }) => {
		await page.goto('/organization/users');

		// Section heading
		await expect(page.getByText('All Users')).toBeVisible();

		// Table headers
		await expect(page.getByText('Name', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('Email', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('Role', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('Status', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('Last Login', { exact: true }).first()).toBeVisible();

		// User rows - check names and emails (scope to table to avoid matching inviter names)
		const usersTable = page.locator('table').first();
		for (const user of mockUsers) {
			await expect(usersTable.getByText(user.name)).toBeVisible();
			await expect(usersTable.getByText(user.email)).toBeVisible();
		}
	});

	test('role badges display correctly', async ({ page }) => {
		await page.goto('/organization/users');

		// Role badges should display in the table
		// Admin User has admin role
		const adminBadge = page.locator('span.rounded-full', { hasText: 'admin' }).first();
		await expect(adminBadge).toBeVisible();

		// Alice Johnson has member role
		const memberBadge = page.locator('span.rounded-full', { hasText: 'member' }).first();
		await expect(memberBadge).toBeVisible();

		// Bob Smith has readonly role
		const readonlyBadge = page.locator('span.rounded-full', { hasText: 'readonly' }).first();
		await expect(readonlyBadge).toBeVisible();
	});

	test('status badges display correctly', async ({ page }) => {
		await page.goto('/organization/users');

		// Wait for table to render by checking a known user is visible
		await expect(page.locator('table').first().getByText('Admin User')).toBeVisible();

		// Active status badges
		const activeBadges = page.locator('span.rounded-full', { hasText: 'active' });
		expect(await activeBadges.count()).toBeGreaterThanOrEqual(2);

		// Disabled status badge for Bob
		const disabledBadge = page.locator('span.rounded-full', { hasText: 'disabled' });
		await expect(disabledBadge.first()).toBeVisible();
	});

	test('current user row shows "You" label', async ({ page }) => {
		await page.goto('/organization/users');

		// Admin User (id=1) is the current user, should show "You" instead of action buttons
		const usersTable = page.locator('table').first();
		await expect(usersTable.getByText('You', { exact: true })).toBeVisible();
	});

	test('action buttons visible for manageable users', async ({ page }) => {
		await page.goto('/organization/users');

		// Wait for table to render by checking a known user is visible
		await expect(page.locator('table').first().getByText('Alice Johnson')).toBeVisible();

		// For non-self, non-owner users, edit button (title="Edit") should be visible
		const editButtons = page.locator('button[title="Edit"]');
		expect(await editButtons.count()).toBeGreaterThanOrEqual(1);

		// Delete button
		const deleteButtons = page.locator('button[title="Delete"]');
		expect(await deleteButtons.count()).toBeGreaterThanOrEqual(1);
	});

	test('pending invitations section is visible for admin', async ({ page }) => {
		await page.goto('/organization/users');

		await expect(page.getByText('Pending Invitations')).toBeVisible();

		// The invitation email should be visible
		await expect(page.getByText('newuser@test.com')).toBeVisible();

		// Bulk invite link
		await expect(page.getByText('Bulk Invite (CSV)')).toBeVisible();
	});

	test('invite user modal opens on button click', async ({ page }) => {
		await page.goto('/organization/users');

		await page.getByRole('button', { name: 'Invite User' }).click();

		// Modal should appear with form fields
		await expect(page.getByRole('heading', { name: 'Invite User' })).toBeVisible();
		await expect(page.locator('input#email')).toBeVisible();
		await expect(page.locator('input#name')).toBeVisible();
		await expect(page.locator('select#role')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Send Invitation', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
	});
});
