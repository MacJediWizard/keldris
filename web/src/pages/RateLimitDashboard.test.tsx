import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useRateLimits', () => ({
	useRateLimitDashboard: vi.fn(),
}));

import { useRateLimitDashboard } from '../hooks/useRateLimits';
import { RateLimitDashboard } from './RateLimitDashboard';

describe('RateLimitDashboard', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useRateLimitDashboard).mockReturnValue({
			stats: undefined,
			isLoading: false,
			error: null,
			refresh: vi.fn(),
		} as ReturnType<typeof useRateLimitDashboard>);
		renderWithProviders(<RateLimitDashboard />);
		expect(screen.getByText('Rate Limit Dashboard')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		vi.mocked(useRateLimitDashboard).mockReturnValue({
			stats: undefined,
			isLoading: false,
			error: null,
			refresh: vi.fn(),
		} as ReturnType<typeof useRateLimitDashboard>);
		renderWithProviders(<RateLimitDashboard />);
		expect(
			screen.getByText('Monitor API rate limiting and client statistics'),
		).toBeInTheDocument();
	});

	it('shows error state', () => {
		vi.mocked(useRateLimitDashboard).mockReturnValue({
			stats: undefined,
			isLoading: false,
			error: new Error('forbidden'),
			refresh: vi.fn(),
		} as unknown as ReturnType<typeof useRateLimitDashboard>);
		renderWithProviders(<RateLimitDashboard />);
		expect(
			screen.getByText('Failed to load rate limit data'),
		).toBeInTheDocument();
	});

	it('shows loading skeletons', () => {
		vi.mocked(useRateLimitDashboard).mockReturnValue({
			stats: undefined,
			isLoading: true,
			error: null,
			refresh: vi.fn(),
		} as ReturnType<typeof useRateLimitDashboard>);
		renderWithProviders(<RateLimitDashboard />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('renders stats when data is available', () => {
		vi.mocked(useRateLimitDashboard).mockReturnValue({
			stats: {
				default_limit: 100,
				default_period: '1m',
				total_requests: 1000,
				total_rejected: 5,
				endpoint_configs: [],
				client_stats: [],
			},
			isLoading: false,
			error: null,
			refresh: vi.fn(),
		} as unknown as ReturnType<typeof useRateLimitDashboard>);
		renderWithProviders(<RateLimitDashboard />);
		expect(screen.getByText('Total Requests')).toBeInTheDocument();
		expect(screen.getByText('1,000')).toBeInTheDocument();
	});
});
