import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

// ---- Shared mock data ----

export const mockUser = {
	id: '1',
	email: 'admin@test.com',
	name: 'Admin User',
	role: 'admin',
	current_org_role: 'admin',
	current_org_id: 'org-1',
	is_superuser: false,
	is_impersonating: false,
	language: 'en',
	created_at: '2024-01-01T00:00:00Z',
	updated_at: '2024-01-01T00:00:00Z',
};

export const mockAgents = [
	{
		id: '1',
		hostname: 'prod-server-1',
		status: 'active',
		os_info: { os: 'linux', arch: 'amd64', version: 'Ubuntu 22.04' },
		last_seen: new Date().toISOString(),
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
		version: '1.0.0',
	},
	{
		id: '2',
		hostname: 'staging-server',
		status: 'offline',
		os_info: { os: 'linux', arch: 'arm64', version: 'Debian 12' },
		last_seen: '2024-01-01T00:00:00Z',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
		version: '1.0.0',
	},
];

export const mockRepositories = [
	{
		id: '1',
		name: 'local-backups',
		type: 'local',
		org_id: 'org-1',
		status: 'healthy',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
	{
		id: '2',
		name: 's3-archive',
		type: 's3',
		org_id: 'org-1',
		status: 'healthy',
		created_at: '2024-02-01T00:00:00Z',
		updated_at: '2024-02-01T00:00:00Z',
	},
];

export const mockSchedules = [
	{
		id: '1',
		name: 'Daily Backup',
		cron: '0 2 * * *',
		enabled: true,
		agent_id: '1',
		repository_id: '1',
		priority: 1,
		preemptible: false,
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
		paths: ['/var/data'],
		excludes: [],
		repositories: [],
	},
	{
		id: '2',
		name: 'Weekly Full',
		cron: '0 0 * * 0',
		enabled: false,
		agent_id: '2',
		repository_id: '2',
		priority: 2,
		preemptible: false,
		org_id: 'org-1',
		created_at: '2024-02-01T00:00:00Z',
		updated_at: '2024-02-01T00:00:00Z',
		paths: ['/home'],
		excludes: [],
		repositories: [],
	},
];

export const mockBackups = [
	{
		id: '1',
		snapshot_id: 'abc12345',
		status: 'success',
		agent_id: '1',
		schedule_id: '1',
		repository_id: '1',
		started_at: new Date(Date.now() - 3600000).toISOString(),
		finished_at: new Date().toISOString(),
		size_bytes: 1073741824,
		org_id: 'org-1',
	},
	{
		id: '2',
		snapshot_id: 'def67890',
		status: 'failed',
		agent_id: '2',
		schedule_id: '2',
		repository_id: '2',
		started_at: new Date(Date.now() - 7200000).toISOString(),
		finished_at: new Date(Date.now() - 7100000).toISOString(),
		size_bytes: 0,
		error_message: 'Connection timeout',
		org_id: 'org-1',
	},
	{
		id: '3',
		snapshot_id: 'ghi11223',
		status: 'running',
		agent_id: '1',
		schedule_id: '1',
		repository_id: '1',
		started_at: new Date().toISOString(),
		finished_at: null,
		size_bytes: 0,
		org_id: 'org-1',
	},
];

export const mockDashboardStats = {
	agent_total: 2,
	agent_online: 1,
	agent_offline: 1,
	repository_count: 2,
	schedule_count: 2,
	schedule_enabled: 1,
	backup_total: 50,
	backup_running: 1,
	backup_failed_24h: 1,
	success_rate_7d: 95.5,
	success_rate_30d: 92.3,
	avg_dedup_ratio: 2.5,
	total_backup_size: 10737418240,
	total_raw_size: 4294967296,
	total_space_saved: 6442450944,
};

export const mockOrganizations = [
	{ id: 'org-1', name: 'Test Organization', role: 'admin' },
];

export const mockLicense = {
	tier: 'community',
	status: 'active',
	features: [],
};

export const mockVersion = {
	version: '1.0.0',
	commit: 'abc123',
	build_date: '2024-01-01T00:00:00Z',
};

// ---- Helpers ----

/**
 * Sets up route mocks for an authenticated user session.
 * Covers all API calls that the Layout component makes on load.
 */
export async function mockAuthenticatedUser(page: Page) {
	// Auth
	await page.route('**/auth/me', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockUser) }),
	);
	await page.route('**/auth/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ oidc_enabled: false, password_enabled: true }) }),
	);

	// Onboarding
	await page.route('**/api/v1/onboarding/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ needs_onboarding: false, completed_steps: [] }) }),
	);

	// License
	await page.route('**/api/v1/license', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockLicense) }),
	);
	await page.route('**/api/v1/license/features/*/check', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ result: { allowed: false, tier_required: 'professional' } }) }),
	);

	// Version
	await page.route('**/api/v1/version', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockVersion) }),
	);

	// Organizations
	await page.route('**/api/v1/organizations', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: mockOrganizations }) }),
	);

	// Branding
	await page.route('**/api/v1/branding', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ product_name: 'Keldris', logo_url: '', primary_color: '', secondary_color: '', custom_css: '', favicon_url: '' }) }),
	);

	// Alerts count (header badge)
	await page.route('**/api/v1/alerts/count', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ count: 3 }) }),
	);

	// Changelog
	await page.route('**/api/v1/changelog', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: [] }) }),
	);

	// Recent items (header)
	await page.route('**/api/v1/recent-items*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [] }) }),
	);

	// Active maintenance window
	await page.route('**/api/v1/maintenance/active', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ active: false }) }),
	);

	// Announcements (banner)
	await page.route('**/api/v1/announcements*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ announcements: [] }) }),
	);

	// Password expiration check
	await page.route('**/api/v1/password-expiration*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ expires_soon: false }) }),
	);

	// Trial banner
	await page.route('**/api/v1/license/trial*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) }),
	);

	// Favorites
	await page.route('**/api/v1/favorites*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ favorites: [] }) }),
	);

	// Search
	await page.route('**/api/v1/search*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ results: [], total: 0 }) }),
	);

	// Air-gap status
	await page.route('**/api/public/airgap/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ air_gap_mode: false }) }),
	);
}

/**
 * Sets up route mocks for an unauthenticated session (API returns 401).
 */
export async function mockUnauthenticatedUser(page: Page) {
	await page.route('**/auth/me', (route) =>
		route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ error: 'Unauthorized' }) }),
	);
	await page.route('**/auth/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ oidc_enabled: false, password_enabled: true }) }),
	);
}

/**
 * Simulates a login by mocking the login endpoint and /auth/me.
 */
export async function login(page: Page, email: string, password: string) {
	await page.route('**/auth/login/password', (route) => {
		const body = route.request().postDataJSON();
		if (body.email === email && body.password === password) {
			return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ message: 'Login successful' }) });
		}
		return route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ error: 'Invalid email or password' }) });
	});
}

/**
 * Asserts that the user appears to be logged in (sidebar visible).
 */
export async function expectLoggedIn(page: Page) {
	await expect(page.locator('aside')).toBeVisible();
}

/**
 * Asserts that the user appears to be logged out (login form visible).
 */
export async function expectLoggedOut(page: Page) {
	await expect(page.getByText('Sign in to your account')).toBeVisible();
}
