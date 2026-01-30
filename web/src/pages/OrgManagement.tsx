import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useAdminCreateOrganization,
	useAdminDeleteOrganization,
	useAdminOrganizations,
	useAdminOrgUsageStats,
	useAdminTransferOwnership,
	useAdminUpdateOrganization,
} from '../hooks/useAdminOrganizations';
import type {
	AdminCreateOrgRequest,
	AdminOrganization,
	AdminOrgSettings,
	OrgFeatureFlags,
} from '../lib/types';

function formatBytes(bytes: number | undefined): string {
	if (bytes === undefined || bytes === null) return '-';
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

function formatDate(dateStr: string): string {
	return new Date(dateStr).toLocaleDateString(undefined, {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
	});
}

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

interface CreateOrgModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateOrgModal({ isOpen, onClose }: CreateOrgModalProps) {
	const createOrganization = useAdminCreateOrganization();
	const [formData, setFormData] = useState<AdminCreateOrgRequest>({
		name: '',
		slug: '',
		owner_email: '',
		storage_quota_bytes: undefined,
		agent_limit: undefined,
		feature_flags: {},
	});
	const [autoSlug, setAutoSlug] = useState(true);

	const handleNameChange = (value: string) => {
		setFormData((prev) => ({ ...prev, name: value }));
		if (autoSlug) {
			setFormData((prev) => ({
				...prev,
				slug: value
					.toLowerCase()
					.replace(/[^a-z0-9]+/g, '-')
					.replace(/^-|-$/g, ''),
			}));
		}
	};

	const handleSlugChange = (value: string) => {
		setAutoSlug(false);
		setFormData((prev) => ({
			...prev,
			slug: value.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
		}));
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createOrganization.mutateAsync(formData);
			onClose();
			setFormData({
				name: '',
				slug: '',
				owner_email: '',
				storage_quota_bytes: undefined,
				agent_limit: undefined,
				feature_flags: {},
			});
			setAutoSlug(true);
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Create New Organization
				</h3>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div>
						<label
							htmlFor="name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Organization Name *
						</label>
						<input
							type="text"
							id="name"
							value={formData.name}
							onChange={(e) => handleNameChange(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>
					<div>
						<label
							htmlFor="slug"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							URL Slug *
						</label>
						<input
							type="text"
							id="slug"
							value={formData.slug}
							onChange={(e) => handleSlugChange(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							pattern="[a-z0-9-]+"
							required
						/>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Only lowercase letters, numbers, and hyphens
						</p>
					</div>
					<div>
						<label
							htmlFor="owner_email"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Owner Email *
						</label>
						<input
							type="email"
							id="owner_email"
							value={formData.owner_email}
							onChange={(e) =>
								setFormData((prev) => ({
									...prev,
									owner_email: e.target.value,
								}))
							}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>
					<div>
						<label
							htmlFor="logo_url"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Logo URL
						</label>
						<input
							type="url"
							id="logo_url"
							value={formData.logo_url || ''}
							onChange={(e) =>
								setFormData((prev) => ({
									...prev,
									logo_url: e.target.value || undefined,
								}))
							}
							placeholder="https://example.com/logo.png"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="storage_quota"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Storage Quota (GB)
							</label>
							<input
								type="number"
								id="storage_quota"
								value={
									formData.storage_quota_bytes
										? formData.storage_quota_bytes / (1024 * 1024 * 1024)
										: ''
								}
								onChange={(e) =>
									setFormData((prev) => ({
										...prev,
										storage_quota_bytes: e.target.value
											? Number(e.target.value) * 1024 * 1024 * 1024
											: undefined,
									}))
								}
								min="0"
								placeholder="Unlimited"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="agent_limit"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Agent Limit
							</label>
							<input
								type="number"
								id="agent_limit"
								value={formData.agent_limit ?? ''}
								onChange={(e) =>
									setFormData((prev) => ({
										...prev,
										agent_limit: e.target.value
											? Number(e.target.value)
											: undefined,
									}))
								}
								min="0"
								placeholder="Unlimited"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
					</div>

					<div>
						<label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Feature Flags
						</label>
						<div className="space-y-2">
							{[
								{ key: 'sso_enabled', label: 'SSO Enabled' },
								{ key: 'api_access', label: 'API Access' },
								{ key: 'advanced_reporting', label: 'Advanced Reporting' },
								{ key: 'custom_branding', label: 'Custom Branding' },
								{ key: 'priority_support', label: 'Priority Support' },
							].map(({ key, label }) => (
								<label key={key} className="flex items-center gap-2">
									<input
										type="checkbox"
										checked={
											formData.feature_flags?.[
												key as keyof OrgFeatureFlags
											] ?? false
										}
										onChange={(e) =>
											setFormData((prev) => ({
												...prev,
												feature_flags: {
													...prev.feature_flags,
													[key]: e.target.checked,
												},
											}))
										}
										className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="text-sm text-gray-700 dark:text-gray-300">
										{label}
									</span>
								</label>
							))}
						</div>
					</div>

					{createOrganization.isError && (
						<p className="text-sm text-red-600 dark:text-red-400">
							Failed to create organization. Please try again.
						</p>
					)}

					<div className="flex justify-end gap-3 pt-4">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createOrganization.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createOrganization.isPending ? 'Creating...' : 'Create Organization'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface EditOrgModalProps {
	isOpen: boolean;
	onClose: () => void;
	organization: AdminOrganization;
}

function EditOrgModal({ isOpen, onClose, organization }: EditOrgModalProps) {
	const updateOrganization = useAdminUpdateOrganization();
	const { data: usageStats, isLoading: usageLoading } = useAdminOrgUsageStats(
		organization.id,
	);
	const [formData, setFormData] = useState<AdminOrgSettings>({
		name: organization.name,
		slug: organization.slug,
		logo_url: organization.logo_url,
		storage_quota_bytes: organization.storage_quota_bytes,
		agent_limit: organization.agent_limit,
		feature_flags: organization.feature_flags,
	});

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updateOrganization.mutateAsync({
				id: organization.id,
				data: formData,
			});
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Edit Organization: {organization.name}
				</h3>

				{/* Usage Stats */}
				<div className="mb-6 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
					<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
						Current Usage
					</h4>
					{usageLoading ? (
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
							{[1, 2, 3, 4].map((i) => (
								<div key={i} className="animate-pulse">
									<div className="h-3 w-16 bg-gray-200 dark:bg-gray-700 rounded mb-1" />
									<div className="h-5 w-12 bg-gray-200 dark:bg-gray-700 rounded" />
								</div>
							))}
						</div>
					) : usageStats ? (
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
							<div>
								<p className="text-xs text-gray-500 dark:text-gray-400">
									Storage Used
								</p>
								<p className="text-sm font-medium text-gray-900 dark:text-white">
									{formatBytes(usageStats.storage_used_bytes)}
									{usageStats.storage_quota_bytes && (
										<span className="text-gray-500 dark:text-gray-400">
											{' '}
											/ {formatBytes(usageStats.storage_quota_bytes)}
										</span>
									)}
								</p>
							</div>
							<div>
								<p className="text-xs text-gray-500 dark:text-gray-400">
									Agents
								</p>
								<p className="text-sm font-medium text-gray-900 dark:text-white">
									{usageStats.agent_count}
									{usageStats.agent_limit && (
										<span className="text-gray-500 dark:text-gray-400">
											{' '}
											/ {usageStats.agent_limit}
										</span>
									)}
								</p>
							</div>
							<div>
								<p className="text-xs text-gray-500 dark:text-gray-400">
									Members
								</p>
								<p className="text-sm font-medium text-gray-900 dark:text-white">
									{usageStats.member_count}
								</p>
							</div>
							<div>
								<p className="text-xs text-gray-500 dark:text-gray-400">
									Backups
								</p>
								<p className="text-sm font-medium text-gray-900 dark:text-white">
									{usageStats.backup_count}
								</p>
							</div>
						</div>
					) : null}
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div>
						<label
							htmlFor="edit-name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Organization Name
						</label>
						<input
							type="text"
							id="edit-name"
							value={formData.name ?? ''}
							onChange={(e) =>
								setFormData((prev) => ({ ...prev, name: e.target.value }))
							}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div>
						<label
							htmlFor="edit-slug"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							URL Slug
						</label>
						<input
							type="text"
							id="edit-slug"
							value={formData.slug ?? ''}
							onChange={(e) =>
								setFormData((prev) => ({
									...prev,
									slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
								}))
							}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							pattern="[a-z0-9-]+"
						/>
					</div>
					<div>
						<label
							htmlFor="edit-logo"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Logo URL
						</label>
						<input
							type="url"
							id="edit-logo"
							value={formData.logo_url ?? ''}
							onChange={(e) =>
								setFormData((prev) => ({
									...prev,
									logo_url: e.target.value || undefined,
								}))
							}
							placeholder="https://example.com/logo.png"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="edit-storage"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Storage Quota (GB)
							</label>
							<input
								type="number"
								id="edit-storage"
								value={
									formData.storage_quota_bytes
										? formData.storage_quota_bytes / (1024 * 1024 * 1024)
										: ''
								}
								onChange={(e) =>
									setFormData((prev) => ({
										...prev,
										storage_quota_bytes: e.target.value
											? Number(e.target.value) * 1024 * 1024 * 1024
											: undefined,
									}))
								}
								min="0"
								placeholder="Unlimited"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="edit-agents"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Agent Limit
							</label>
							<input
								type="number"
								id="edit-agents"
								value={formData.agent_limit ?? ''}
								onChange={(e) =>
									setFormData((prev) => ({
										...prev,
										agent_limit: e.target.value
											? Number(e.target.value)
											: undefined,
									}))
								}
								min="0"
								placeholder="Unlimited"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
					</div>

					<div>
						<label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Feature Flags
						</label>
						<div className="grid grid-cols-2 gap-2">
							{[
								{ key: 'sso_enabled', label: 'SSO Enabled' },
								{ key: 'api_access', label: 'API Access' },
								{ key: 'advanced_reporting', label: 'Advanced Reporting' },
								{ key: 'custom_branding', label: 'Custom Branding' },
								{ key: 'priority_support', label: 'Priority Support' },
							].map(({ key, label }) => (
								<label key={key} className="flex items-center gap-2">
									<input
										type="checkbox"
										checked={
											formData.feature_flags?.[
												key as keyof OrgFeatureFlags
											] ?? false
										}
										onChange={(e) =>
											setFormData((prev) => ({
												...prev,
												feature_flags: {
													...prev.feature_flags,
													[key]: e.target.checked,
												},
											}))
										}
										className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="text-sm text-gray-700 dark:text-gray-300">
										{label}
									</span>
								</label>
							))}
						</div>
					</div>

					{/* Billing Settings (Future) */}
					<div className="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
						<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Billing Settings
						</h4>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							Billing configuration will be available in a future release.
						</p>
					</div>

					{updateOrganization.isError && (
						<p className="text-sm text-red-600 dark:text-red-400">
							Failed to update organization. Please try again.
						</p>
					)}

					<div className="flex justify-end gap-3 pt-4">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={updateOrganization.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{updateOrganization.isPending ? 'Saving...' : 'Save Changes'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface DeleteOrgModalProps {
	isOpen: boolean;
	onClose: () => void;
	organization: AdminOrganization;
}

function DeleteOrgModal({ isOpen, onClose, organization }: DeleteOrgModalProps) {
	const deleteOrganization = useAdminDeleteOrganization();
	const [confirmText, setConfirmText] = useState('');

	const handleDelete = async () => {
		if (confirmText !== organization.name) return;
		try {
			await deleteOrganization.mutateAsync(organization.id);
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-red-100 rounded-full">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-red-600"
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
					</div>
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Delete Organization
					</h3>
				</div>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					This action cannot be undone. This will permanently delete the{' '}
					<strong>{organization.name}</strong> organization and all of its data
					including agents, backups, schedules, and members.
				</p>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					Please type <strong>{organization.name}</strong> to confirm.
				</p>
				<input
					type="text"
					value={confirmText}
					onChange={(e) => setConfirmText(e.target.value)}
					placeholder="Type organization name to confirm"
					className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-red-500 focus:border-red-500 mb-4"
				/>
				{deleteOrganization.isError && (
					<p className="text-sm text-red-600 dark:text-red-400 mb-4">
						Failed to delete organization. Please try again.
					</p>
				)}
				<div className="flex justify-end gap-3">
					<button
						type="button"
						onClick={() => {
							onClose();
							setConfirmText('');
						}}
						className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
					>
						Cancel
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={
							confirmText !== organization.name || deleteOrganization.isPending
						}
						className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
					>
						{deleteOrganization.isPending ? 'Deleting...' : 'Delete Organization'}
					</button>
				</div>
			</div>
		</div>
	);
}

interface TransferOwnershipModalProps {
	isOpen: boolean;
	onClose: () => void;
	organization: AdminOrganization;
}

function TransferOwnershipModal({
	isOpen,
	onClose,
	organization,
}: TransferOwnershipModalProps) {
	const transferOwnership = useAdminTransferOwnership();
	const [newOwnerUserId, setNewOwnerUserId] = useState('');

	const handleTransfer = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await transferOwnership.mutateAsync({
				orgId: organization.id,
				data: { new_owner_user_id: newOwnerUserId },
			});
			onClose();
			setNewOwnerUserId('');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Transfer Ownership
				</h3>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					Transfer ownership of <strong>{organization.name}</strong> to another
					user. The new owner will have full administrative control.
				</p>
				{organization.owner_email && (
					<p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
						Current owner: {organization.owner_email}
					</p>
				)}
				<form onSubmit={handleTransfer}>
					<div className="mb-4">
						<label
							htmlFor="new_owner"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							New Owner User ID
						</label>
						<input
							type="text"
							id="new_owner"
							value={newOwnerUserId}
							onChange={(e) => setNewOwnerUserId(e.target.value)}
							placeholder="Enter user ID"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>
					{transferOwnership.isError && (
						<p className="text-sm text-red-600 dark:text-red-400 mb-4">
							Failed to transfer ownership. Please verify the user ID.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={() => {
								onClose();
								setNewOwnerUserId('');
							}}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={!newOwnerUserId || transferOwnership.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{transferOwnership.isPending ? 'Transferring...' : 'Transfer'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

export function OrgManagement() {
	const { data: user, isLoading: userLoading } = useMe();
	const [search, setSearch] = useState('');
	const [searchInput, setSearchInput] = useState('');
	const [page, setPage] = useState(0);
	const pageSize = 20;

	const { data, isLoading, isError } = useAdminOrganizations({
		search: search || undefined,
		limit: pageSize,
		offset: page * pageSize,
	});

	const [showCreateModal, setShowCreateModal] = useState(false);
	const [editingOrg, setEditingOrg] = useState<AdminOrganization | null>(null);
	const [deletingOrg, setDeletingOrg] = useState<AdminOrganization | null>(null);
	const [transferringOrg, setTransferringOrg] =
		useState<AdminOrganization | null>(null);

	const handleSearch = () => {
		setSearch(searchInput);
		setPage(0);
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter') {
			handleSearch();
		}
	};

	// Check if user is a superuser
	if (userLoading) {
		return (
			<div className="space-y-6">
				<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div
								key={i}
								className="h-12 bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
							/>
						))}
					</div>
				</div>
			</div>
		);
	}

	if (!user?.is_superuser) {
		return (
			<div className="text-center py-12">
				<svg
					aria-hidden="true"
					className="w-16 h-16 mx-auto mb-4 text-gray-300"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
					/>
				</svg>
				<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
					Access Denied
				</h2>
				<p className="text-gray-500 dark:text-gray-400">
					You need superuser privileges to access this page.
				</p>
			</div>
		);
	}

	const totalPages = data ? Math.ceil(data.total_count / pageSize) : 0;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Organization Management
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Manage all organizations in the system
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowCreateModal(true)}
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
					Create Organization
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-4 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<div className="flex-1">
							<input
								type="text"
								placeholder="Search organizations..."
								value={searchInput}
								onChange={(e) => setSearchInput(e.target.value)}
								onKeyDown={handleKeyDown}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<button
							type="button"
							onClick={handleSearch}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Search
						</button>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load organizations</p>
						<p className="text-sm">Please try again later</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Organization
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Owner
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Members
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Agents
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Storage
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Created
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
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
				) : data && data.organizations.length > 0 ? (
					<>
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Organization
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Owner
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Members
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Agents
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Storage
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{data.organizations.map((org) => (
										<tr
											key={org.id}
											className="hover:bg-gray-50 dark:hover:bg-gray-700"
										>
											<td className="px-6 py-4">
												<div className="flex items-center gap-3">
													{org.logo_url ? (
														<img
															src={org.logo_url}
															alt={org.name}
															className="w-8 h-8 rounded object-cover"
														/>
													) : (
														<div className="w-8 h-8 rounded bg-indigo-100 dark:bg-indigo-900 flex items-center justify-center">
															<span className="text-indigo-600 dark:text-indigo-400 text-sm font-medium">
																{org.name.charAt(0).toUpperCase()}
															</span>
														</div>
													)}
													<div>
														<p className="font-medium text-gray-900 dark:text-white">
															{org.name}
														</p>
														<p className="text-sm text-gray-500 dark:text-gray-400">
															{org.slug}
														</p>
													</div>
												</div>
											</td>
											<td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
												{org.owner_email || '-'}
											</td>
											<td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
												{org.member_count}
											</td>
											<td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
												{org.agent_count ?? 0}
												{org.agent_limit && (
													<span className="text-gray-400">
														{' '}
														/ {org.agent_limit}
													</span>
												)}
											</td>
											<td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
												{formatBytes(org.storage_used_bytes)}
												{org.storage_quota_bytes && (
													<span className="text-gray-400">
														{' '}
														/ {formatBytes(org.storage_quota_bytes)}
													</span>
												)}
											</td>
											<td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
												{formatDate(org.created_at)}
											</td>
											<td className="px-6 py-4">
												<div className="flex items-center gap-2">
													<button
														type="button"
														onClick={() => setEditingOrg(org)}
														className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
													>
														Edit
													</button>
													<button
														type="button"
														onClick={() => setTransferringOrg(org)}
														className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 text-sm font-medium"
													>
														Transfer
													</button>
													<button
														type="button"
														onClick={() => setDeletingOrg(org)}
														className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 text-sm font-medium"
													>
														Delete
													</button>
												</div>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>

						{totalPages > 1 && (
							<div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
								<div className="text-sm text-gray-500 dark:text-gray-400">
									Showing {page * pageSize + 1} to{' '}
									{Math.min((page + 1) * pageSize, data.total_count)} of{' '}
									{data.total_count} organizations
								</div>
								<div className="flex items-center gap-2">
									<button
										type="button"
										onClick={() => setPage((p) => Math.max(0, p - 1))}
										disabled={page === 0}
										className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
									>
										Previous
									</button>
									<span className="text-sm text-gray-700 dark:text-gray-300">
										Page {page + 1} of {totalPages}
									</span>
									<button
										type="button"
										onClick={() => setPage((p) => p + 1)}
										disabled={page >= totalPages - 1}
										className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
									>
										Next
									</button>
								</div>
							</div>
						)}
					</>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<svg
							aria-hidden="true"
							className="w-16 h-16 mx-auto mb-4 text-gray-300"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No organizations found
						</h3>
						<p>
							{search
								? 'Try adjusting your search criteria'
								: 'Create your first organization to get started'}
						</p>
					</div>
				)}
			</div>

			<CreateOrgModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			{editingOrg && (
				<EditOrgModal
					isOpen={!!editingOrg}
					onClose={() => setEditingOrg(null)}
					organization={editingOrg}
				/>
			)}

			{deletingOrg && (
				<DeleteOrgModal
					isOpen={!!deletingOrg}
					onClose={() => setDeletingOrg(null)}
					organization={deletingOrg}
				/>
			)}

			{transferringOrg && (
				<TransferOwnershipModal
					isOpen={!!transferringOrg}
					onClose={() => setTransferringOrg(null)}
					organization={transferringOrg}
				/>
			)}
		</div>
	);
}
