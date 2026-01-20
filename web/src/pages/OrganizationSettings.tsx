import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMe } from '../hooks/useAuth';
import {
	useCurrentOrganization,
	useDeleteOrganization,
	useUpdateOrganization,
} from '../hooks/useOrganizations';
import type { OrgRole } from '../lib/types';

export function OrganizationSettings() {
	const navigate = useNavigate();
	const { data: user } = useMe();
	const { data: currentOrg, isLoading } = useCurrentOrganization();
	const updateOrganization = useUpdateOrganization();
	const deleteOrganization = useDeleteOrganization();

	const [name, setName] = useState('');
	const [slug, setSlug] = useState('');
	const [isEditing, setIsEditing] = useState(false);
	const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
	const [deleteConfirmText, setDeleteConfirmText] = useState('');

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isOwner = currentUserRole === 'owner';
	const canEdit = isOwner || currentUserRole === 'admin';
	const orgId = user?.current_org_id ?? '';

	const handleEdit = () => {
		if (currentOrg) {
			setName(currentOrg.organization.name);
			setSlug(currentOrg.organization.slug);
			setIsEditing(true);
		}
	};

	const handleCancel = () => {
		setIsEditing(false);
		setName('');
		setSlug('');
	};

	const handleSave = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updateOrganization.mutateAsync({
				id: orgId,
				data: { name, slug },
			});
			setIsEditing(false);
		} catch {
			// Error handled by mutation
		}
	};

	const handleDelete = async () => {
		if (deleteConfirmText !== currentOrg?.organization.name) {
			return;
		}
		try {
			await deleteOrganization.mutateAsync(orgId);
			navigate('/');
		} catch {
			// Error handled by mutation
		}
	};

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="space-y-4">
						<div className="h-4 w-24 bg-gray-200 rounded animate-pulse" />
						<div className="h-10 w-full bg-gray-200 rounded animate-pulse" />
						<div className="h-4 w-24 bg-gray-200 rounded animate-pulse" />
						<div className="h-10 w-full bg-gray-200 rounded animate-pulse" />
					</div>
				</div>
			</div>
		);
	}

	if (!currentOrg) {
		return (
			<div className="text-center py-12">
				<p className="text-gray-500">Organization not found</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">
					Organization Settings
				</h1>
				<p className="text-gray-600 mt-1">
					Manage settings for {currentOrg.organization.name}
				</p>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900">
						General Settings
					</h2>
					{canEdit && !isEditing && (
						<button
							type="button"
							onClick={handleEdit}
							className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
						>
							Edit
						</button>
					)}
				</div>

				<div className="p-6">
					{isEditing ? (
						<form onSubmit={handleSave} className="space-y-4">
							<div>
								<label
									htmlFor="name"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Organization Name
								</label>
								<input
									type="text"
									id="name"
									value={name}
									onChange={(e) => setName(e.target.value)}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									required
								/>
							</div>
							<div>
								<label
									htmlFor="slug"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									URL Slug
								</label>
								<input
									type="text"
									id="slug"
									value={slug}
									onChange={(e) =>
										setSlug(
											e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
										)
									}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									pattern="[a-z0-9-]+"
									required
								/>
								<p className="mt-1 text-xs text-gray-500">
									Only lowercase letters, numbers, and hyphens
								</p>
							</div>
							{updateOrganization.isError && (
								<p className="text-sm text-red-600">
									Failed to update organization. Please try again.
								</p>
							)}
							<div className="flex justify-end gap-3">
								<button
									type="button"
									onClick={handleCancel}
									className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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
					) : (
						<dl className="space-y-4">
							<div>
								<dt className="text-sm font-medium text-gray-500">
									Organization Name
								</dt>
								<dd className="mt-1 text-sm text-gray-900">
									{currentOrg.organization.name}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500">URL Slug</dt>
								<dd className="mt-1 text-sm text-gray-900">
									{currentOrg.organization.slug}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500">Your Role</dt>
								<dd className="mt-1">
									<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize bg-indigo-100 text-indigo-700">
										{currentUserRole}
									</span>
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500">Created</dt>
								<dd className="mt-1 text-sm text-gray-900">
									{new Date(
										currentOrg.organization.created_at,
									).toLocaleDateString()}
								</dd>
							</div>
						</dl>
					)}
				</div>
			</div>

			{isOwner && (
				<div className="bg-white rounded-lg border border-red-200">
					<div className="px-6 py-4 border-b border-red-200 bg-red-50">
						<h2 className="text-lg font-semibold text-red-900">Danger Zone</h2>
					</div>
					<div className="p-6">
						<div className="flex items-start justify-between">
							<div>
								<h3 className="font-medium text-gray-900">
									Delete this organization
								</h3>
								<p className="text-sm text-gray-500 mt-1">
									Once you delete an organization, there is no going back. This
									will permanently delete all agents, repositories, schedules,
									and backups associated with this organization.
								</p>
							</div>
							<button
								type="button"
								onClick={() => setShowDeleteConfirm(true)}
								className="px-4 py-2 border border-red-300 text-red-600 rounded-lg hover:bg-red-50 transition-colors flex-shrink-0"
							>
								Delete Organization
							</button>
						</div>
					</div>
				</div>
			)}

			{showDeleteConfirm && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
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
							<h3 className="text-lg font-semibold text-gray-900">
								Delete Organization
							</h3>
						</div>
						<p className="text-sm text-gray-600 mb-4">
							This action cannot be undone. This will permanently delete the{' '}
							<strong>{currentOrg.organization.name}</strong> organization and
							all of its data.
						</p>
						<p className="text-sm text-gray-600 mb-4">
							Please type <strong>{currentOrg.organization.name}</strong> to
							confirm.
						</p>
						<input
							type="text"
							value={deleteConfirmText}
							onChange={(e) => setDeleteConfirmText(e.target.value)}
							placeholder="Type organization name to confirm"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-red-500 focus:border-red-500 mb-4"
						/>
						{deleteOrganization.isError && (
							<p className="text-sm text-red-600 mb-4">
								Failed to delete organization. Please try again.
							</p>
						)}
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => {
									setShowDeleteConfirm(false);
									setDeleteConfirmText('');
								}}
								className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleDelete}
								disabled={
									deleteConfirmText !== currentOrg.organization.name ||
									deleteOrganization.isPending
								}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{deleteOrganization.isPending
									? 'Deleting...'
									: 'Delete Organization'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
