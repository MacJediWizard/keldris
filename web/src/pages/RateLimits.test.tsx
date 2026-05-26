import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useRateLimits', () => ({
	useRateLimitConfigs: vi.fn(),
	useRateLimitStats: vi.fn(),
	useBlockedRequests: () => ({ data: { blocked_requests: [] } }),
	useIPBans: () => ({ data: [] }),
	useCreateRateLimitConfig: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateRateLimitConfig: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteRateLimitConfig: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useCreateIPBan: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteIPBan: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { useRateLimitConfigs, useRateLimitStats } from '../hooks/useRateLimits';
import { RateLimits } from './RateLimits';

describe('RateLimits page', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows access denied for non-admin role', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useRateLimitConfigs).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useRateLimitConfigs>);
		vi.mocked(useRateLimitStats).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useRateLimitStats>);
		renderWithProviders(<RateLimits />);
		expect(
			screen.getByText('Only administrators can manage rate limits.'),
		).toBeInTheDocument();
	});

	it('renders title for admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useRateLimitConfigs).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useRateLimitConfigs>);
		vi.mocked(useRateLimitStats).mockReturnValue({
			data: { stats: { blocked_today: 0 } },
			isLoading: false,
		} as ReturnType<typeof useRateLimitStats>);
		renderWithProviders(<RateLimits />);
		expect(screen.getByText('Rate Limits')).toBeInTheDocument();
		expect(screen.getByText('Rate Limit Configurations')).toBeInTheDocument();
	});

	it('renders rate limit config from data', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'owner' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useRateLimitConfigs).mockReturnValue({
			data: [
				{
					id: 'cfg-1',
					endpoint: '/api/v1/agents',
					requests_per_period: 100,
					period_seconds: 60,
					enabled: true,
				},
			],
			isLoading: false,
		} as ReturnType<typeof useRateLimitConfigs>);
		vi.mocked(useRateLimitStats).mockReturnValue({
			data: { stats: { blocked_today: 5 } },
			isLoading: false,
		} as ReturnType<typeof useRateLimitStats>);
		renderWithProviders(<RateLimits />);
		expect(screen.getByText('/api/v1/agents')).toBeInTheDocument();
	});
});
