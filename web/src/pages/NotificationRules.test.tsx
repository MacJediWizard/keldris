import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useNotificationRules', () => ({
	useNotificationRules: vi.fn(),
	useCreateNotificationRule: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateNotificationRule: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteNotificationRule: () => ({
		mutate: vi.fn(),
		isPending: false,
	}),
	useTestNotificationRule: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../hooks/useNotifications', () => ({
	useNotificationChannels: () => ({ data: [], isLoading: false }),
}));

import { useNotificationRules } from '../hooks/useNotificationRules';
import { NotificationRules } from './NotificationRules';

describe('NotificationRules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		expect(screen.getByText('Notification Rules')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		expect(
			screen.getByText(
				'Create rules to escalate notifications based on conditions',
			),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		expect(
			screen.getByText('Failed to load notification rules'),
		).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		expect(
			screen.getByText('No notification rules configured'),
		).toBeInTheDocument();
	});

	it('renders rules list', () => {
		vi.mocked(useNotificationRules).mockReturnValue({
			data: [
				{
					id: 'rule-1',
					name: 'Critical Backup Failure',
					description: 'Alert on 3 backup failures',
					trigger_type: 'backup_failed',
					enabled: true,
					priority: 0,
					conditions: { count: 3, time_window_minutes: 60 },
					actions: [{ type: 'notify_channel', channel_id: 'ch-1' }],
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationRules>);
		renderWithProviders(<NotificationRules />);
		expect(screen.getByText('Critical Backup Failure')).toBeInTheDocument();
		expect(screen.getByText('Alert on 3 backup failures')).toBeInTheDocument();
	});
});
