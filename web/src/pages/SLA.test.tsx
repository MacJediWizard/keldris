import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useSLA', () => ({
	useSLAs: vi.fn(),
	useSLADashboard: () => ({ data: undefined }),
	useActiveSLABreaches: () => ({ data: [] }),
	useCreateSLA: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateSLA: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteSLA: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useAcknowledgeBreach: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useResolveBreach: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { useSLAs } from '../hooks/useSLA';
import { SLA } from './SLA';

describe('SLA page', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows access denied for non-admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useSLAs).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAs>);
		renderWithProviders(<SLA />);
		expect(
			screen.getByText('Only administrators can manage SLAs.'),
		).toBeInTheDocument();
	});

	it('renders title and empty state for admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useSLAs).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAs>);
		renderWithProviders(<SLA />);
		expect(screen.getByText('Service Level Agreements')).toBeInTheDocument();
		expect(screen.getByText('No SLAs')).toBeInTheDocument();
	});

	it('renders SLA entries from data', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'owner' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useSLAs).mockReturnValue({
			data: [
				{
					id: 'sla-1',
					name: 'Production Backup SLA',
					description: 'Critical systems',
					rpo_minutes: 60,
					rto_minutes: 30,
					uptime_percentage: 99.9,
					scope: 'agent',
					active: true,
					agent_count: 5,
					repository_count: 2,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAs>);
		renderWithProviders(<SLA />);
		expect(screen.getByText('Production Backup SLA')).toBeInTheDocument();
	});
});
