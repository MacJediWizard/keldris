import { useState, useRef } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useBulkInviteCSV,
	useCurrentOrganization,
	useDeleteInvitation,
	useOrgInvitations,
	useResendInvitation,
} from '../hooks/useOrganizations';
import {
	useDeleteUser,
	useDisableUser,
	useEnableUser,
	useEndImpersonation,
	useInviteUser,
	useResetUserPassword,
	useStartImpersonation,
	useUpdateUser,
	useUserActivity,
	useUsers,
} from '../hooks/useUsers';
import type {
	OrgInvitation,
	OrgRole,
	UserActivityLog,
	UserStatus,
	UserWithMembership,
} from '../lib/types';
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
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
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
}

function InviteModal({ isOpen, onClose }: InviteModalProps) {
	const [email, setEmail] = useState('');
	const [name, setName] = useState('');
	const [role, setRole] = useState<OrgRole>('member');
	const inviteUser = useInviteUser();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await inviteUser.mutateAsync({ email, name: name || undefined, role });
			setEmail('');
			setName('');
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
					Invite User
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="email"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
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
							required
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Name (optional)
						</label>
						<input
							type="text"
							id="name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							placeholder="John Doe"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="role"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Role
						</label>
						<select
							id="role"
							value={role}
							onChange={(e) => setRole(e.target.value as OrgRole)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="member">Member</option>
							<option value="admin">Admin</option>
							<option value="readonly">Read Only</option>
						</select>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Admin: Can manage members and resources. Member: Can create and
							manage resources. Read Only: View-only access.
						</p>
					</div>
					{inviteUser.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to send invitation. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={inviteUser.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{inviteUser.isPending ? 'Sending...' : 'Send Invitation'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface EditUserModalProps {
	isOpen: boolean;
	onClose: () => void;
	user: UserWithMembership | null;
}

function EditUserModal({ isOpen, onClose, user }: EditUserModalProps) {
	const [role, setRole] = useState<OrgRole>(user?.org_role || 'member');
	const [name, setName] = useState(user?.name || '');
	const updateUser = useUpdateUser();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!user) return;
		try {
			await updateUser.mutateAsync({
				id: user.id,
				data: { role, name: name || undefined },
			});
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen || !user) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Edit User
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<span className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Email
						</span>
						<p className="text-gray-900 dark:text-white">{user.email}</p>
					</div>
					<div className="mb-4">
						<label
							htmlFor="edit-name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Name
						</label>
						<input
							type="text"
							id="edit-name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="edit-role"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Role
						</label>
						<select
							id="edit-role"
							value={role}
							onChange={(e) => setRole(e.target.value as OrgRole)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							disabled={user.role === 'owner'}
						>
							<option value="owner" disabled>
								Owner
							</option>
							<option value="admin">Admin</option>
							<option value="member">Member</option>
							<option value="readonly">Read Only</option>
						</select>
					</div>
					{updateUser.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to update user. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={updateUser.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{updateUser.isPending ? 'Saving...' : 'Save Changes'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ResetPasswordModalProps {
	isOpen: boolean;
	onClose: () => void;
	user: UserWithMembership | null;
}

function ResetPasswordModal({
	isOpen,
	onClose,
	user,
}: ResetPasswordModalProps) {
	const [password, setPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [requireChange, setRequireChange] = useState(true);
	const resetPassword = useResetUserPassword();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!user) return;
		if (password !== confirmPassword) {
			return;
		}
		try {
			await resetPassword.mutateAsync({
				id: user.id,
				data: { new_password: password, require_change_on_use: requireChange },
			});
			setPassword('');
			setConfirmPassword('');
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen || !user) return null;

	const passwordsMatch = password === confirmPassword;
	const isValid = password.length >= 8 && passwordsMatch;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Reset Password for {user.name || user.email}
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="new-password"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							New Password
						</label>
						<input
							type="password"
							id="new-password"
							value={password}
							onChange={(e) => setPassword(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
							minLength={8}
						/>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Minimum 8 characters
						</p>
					</div>
					<div className="mb-4">
						<label
							htmlFor="confirm-password"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Confirm Password
						</label>
						<input
							type="password"
							id="confirm-password"
							value={confirmPassword}
							onChange={(e) => setConfirmPassword(e.target.value)}
							className={`w-full px-4 py-2 border ${
								confirmPassword && !passwordsMatch
									? 'border-red-500'
									: 'border-gray-300 dark:border-gray-600'
							} bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500`}
							required
						/>
						{confirmPassword && !passwordsMatch && (
							<p className="mt-1 text-xs text-red-500">
								Passwords do not match
							</p>
						)}
					</div>
					<div className="mb-4">
						<label className="flex items-center gap-2">
							<input
								type="checkbox"
								checked={requireChange}
								onChange={(e) => setRequireChange(e.target.checked)}
								className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
							/>
							<span className="text-sm text-gray-700 dark:text-gray-300">
								Require password change on next login
							</span>
						</label>
					</div>
					{resetPassword.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to reset password. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={resetPassword.isPending || !isValid}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{resetPassword.isPending ? 'Resetting...' : 'Reset Password'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ImpersonateModalProps {
	isOpen: boolean;
	onClose: () => void;
	user: UserWithMembership | null;
}

function ImpersonateModal({ isOpen, onClose, user }: ImpersonateModalProps) {
	const [reason, setReason] = useState('');
	const startImpersonation = useStartImpersonation();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!user) return;
		try {
			await startImpersonation.mutateAsync({
				id: user.id,
				data: { reason },
			});
			// Page will reload after successful impersonation
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen || !user) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Impersonate User
				</h3>
				<div className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
					<p className="text-sm text-yellow-800 dark:text-yellow-200">
						You are about to impersonate{' '}
						<strong>{user.name || user.email}</strong>. All actions will be
						logged for audit purposes.
					</p>
				</div>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="reason"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Reason for impersonation
						</label>
						<textarea
							id="reason"
							value={reason}
							onChange={(e) => setReason(e.target.value)}
							placeholder="e.g., Investigating support ticket #12345"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							rows={3}
							required
						/>
					</div>
					{startImpersonation.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to start impersonation. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={startImpersonation.isPending}
							className="px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 transition-colors disabled:opacity-50"
						>
							{startImpersonation.isPending
								? 'Starting...'
								: 'Start Impersonation'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ActivityModalProps {
	isOpen: boolean;
	onClose: () => void;
	user: UserWithMembership | null;
}

function ActivityModal({ isOpen, onClose, user }: ActivityModalProps) {
	const { data: activity, isLoading } = useUserActivity(user?.id || '', 50, 0);

	if (!isOpen || !user) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Activity Log for {user.name || user.email}
					</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>
				<div className="flex-1 overflow-y-auto">
					{isLoading ? (
						<div className="flex items-center justify-center py-8">
							<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
						</div>
					) : activity && activity.length > 0 ? (
						<div className="space-y-3">
							{activity.map((log: UserActivityLog) => (
								<div
									key={log.id}
									className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg"
								>
									<div className="flex items-center justify-between mb-1">
										<span className="text-sm font-medium text-gray-900 dark:text-white">
											{log.action}
										</span>
										<span className="text-xs text-gray-500 dark:text-gray-400">
											{formatDate(log.created_at)}
										</span>
									</div>
									{log.resource_type && (
										<p className="text-xs text-gray-500 dark:text-gray-400">
											{log.resource_type}
											{log.resource_id && `: ${log.resource_id}`}
										</p>
									)}
									{log.ip_address && (
										<p className="text-xs text-gray-400 dark:text-gray-500">
											IP: {log.ip_address}
										</p>
									)}
								</div>
							))}
						</div>
					) : (
						<div className="text-center py-8 text-gray-500 dark:text-gray-400">
							No activity found
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

interface BulkInviteModalProps {
	isOpen: boolean;
	onClose: () => void;
	orgId: string;
}

function BulkInviteModal({ isOpen, onClose, orgId }: BulkInviteModalProps) {
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [dragActive, setDragActive] = useState(false);
	const [selectedFile, setSelectedFile] = useState<File | null>(null);
	const bulkInviteCSV = useBulkInviteCSV();
	const [result, setResult] = useState<{
		successful: number;
		failed: { email: string; error: string }[];
	} | null>(null);

	const handleDrag = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		if (e.type === 'dragenter' || e.type === 'dragover') {
			setDragActive(true);
		} else if (e.type === 'dragleave') {
			setDragActive(false);
		}
	};

	const handleDrop = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		setDragActive(false);

		if (e.dataTransfer.files?.[0]) {
			setSelectedFile(e.dataTransfer.files[0]);
		}
	};

	const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		if (e.target.files?.[0]) {
			setSelectedFile(e.target.files[0]);
		}
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!selectedFile) return;

		try {
			const response = await bulkInviteCSV.mutateAsync({
				orgId,
				file: selectedFile,
			});
			setResult({
				successful: response.successful.length,
				failed: response.failed,
			});
		} catch {
			// Error handled by mutation
		}
	};

	const handleClose = () => {
		setSelectedFile(null);
		setResult(null);
		onClose();
	};

	const downloadTemplate = () => {
		const csvContent = 'email,role\nexample@company.com,member\n';
		const blob = new Blob([csvContent], { type: 'text/csv' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = 'invite_template.csv';
		a.click();
		URL.revokeObjectURL(url);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Bulk Invite Users
				</h3>

				{result ? (
					<div>
						<div className="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
							<p className="text-green-800 dark:text-green-200">
								Successfully invited {result.successful} user(s)
							</p>
						</div>
						{result.failed.length > 0 && (
							<div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
								<p className="text-red-800 dark:text-red-200 font-medium mb-2">
									Failed to invite {result.failed.length} user(s):
								</p>
								<ul className="text-sm text-red-700 dark:text-red-300 list-disc list-inside">
									{result.failed.map((f, i) => (
										<li key={i}>
											{f.email}: {f.error}
										</li>
									))}
								</ul>
							</div>
						)}
						<button
							type="button"
							onClick={handleClose}
							className="w-full px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Done
						</button>
					</div>
				) : (
					<form onSubmit={handleSubmit}>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							Upload a CSV file with email and role columns.{' '}
							<button
								type="button"
								onClick={downloadTemplate}
								className="text-indigo-600 dark:text-indigo-400 hover:underline"
							>
								Download template
							</button>
						</p>

						<div
							className={`mb-4 border-2 border-dashed rounded-lg p-6 text-center transition-colors ${
								dragActive
									? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
									: 'border-gray-300 dark:border-gray-600'
							}`}
							onDragEnter={handleDrag}
							onDragLeave={handleDrag}
							onDragOver={handleDrag}
							onDrop={handleDrop}
						>
							{selectedFile ? (
								<div>
									<p className="text-gray-900 dark:text-white font-medium">
										{selectedFile.name}
									</p>
									<button
										type="button"
										onClick={() => setSelectedFile(null)}
										className="text-sm text-red-600 hover:underline mt-1"
									>
										Remove
									</button>
								</div>
							) : (
								<div>
									<svg
										aria-hidden="true"
										className="w-10 h-10 mx-auto text-gray-400 mb-2"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
										/>
									</svg>
									<p className="text-gray-600 dark:text-gray-400">
										Drag and drop CSV file here, or{' '}
										<button
											type="button"
											onClick={() => fileInputRef.current?.click()}
											className="text-indigo-600 dark:text-indigo-400 hover:underline"
										>
											browse
										</button>
									</p>
									<input
										ref={fileInputRef}
										type="file"
										accept=".csv"
										onChange={handleFileChange}
										className="hidden"
									/>
								</div>
							)}
						</div>

						{bulkInviteCSV.isError && (
							<p className="text-sm text-red-600 mb-4">
								Failed to process invitations. Please check your CSV file.
							</p>
						)}

						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={handleClose}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="submit"
								disabled={!selectedFile || bulkInviteCSV.isPending}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								{bulkInviteCSV.isPending ? 'Sending...' : 'Send Invitations'}
							</button>
						</div>
					</form>
				)}
			</div>
		</div>
	);
}

function getStatusBadgeColor(status: UserStatus) {
	switch (status) {
		case 'active':
			return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400';
		case 'disabled':
			return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400';
		case 'pending':
			return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400';
		case 'locked':
			return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400';
		default:
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300';
	}
}

function getRoleBadgeColor(role: OrgRole) {
	switch (role) {
		case 'owner':
			return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400';
		case 'admin':
			return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400';
		case 'member':
			return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400';
		case 'readonly':
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300';
		default:
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300';
	}
}

interface UserRowProps {
	user: UserWithMembership;
	currentUserId: string;
	currentUserRole: OrgRole;
	isSuperuser: boolean;
	onEdit: (user: UserWithMembership) => void;
	onResetPassword: (user: UserWithMembership) => void;
	onViewActivity: (user: UserWithMembership) => void;
	onImpersonate: (user: UserWithMembership) => void;
}

function UserRow({
	user,
	currentUserId,
	currentUserRole,
	isSuperuser,
	onEdit,
	onResetPassword,
	onViewActivity,
	onImpersonate,
}: UserRowProps) {
	const disableUser = useDisableUser();
	const enableUser = useEnableUser();
	const deleteUser = useDeleteUser();

	const canManage =
		(currentUserRole === 'owner' || currentUserRole === 'admin') &&
		user.id !== currentUserId &&
		user.org_role !== 'owner';

	const handleDisable = () => {
		if (
			confirm(`Are you sure you want to disable ${user.name || user.email}?`)
		) {
			disableUser.mutate(user.id);
		}
	};

	const handleEnable = () => {
		enableUser.mutate(user.id);
	};

	const handleDelete = () => {
		if (
			confirm(
				`Are you sure you want to delete ${user.name || user.email}? This action cannot be undone.`,
			)
		) {
			deleteUser.mutate(user.id);
		}
	};

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">
					{user.name || 'No name'}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{user.email}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getRoleBadgeColor(user.org_role)}`}
				>
					{user.org_role}
				</span>
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getStatusBadgeColor(user.status)}`}
				>
					{user.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{user.last_login_at ? formatDate(user.last_login_at) : 'Never'}
			</td>
			<td className="px-6 py-4 text-right">
				{user.id === currentUserId ? (
					<span className="text-xs text-gray-400">You</span>
				) : (
					<div className="flex items-center justify-end gap-2">
						<button
							type="button"
							onClick={() => onViewActivity(user)}
							className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-sm"
							title="View Activity"
						>
							<svg
								aria-hidden="true"
								className="w-4 h-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
								/>
							</svg>
						</button>
						{canManage && (
							<>
								<button
									type="button"
									onClick={() => onEdit(user)}
									className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm"
									title="Edit"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
										/>
									</svg>
								</button>
								{!user.oidc_subject && (
									<button
										type="button"
										onClick={() => onResetPassword(user)}
										className="text-yellow-600 dark:text-yellow-400 hover:text-yellow-800 dark:hover:text-yellow-300 text-sm"
										title="Reset Password"
									>
										<svg
											aria-hidden="true"
											className="w-4 h-4"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
											/>
										</svg>
									</button>
								)}
								{user.status === 'active' ? (
									<button
										type="button"
										onClick={handleDisable}
										disabled={disableUser.isPending}
										className="text-orange-600 dark:text-orange-400 hover:text-orange-800 dark:hover:text-orange-300 text-sm disabled:opacity-50"
										title="Disable"
									>
										<svg
											aria-hidden="true"
											className="w-4 h-4"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
											/>
										</svg>
									</button>
								) : (
									<button
										type="button"
										onClick={handleEnable}
										disabled={enableUser.isPending}
										className="text-green-600 dark:text-green-400 hover:text-green-800 dark:hover:text-green-300 text-sm disabled:opacity-50"
										title="Enable"
									>
										<svg
											aria-hidden="true"
											className="w-4 h-4"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
											/>
										</svg>
									</button>
								)}
								<button
									type="button"
									onClick={handleDelete}
									disabled={deleteUser.isPending}
									className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
									title="Delete"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
								</button>
							</>
						)}
						{isSuperuser && user.status === 'active' && (
							<button
								type="button"
								onClick={() => onImpersonate(user)}
								className="text-purple-600 dark:text-purple-400 hover:text-purple-800 dark:hover:text-purple-300 text-sm"
								title="Impersonate"
							>
								<svg
									aria-hidden="true"
									className="w-4 h-4"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
									/>
								</svg>
							</button>
						)}
					</div>
				)}
			</td>
		</tr>
	);
}

export function UserManagement() {
	const [showInviteModal, setShowInviteModal] = useState(false);
	const [editingUser, setEditingUser] = useState<UserWithMembership | null>(
		null,
	);
	const [resetPasswordUser, setResetPasswordUser] =
		useState<UserWithMembership | null>(null);
	const [activityUser, setActivityUser] = useState<UserWithMembership | null>(
		null,
	);
	const [impersonateUser, setImpersonateUser] =
		useState<UserWithMembership | null>(null);
	const [showBulkInviteModal, setShowBulkInviteModal] = useState(false);

	const { data: me } = useMe();
	const { data: currentOrg, isLoading: orgLoading } = useCurrentOrganization();
	const {
		data: users,
		isLoading: usersLoading,
		isError: usersError,
	} = useUsers();
	const endImpersonation = useEndImpersonation();
	const orgId = currentOrg?.organization.id ?? '';
	const { data: invitations, isLoading: invitationsLoading } =
		useOrgInvitations(orgId);
	const deleteInvitation = useDeleteInvitation();
	const resendInvitation = useResendInvitation();

	const currentUserRole = (me?.current_org_role ?? 'member') as OrgRole;
	const canInvite = currentUserRole === 'owner' || currentUserRole === 'admin';
	const isSuperuser = me?.is_superuser ?? false;
	const isImpersonating = me?.is_impersonating ?? false;

	const isLoading = orgLoading || usersLoading;

	const pendingInvitations = invitations?.filter(
		(inv) => !inv.accepted_at && new Date(inv.expires_at) > new Date(),
	);

	const handleRevokeInvitation = (invitationId: string) => {
		if (confirm('Are you sure you want to revoke this invitation?')) {
			deleteInvitation.mutate({ orgId, invitationId });
		}
	};

	const handleResendInvitation = (invitationId: string) => {
		resendInvitation.mutate({ orgId, invitationId });
	};

	const copyInviteLink = (token: string) => {
		const link = `${window.location.origin}/invite/${token}`;
		navigator.clipboard.writeText(link);
	};

	const handleEndImpersonation = () => {
		if (confirm('Are you sure you want to end the impersonation session?')) {
			endImpersonation.mutate();
		}
	};

	return (
		<div className="space-y-6">
			{isImpersonating && (
				<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 flex items-center justify-between">
					<div className="flex items-center gap-3">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-yellow-600 dark:text-yellow-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
						<span className="text-yellow-800 dark:text-yellow-200 font-medium">
							You are currently impersonating a user. All actions are being
							logged.
						</span>
					</div>
					<button
						type="button"
						onClick={handleEndImpersonation}
						disabled={endImpersonation.isPending}
						className="px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 transition-colors disabled:opacity-50"
					>
						{endImpersonation.isPending ? 'Ending...' : 'End Impersonation'}
					</button>
				</div>
			)}

			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						User Management
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Manage users in{' '}
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
						Invite User
					</button>
				)}
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						All Users
					</h2>
				</div>

				{usersError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load users</p>
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
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Last Login
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : users && users.length > 0 ? (
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
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Last Login
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{users.map((user) => (
								<UserRow
									key={user.id}
									user={user}
									currentUserId={me?.id ?? ''}
									currentUserRole={currentUserRole}
									isSuperuser={isSuperuser}
									onEdit={setEditingUser}
									onResetPassword={setResetPasswordUser}
									onViewActivity={setActivityUser}
									onImpersonate={setImpersonateUser}
								/>
							))}
						</tbody>
					</table>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<p>No users found</p>
					</div>
				)}
			</div>

			{/* Pending Invitations Section */}
			{canInvite && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Pending Invitations
						</h2>
						<button
							type="button"
							onClick={() => setShowBulkInviteModal(true)}
							className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
						>
							Bulk Invite (CSV)
						</button>
					</div>

					{invitationsLoading ? (
						<div className="p-6 text-center">
							<div className="w-6 h-6 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto" />
						</div>
					) : pendingInvitations && pendingInvitations.length > 0 ? (
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
										Expires
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{pendingInvitations.map((inv: OrgInvitation) => (
									<tr
										key={inv.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
											{inv.email}
										</td>
										<td className="px-6 py-4">
											<span
												className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getRoleBadgeColor(inv.role)}`}
											>
												{inv.role}
											</span>
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											{inv.inviter_name}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											{formatDate(inv.expires_at)}
										</td>
										<td className="px-6 py-4 text-right">
											<div className="flex items-center justify-end gap-2">
												<button
													type="button"
													onClick={() => copyInviteLink(inv.id)}
													className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-sm"
													title="Copy invite link"
												>
													<svg
														aria-hidden="true"
														className="w-4 h-4"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
														/>
													</svg>
												</button>
												<button
													type="button"
													onClick={() => handleResendInvitation(inv.id)}
													disabled={resendInvitation.isPending}
													className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm disabled:opacity-50"
													title="Resend invitation"
												>
													<svg
														aria-hidden="true"
														className="w-4 h-4"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
														/>
													</svg>
												</button>
												<button
													type="button"
													onClick={() => handleRevokeInvitation(inv.id)}
													disabled={deleteInvitation.isPending}
													className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
													title="Revoke invitation"
												>
													<svg
														aria-hidden="true"
														className="w-4 h-4"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
														/>
													</svg>
												</button>
											</div>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-6 text-center text-gray-500 dark:text-gray-400">
							No pending invitations
						</div>
					)}
				</div>
			)}

			<InviteModal
				isOpen={showInviteModal}
				onClose={() => setShowInviteModal(false)}
			/>
			<BulkInviteModal
				isOpen={showBulkInviteModal}
				onClose={() => setShowBulkInviteModal(false)}
				orgId={orgId}
			/>
			<EditUserModal
				isOpen={!!editingUser}
				onClose={() => setEditingUser(null)}
				user={editingUser}
			/>
			<ResetPasswordModal
				isOpen={!!resetPasswordUser}
				onClose={() => setResetPasswordUser(null)}
				user={resetPasswordUser}
			/>
			<ActivityModal
				isOpen={!!activityUser}
				onClose={() => setActivityUser(null)}
				user={activityUser}
			/>
			<ImpersonateModal
				isOpen={!!impersonateUser}
				onClose={() => setImpersonateUser(null)}
				user={impersonateUser}
			/>
		</div>
	);
}
