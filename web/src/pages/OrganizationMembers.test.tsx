import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useOrganizations', () => ({
	useCurrentOrganization: vi.fn(() => ({ data: { organization: { id: 'org1', name: 'Test Org' } } })),
	useOrgMembers: vi.fn(),
	useOrgInvitations: vi.fn(() => ({ data: [], isLoading: false })),
	useUpdateMember: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRemoveMember: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useCreateInvitation: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false }),
	useDeleteInvitation: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { useOrgMembers, useOrgInvitations } from '../hooks/useOrganizations';

const { default: OrganizationMembers } = await import('./OrganizationMembers');

function renderPage() {
	return render(
		<BrowserRouter>
			<OrganizationMembers />
		</BrowserRouter>,
	);
}

describe('OrganizationMembers', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Members')).toBeInTheDocument();
	});

	it('shows subtitle with org name', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText(/Manage members of/)).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Failed to load members')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('No members found')).toBeInTheDocument();
	});

	it('renders members', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [
				{ user_id: 'u1', name: 'Alice Smith', email: 'alice@example.com', role: 'admin', joined_at: '2024-01-01T00:00:00Z' },
				{ user_id: 'u2', name: 'Bob Jones', email: 'bob@example.com', role: 'member', joined_at: '2024-01-02T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Alice Smith')).toBeInTheDocument();
		expect(screen.getByText('Bob Jones')).toBeInTheDocument();
	});

	it('shows member emails', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [
				{ user_id: 'u2', name: 'Bob Jones', email: 'bob@example.com', role: 'member', joined_at: '2024-01-02T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('bob@example.com')).toBeInTheDocument();
	});

	it('shows role badges', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [
				{ user_id: 'u2', name: 'Bob', email: 'bob@example.com', role: 'member', joined_at: '2024-01-02T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('member')).toBeInTheDocument();
	});

	it('shows invite button for admin', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Invite Member')).toBeInTheDocument();
	});

	it('hides invite button for non-admin', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'member' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.queryByText('Invite Member')).not.toBeInTheDocument();
	});

	it('shows Organization Members section header', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Organization Members')).toBeInTheDocument();
	});

	it('shows Pending Invitations section for admin', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Pending Invitations')).toBeInTheDocument();
	});

	it('shows no pending invitations message', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		vi.mocked(useOrgInvitations).mockReturnValue({ data: [], isLoading: false } as ReturnType<typeof useOrgInvitations>);
		renderPage();
		expect(screen.getByText('No pending invitations')).toBeInTheDocument();
	});

	it('shows edit and remove buttons for manageable members', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [
				{ user_id: 'u2', name: 'Bob Jones', email: 'bob@example.com', role: 'member', joined_at: '2024-01-02T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('Edit')).toBeInTheDocument();
		expect(screen.getByText('Remove')).toBeInTheDocument();
	});

	it('shows You label for current user', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [
				{ user_id: 'u1', name: 'Alice Smith', email: 'alice@example.com', role: 'admin', joined_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getByText('You')).toBeInTheDocument();
	});

	it('opens invite modal on button click', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		renderPage();
		await user.click(screen.getByText('Invite Member'));
		expect(screen.getAllByText('Invite Member').length).toBeGreaterThan(1);
		expect(screen.getByLabelText('Email Address')).toBeInTheDocument();
		expect(screen.getByLabelText('Role')).toBeInTheDocument();
	});

	it('shows table headers when members exist', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({
			data: [{ user_id: 'u2', name: 'Bob', email: 'bob@example.com', role: 'member', joined_at: '2024-01-02T00:00:00Z' }],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useOrgMembers>);
		renderPage();
		expect(screen.getAllByText('Name').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Email').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Role').length).toBeGreaterThan(0);
	});

	it('shows invitations when present', () => {
		vi.mocked(useMe).mockReturnValue({ data: { id: 'u1', current_org_id: 'org1', current_org_role: 'owner' } } as ReturnType<typeof useMe>);
		vi.mocked(useOrgMembers).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useOrgMembers>);
		vi.mocked(useOrgInvitations).mockReturnValue({
			data: [{ id: 'inv1', email: 'new@example.com', role: 'member', inviter_name: 'Alice', expires_at: new Date(Date.now() + 86400000).toISOString() }],
			isLoading: false,
		} as ReturnType<typeof useOrgInvitations>);
		renderPage();
		expect(screen.getByText('new@example.com')).toBeInTheDocument();
		expect(screen.getByText('Alice')).toBeInTheDocument();
	});
});
