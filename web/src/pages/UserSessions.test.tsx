import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useUserSessions', () => ({
	useUserSessions: vi.fn(),
	useRevokeSession: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRevokeAllSessions: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
}));

import { useUserSessions } from '../hooks/useUserSessions';
import { UserSessions } from './UserSessions';

describe('UserSessions', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		expect(screen.getByText('Active Sessions')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		expect(
			screen.getByText(
				'View and manage your active sessions across all devices',
			),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		expect(
			screen.getByText('Failed to load sessions. Please try again.'),
		).toBeInTheDocument();
	});

	it('shows current session', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: [
				{
					id: 'sess-1',
					user_id: 'user-1',
					ip_address: '192.168.1.1',
					user_agent:
						'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0.0.0',
					is_current: true,
					last_active_at: '2024-01-01T12:00:00Z',
					created_at: '2024-01-01T10:00:00Z',
					expires_at: '2024-01-02T10:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		expect(screen.getByText('This Device')).toBeInTheDocument();
		expect(screen.getByText('Current session')).toBeInTheDocument();
	});

	it('renders other sessions section', () => {
		vi.mocked(useUserSessions).mockReturnValue({
			data: [
				{
					id: 'sess-1',
					user_id: 'user-1',
					ip_address: '192.168.1.1',
					user_agent: 'Mozilla/5.0 Chrome/120',
					is_current: true,
					last_active_at: '2024-01-01T12:00:00Z',
					created_at: '2024-01-01T10:00:00Z',
					expires_at: '2024-01-02T10:00:00Z',
				},
				{
					id: 'sess-2',
					user_id: 'user-1',
					ip_address: '10.0.0.1',
					user_agent: 'Mozilla/5.0 Firefox/115',
					is_current: false,
					last_active_at: '2024-01-01T11:00:00Z',
					created_at: '2024-01-01T09:00:00Z',
					expires_at: '2024-01-02T09:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useUserSessions>);
		renderWithProviders(<UserSessions />);
		expect(screen.getByText('Other Sessions (1)')).toBeInTheDocument();
		expect(screen.getByText('Revoke All Other Sessions')).toBeInTheDocument();
	});
});
