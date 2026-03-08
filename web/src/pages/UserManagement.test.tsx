import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { UserManagement } from './UserManagement';

// --- Mock data builders ---

function makeUser(overrides: Record<string, unknown> = {}) {
	return {
		id: 'user-2',
		org_id: 'org-1',
		email: 'jane@example.com',
		name: 'Jane Doe',
		role: 'member',
		status: 'active' as const,
		org_role: 'member' as const,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-02T00:00:00Z',
		last_login_at: '2026-03-01T12:00:00Z',
		...overrides,
	};
}

function makeInvitation(overrides: Record<string, unknown> = {}) {
	return {
		id: 'inv-1',
		org_id: 'org-1',
		org_name: 'My Org',
		email: 'invited@example.com',
		role: 'member' as const,
		invited_by: 'user-1',
		inviter_name: 'Admin User',
		expires_at: new Date(Date.now() + 86400000).toISOString(),
		created_at: '2026-03-01T00:00:00Z',
		...overrides,
	};
}

// --- Mock state ---

const mockMe = {
	data: null as Record<string, unknown> | null,
};

const mockCurrentOrg = {
	data: null as { organization: { id: string; name: string } } | null,
	isLoading: false,
};

const mockUsers = {
	data: null as ReturnType<typeof makeUser>[] | null,
	isLoading: false,
	isError: false,
};

const mockInvitations = {
	data: null as ReturnType<typeof makeInvitation>[] | null,
	isLoading: false,
};

const mockInviteUser = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
	isError: false,
};

const mockUpdateUser = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
	isError: false,
};

const mockDeleteUser = { mutate: vi.fn(), isPending: false };
const mockDisableUser = { mutate: vi.fn(), isPending: false };
const mockEnableUser = { mutate: vi.fn(), isPending: false };
const mockResetPassword = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
	isError: false,
};
const mockStartImpersonation = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
	isError: false,
};
const mockEndImpersonation = { mutate: vi.fn(), isPending: false };
const mockDeleteInvitation = { mutate: vi.fn(), isPending: false };
const mockResendInvitation = { mutate: vi.fn(), isPending: false };
const mockBulkInviteCSV = {
	mutateAsync: vi.fn(),
	isPending: false,
	isError: false,
};
const mockUserActivity = { data: null, isLoading: false };

// --- Mocks ---

vi.mock('../hooks/useAuth', () => ({
	useMe: () => mockMe,
}));

vi.mock('../hooks/useOrganizations', () => ({
	useCurrentOrganization: () => mockCurrentOrg,
	useOrgInvitations: () => mockInvitations,
	useDeleteInvitation: () => mockDeleteInvitation,
	useResendInvitation: () => mockResendInvitation,
	useBulkInviteCSV: () => mockBulkInviteCSV,
}));

vi.mock('../hooks/useUsers', () => ({
	useUsers: () => mockUsers,
	useInviteUser: () => mockInviteUser,
	useUpdateUser: () => mockUpdateUser,
	useDeleteUser: () => mockDeleteUser,
	useDisableUser: () => mockDisableUser,
	useEnableUser: () => mockEnableUser,
	useResetUserPassword: () => mockResetPassword,
	useStartImpersonation: () => mockStartImpersonation,
	useEndImpersonation: () => mockEndImpersonation,
	useUserActivity: () => mockUserActivity,
}));

vi.mock('../lib/utils', async () => {
	const actual =
		await vi.importActual<typeof import('../lib/utils')>('../lib/utils');
	return { ...actual };
});

// --- Helpers ---

function createQueryClient() {
	return new QueryClient({
		defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
	});
}

function renderUserManagement() {
	const queryClient = createQueryClient();
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter>
				<UserManagement />
			</MemoryRouter>
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	mockMe.data = {
		id: 'user-1',
		email: 'admin@test.com',
		name: 'Admin User',
		current_org_role: 'owner',
		is_superuser: true,
		is_impersonating: false,
	};
	mockCurrentOrg.data = {
		organization: { id: 'org-1', name: 'Test Org' },
	};
	mockCurrentOrg.isLoading = false;
	mockUsers.data = [
		makeUser({
			id: 'user-1',
			email: 'admin@test.com',
			name: 'Admin User',
			org_role: 'owner',
			role: 'owner',
		}),
		makeUser({
			id: 'user-2',
			email: 'jane@example.com',
			name: 'Jane Doe',
			org_role: 'member',
		}),
		makeUser({
			id: 'user-3',
			email: 'bob@example.com',
			name: 'Bob Smith',
			org_role: 'admin',
		}),
	];
	mockUsers.isLoading = false;
	mockUsers.isError = false;
	mockInvitations.data = [];
	mockInvitations.isLoading = false;
	mockInviteUser.mutateAsync.mockResolvedValue(undefined);
	mockInviteUser.isPending = false;
	mockInviteUser.isError = false;
	mockUpdateUser.mutateAsync.mockResolvedValue(undefined);
	mockUpdateUser.isPending = false;
	mockUpdateUser.isError = false;
	mockDeleteUser.mutate.mockReset();
	mockDisableUser.mutate.mockReset();
	mockEnableUser.mutate.mockReset();
});

// --- Tests ---

describe('UserManagement page', () => {
	describe('page header', () => {
		it('renders the User Management heading', () => {
			renderUserManagement();
			expect(screen.getByText('User Management')).toBeInTheDocument();
		});

		it('shows the organization name in the subtitle', () => {
			renderUserManagement();
			expect(screen.getByText(/Manage users in Test Org/)).toBeInTheDocument();
		});
	});

	describe('loading state', () => {
		it('renders loading skeleton rows when users are loading', () => {
			mockUsers.isLoading = true;
			mockUsers.data = null;
			renderUserManagement();
			const pulseRows = document.querySelectorAll('tr.animate-pulse');
			expect(pulseRows.length).toBe(3);
		});
	});

	describe('error state', () => {
		it('shows error message when users fail to load', () => {
			mockUsers.isError = true;
			mockUsers.data = null;
			renderUserManagement();
			expect(screen.getByText('Failed to load users')).toBeInTheDocument();
		});
	});

	describe('empty state', () => {
		it('shows no users message when list is empty', () => {
			mockUsers.data = [];
			renderUserManagement();
			expect(screen.getByText('No users found')).toBeInTheDocument();
		});
	});

	describe('user list rendering', () => {
		it('renders all users in the table', () => {
			renderUserManagement();
			expect(screen.getByText('Admin User')).toBeInTheDocument();
			expect(screen.getByText('Jane Doe')).toBeInTheDocument();
			expect(screen.getByText('Bob Smith')).toBeInTheDocument();
		});

		it('renders email addresses', () => {
			renderUserManagement();
			expect(screen.getByText('admin@test.com')).toBeInTheDocument();
			expect(screen.getByText('jane@example.com')).toBeInTheDocument();
			expect(screen.getByText('bob@example.com')).toBeInTheDocument();
		});

		it('renders role badges for each user', () => {
			renderUserManagement();
			expect(screen.getByText('owner')).toBeInTheDocument();
			expect(screen.getByText('member')).toBeInTheDocument();
			expect(screen.getByText('admin')).toBeInTheDocument();
		});

		it('renders status badges', () => {
			renderUserManagement();
			// All three users are active
			const activeBadges = screen.getAllByText('active');
			expect(activeBadges.length).toBe(3);
		});

		it('shows "You" label for the current user row', () => {
			renderUserManagement();
			expect(screen.getByText('You')).toBeInTheDocument();
		});

		it('renders table headers', () => {
			renderUserManagement();
			expect(screen.getAllByText('Name').length).toBeGreaterThan(0);
			expect(screen.getAllByText('Email').length).toBeGreaterThan(0);
			expect(screen.getAllByText('Role').length).toBeGreaterThan(0);
			expect(screen.getAllByText('Status').length).toBeGreaterThan(0);
		});
	});

	describe('Invite User button and modal', () => {
		it('shows Invite User button for owner role', () => {
			renderUserManagement();
			expect(
				screen.getByRole('button', { name: /invite user/i }),
			).toBeInTheDocument();
		});

		it('shows Invite User button for admin role', () => {
			mockMe.data = { ...mockMe.data, current_org_role: 'admin' };
			renderUserManagement();
			expect(
				screen.getByRole('button', { name: /invite user/i }),
			).toBeInTheDocument();
		});

		it('does NOT show Invite User button for member role', () => {
			mockMe.data = { ...mockMe.data, current_org_role: 'member' };
			renderUserManagement();
			expect(
				screen.queryByRole('button', { name: /invite user/i }),
			).not.toBeInTheDocument();
		});

		it('opens invite modal on click', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			await user.click(screen.getByRole('button', { name: /invite user/i }));
			// The modal has a heading "Invite User" (h3) plus the button text.
			// Look for the modal heading specifically.
			expect(
				screen.getByRole('heading', { name: 'Invite User' }),
			).toBeInTheDocument();
			expect(screen.getByLabelText('Email Address')).toBeInTheDocument();
		});

		it('modal has role selector with correct options', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			await user.click(screen.getByRole('button', { name: /invite user/i }));
			const roleSelect = screen.getByLabelText('Role');
			expect(roleSelect).toBeInTheDocument();
			const options = within(roleSelect as HTMLElement).getAllByRole('option');
			expect(options.map((o) => o.textContent)).toEqual([
				'Member',
				'Admin',
				'Read Only',
			]);
		});

		it('submits invite form and closes modal', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			await user.click(screen.getByRole('button', { name: /invite user/i }));
			await user.type(
				screen.getByLabelText('Email Address'),
				'new@example.com',
			);
			await user.click(
				screen.getByRole('button', { name: /send invitation/i }),
			);
			expect(mockInviteUser.mutateAsync).toHaveBeenCalledWith({
				email: 'new@example.com',
				name: undefined,
				role: 'member',
			});
		});

		it('cancel closes the modal', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			await user.click(screen.getByRole('button', { name: /invite user/i }));
			expect(screen.getByLabelText('Email Address')).toBeInTheDocument();
			await user.click(screen.getByRole('button', { name: /cancel/i }));
			expect(screen.queryByLabelText('Email Address')).not.toBeInTheDocument();
		});
	});

	describe('action buttons per user row', () => {
		it('renders edit button for manageable users (not self, not owner)', () => {
			renderUserManagement();
			// Jane (member) and Bob (admin) should have edit buttons,
			// but not the current user (Admin User / owner).
			const editButtons = screen.getAllByTitle('Edit');
			expect(editButtons.length).toBe(2);
		});

		it('renders delete button for manageable users', () => {
			renderUserManagement();
			const deleteButtons = screen.getAllByTitle('Delete');
			expect(deleteButtons.length).toBe(2);
		});

		it('renders disable button for active manageable users', () => {
			renderUserManagement();
			const disableButtons = screen.getAllByTitle('Disable');
			expect(disableButtons.length).toBe(2);
		});

		it('renders enable button for disabled users', () => {
			mockUsers.data = [
				makeUser({
					id: 'user-1',
					email: 'admin@test.com',
					name: 'Admin User',
					org_role: 'owner',
					role: 'owner',
				}),
				makeUser({
					id: 'user-4',
					email: 'disabled@example.com',
					name: 'Disabled User',
					org_role: 'member',
					status: 'disabled',
				}),
			];
			renderUserManagement();
			expect(screen.getByTitle('Enable')).toBeInTheDocument();
		});

		it('renders impersonate button for superuser viewing active non-self users', () => {
			renderUserManagement();
			const impersonateButtons = screen.getAllByTitle('Impersonate');
			expect(impersonateButtons.length).toBe(2);
		});

		it('does not render impersonate button when not a superuser', () => {
			mockMe.data = { ...mockMe.data, is_superuser: false };
			renderUserManagement();
			expect(screen.queryByTitle('Impersonate')).not.toBeInTheDocument();
		});

		it('renders reset password button for non-OIDC users', () => {
			renderUserManagement();
			const resetButtons = screen.getAllByTitle('Reset Password');
			expect(resetButtons.length).toBe(2);
		});

		it('does NOT render reset password for OIDC users', () => {
			mockUsers.data = [
				makeUser({
					id: 'user-1',
					email: 'admin@test.com',
					name: 'Admin User',
					org_role: 'owner',
					role: 'owner',
				}),
				makeUser({
					id: 'user-5',
					email: 'oidc@example.com',
					name: 'OIDC User',
					org_role: 'member',
					oidc_subject: 'oidc-sub-123',
				}),
			];
			renderUserManagement();
			expect(screen.queryByTitle('Reset Password')).not.toBeInTheDocument();
		});
	});

	describe('edit user modal', () => {
		it('opens edit modal when edit button is clicked', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			const editButtons = screen.getAllByTitle('Edit');
			await user.click(editButtons[0]);
			expect(screen.getByText('Edit User')).toBeInTheDocument();
		});
	});

	describe('delete user', () => {
		it('calls deleteUser when delete is confirmed', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			const deleteButtons = screen.getAllByTitle('Delete');
			await user.click(deleteButtons[0]);
			// window.confirm is mocked to return true in setup.ts
			expect(mockDeleteUser.mutate).toHaveBeenCalled();
		});
	});

	describe('disable user', () => {
		it('calls disableUser when disable is confirmed', async () => {
			const user = userEvent.setup();
			renderUserManagement();
			const disableButtons = screen.getAllByTitle('Disable');
			await user.click(disableButtons[0]);
			expect(mockDisableUser.mutate).toHaveBeenCalled();
		});
	});

	describe('pending invitations section', () => {
		it('renders pending invitations section for owner', () => {
			renderUserManagement();
			expect(screen.getByText('Pending Invitations')).toBeInTheDocument();
		});

		it('shows "No pending invitations" when empty', () => {
			renderUserManagement();
			expect(screen.getByText('No pending invitations')).toBeInTheDocument();
		});

		it('does not render invitations section for member role', () => {
			mockMe.data = { ...mockMe.data, current_org_role: 'member' };
			renderUserManagement();
			expect(screen.queryByText('Pending Invitations')).not.toBeInTheDocument();
		});

		it('renders invitation rows', () => {
			mockInvitations.data = [makeInvitation()];
			renderUserManagement();
			expect(screen.getByText('invited@example.com')).toBeInTheDocument();
		});

		it('renders Bulk Invite (CSV) button', () => {
			renderUserManagement();
			expect(screen.getByText('Bulk Invite (CSV)')).toBeInTheDocument();
		});
	});

	describe('impersonation banner', () => {
		it('shows impersonation banner when is_impersonating is true', () => {
			mockMe.data = { ...mockMe.data, is_impersonating: true };
			renderUserManagement();
			expect(
				screen.getByText(/you are currently impersonating a user/i),
			).toBeInTheDocument();
			expect(
				screen.getByRole('button', { name: /end impersonation/i }),
			).toBeInTheDocument();
		});

		it('does not show impersonation banner normally', () => {
			renderUserManagement();
			expect(
				screen.queryByText(/you are currently impersonating a user/i),
			).not.toBeInTheDocument();
		});

		it('calls endImpersonation on click', async () => {
			mockMe.data = { ...mockMe.data, is_impersonating: true };
			const user = userEvent.setup();
			renderUserManagement();
			await user.click(
				screen.getByRole('button', { name: /end impersonation/i }),
			);
			// window.confirm returns true from setup.ts
			expect(mockEndImpersonation.mutate).toHaveBeenCalled();
		});
	});

	describe('view activity', () => {
		it('renders view activity button for non-self users', () => {
			renderUserManagement();
			const activityButtons = screen.getAllByTitle('View Activity');
			expect(activityButtons.length).toBe(2);
		});
	});
});
