import type { Page } from '@playwright/test';
import {
	mockAgents,
	mockBackups,
	mockDashboardStats,
	mockRepositories,
	mockSchedules,
} from './auth';

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
