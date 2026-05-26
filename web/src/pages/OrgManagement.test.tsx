import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useAdminOrganizations', () => ({
	useAdminOrganizations: vi.fn(),
	useAdminOrgUsageStats: () => ({ data: undefined, isLoading: false }),
	useAdminCreateOrganization: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useAdminUpdateOrganization: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useAdminDeleteOrganization: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useAdminTransferOwnership: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
}));

import { useAdminOrganizations } from '../hooks/useAdminOrganizations';
import { useMe } from '../hooks/useAuth';
import { OrgManagement } from './OrgManagement';

describe('OrgManagement page', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows access denied for non-superuser', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { is_superuser: false },
			isLoading: false,
		} as ReturnType<typeof useMe>);
		vi.mocked(useAdminOrganizations).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAdminOrganizations>);
		renderWithProviders(<OrgManagement />);
		expect(screen.getByText('Access Denied')).toBeInTheDocument();
	});

	it('renders title for superuser', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { is_superuser: true },
			isLoading: false,
		} as ReturnType<typeof useMe>);
		vi.mocked(useAdminOrganizations).mockReturnValue({
			data: { organizations: [], total_count: 0 },
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAdminOrganizations>);
		renderWithProviders(<OrgManagement />);
		expect(screen.getByText('Organization Management')).toBeInTheDocument();
	});

	it('renders organization rows from data', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { is_superuser: true },
			isLoading: false,
		} as ReturnType<typeof useMe>);
		vi.mocked(useAdminOrganizations).mockReturnValue({
			data: {
				organizations: [
					{
						id: 'org-1',
						name: 'Acme Corp',
						slug: 'acme',
						owner_email: 'owner@acme.com',
						member_count: 3,
						agent_count: 2,
						storage_used_bytes: 1024,
						created_at: '2024-01-01T00:00:00Z',
					},
				],
				total_count: 1,
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAdminOrganizations>);
		renderWithProviders(<OrgManagement />);
		expect(screen.getByText('Acme Corp')).toBeInTheDocument();
		expect(screen.getByText('owner@acme.com')).toBeInTheDocument();
	});
});
