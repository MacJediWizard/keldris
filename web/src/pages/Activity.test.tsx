import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useActivity', () => ({
	useRecentActivity: vi.fn(),
	useActivityFeed: vi.fn().mockReturnValue({
		events: [],
		isConnected: false,
		error: null,
		clearEvents: vi.fn(),
	}),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string) => key,
		formatRelativeTime: (d: string) => d || 'N/A',
		formatDate: (d: string) => d || 'N/A',
		formatDateTime: (d: string) => d || 'N/A',
		formatNumber: (v: number) => String(v),
		formatBytes: (v: number) => `${v} B`,
		formatPercent: (v: number) => `${v}%`,
		formatDuration: () => '0s',
		language: 'en',
		setLanguage: vi.fn(),
	}),
}));

import { useActivityFeed, useRecentActivity } from '../hooks/useActivity';
import { Activity } from './Activity';

const mockEvents = [
	{
		id: 'evt_1',
		org_id: 'org_1',
		type: 'backup_completed' as const,
		category: 'backup' as const,
		title: 'Backup completed',
		description: 'Daily backup of production database finished',
		agent_name: 'prod-server-01',
		user_name: 'admin',
		created_at: '2024-06-15T10:30:00Z',
	},
	{
		id: 'evt_2',
		org_id: 'org_1',
		type: 'agent_connected' as const,
		category: 'agent' as const,
		title: 'Agent connected',
		description: 'Agent prod-server-02 came online',
		agent_name: 'prod-server-02',
		created_at: '2024-06-15T09:15:00Z',
	},
	{
		id: 'evt_3',
		org_id: 'org_1',
		type: 'alert_triggered' as const,
		category: 'alert' as const,
		title: 'Storage alert triggered',
		description: 'Disk usage exceeded 90%',
		created_at: '2024-06-15T08:00:00Z',
	},
];

function setRecentActivityMock(
	events?: typeof mockEvents,
	loading = false,
	fetching = false,
) {
	vi.mocked(useRecentActivity).mockReturnValue({
		data: events,
		isLoading: loading,
		isFetching: fetching,
		isError: false,
		error: null,
	} as ReturnType<typeof useRecentActivity>);
}

describe('Activity page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		setRecentActivityMock(mockEvents);
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [],
			isConnected: false,
			error: null,
			clearEvents: vi.fn(),
		});
	});

	it('renders the page title', () => {
		renderWithProviders(<Activity />);
		expect(screen.getByText('Activity Feed')).toBeInTheDocument();
	});

	it('shows loading skeleton when data is loading', () => {
		setRecentActivityMock(undefined, true);
		renderWithProviders(<Activity />);
		const pulses = document.querySelectorAll('.animate-pulse');
		expect(pulses.length).toBeGreaterThan(0);
	});

	it('shows empty state when no events exist', () => {
		setRecentActivityMock([]);
		renderWithProviders(<Activity />);
		expect(screen.getByText('No activity found')).toBeInTheDocument();
		expect(
			screen.getByText('Events will appear here as they happen'),
		).toBeInTheDocument();
	});

	it('renders activity events with titles and descriptions', () => {
		renderWithProviders(<Activity />);
		expect(screen.getByText('Backup completed')).toBeInTheDocument();
		expect(
			screen.getByText('Daily backup of production database finished'),
		).toBeInTheDocument();
		expect(screen.getByText('Agent connected')).toBeInTheDocument();
		expect(
			screen.getByText('Agent prod-server-02 came online'),
		).toBeInTheDocument();
		expect(screen.getByText('Storage alert triggered')).toBeInTheDocument();
		expect(screen.getByText('Disk usage exceeded 90%')).toBeInTheDocument();
	});

	it('displays agent names in event items', () => {
		renderWithProviders(<Activity />);
		expect(screen.getByText('prod-server-01')).toBeInTheDocument();
		expect(screen.getByText('prod-server-02')).toBeInTheDocument();
	});

	it('displays user names in event items', () => {
		renderWithProviders(<Activity />);
		expect(screen.getByText('admin')).toBeInTheDocument();
	});

	it('renders category filter pills', () => {
		renderWithProviders(<Activity />);
		expect(screen.getByText('All')).toBeInTheDocument();
		expect(screen.getByText('Backups')).toBeInTheDocument();
		expect(screen.getByText('Restores')).toBeInTheDocument();
		expect(screen.getByText('Agents')).toBeInTheDocument();
		expect(screen.getByText('Users')).toBeInTheDocument();
		expect(screen.getByText('Alerts')).toBeInTheDocument();
		expect(screen.getByText('Schedules')).toBeInTheDocument();
		expect(screen.getByText('Repositories')).toBeInTheDocument();
		expect(screen.getByText('Maintenance')).toBeInTheDocument();
		expect(screen.getByText('System')).toBeInTheDocument();
	});

	it('filters events by category when a filter pill is clicked', async () => {
		const user = userEvent.setup();
		renderWithProviders(<Activity />);

		// Click the "Agents" category filter
		await user.click(screen.getByText('Agents'));

		// Only agent events should be visible (backup and alert filtered out)
		expect(screen.getByText('Agent connected')).toBeInTheDocument();
		expect(screen.queryByText('Backup completed')).not.toBeInTheDocument();
		expect(
			screen.queryByText('Storage alert triggered'),
		).not.toBeInTheDocument();
	});

	it('shows category-specific empty message when filter has no matches', async () => {
		const user = userEvent.setup();
		renderWithProviders(<Activity />);

		// Click "Restores" — no restore events in our mock data
		await user.click(screen.getByText('Restores'));

		expect(screen.getByText('No activity found')).toBeInTheDocument();
		expect(screen.getByText('No restore events found')).toBeInTheDocument();
	});

	it("returns to showing all events when 'All' filter is clicked after filtering", async () => {
		const user = userEvent.setup();
		renderWithProviders(<Activity />);

		// Filter to Agents
		await user.click(screen.getByText('Agents'));
		expect(screen.queryByText('Backup completed')).not.toBeInTheDocument();

		// Click All to show everything again
		await user.click(screen.getByText('All'));
		expect(screen.getByText('Backup completed')).toBeInTheDocument();
		expect(screen.getByText('Agent connected')).toBeInTheDocument();
		expect(screen.getByText('Storage alert triggered')).toBeInTheDocument();
	});

	it('shows live indicator when connected via websocket', () => {
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [],
			isConnected: true,
			error: null,
			clearEvents: vi.fn(),
		});
		renderWithProviders(<Activity />);
		expect(screen.getByText('Live')).toBeInTheDocument();
	});

	it("shows 'Connecting...' when websocket is not connected", () => {
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [],
			isConnected: false,
			error: null,
			clearEvents: vi.fn(),
		});
		renderWithProviders(<Activity />);
		expect(screen.getByText('Connecting...')).toBeInTheDocument();
	});

	it("shows 'Clear new events' button when live events exist", () => {
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [mockEvents[0]],
			isConnected: true,
			error: null,
			clearEvents: vi.fn(),
		});
		renderWithProviders(<Activity />);
		expect(screen.getByText('Clear new events')).toBeInTheDocument();
	});

	it("calls clearEvents when 'Clear new events' button is clicked", async () => {
		const user = userEvent.setup();
		const mockClearEvents = vi.fn();
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [mockEvents[0]],
			isConnected: true,
			error: null,
			clearEvents: mockClearEvents,
		});
		renderWithProviders(<Activity />);

		await user.click(screen.getByText('Clear new events'));
		expect(mockClearEvents).toHaveBeenCalled();
	});

	it('merges live events with recent events without duplicates', () => {
		const liveEvent = {
			id: 'evt_live',
			org_id: 'org_1',
			type: 'backup_started' as const,
			category: 'backup' as const,
			title: 'Backup started',
			description: 'Starting daily backup',
			created_at: '2024-06-15T11:00:00Z',
		};
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [liveEvent],
			isConnected: true,
			error: null,
			clearEvents: vi.fn(),
		});
		renderWithProviders(<Activity />);

		// Both live and recent events should be visible
		expect(screen.getByText('Backup started')).toBeInTheDocument();
		expect(screen.getByText('Backup completed')).toBeInTheDocument();
	});

	it('deduplicates events when live and recent share the same id', () => {
		// Return a live event with the same id as a recent event
		vi.mocked(useActivityFeed).mockReturnValue({
			events: [
				{
					...mockEvents[0],
					title: 'Backup completed (live)',
				},
			],
			isConnected: true,
			error: null,
			clearEvents: vi.fn(),
		});
		renderWithProviders(<Activity />);

		// The live version should appear (it comes first in the merge), not the recent
		expect(screen.getByText('Backup completed (live)')).toBeInTheDocument();
		expect(screen.queryByText(/^Backup completed$/)).not.toBeInTheDocument();
	});

	it('shows refreshing indicator when refetching', () => {
		setRecentActivityMock(mockEvents, false, true);
		renderWithProviders(<Activity />);
		expect(screen.getByText('Refreshing...')).toBeInTheDocument();
	});
});
