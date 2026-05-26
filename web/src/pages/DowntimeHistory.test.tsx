import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useDowntime', () => ({
	useDowntimeEvents: vi.fn().mockReturnValue({
		data: [
			{
				id: 'd1',
				component_id: 'c1',
				component_type: 'agent',
				component_name: 'agent-1',
				severity: 'critical',
				cause: 'Network unreachable',
				started_at: '2024-01-01T00:00:00Z',
				ended_at: '2024-01-01T01:00:00Z',
				duration_seconds: 3600,
				notes: null,
			},
		],
		isLoading: false,
		isError: false,
	}),
	useActiveDowntime: vi.fn().mockReturnValue({
		data: [],
	}),
	useUptimeSummary: vi.fn().mockReturnValue({
		data: {
			overall_uptime_7d: 99.9,
			overall_uptime_30d: 99.5,
			overall_uptime_90d: 99.0,
			total_components: 5,
		},
	}),
	useMonthlyUptimeReport: vi.fn().mockReturnValue({
		data: undefined,
	}),
	useResolveDowntimeEvent: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

import { DowntimeHistory } from './DowntimeHistory';

describe('DowntimeHistory page', () => {
	it('renders the title', () => {
		renderWithProviders(<DowntimeHistory />);
		expect(screen.getByText('Downtime History')).toBeInTheDocument();
	});

	it('renders downtime events from data', () => {
		renderWithProviders(<DowntimeHistory />);
		expect(screen.getByText('agent-1')).toBeInTheDocument();
	});
});
