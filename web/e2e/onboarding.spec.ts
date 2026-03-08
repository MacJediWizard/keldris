import { expect, test } from '@playwright/test';
import { mockAuthenticatedUser } from './fixtures/auth';
import {
	mockOnboardingAPIs,
	mockOnboardingStatusAgentStep,
	mockOnboardingStatusIncomplete,
	mockOnboardingStatusOrgStep,
	mockOnboardingStatusRepoStep,
} from './fixtures/mock-api';

test.describe('Onboarding Page', () => {
	test.beforeEach(async ({ page }) => {
		// Onboarding is inside the authenticated Layout
		await mockAuthenticatedUser(page);
	});

	test('onboarding page loads with stepper and welcome step', async ({ page }) => {
		await mockOnboardingAPIs(page, mockOnboardingStatusIncomplete);

		await page.goto('/onboarding');

		// Page header
		await expect(page.getByText('Getting Started')).toBeVisible();
		await expect(page.getByText('Complete these steps to set up your first backup')).toBeVisible();

		// Stepper items should be visible in the sidebar
		const stepper = page.locator('nav[aria-label="Progress"]');
		await expect(stepper.getByText('Welcome', { exact: true })).toBeVisible();
		await expect(stepper.getByText('License', { exact: true })).toBeVisible();
		await expect(stepper.getByText('Organization', { exact: true })).toBeVisible();
		await expect(stepper.getByText('Repository', { exact: true })).toBeVisible();
		await expect(stepper.getByText('Install Agent', { exact: true })).toBeVisible();
		await expect(stepper.getByText('Schedule', { exact: true })).toBeVisible();
		await expect(stepper.getByText('Verify', { exact: true })).toBeVisible();

		// Welcome step content
		await expect(page.getByRole('heading', { name: 'Welcome to Keldris' })).toBeVisible();
		await expect(
			page.getByText('Keldris is your self-hosted backup solution.', { exact: false }),
		).toBeVisible();

		// Skip button should be visible
		await expect(page.getByText('Skip setup wizard')).toBeVisible();
	});

	test('organization step renders', async ({ page }) => {
		await mockOnboardingAPIs(page, mockOnboardingStatusOrgStep);

		await page.goto('/onboarding');

		// The organization step should show the org name input or org info
		// The step heading references organization setup (scoped to stepper to avoid sidebar matches)
		await expect(page.locator('nav[aria-label="Progress"]').getByText('Organization', { exact: true })).toBeVisible();

		// The step should be in the main content area
		// OrganizationStep shows a "Continue" or "Next" button
		// It also shows org name editing UI
		const mainContent = page.locator('.bg-white.rounded-lg.border');
		await expect(mainContent.first()).toBeVisible();
	});

	test('repository step renders', async ({ page }) => {
		await mockOnboardingAPIs(page, mockOnboardingStatusRepoStep);

		await page.goto('/onboarding');

		// Repository step should show repository type selection or creation form
		// The stepper should highlight the Repository step
		const mainContent = page.locator('.bg-white.rounded-lg.border').first();
		await expect(mainContent).toBeVisible();

		// The step should show a repository creation form or existing repos
		// "Repository" should appear in the stepper as the current step
		await expect(page.getByText('Repository').first()).toBeVisible();
	});

	test('agent install step renders', async ({ page }) => {
		await mockOnboardingAPIs(page, mockOnboardingStatusAgentStep);

		await page.goto('/onboarding');

		// Agent step should show agent installation instructions
		const mainContent = page.locator('.bg-white.rounded-lg.border').first();
		await expect(mainContent).toBeVisible();

		// "Install Agent" should appear in the stepper
		await expect(page.getByText('Install Agent')).toBeVisible();
	});
});
