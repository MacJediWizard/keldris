import type { Page } from '@playwright/test';
import {
	mockAgents,
	mockBackups,
	mockDashboardStats,
	mockRepositories,
	mockSchedules,
} from './auth';

// ---- Mock data for Restore page ----

export const mockSnapshots = [
	{
		id: 'snap-1',
		short_id: 'abc123',
		agent_id: '1',
		repository_id: '1',
		time: '2024-06-01T02:00:00Z',
		hostname: 'prod-server-1',
		paths: ['/var/data'],
		size_bytes: 1073741824,
		tags: [],
	},
	{
		id: 'snap-2',
		short_id: 'def456',
		agent_id: '2',
		repository_id: '2',
		time: '2024-06-02T02:00:00Z',
		hostname: 'staging-server',
		paths: ['/home'],
		size_bytes: 536870912,
		tags: [],
	},
];

export const mockRestoreJobs = [
	{
		id: 'restore-1',
		snapshot_id: 'abc12345deadbeef',
		agent_id: '1',
		repository_id: '1',
		target_path: '/tmp/restore',
		status: 'completed',
		is_cross_agent: false,
		include_paths: null,
		path_mappings: null,
		progress: null,
		created_at: '2024-06-01T03:00:00Z',
		updated_at: '2024-06-01T03:05:00Z',
	},
];

// ---- Mock data for Alerts page ----

export const mockAlerts = [
	{
		id: 'alert-1',
		title: 'Backup Failed: prod-server-1',
		message: 'Backup job for /var/data failed with timeout error',
		severity: 'critical',
		status: 'active',
		type: 'backup_failure',
		agent_id: '1',
		created_at: '2024-06-01T10:00:00Z',
		acknowledged_at: null,
		resolved_at: null,
	},
	{
		id: 'alert-2',
		title: 'Disk Space Low: staging-server',
		message: 'Available disk space below 10%',
		severity: 'warning',
		status: 'acknowledged',
		type: 'disk_space',
		agent_id: '2',
		created_at: '2024-05-30T08:00:00Z',
		acknowledged_at: '2024-05-30T09:00:00Z',
		resolved_at: null,
	},
	{
		id: 'alert-3',
		title: 'Agent Offline: dev-server',
		message: 'Agent has been unreachable for 30 minutes',
		severity: 'info',
		status: 'resolved',
		type: 'agent_offline',
		agent_id: '3',
		created_at: '2024-05-28T12:00:00Z',
		acknowledged_at: '2024-05-28T12:15:00Z',
		resolved_at: '2024-05-28T13:00:00Z',
	},
];

// ---- Mock data for Reports page ----

export const mockReportSchedules = [
	{
		id: 'sched-1',
		name: 'Weekly Summary',
		frequency: 'weekly',
		recipients: ['admin@test.com'],
		timezone: 'UTC',
		enabled: true,
		channel_id: 'ch-1',
		last_sent_at: '2024-06-01T08:00:00Z',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-06-01T08:00:00Z',
	},
];

export const mockReportHistory = [
	{
		id: 'hist-1',
		schedule_id: 'sched-1',
		frequency: 'weekly',
		period_start: '2024-05-27T00:00:00Z',
		period_end: '2024-06-02T23:59:59Z',
		recipients: ['admin@test.com'],
		status: 'sent',
		sent_at: '2024-06-03T08:00:00Z',
		created_at: '2024-06-03T08:00:00Z',
	},
];

// ---- Mock data for Audit Logs page ----

export const mockAuditLogs = [
	{
		id: 'log-1',
		user_id: '1',
		user_email: 'admin@test.com',
		action: 'login',
		resource_type: 'session',
		resource_id: 'sess-abc123',
		ip_address: '192.168.1.100',
		result: 'success',
		details: 'Password login',
		created_at: '2024-06-01T10:00:00Z',
	},
	{
		id: 'log-2',
		user_id: '1',
		user_email: 'admin@test.com',
		action: 'create',
		resource_type: 'agent',
		resource_id: 'agent-new-1',
		ip_address: '192.168.1.100',
		result: 'success',
		details: 'Created agent prod-db-1',
		created_at: '2024-06-01T10:05:00Z',
	},
	{
		id: 'log-3',
		user_id: '2',
		user_email: 'alice@test.com',
		action: 'delete',
		resource_type: 'schedule',
		resource_id: 'sched-old-1',
		ip_address: '10.0.0.50',
		result: 'failure',
		details: 'Insufficient permissions',
		created_at: '2024-06-01T11:00:00Z',
	},
];

// ---- Shared mock data for organization pages ----

export const mockCurrentOrg = {
	organization: {
		id: 'org-1',
		name: 'Test Organization',
		slug: 'test-organization',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
};

export const mockConcurrency = {
	max_concurrent_backups: 5,
	running_count: 2,
};

export const mockQueueSummary = {
	total_queued: 1,
	avg_wait_minutes: 3,
};

export const mockUsers = [
	{
		id: '1',
		email: 'admin@test.com',
		name: 'Admin User',
		role: 'admin',
		org_role: 'admin',
		status: 'active' as const,
		last_login_at: '2024-06-15T10:30:00Z',
		oidc_subject: null,
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
	{
		id: '2',
		email: 'alice@test.com',
		name: 'Alice Johnson',
		role: 'member',
		org_role: 'member',
		status: 'active' as const,
		last_login_at: '2024-06-14T08:00:00Z',
		oidc_subject: null,
		created_at: '2024-02-01T00:00:00Z',
		updated_at: '2024-02-01T00:00:00Z',
	},
	{
		id: '3',
		email: 'bob@test.com',
		name: 'Bob Smith',
		role: 'readonly',
		org_role: 'readonly',
		status: 'disabled' as const,
		last_login_at: null,
		oidc_subject: null,
		created_at: '2024-03-01T00:00:00Z',
		updated_at: '2024-03-01T00:00:00Z',
	},
];

export const mockInvitations = [
	{
		id: 'inv-1',
		email: 'newuser@test.com',
		role: 'member',
		inviter_name: 'Admin User',
		accepted_at: null,
		expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
		created_at: '2024-06-01T00:00:00Z',
	},
];

export const mockSSOGroupMappings = [
	{
		id: 'map-1',
		oidc_group_name: 'engineering',
		role: 'admin',
		auto_create_org: false,
		org_id: 'org-1',
		created_at: '2024-01-15T00:00:00Z',
		updated_at: '2024-01-15T00:00:00Z',
	},
	{
		id: 'map-2',
		oidc_group_name: 'support',
		role: 'member',
		auto_create_org: true,
		org_id: 'org-1',
		created_at: '2024-02-01T00:00:00Z',
		updated_at: '2024-02-01T00:00:00Z',
	},
];

export const mockSSOSettings = {
	default_role: null,
	auto_create_orgs: false,
};

/**
 * Mocks all dashboard-related API endpoints.
 */
export async function mockDashboardAPIs(page: Page) {
	await page.route('**/api/v1/dashboard-metrics/stats', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockDashboardStats) }),
	);

	await page.route('**/api/v1/agents', (route) => {
		if (route.request().url().includes('/agents/') && !route.request().url().includes('/agents?')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) });
	});

	await page.route('**/api/v1/repositories', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) }),
	);

	await page.route('**/api/v1/schedules', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockSchedules }) }),
	);

	await page.route('**/api/v1/backups', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ backups: mockBackups }) }),
	);

	await page.route('**/api/v1/daily-backup-stats*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ stats: [] }) }),
	);

	await page.route('**/api/v1/storage-growth*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }),
	);

	await page.route('**/api/v1/backup-duration-trend*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }),
	);

	await page.route('**/api/v1/storage-stats/summary', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				repository_count: 2,
				total_snapshots: 50,
				total_raw_size: 4294967296,
				total_restore_size: 10737418240,
				total_space_saved: 6442450944,
				avg_dedup_ratio: 2.5,
			}),
		}),
	);

	await page.route('**/api/v1/dr/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ active_runbooks: 1, total_runbooks: 2, last_test_at: null, next_test_at: null }) }),
	);

	await page.route('**/api/v1/fleet-health*', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				total_agents: 2,
				healthy_count: 1,
				warning_count: 0,
				critical_count: 0,
				unknown_count: 1,
				avg_cpu_usage: 25.5,
				avg_memory_usage: 45.2,
				avg_disk_usage: 60.1,
			}),
		}),
	);

	await page.route('**/api/v1/activity*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ events: [], total: 0 }) }),
	);

	await page.route('**/api/v1/backup-calendar*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ days: [] }) }),
	);
}

/**
 * Mocks the agents list and detail APIs.
 */
export async function mockAgentsAPIs(page: Page) {
	await page.route('**/api/v1/agents', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /agents/1
		if (/\/agents\/[^/]+/.test(url) && !url.endsWith('/agents')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) });
	});

	await page.route('**/api/v1/agents/1', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockAgents[0]) }),
	);

	await page.route('**/api/v1/agents/1/stats', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ cpu_usage: 25, memory_usage: 45, disk_usage: 60 }) }),
	);

	await page.route('**/api/v1/agents/1/backups*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ backups: mockBackups.slice(0, 1) }) }),
	);

	await page.route('**/api/v1/agents/1/schedules*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockSchedules.slice(0, 1) }) }),
	);

	await page.route('**/api/v1/agents/1/commands*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ commands: [] }) }),
	);

	await page.route('**/api/v1/agents/1/health-history*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ history: [] }) }),
	);

	await page.route('**/api/v1/agents/1/logs*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ logs: [], total: 0 }) }),
	);

	await page.route('**/api/v1/agent-groups*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ groups: [] }) }),
	);

	await page.route('**/api/v1/agents/registration-codes*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pending: [] }) }),
	);

	await page.route('**/api/v1/agents/pending*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pending: [] }) }),
	);
}

/**
 * Mocks the repositories APIs.
 */
export async function mockRepositoriesAPIs(page: Page) {
	await page.route('**/api/v1/repositories', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) }),
	);

	await page.route('**/api/v1/verification-status*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ statuses: {} }) }),
	);
}

/**
 * Mocks the schedules APIs.
 */
export async function mockSchedulesAPIs(page: Page) {
	await page.route('**/api/v1/schedules', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockSchedules }) }),
	);

	await page.route('**/api/v1/agents', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) }),
	);

	await page.route('**/api/v1/repositories', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) }),
	);

	await page.route('**/api/v1/policies*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ policies: [] }) }),
	);
}

/**
 * Mocks the backups APIs.
 */
export async function mockBackupsAPIs(page: Page) {
	await page.route('**/api/v1/backups', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ backups: mockBackups }) }),
	);

	await page.route('**/api/v1/agents', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) }),
	);

	await page.route('**/api/v1/repositories', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) }),
	);

	await page.route('**/api/v1/schedules', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockSchedules }) }),
	);

	await page.route('**/api/v1/tags*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ tags: [] }) }),
	);

	await page.route('**/api/v1/backup-calendar*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ days: [] }) }),
	);
}

/**
 * Mocks the Organization Settings page APIs.
 */
export async function mockOrgSettingsAPIs(page: Page) {
	await page.route('**/api/v1/organizations/current', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockCurrentOrg) }),
	);

	await page.route('**/api/v1/organizations/*/concurrency', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockConcurrency) }),
	);

	await page.route('**/api/v1/backup-queue/summary', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockQueueSummary) }),
	);

	await page.route('**/api/v1/support/bundle', (route) =>
		route.fulfill({ status: 200, contentType: 'application/octet-stream', body: '' }),
	);
}

/**
 * Mocks the User Management page APIs.
 */
export async function mockUserManagementAPIs(page: Page) {
	await page.route('**/api/v1/organizations/current', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockCurrentOrg) }),
	);

	await page.route('**/api/v1/users', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /users/1
		if (/\/users\/[^/?]+/.test(url) && !url.endsWith('/users')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ users: mockUsers }) });
	});

	await page.route('**/api/v1/organizations/*/invitations', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ invitations: mockInvitations }) }),
	);
}

/**
 * Mocks the SSO Settings page APIs.
 */
export async function mockSSOSettingsAPIs(page: Page) {
	await page.route('**/api/v1/organizations/current', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockCurrentOrg) }),
	);

	await page.route('**/api/v1/organizations/*/sso-group-mappings', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /sso-group-mappings/map-1
		if (/\/sso-group-mappings\/[^/?]+/.test(url)) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ mappings: mockSSOGroupMappings }) });
	});

	await page.route('**/api/v1/organizations/*/sso-settings', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockSSOSettings) }),
	);
}

/**
 * Mocks the Restore page APIs (snapshots, restores, agents, repositories, legal holds, comments).
 */
export async function mockRestoreAPIs(page: Page) {
	await page.route('**/api/v1/snapshots*', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /snapshots/snap-1/files or /snapshots/compare
		if (/\/snapshots\/[^/?]+/.test(url) && !url.includes('/snapshots?')) {
			// Handle comment endpoints
			if (url.includes('/comments')) {
				return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ comments: [] }) });
			}
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ snapshots: mockSnapshots }) });
	});

	await page.route('**/api/v1/restores*', (route) => {
		const url = route.request().url();
		if (/\/restores\/[^/?]+/.test(url) && !url.includes('/restores?')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ restores: mockRestoreJobs }) });
	});

	await page.route('**/api/v1/agents*', (route) => {
		const url = route.request().url();
		if (/\/agents\/[^/?]+/.test(url) && !url.endsWith('/agents')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) });
	});

	await page.route('**/api/v1/repositories*', (route) => {
		const url = route.request().url();
		if (/\/repositories\/[^/?]+/.test(url) && !url.endsWith('/repositories')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) });
	});

	await page.route('**/api/v1/legal-holds*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ legal_holds: [] }) }),
	);
}

/**
 * Mocks the Alerts page APIs.
 */
export async function mockAlertsAPIs(page: Page) {
	await page.route('**/api/v1/alerts', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /alerts/alert-1
		if (/\/alerts\/[^/?]+/.test(url) && !url.includes('/alerts?') && !url.includes('/alerts/count') && !url.includes('/alerts/active')) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ alerts: mockAlerts }) });
	});

	await page.route('**/api/v1/filters*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ filters: [] }) }),
	);
}

/**
 * Mocks the Reports page APIs.
 */
export async function mockReportsAPIs(page: Page) {
	await page.route('**/api/v1/reports/schedules*', (route) => {
		const url = route.request().url();
		if (/\/schedules\/[^/?]+/.test(url)) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockReportSchedules }) });
	});

	await page.route('**/api/v1/reports/history*', (route) => {
		const url = route.request().url();
		if (/\/history\/[^/?]+/.test(url)) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ history: mockReportHistory }) });
	});

	await page.route('**/api/v1/notifications/channels*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ channels: [] }) }),
	);
}

/**
 * Mocks the Audit Logs page APIs.
 */
export async function mockAuditLogsAPIs(page: Page) {
	await page.route('**/api/v1/audit-logs*', (route) => {
		const url = route.request().url();
		// Don't intercept export endpoints or detail endpoints
		if (url.includes('/export/')) {
			return route.continue();
		}
		if (/\/audit-logs\/[^/?]+/.test(url) && !url.includes('/audit-logs?')) {
			return route.continue();
		}
		return route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ audit_logs: mockAuditLogs, total_count: mockAuditLogs.length }),
		});
	});
}

// ---- Mock data for Setup, Onboarding, License, and Storage Stats pages ----

export const mockSetupStatusIncomplete = {
	needs_setup: true,
	setup_completed: false,
	current_step: 'database',
	completed_steps: [] as string[],
	database_ok: false,
	has_superuser: false,
};

export const mockSetupStatusComplete = {
	needs_setup: false,
	setup_completed: true,
	current_step: 'complete',
	completed_steps: ['database', 'superuser'],
	database_ok: true,
	has_superuser: true,
};

export const mockSetupStatusSuperuserStep = {
	needs_setup: true,
	setup_completed: false,
	current_step: 'superuser',
	completed_steps: ['database'],
	database_ok: true,
	has_superuser: false,
};

export const mockOnboardingStatusIncomplete = {
	needs_onboarding: true,
	current_step: 'welcome',
	completed_steps: [] as string[],
	skipped: false,
	is_complete: false,
	license_tier: 'free',
};

export const mockOnboardingStatusOrgStep = {
	needs_onboarding: true,
	current_step: 'organization',
	completed_steps: ['welcome', 'license'],
	skipped: false,
	is_complete: false,
	license_tier: 'free',
};

export const mockOnboardingStatusRepoStep = {
	needs_onboarding: true,
	current_step: 'repository',
	completed_steps: ['welcome', 'license', 'organization', 'oidc', 'smtp'],
	skipped: false,
	is_complete: false,
	license_tier: 'free',
};

export const mockOnboardingStatusAgentStep = {
	needs_onboarding: true,
	current_step: 'agent',
	completed_steps: ['welcome', 'license', 'organization', 'oidc', 'smtp', 'repository'],
	skipped: false,
	is_complete: false,
	license_tier: 'free',
};

export const mockLicenseInfo = {
	tier: 'free',
	customer_id: '',
	customer_name: '',
	company: '',
	expires_at: '',
	issued_at: '2024-01-01T00:00:00Z',
	features: [] as string[],
	limits: {
		max_agents: 3,
		max_servers: 1,
		max_users: 5,
		max_orgs: 1,
		max_storage_bytes: 0,
	},
	license_key_source: 'none' as const,
	is_trial: false,
};

export const mockLicenseInfoPro = {
	tier: 'pro',
	customer_id: 'cust-123',
	customer_name: 'Acme Corp',
	company: 'Acme Corp',
	expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
	issued_at: '2024-01-01T00:00:00Z',
	features: ['oidc', 'api_access', 'custom_reports', 'white_label', 'priority_support'],
	limits: {
		max_agents: 50,
		max_servers: 10,
		max_users: 100,
		max_orgs: 5,
		max_storage_bytes: 1099511627776,
	},
	license_key_source: 'database' as const,
	is_trial: false,
};

export const mockPricingPlans = [
	{
		id: 'plan-pro',
		product_id: 'prod-pro',
		name: 'Professional',
		base_price_cents: 4900,
		included_agents: 10,
		included_servers: 5,
		agent_price_cents: 500,
		server_price_cents: 1000,
	},
	{
		id: 'plan-ent',
		product_id: 'prod-ent',
		name: 'Enterprise',
		base_price_cents: 19900,
		included_agents: 50,
		included_servers: 25,
		agent_price_cents: 300,
		server_price_cents: 700,
	},
];

export const mockStorageSummary = {
	total_raw_size: 4294967296,
	total_restore_size: 10737418240,
	total_space_saved: 6442450944,
	avg_dedup_ratio: 2.5,
	repository_count: 2,
	total_snapshots: 50,
};

export const mockStorageGrowth = [
	{ date: '2024-06-01', raw_data_size: 1073741824, restore_size: 2684354560 },
	{ date: '2024-06-08', raw_data_size: 2147483648, restore_size: 5368709120 },
	{ date: '2024-06-15', raw_data_size: 3221225472, restore_size: 8053063680 },
	{ date: '2024-06-22', raw_data_size: 4294967296, restore_size: 10737418240 },
];

export const mockRepositoryStatsList = [
	{
		id: 'stat-1',
		repository_id: '1',
		repository_name: 'local-backups',
		total_size: 2147483648,
		total_file_count: 10000,
		raw_data_size: 2147483648,
		restore_size: 5368709120,
		dedup_ratio: 2.5,
		space_saved: 3221225472,
		space_saved_pct: 60,
		snapshot_count: 30,
		collected_at: '2024-06-22T00:00:00Z',
		created_at: '2024-06-22T00:00:00Z',
	},
	{
		id: 'stat-2',
		repository_id: '2',
		repository_name: 's3-archive',
		total_size: 2147483648,
		total_file_count: 5000,
		raw_data_size: 2147483648,
		restore_size: 5368709120,
		dedup_ratio: 2.5,
		space_saved: 3221225472,
		space_saved_pct: 60,
		snapshot_count: 20,
		collected_at: '2024-06-22T00:00:00Z',
		created_at: '2024-06-22T00:00:00Z',
	},
];

/**
 * Mocks the Setup page APIs (setup runs before auth exists).
 */
export async function mockSetupAPIs(page: Page, status = mockSetupStatusIncomplete) {
	await page.route('**/api/v1/setup/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(status) }),
	);

	await page.route('**/api/v1/setup/database/test', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true, message: 'Connection successful' }) }),
	);

	await page.route('**/api/v1/setup/superuser', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ message: 'Superuser created' }) }),
	);

	await page.route('**/api/v1/setup/complete', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ redirect: '/login' }) }),
	);

	// Branding (public, may be called even on setup page)
	await page.route('**/api/v1/branding', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ product_name: 'Keldris', logo_url: '', primary_color: '', secondary_color: '', custom_css: '', favicon_url: '' }) }),
	);
}

/**
 * Mocks the Onboarding page APIs.
 */
export async function mockOnboardingAPIs(page: Page, status = mockOnboardingStatusIncomplete) {
	// Override the onboarding status from mockAuthenticatedUser
	await page.route('**/api/v1/onboarding/status', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(status) }),
	);

	await page.route('**/api/v1/onboarding/step/*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(status) }),
	);

	await page.route('**/api/v1/onboarding/skip', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ...status, is_complete: true }) }),
	);

	// Organization step data
	await page.route('**/api/v1/agents', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ agents: mockAgents }) }),
	);

	await page.route('**/api/v1/repositories', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ repositories: mockRepositories }) }),
	);

	await page.route('**/api/v1/schedules', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedules: mockSchedules }) }),
	);

	await page.route('**/api/v1/backups', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ backups: mockBackups }) }),
	);
}

/**
 * Mocks the License page APIs.
 */
export async function mockLicenseAPIs(page: Page, licenseData = mockLicenseInfo) {
	// Override the license endpoint from mockAuthenticatedUser
	await page.route('**/api/v1/license', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(licenseData) }),
	);

	await page.route('**/api/v1/system/license/plans', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPricingPlans) }),
	);

	await page.route('**/api/v1/system/license/activate', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'activated', tier: 'pro' }) }),
	);

	await page.route('**/api/v1/system/license/deactivate', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'deactivated', tier: 'free' }) }),
	);

	await page.route('**/api/v1/system/license/trial/start', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'trial_started', tier: 'pro', expires_at: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString() }) }),
	);

	await page.route('**/api/v1/licenses/history*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ history: [], total_count: 0 }) }),
	);

	await page.route('**/api/v1/licenses/current', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ license: licenseData }) }),
	);
}

/**
 * Mocks the Storage Stats page APIs.
 */
export async function mockStorageStatsAPIs(page: Page) {
	await page.route('**/api/v1/stats/summary', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockStorageSummary) }),
	);

	await page.route('**/api/v1/stats/growth*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ growth: mockStorageGrowth }) }),
	);

	await page.route('**/api/v1/stats/repositories', (route) => {
		const url = route.request().url();
		// Don't intercept sub-paths like /stats/repositories/1
		if (/\/stats\/repositories\/[^/?]+/.test(url)) {
			return route.continue();
		}
		return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ stats: mockRepositoryStatsList }) });
	});
}
