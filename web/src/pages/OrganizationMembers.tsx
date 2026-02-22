import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useCreateInvitation,
	useCurrentOrganization,
	useDeleteInvitation,
	useOrgInvitations,
	useOrgMembers,
	useRemoveMember,
	useUpdateMember,
} from '../hooks/useOrganizations';
import type { OrgInvitation, OrgMember, OrgRole } from '../lib/types';
import { formatDate } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-40 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-20 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
				<div className="h-4 w-32 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-40 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-20 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

interface InviteModalProps {
	isOpen: boolean;
	onClose: () => void;
	orgId: string;
}

function InviteModal({ isOpen, onClose, orgId }: InviteModalProps) {
	const [email, setEmail] = useState('');
	const [role, setRole] = useState<OrgRole>('member');
	const createInvitation = useCreateInvitation();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createInvitation.mutateAsync({ orgId, data: { email, role } });
			setEmail('');
			setRole('member');
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Invite Member
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="email"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Email Address
						</label>
						<input
							type="email"
							id="email"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
							placeholder="user@example.com"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="role"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Role
						</label>
						<select
							id="role"
							value={role}
							onChange={(e) => setRole(e.target.value as OrgRole)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="member">Member</option>
							<option value="admin">Admin</option>
							<option value="readonly">Read Only</option>
						</select>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
						<p className="mt-1 text-xs text-gray-500">
							Admin: Can manage members and resources. Member: Can create and
							manage resources. Read Only: View-only access.
						</p>
					</div>
					{createInvitation.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to send invitation. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createInvitation.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createInvitation.isPending ? 'Sending...' : 'Send Invitation'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

function getRoleBadgeColor(role: OrgRole) {
	switch (role) {
		case 'owner':
			return 'bg-purple-100 text-purple-700';
		case 'admin':
			return 'bg-blue-100 text-blue-700';
		case 'member':
			return 'bg-green-100 text-green-700';
		case 'readonly':
			return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300';
		default:
			return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300';
			return 'bg-gray-100 text-gray-700';
		default:
			return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300';
	}
}

interface MemberRowProps {
	member: OrgMember;
	currentUserId: string;
	currentUserRole: OrgRole;
	orgId: string;
}

function MemberRow({
	member,
	currentUserId,
	currentUserRole,
	orgId,
}: MemberRowProps) {
	const [isEditing, setIsEditing] = useState(false);
	const [selectedRole, setSelectedRole] = useState<OrgRole>(member.role);
	const updateMember = useUpdateMember();
	const removeMember = useRemoveMember();

	const canManage =
		(currentUserRole === 'owner' || currentUserRole === 'admin') &&
		member.user_id !== currentUserId &&
		member.role !== 'owner';

	const handleRoleChange = async () => {
		if (selectedRole !== member.role) {
			await updateMember.mutateAsync({
				orgId,
				userId: member.user_id,
				role: selectedRole,
			});
		}
		setIsEditing(false);
	};

	const handleRemove = () => {
		if (
			confirm(
				`Are you sure you want to remove ${member.name || member.email} from the organization?`,
			)
		) {
			removeMember.mutate({ orgId, userId: member.user_id });
		}
	};

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">
					{member.name || 'No name'}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{member.email}
			</td>
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">
					{member.name || 'No name'}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{member.email}
			</td>
			<td className="px-6 py-4">
				{isEditing ? (
					<div className="flex items-center gap-2">
						<select
							value={selectedRole}
							onChange={(e) => setSelectedRole(e.target.value as OrgRole)}
							className="text-sm border border-gray-300 rounded px-2 py-1"
						>
							<option value="admin">Admin</option>
							<option value="member">Member</option>
							<option value="readonly">Read Only</option>
						</select>
						<button
							type="button"
							onClick={handleRoleChange}
							disabled={updateMember.isPending}
							className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
							className="text-sm text-indigo-600 hover:text-indigo-800"
						>
							Save
						</button>
						<button
							type="button"
							onClick={() => {
								setIsEditing(false);
								setSelectedRole(member.role);
							}}
							className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800"
							className="text-sm text-gray-600 hover:text-gray-800"
						>
							Cancel
						</button>
					</div>
				) : (
					<span
						className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getRoleBadgeColor(member.role)}`}
					>
						{member.role}
					</span>
				)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(member.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				{canManage && !isEditing && (
					<div className="flex items-center justify-end gap-3">
						<button
							type="button"
							onClick={() => setIsEditing(true)}
							className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
							className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
						>
							Edit
						</button>
						<button
							type="button"
							onClick={handleRemove}
							disabled={removeMember.isPending}
							className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
						>
							Remove
						</button>
					</div>
				)}
				{member.user_id === currentUserId && (
					<span className="text-xs text-gray-400">You</span>
				)}
			</td>
		</tr>
	);
}

interface InvitationRowProps {
	invitation: OrgInvitation;
	orgId: string;
}

function InvitationRow({ invitation, orgId }: InvitationRowProps) {
	const deleteInvitation = useDeleteInvitation();

	const handleCancel = () => {
		if (confirm('Are you sure you want to cancel this invitation?')) {
			deleteInvitation.mutate({ orgId, invitationId: invitation.id });
		}
	};

	const isExpired = new Date(invitation.expires_at) < new Date();

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{invitation.email}
			</td>
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4 text-sm text-gray-500">{invitation.email}</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getRoleBadgeColor(invitation.role)}`}
				>
					{invitation.role}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
			<td className="px-6 py-4 text-sm text-gray-500">
				{invitation.inviter_name}
			</td>
			<td className="px-6 py-4 text-sm">
				{isExpired ? (
					<span className="text-red-600 dark:text-red-400">Expired</span>
				) : (
					<span className="text-gray-500 dark:text-gray-400">
					<span className="text-red-600">Expired</span>
				) : (
					<span className="text-gray-500 dark:text-gray-400">
						Expires {formatDate(invitation.expires_at)}
					</span>
				)}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={handleCancel}
					disabled={deleteInvitation.isPending}
					className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
				>
					Cancel
				</button>
			</td>
		</tr>
	);
}

export function OrganizationMembers() {
	const [showInviteModal, setShowInviteModal] = useState(false);
	const { data: user } = useMe();
	const { data: currentOrg, isLoading: orgLoading } = useCurrentOrganization();

	const orgId = user?.current_org_id ?? '';
	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;

	const {
		data: members,
		isLoading: membersLoading,
		isError: membersError,
	} = useOrgMembers(orgId);
	const { data: invitations, isLoading: invitationsLoading } =
		useOrgInvitations(orgId);

	const canInvite = currentUserRole === 'owner' || currentUserRole === 'admin';

	const isLoading = orgLoading || membersLoading;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Members
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
					<h1 className="text-2xl font-bold text-gray-900">Members</h1>
					<p className="text-gray-600 mt-1">
						Manage members of{' '}
						{currentOrg?.organization.name ?? 'your organization'}
					</p>
				</div>
				{canInvite && (
					<button
						type="button"
						onClick={() => setShowInviteModal(true)}
						className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 4v16m8-8H4"
							/>
						</svg>
						Invite Member
					</button>
				)}
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-6 py-4 border-b border-gray-200">
					<h2 className="text-lg font-semibold text-gray-900">
						Organization Members
					</h2>
				</div>

				{membersError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400 dark:text-red-400">
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">Failed to load members</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Email
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Role
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Joined
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Email
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Role
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Joined
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
						<tbody className="divide-y divide-gray-200">
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : members && members.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Email
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Role
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Joined
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Email
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Role
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Joined
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{members.map((member) => (
								<MemberRow
									key={member.user_id}
						<tbody className="divide-y divide-gray-200">
							{members.map((member) => (
								<MemberRow
									key={member.id}
									member={member}
									currentUserId={user?.id ?? ''}
									currentUserRole={currentUserRole}
									orgId={orgId}
								/>
							))}
						</tbody>
					</table>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
					<div className="p-12 text-center text-gray-500">
						<p>No members found</p>
					</div>
				)}
			</div>

			{canInvite && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
				<div className="bg-white rounded-lg border border-gray-200">
					<div className="px-6 py-4 border-b border-gray-200">
						<h2 className="text-lg font-semibold text-gray-900">
							Pending Invitations
						</h2>
					</div>

					{invitationsLoading ? (
						<div className="p-6 text-center">
							<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto" />
						</div>
					) : invitations && invitations.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Email
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Role
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Invited By
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Email
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Role
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Invited By
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							<tbody className="divide-y divide-gray-200">
								{invitations.map((invitation) => (
									<InvitationRow
										key={invitation.id}
										invitation={invitation}
										orgId={orgId}
									/>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-8 text-center text-gray-500 dark:text-gray-400">
						<div className="p-8 text-center text-gray-500">
							<p>No pending invitations</p>
						</div>
					)}
				</div>
			)}

			<InviteModal
				isOpen={showInviteModal}
				onClose={() => setShowInviteModal(false)}
				orgId={orgId}
			/>
		</div>
	);
}

export default OrganizationMembers;
