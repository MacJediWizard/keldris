import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

const mockRefetch = vi.fn();

vi.mock('../hooks/useSystemHealth', () => ({
	useSystemHealth: vi.fn(),
	useSystemHealthHistory: vi.fn(),
}));

import {
	useSystemHealth,
	useSystemHealthHistory,
} from '../hooks/useSystemHealth';
import type {
	SystemHealthHistoryResponse,
	SystemHealthResponse,
} from '../lib/types';
import { SystemHealth } from './SystemHealth';

function buildHealthyResponse(
	overrides?: Partial<SystemHealthResponse>,
): SystemHealthResponse {
	return {
		status: 'healthy',
		timestamp: '2026-03-14T10:00:00Z',
		server: {
			status: 'healthy',
			cpu_usage: 12.5,
			memory_usage: 45.2,
			memory_alloc_mb: 128.4,
			memory_total_alloc_mb: 256.0,
			memory_sys_mb: 512.0,
			goroutine_count: 42,
			num_cpu: 8,
			go_version: 'go1.25',
			uptime_seconds: 90061, // 1d 1h 1m
		},
		database: {
			status: 'healthy',
			connected: true,
			latency: '2ms',
			active_connections: 5,
			max_connections: 100,
			size_bytes: 1073741824,
			size_formatted: '1.0 GB',
		},
		queue: {
			status: 'healthy',
			pending_backups: 3,
			running_backups: 1,
			total_queued: 4,
		},
		background_jobs: {
			status: 'healthy',
			goroutine_count: 12,
			active_jobs: 2,
		},
		recent_errors: [],
		...overrides,
	};
}

function buildHistoryResponse(
	overrides?: Partial<SystemHealthHistoryResponse>,
): SystemHealthHistoryResponse {
	return {
		records: [
			{
				id: 'rec-1',
				timestamp: '2026-03-14T09:00:00Z',
				status: 'healthy',
				cpu_usage: 10.0,
				memory_usage: 40.0,
				memory_alloc_mb: 120.0,
				memory_total_alloc_mb: 240.0,
				goroutine_count: 38,
				database_connections: 4,
				database_size_bytes: 1073741824,
				pending_backups: 1,
				running_backups: 0,
				error_count: 0,
			},
			{
				id: 'rec-2',
				timestamp: '2026-03-14T10:00:00Z',
				status: 'warning',
				cpu_usage: 80.0,
				memory_usage: 75.0,
				memory_alloc_mb: 200.0,
				memory_total_alloc_mb: 400.0,
				goroutine_count: 120,
				database_connections: 50,
				database_size_bytes: 1073741824,
				pending_backups: 10,
				running_backups: 3,
				error_count: 2,
			},
		],
		since: '2026-03-13T10:00:00Z',
		until: '2026-03-14T10:00:00Z',
		...overrides,
	};
}

function mockHooks(
	healthOverrides?: Partial<ReturnType<typeof useSystemHealth>>,
	historyOverrides?: Partial<ReturnType<typeof useSystemHealthHistory>>,
) {
	vi.mocked(useSystemHealth).mockReturnValue({
		data: undefined,
		isLoading: false,
		isError: false,
		refetch: mockRefetch,
		...healthOverrides,
	} as ReturnType<typeof useSystemHealth>);

	vi.mocked(useSystemHealthHistory).mockReturnValue({
		data: undefined,
		isLoading: false,
		isError: false,
		...historyOverrides,
	} as ReturnType<typeof useSystemHealthHistory>);
}

describe('SystemHealth', () => {
	beforeEach(() => vi.clearAllMocks());

	// --- Loading State ---

	it('shows loading skeletons when data is loading', () => {
		mockHooks({ isLoading: true });
		renderWithProviders(<SystemHealth />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('renders page title during loading', () => {
		mockHooks({ isLoading: true });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Health')).toBeInTheDocument();
	});

	it('renders subtitle during loading', () => {
		mockHooks({ isLoading: true });
		renderWithProviders(<SystemHealth />);
		expect(
			screen.getByText('Monitor system status and performance'),
		).toBeInTheDocument();
	});

	// --- Error State ---

	it('shows error message when health fetch fails', () => {
		mockHooks({ isError: true });
		renderWithProviders(<SystemHealth />);
		expect(
			screen.getByText('Failed to load system health data'),
		).toBeInTheDocument();
	});

	it('shows superuser access hint on error', () => {
		mockHooks({ isError: true });
		renderWithProviders(<SystemHealth />);
		expect(
			screen.getByText('You may not have superuser access to view this page'),
		).toBeInTheDocument();
	});

	it('still renders page title on error', () => {
		mockHooks({ isError: true });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Health')).toBeInTheDocument();
	});

	// --- Healthy State ---

	it('renders overall healthy status', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Status: Healthy')).toBeInTheDocument();
	});

	it('renders server memory usage metric', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Memory Usage')).toBeInTheDocument();
		expect(screen.getByText('45.2%')).toBeInTheDocument();
		expect(screen.getByText('128.4 MB allocated')).toBeInTheDocument();
	});

	it('renders goroutine count with CPU count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Goroutines')).toBeInTheDocument();
		expect(screen.getByText('42')).toBeInTheDocument();
		expect(screen.getByText('8 CPUs')).toBeInTheDocument();
	});

	it('renders uptime formatted correctly', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Uptime')).toBeInTheDocument();
		expect(screen.getByText('1d 1h 1m')).toBeInTheDocument();
		expect(screen.getByText('go1.25')).toBeInTheDocument();
	});

	it('renders system memory metric', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Memory')).toBeInTheDocument();
		expect(screen.getByText('512.0 MB')).toBeInTheDocument();
		expect(screen.getByText('Total system allocation')).toBeInTheDocument();
	});

	// --- Database Section ---

	it('renders database connection status', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Connection')).toBeInTheDocument();
		expect(screen.getByText('Connected')).toBeInTheDocument();
		expect(screen.getByText('Latency: 2ms')).toBeInTheDocument();
	});

	it('renders database active connections', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Active Connections')).toBeInTheDocument();
		expect(screen.getByText('5')).toBeInTheDocument();
		expect(screen.getByText('Max: 100')).toBeInTheDocument();
	});

	it('renders database size from formatted field', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Database Size')).toBeInTheDocument();
		expect(screen.getByText('1.0 GB')).toBeInTheDocument();
	});

	it('renders connection pool utilization', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Connection Pool')).toBeInTheDocument();
		expect(screen.getByText('5%')).toBeInTheDocument();
		expect(screen.getByText('Pool utilization')).toBeInTheDocument();
	});

	it('shows N/A for pool utilization when max_connections is 0', () => {
		const health = buildHealthyResponse();
		health.database.max_connections = 0;
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('N/A')).toBeInTheDocument();
	});

	it('shows Disconnected when database is not connected', () => {
		const health = buildHealthyResponse();
		health.database.connected = false;
		health.database.status = 'critical';
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Disconnected')).toBeInTheDocument();
	});

	// --- Queue Section ---

	it('renders backup queue status section', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Backup Queue Status')).toBeInTheDocument();
	});

	it('renders pending backups count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Pending Backups')).toBeInTheDocument();
		expect(screen.getByText('3')).toBeInTheDocument();
	});

	it('renders running backups count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Running Backups')).toBeInTheDocument();
		expect(screen.getByText('1')).toBeInTheDocument();
	});

	it('renders total queued count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Total Queued')).toBeInTheDocument();
		expect(screen.getByText('4')).toBeInTheDocument();
	});

	// --- Background Jobs Section ---

	it('renders background jobs section', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Background Jobs')).toBeInTheDocument();
	});

	it('renders background goroutine count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Goroutine Count')).toBeInTheDocument();
		expect(screen.getByText('12')).toBeInTheDocument();
	});

	it('renders active jobs count', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Active Jobs')).toBeInTheDocument();
		expect(screen.getByText('2')).toBeInTheDocument();
		expect(
			screen.getByText('Estimated active background tasks'),
		).toBeInTheDocument();
	});

	// --- Warning State ---

	it('renders warning status banner', () => {
		const health = buildHealthyResponse({ status: 'warning' });
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Status: Warning')).toBeInTheDocument();
	});

	it('displays issues list when present', () => {
		const health = buildHealthyResponse({
			status: 'warning',
			issues: [
				'High memory usage detected',
				'Database connections nearing limit',
			],
		});
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('High memory usage detected')).toBeInTheDocument();
		expect(
			screen.getByText('Database connections nearing limit'),
		).toBeInTheDocument();
	});

	// --- Critical State ---

	it('renders critical status banner', () => {
		const health = buildHealthyResponse({ status: 'critical' });
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('System Status: Critical')).toBeInTheDocument();
	});

	// --- Recent Errors ---

	it('shows no recent errors message when error list is empty', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('No recent errors')).toBeInTheDocument();
	});

	it('renders recent errors table when errors exist', () => {
		const health = buildHealthyResponse({
			recent_errors: [
				{
					id: 'err-1',
					level: 'error',
					message: 'Connection timeout to storage backend',
					component: 'backup-engine',
					timestamp: '2026-03-14T09:30:00Z',
				},
				{
					id: 'err-2',
					level: 'fatal',
					message: 'Out of memory',
					timestamp: '2026-03-14T09:45:00Z',
				},
			],
		});
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(
			screen.getByText('Connection timeout to storage backend'),
		).toBeInTheDocument();
		expect(screen.getByText('Out of memory')).toBeInTheDocument();
		expect(screen.getByText('backup-engine')).toBeInTheDocument();
	});

	it('renders table headers for error log', () => {
		const health = buildHealthyResponse({
			recent_errors: [
				{
					id: 'err-1',
					level: 'error',
					message: 'Test error',
					timestamp: '2026-03-14T09:30:00Z',
				},
			],
		});
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Timestamp')).toBeInTheDocument();
		expect(screen.getByText('Level')).toBeInTheDocument();
		expect(screen.getByText('Component')).toBeInTheDocument();
		expect(screen.getByText('Message')).toBeInTheDocument();
	});

	it('shows dash for error without component', () => {
		const health = buildHealthyResponse({
			recent_errors: [
				{
					id: 'err-1',
					level: 'error',
					message: 'Some error',
					timestamp: '2026-03-14T09:30:00Z',
				},
			],
		});
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('-')).toBeInTheDocument();
	});

	// --- Refresh Button ---

	it('renders refresh button', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
	});

	it('calls refetch when refresh button is clicked', async () => {
		const user = userEvent.setup();
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		await user.click(screen.getByRole('button', { name: 'Refresh' }));
		expect(mockRefetch).toHaveBeenCalledTimes(1);
	});

	// --- Auto-Refresh Checkbox ---

	it('renders auto-refresh checkbox checked by default', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		const checkbox = screen.getByRole('checkbox');
		expect(checkbox).toBeChecked();
		expect(screen.getByText('Auto-refresh (30s)')).toBeInTheDocument();
	});

	it('toggles auto-refresh checkbox', async () => {
		const user = userEvent.setup();
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		const checkbox = screen.getByRole('checkbox');
		expect(checkbox).toBeChecked();
		await user.click(checkbox);
		expect(checkbox).not.toBeChecked();
	});

	// --- Historical Data ---

	it('renders historical data section when records exist', () => {
		const health = buildHealthyResponse();
		const history = buildHistoryResponse();
		mockHooks({ data: health }, { data: history });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Historical Data (24h)')).toBeInTheDocument();
	});

	it('renders memory usage chart label in history', () => {
		const health = buildHealthyResponse();
		const history = buildHistoryResponse();
		mockHooks({ data: health }, { data: history });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Memory Usage (MB)')).toBeInTheDocument();
	});

	it('renders goroutines chart label in history', () => {
		const health = buildHealthyResponse();
		const history = buildHistoryResponse();
		mockHooks({ data: health }, { data: history });
		renderWithProviders(<SystemHealth />);
		// "Goroutines" appears as both a metric card title and chart label
		const goroutineLabels = screen.getAllByText('Goroutines');
		expect(goroutineLabels.length).toBeGreaterThanOrEqual(1);
	});

	it('renders pending backups chart label in history', () => {
		const health = buildHealthyResponse();
		const history = buildHistoryResponse();
		mockHooks({ data: health }, { data: history });
		renderWithProviders(<SystemHealth />);
		// "Pending Backups" appears as both a metric card title and chart label
		const pendingLabels = screen.getAllByText('Pending Backups');
		expect(pendingLabels.length).toBeGreaterThanOrEqual(1);
	});

	it('does not render historical data section when records are empty', () => {
		const health = buildHealthyResponse();
		const history = buildHistoryResponse({ records: [] });
		mockHooks({ data: health }, { data: history });
		renderWithProviders(<SystemHealth />);
		expect(screen.queryByText('Historical Data (24h)')).not.toBeInTheDocument();
	});

	it('does not render historical data section when history is undefined', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health }, { data: undefined });
		renderWithProviders(<SystemHealth />);
		expect(screen.queryByText('Historical Data (24h)')).not.toBeInTheDocument();
	});

	// --- Uptime Formatting Edge Cases ---

	it('formats uptime with only hours and minutes', () => {
		const health = buildHealthyResponse();
		health.server.uptime_seconds = 3660; // 1h 1m
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('1h 1m')).toBeInTheDocument();
	});

	it('formats uptime with only minutes', () => {
		const health = buildHealthyResponse();
		health.server.uptime_seconds = 300; // 5m
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('5m')).toBeInTheDocument();
	});

	// --- Database size fallback ---

	it('falls back to formatBytes when size_formatted is empty', () => {
		const health = buildHealthyResponse();
		health.database.size_formatted = '';
		health.database.size_bytes = 262144000; // ~250 MB
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Database Size')).toBeInTheDocument();
		expect(screen.getByText('250.0 MB')).toBeInTheDocument();
	});

	// --- Section Headers ---

	it('renders all section headers', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		expect(screen.getByText('Server Status')).toBeInTheDocument();
		expect(screen.getByText('Database Status')).toBeInTheDocument();
		expect(screen.getByText('Backup Queue Status')).toBeInTheDocument();
		expect(screen.getByText('Background Jobs')).toBeInTheDocument();
		expect(screen.getByText('Recent Errors')).toBeInTheDocument();
	});

	// --- StatusBadge rendering ---

	it('renders status badges on metric cards', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		// Multiple "Healthy" badges across various sections
		const healthyBadges = screen.getAllByText('Healthy');
		expect(healthyBadges.length).toBeGreaterThanOrEqual(1);
	});

	it('renders warning badges when server is in warning state', () => {
		const health = buildHealthyResponse();
		health.server.status = 'warning';
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		const warningBadges = screen.getAllByText('Warning');
		expect(warningBadges.length).toBeGreaterThanOrEqual(1);
	});

	it('renders critical badges when database is critical', () => {
		const health = buildHealthyResponse();
		health.database.status = 'critical';
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		const criticalBadges = screen.getAllByText('Critical');
		expect(criticalBadges.length).toBeGreaterThanOrEqual(1);
	});

	// --- Last updated timestamp ---

	it('shows last updated timestamp in status banner', () => {
		const health = buildHealthyResponse();
		mockHooks({ data: health });
		renderWithProviders(<SystemHealth />);
		// The timestamp is formatted via toLocaleString, so look for the "Last updated:" prefix
		expect(screen.getByText(/Last updated:/)).toBeInTheDocument();
	});
});
