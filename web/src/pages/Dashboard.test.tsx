import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [
			{ id: '1', hostname: 'server-1', status: 'active' },
			{ id: '2', hostname: 'server-2', status: 'offline' },
		],
		isLoading: false,
		error: null,
	}),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: vi.fn().mockReturnValue({
		data: [{ id: 'repo-1', name: 'Local Backup', type: 'local' }],
		isLoading: false,
		error: null,
	}),
}));

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: vi.fn().mockReturnValue({
		data: [{ id: 'sched-1', name: 'Daily Backup', enabled: true }],
		isLoading: false,
		error: null,
	}),
}));

vi.mock('../hooks/useBackups', () => ({
	useBackups: vi.fn().mockReturnValue({
		data: [
			{
				id: 'backup-1',
				status: 'completed',
				started_at: '2024-06-15T12:00:00Z',
				completed_at: '2024-06-15T12:30:00Z',
				snapshot_id: 'abc12345',
			},
		],
		isLoading: false,
		error: null,
	}),
}));

vi.mock('../hooks/useStorageStats', () => ({
	useStorageStatsSummary: vi.fn().mockReturnValue({
		data: {
			total_size: 1024 * 1024 * 1024,
			total_dedup_size: 512 * 1024 * 1024,
			dedup_ratio: 2.0,
			space_saved_percent: 50,
		},
		isLoading: false,
	}),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string) => {
			const translations: Record<string, string> = {
				'dashboard.title': 'Dashboard',
				'dashboard.subtitle': 'Overview of your backup infrastructure',
				'dashboard.activeAgents': 'Active Agents',
				'dashboard.connectedAgents': 'Connected agents',
				'dashboard.repositories': 'Repositories',
				'dashboard.configuredRepos': 'Configured repos',
				'dashboard.enabledSchedules': 'Enabled Schedules',
				'dashboard.automatedBackups': 'Automated backups',
				'dashboard.totalBackups': 'Total Backups',
				'dashboard.allTimeBackups': 'All time backups',
				'dashboard.recentBackups': 'Recent Backups',
				'dashboard.noRecentBackups': 'No recent backups',
				'dashboard.viewAll': 'View All',
				'dashboard.storageOverview': 'Storage Overview',
			};
			return translations[key] || key;
		},
		formatRelativeTime: (d: string) => d || 'Never',
		formatBytes: (b: number) => `${b} B`,
		formatPercent: (p: number) => `${p}%`,
	}),
}));

// Import after mocks
import { Dashboard } from './Dashboard';

describe('Dashboard', () => {
	it('renders the dashboard title', () => {
		renderWithProviders(<Dashboard />);
		expect(screen.getByText('Dashboard')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		renderWithProviders(<Dashboard />);
		expect(
			screen.getByText('Overview of your backup infrastructure'),
		).toBeInTheDocument();
	});

	it('renders stat cards', () => {
		renderWithProviders(<Dashboard />);
		expect(screen.getByText('Active Agents')).toBeInTheDocument();
		expect(screen.getByText('Repositories')).toBeInTheDocument();
	});

	it('shows stat values', () => {
		renderWithProviders(<Dashboard />);
		// Multiple stat cards show "1" (1 active agent, 1 repo, 1 schedule, 1 backup)
		const statValues = screen.getAllByText('1');
		expect(statValues.length).toBeGreaterThanOrEqual(1);
	});
});
