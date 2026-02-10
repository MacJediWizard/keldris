import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { OrganizationSSOSettings } from './OrganizationSSOSettings';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useSSOGroupMappings', () => ({
	useSSOGroupMappings: vi.fn(),
	useSSOSettings: vi.fn(),
	useCreateSSOGroupMapping: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateSSOGroupMapping: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteSSOGroupMapping: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateSSOSettings: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { useSSOGroupMappings, useSSOSettings } from '../hooks/useSSOGroupMappings';

function renderPage() {
	return render(
		<BrowserRouter>
			<OrganizationSSOSettings />
		</BrowserRouter>,
	);
}

describe('OrganizationSSOSettings', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useSSOGroupMappings).mockReturnValue({ data: [], isLoading: false } as ReturnType<typeof useSSOGroupMappings>);
		vi.mocked(useSSOSettings).mockReturnValue({ data: { default_role: 'member', auto_create_orgs: false }, isLoading: false } as ReturnType<typeof useSSOSettings>);
		renderPage();
		expect(screen.getByText('SSO Group Sync Settings')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useSSOGroupMappings).mockReturnValue({ data: undefined, isLoading: true } as ReturnType<typeof useSSOGroupMappings>);
		vi.mocked(useSSOSettings).mockReturnValue({ data: undefined, isLoading: true } as ReturnType<typeof useSSOSettings>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useSSOGroupMappings).mockReturnValue({ data: [], isLoading: false } as ReturnType<typeof useSSOGroupMappings>);
		vi.mocked(useSSOSettings).mockReturnValue({ data: { default_role: 'member', auto_create_orgs: false }, isLoading: false } as ReturnType<typeof useSSOSettings>);
		renderPage();
		expect(screen.getByText('No group mappings configured yet')).toBeInTheDocument();
	});

	it('renders mappings', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useSSOGroupMappings).mockReturnValue({
			data: [
				{ id: 'm1', oidc_group_name: 'engineering', role: 'admin', auto_create_org: false, created_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
		} as ReturnType<typeof useSSOGroupMappings>);
		vi.mocked(useSSOSettings).mockReturnValue({ data: { default_role: 'member', auto_create_orgs: false }, isLoading: false } as ReturnType<typeof useSSOSettings>);
		renderPage();
		expect(screen.getByText('engineering')).toBeInTheDocument();
	});

	it('shows add mapping button for admin', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useSSOGroupMappings).mockReturnValue({ data: [], isLoading: false } as ReturnType<typeof useSSOGroupMappings>);
		vi.mocked(useSSOSettings).mockReturnValue({ data: { default_role: 'member', auto_create_orgs: false }, isLoading: false } as ReturnType<typeof useSSOSettings>);
		renderPage();
		expect(screen.getByText('Add Mapping')).toBeInTheDocument();
	});
});
