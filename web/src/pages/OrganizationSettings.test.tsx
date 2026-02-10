import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrganizationSettings } from './OrganizationSettings';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useOrganizations', () => ({
	useCurrentOrganization: vi.fn(),
	useUpdateOrganization: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteOrganization: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { useCurrentOrganization } from '../hooks/useOrganizations';

function renderPage() {
	return render(
		<BrowserRouter>
			<OrganizationSettings />
		</BrowserRouter>,
	);
}

describe('OrganizationSettings', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useCurrentOrganization).mockReturnValue({
			data: {
				organization: {
					id: 'org1',
					name: 'Test Org',
					slug: 'test-org',
					created_at: '2024-01-01T00:00:00Z',
				},
			},
			isLoading: false,
		} as ReturnType<typeof useCurrentOrganization>);
		renderPage();
		expect(screen.getByText('Organization Settings')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useCurrentOrganization).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useCurrentOrganization>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows organization name', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { id: 'u1', current_org_id: 'org1', current_org_role: 'owner' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useCurrentOrganization).mockReturnValue({
			data: {
				organization: {
					id: 'org1',
					name: 'My Org',
					slug: 'my-org',
					created_at: '2024-01-01T00:00:00Z',
				},
			},
			isLoading: false,
		} as ReturnType<typeof useCurrentOrganization>);
		renderPage();
		expect(screen.getByText('My Org')).toBeInTheDocument();
		expect(screen.getByText('my-org')).toBeInTheDocument();
	});

	it('shows danger zone for owner', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { id: 'u1', current_org_id: 'org1', current_org_role: 'owner' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useCurrentOrganization).mockReturnValue({
			data: {
				organization: {
					id: 'org1',
					name: 'My Org',
					slug: 'my-org',
					created_at: '2024-01-01T00:00:00Z',
				},
			},
			isLoading: false,
		} as ReturnType<typeof useCurrentOrganization>);
		renderPage();
		expect(screen.getByText('Danger Zone')).toBeInTheDocument();
	});

	it('shows edit button for admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useCurrentOrganization).mockReturnValue({
			data: {
				organization: {
					id: 'org1',
					name: 'My Org',
					slug: 'my-org',
					created_at: '2024-01-01T00:00:00Z',
				},
			},
			isLoading: false,
		} as ReturnType<typeof useCurrentOrganization>);
		renderPage();
		expect(screen.getByText('Edit')).toBeInTheDocument();
	});
});
