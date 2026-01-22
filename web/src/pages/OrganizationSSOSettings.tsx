import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useCreateSSOGroupMapping,
	useDeleteSSOGroupMapping,
	useSSOGroupMappings,
	useSSOSettings,
	useUpdateSSOGroupMapping,
	useUpdateSSOSettings,
} from '../hooks/useSSOGroupMappings';
import type { OrgRole, SSOGroupMapping } from '../lib/types';

const roleOptions: OrgRole[] = ['owner', 'admin', 'member', 'readonly'];

export function OrganizationSSOSettings() {
	const { data: user } = useMe();
	const orgId = user?.current_org_id ?? '';
	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const canEdit = currentUserRole === 'owner' || currentUserRole === 'admin';

	const { data: mappings, isLoading: mappingsLoading } =
		useSSOGroupMappings(orgId);
	const { data: settings, isLoading: settingsLoading } = useSSOSettings(orgId);
	const createMapping = useCreateSSOGroupMapping();
	const updateMapping = useUpdateSSOGroupMapping();
	const deleteMapping = useDeleteSSOGroupMapping();
	const updateSettings = useUpdateSSOSettings();

	const [showAddModal, setShowAddModal] = useState(false);
	const [editingMapping, setEditingMapping] = useState<SSOGroupMapping | null>(
		null,
	);
	const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

	// Form state for new mapping
	const [newGroupName, setNewGroupName] = useState('');
	const [newRole, setNewRole] = useState<OrgRole>('member');
	const [newAutoCreate, setNewAutoCreate] = useState(false);

	// Form state for settings
	const [defaultRole, setDefaultRole] = useState<OrgRole | ''>('');
	const [autoCreateOrgs, setAutoCreateOrgs] = useState(false);
	const [settingsEditing, setSettingsEditing] = useState(false);

	const handleAddMapping = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!newGroupName.trim()) return;

		try {
			await createMapping.mutateAsync({
				orgId,
				data: {
					oidc_group_name: newGroupName.trim(),
					role: newRole,
					auto_create_org: newAutoCreate,
				},
			});
			setShowAddModal(false);
			setNewGroupName('');
			setNewRole('member');
			setNewAutoCreate(false);
		} catch {
			// Error handled by mutation
		}
	};

	const handleUpdateMapping = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!editingMapping) return;

		try {
			await updateMapping.mutateAsync({
				orgId,
				id: editingMapping.id,
				data: {
					role: editingMapping.role,
					auto_create_org: editingMapping.auto_create_org,
				},
			});
			setEditingMapping(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleDeleteMapping = async (id: string) => {
		try {
			await deleteMapping.mutateAsync({ orgId, id });
			setDeleteConfirm(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleSaveSettings = async () => {
		try {
			await updateSettings.mutateAsync({
				orgId,
				data: {
					default_role: defaultRole === '' ? null : defaultRole,
					auto_create_orgs: autoCreateOrgs,
				},
			});
			setSettingsEditing(false);
		} catch {
			// Error handled by mutation
		}
	};

	const startEditSettings = () => {
		setDefaultRole(settings?.default_role ?? '');
		setAutoCreateOrgs(settings?.auto_create_orgs ?? false);
		setSettingsEditing(true);
	};

	if (mappingsLoading || settingsLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div
								key={i}
								className="h-12 w-full bg-gray-200 rounded animate-pulse"
							/>
						))}
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">
					SSO Group Sync Settings
				</h1>
				<p className="text-gray-600 mt-1">
					Map OIDC groups from your identity provider to Keldris organization
					roles
				</p>
			</div>

			{/* SSO Settings */}
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900">
						Default Settings
					</h2>
					{canEdit && !settingsEditing && (
						<button
							type="button"
							onClick={startEditSettings}
							className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
						>
							Edit
						</button>
					)}
				</div>
				<div className="p-6">
					{settingsEditing ? (
						<div className="space-y-4">
							<div>
								<label
									htmlFor="defaultRole"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Default Role for Unmapped Groups
								</label>
								<select
									id="defaultRole"
									value={defaultRole}
									onChange={(e) =>
										setDefaultRole(e.target.value as OrgRole | '')
									}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									<option value="">No default (require explicit mapping)</option>
									{roleOptions.map((role) => (
										<option key={role} value={role}>
											{role.charAt(0).toUpperCase() + role.slice(1)}
										</option>
									))}
								</select>
								<p className="mt-1 text-xs text-gray-500">
									Users with unmapped OIDC groups will be assigned this role
								</p>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="autoCreateOrgs"
									checked={autoCreateOrgs}
									onChange={(e) => setAutoCreateOrgs(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="autoCreateOrgs"
									className="text-sm text-gray-700"
								>
									Auto-create organizations from OIDC groups
								</label>
							</div>
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => setSettingsEditing(false)}
									className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleSaveSettings}
									disabled={updateSettings.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateSettings.isPending ? 'Saving...' : 'Save Settings'}
								</button>
							</div>
						</div>
					) : (
						<dl className="space-y-4">
							<div>
								<dt className="text-sm font-medium text-gray-500">
									Default Role for Unmapped Groups
								</dt>
								<dd className="mt-1 text-sm text-gray-900">
									{settings?.default_role ? (
										<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize bg-indigo-100 text-indigo-700">
											{settings.default_role}
										</span>
									) : (
										<span className="text-gray-500">
											No default (require explicit mapping)
										</span>
									)}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500">
									Auto-create Organizations
								</dt>
								<dd className="mt-1 text-sm text-gray-900">
									{settings?.auto_create_orgs ? 'Enabled' : 'Disabled'}
								</dd>
							</div>
						</dl>
					)}
				</div>
			</div>

			{/* Group Mappings */}
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900">Group Mappings</h2>
					{canEdit && (
						<button
							type="button"
							onClick={() => setShowAddModal(true)}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors text-sm"
						>
							Add Mapping
						</button>
					)}
				</div>
				<div className="p-6">
					{mappings && mappings.length > 0 ? (
						<table className="w-full">
							<thead>
								<tr className="text-left text-sm text-gray-500 border-b border-gray-200">
									<th className="pb-3 font-medium">OIDC Group Name</th>
									<th className="pb-3 font-medium">Role</th>
									<th className="pb-3 font-medium">Auto Create Org</th>
									{canEdit && (
										<th className="pb-3 font-medium text-right">Actions</th>
									)}
								</tr>
							</thead>
							<tbody>
								{mappings.map((mapping) => (
									<tr key={mapping.id} className="border-b border-gray-100">
										<td className="py-3 text-sm text-gray-900">
											{mapping.oidc_group_name}
										</td>
										<td className="py-3">
											<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize bg-indigo-100 text-indigo-700">
												{mapping.role}
											</span>
										</td>
										<td className="py-3 text-sm text-gray-600">
											{mapping.auto_create_org ? 'Yes' : 'No'}
										</td>
										{canEdit && (
											<td className="py-3 text-right">
												<button
													type="button"
													onClick={() => setEditingMapping({ ...mapping })}
													className="text-indigo-600 hover:text-indigo-800 text-sm mr-3"
												>
													Edit
												</button>
												<button
													type="button"
													onClick={() => setDeleteConfirm(mapping.id)}
													className="text-red-600 hover:text-red-800 text-sm"
												>
													Delete
												</button>
											</td>
										)}
									</tr>
								))}
							</tbody>
						</table>
					) : (
						<div className="text-center py-8">
							<p className="text-gray-500">No group mappings configured yet</p>
							{canEdit && (
								<button
									type="button"
									onClick={() => setShowAddModal(true)}
									className="mt-4 text-indigo-600 hover:text-indigo-800 text-sm font-medium"
								>
									Add your first mapping
								</button>
							)}
						</div>
					)}
				</div>
			</div>

			{/* Add Mapping Modal */}
			{showAddModal && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Add Group Mapping
						</h3>
						<form onSubmit={handleAddMapping} className="space-y-4">
							<div>
								<label
									htmlFor="groupName"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									OIDC Group Name
								</label>
								<input
									type="text"
									id="groupName"
									value={newGroupName}
									onChange={(e) => setNewGroupName(e.target.value)}
									placeholder="e.g., engineering, admins"
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									required
								/>
								<p className="mt-1 text-xs text-gray-500">
									The exact group name as it appears in your OIDC provider
								</p>
							</div>
							<div>
								<label
									htmlFor="role"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Keldris Role
								</label>
								<select
									id="role"
									value={newRole}
									onChange={(e) => setNewRole(e.target.value as OrgRole)}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									{roleOptions.map((role) => (
										<option key={role} value={role}>
											{role.charAt(0).toUpperCase() + role.slice(1)}
										</option>
									))}
								</select>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="autoCreate"
									checked={newAutoCreate}
									onChange={(e) => setNewAutoCreate(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label htmlFor="autoCreate" className="text-sm text-gray-700">
									Auto-create organization for this group
								</label>
							</div>
							{createMapping.isError && (
								<p className="text-sm text-red-600">
									Failed to create mapping. The group may already be mapped.
								</p>
							)}
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => {
										setShowAddModal(false);
										setNewGroupName('');
										setNewRole('member');
										setNewAutoCreate(false);
									}}
									className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={createMapping.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{createMapping.isPending ? 'Adding...' : 'Add Mapping'}
								</button>
							</div>
						</form>
					</div>
				</div>
			)}

			{/* Edit Mapping Modal */}
			{editingMapping && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Edit Group Mapping
						</h3>
						<form onSubmit={handleUpdateMapping} className="space-y-4">
							<div>
								<label className="block text-sm font-medium text-gray-700 mb-1">
									OIDC Group Name
								</label>
								<p className="text-sm text-gray-900 py-2">
									{editingMapping.oidc_group_name}
								</p>
							</div>
							<div>
								<label
									htmlFor="editRole"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Keldris Role
								</label>
								<select
									id="editRole"
									value={editingMapping.role}
									onChange={(e) =>
										setEditingMapping({
											...editingMapping,
											role: e.target.value as OrgRole,
										})
									}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									{roleOptions.map((role) => (
										<option key={role} value={role}>
											{role.charAt(0).toUpperCase() + role.slice(1)}
										</option>
									))}
								</select>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="editAutoCreate"
									checked={editingMapping.auto_create_org}
									onChange={(e) =>
										setEditingMapping({
											...editingMapping,
											auto_create_org: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="editAutoCreate"
									className="text-sm text-gray-700"
								>
									Auto-create organization for this group
								</label>
							</div>
							{updateMapping.isError && (
								<p className="text-sm text-red-600">
									Failed to update mapping. Please try again.
								</p>
							)}
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => setEditingMapping(null)}
									className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={updateMapping.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateMapping.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						</form>
					</div>
				</div>
			)}

			{/* Delete Confirmation Modal */}
			{deleteConfirm && (
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
								Delete Mapping
							</h3>
						</div>
						<p className="text-sm text-gray-600 mb-4">
							Are you sure you want to delete this group mapping? Users in this
							OIDC group will no longer be automatically assigned to this
							organization.
						</p>
						{deleteMapping.isError && (
							<p className="text-sm text-red-600 mb-4">
								Failed to delete mapping. Please try again.
							</p>
						)}
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setDeleteConfirm(null)}
								className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={() => handleDeleteMapping(deleteConfirm)}
								disabled={deleteMapping.isPending}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{deleteMapping.isPending ? 'Deleting...' : 'Delete'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
