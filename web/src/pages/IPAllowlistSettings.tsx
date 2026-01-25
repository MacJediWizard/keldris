import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useCreateIPAllowlist,
	useDeleteIPAllowlist,
	useIPAllowlistSettings,
	useIPAllowlists,
	useIPBlockedAttempts,
	useUpdateIPAllowlist,
	useUpdateIPAllowlistSettings,
} from '../hooks/useIPAllowlists';
import type { IPAllowlist, IPAllowlistType, OrgRole } from '../lib/types';

const typeOptions: IPAllowlistType[] = ['ui', 'agent', 'both'];
const typeLabels: Record<IPAllowlistType, string> = {
	ui: 'UI Only',
	agent: 'Agent Only',
	both: 'Both',
};

export function IPAllowlistSettings() {
	const { data: user } = useMe();
	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const canEdit = currentUserRole === 'owner' || currentUserRole === 'admin';

	const { data: allowlists, isLoading: allowlistsLoading } = useIPAllowlists();
	const { data: settings, isLoading: settingsLoading } =
		useIPAllowlistSettings();
	const { data: blockedAttempts, isLoading: attemptsLoading } =
		useIPBlockedAttempts(20, 0);

	const createAllowlist = useCreateIPAllowlist();
	const updateAllowlist = useUpdateIPAllowlist();
	const deleteAllowlist = useDeleteIPAllowlist();
	const updateSettings = useUpdateIPAllowlistSettings();

	const [showAddModal, setShowAddModal] = useState(false);
	const [editingAllowlist, setEditingAllowlist] = useState<IPAllowlist | null>(
		null,
	);
	const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

	// Form state for new allowlist
	const [newCIDR, setNewCIDR] = useState('');
	const [newDescription, setNewDescription] = useState('');
	const [newType, setNewType] = useState<IPAllowlistType>('both');
	const [newEnabled, setNewEnabled] = useState(true);

	// Settings state
	const [settingsEditing, setSettingsEditing] = useState(false);
	const [localEnabled, setLocalEnabled] = useState(false);
	const [localEnforceUI, setLocalEnforceUI] = useState(true);
	const [localEnforceAgent, setLocalEnforceAgent] = useState(true);
	const [localAdminBypass, setLocalAdminBypass] = useState(true);

	const handleAddAllowlist = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!newCIDR.trim()) return;

		try {
			await createAllowlist.mutateAsync({
				cidr: newCIDR.trim(),
				description: newDescription.trim() || undefined,
				type: newType,
				enabled: newEnabled,
			});
			setShowAddModal(false);
			resetForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleUpdateAllowlist = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!editingAllowlist) return;

		try {
			await updateAllowlist.mutateAsync({
				id: editingAllowlist.id,
				data: {
					cidr: editingAllowlist.cidr,
					description: editingAllowlist.description,
					type: editingAllowlist.type,
					enabled: editingAllowlist.enabled,
				},
			});
			setEditingAllowlist(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleDeleteAllowlist = async (id: string) => {
		try {
			await deleteAllowlist.mutateAsync(id);
			setDeleteConfirm(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleSaveSettings = async () => {
		try {
			await updateSettings.mutateAsync({
				enabled: localEnabled,
				enforce_for_ui: localEnforceUI,
				enforce_for_agent: localEnforceAgent,
				allow_admin_bypass: localAdminBypass,
			});
			setSettingsEditing(false);
		} catch {
			// Error handled by mutation
		}
	};

	const startEditSettings = () => {
		setLocalEnabled(settings?.enabled ?? false);
		setLocalEnforceUI(settings?.enforce_for_ui ?? true);
		setLocalEnforceAgent(settings?.enforce_for_agent ?? true);
		setLocalAdminBypass(settings?.allow_admin_bypass ?? true);
		setSettingsEditing(true);
	};

	const resetForm = () => {
		setNewCIDR('');
		setNewDescription('');
		setNewType('both');
		setNewEnabled(true);
	};

	if (allowlistsLoading || settingsLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div
								key={i}
								className="h-12 w-full bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
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
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					IP Allowlist Settings
				</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
					Restrict access to your organization by IP address or CIDR range
				</p>
			</div>

			{/* Settings Card */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Enforcement Settings
					</h2>
					{canEdit && !settingsEditing && (
						<button
							type="button"
							onClick={startEditSettings}
							className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
						>
							Edit
						</button>
					)}
				</div>
				<div className="p-6">
					{settingsEditing ? (
						<div className="space-y-4">
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="enabled"
									checked={localEnabled}
									onChange={(e) => setLocalEnabled(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="enabled"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enable IP allowlist enforcement
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="enforceUI"
									checked={localEnforceUI}
									onChange={(e) => setLocalEnforceUI(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="enforceUI"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enforce for web UI access
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="enforceAgent"
									checked={localEnforceAgent}
									onChange={(e) => setLocalEnforceAgent(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="enforceAgent"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enforce for agent connections
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="adminBypass"
									checked={localAdminBypass}
									onChange={(e) => setLocalAdminBypass(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="adminBypass"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Allow admin/owner bypass (emergency access)
								</label>
							</div>
							{localEnabled && !localAdminBypass && (
								<p className="text-sm text-yellow-600 dark:text-yellow-400 bg-yellow-50 dark:bg-yellow-900/20 p-3 rounded-lg">
									Warning: Disabling admin bypass could lock you out if you
									access from an unlisted IP. Make sure your IP is in the
									allowlist.
								</p>
							)}
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => setSettingsEditing(false)}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
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
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Status
								</dt>
								<dd className="mt-1">
									<span
										className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
											settings?.enabled
												? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
												: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
										}`}
									>
										{settings?.enabled ? 'Enabled' : 'Disabled'}
									</span>
								</dd>
							</div>
							<div className="grid grid-cols-3 gap-4">
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										UI Enforcement
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{settings?.enforce_for_ui ? 'Yes' : 'No'}
									</dd>
								</div>
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Agent Enforcement
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{settings?.enforce_for_agent ? 'Yes' : 'No'}
									</dd>
								</div>
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Admin Bypass
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{settings?.allow_admin_bypass ? 'Enabled' : 'Disabled'}
									</dd>
								</div>
							</div>
						</dl>
					)}
				</div>
			</div>

			{/* Allowlist Entries */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Allowed IP Ranges
					</h2>
					{canEdit && (
						<button
							type="button"
							onClick={() => setShowAddModal(true)}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors text-sm"
						>
							Add IP Range
						</button>
					)}
				</div>
				<div className="p-6">
					{allowlists && allowlists.length > 0 ? (
						<table className="w-full">
							<thead>
								<tr className="text-left text-sm text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
									<th className="pb-3 font-medium">CIDR / IP</th>
									<th className="pb-3 font-medium">Description</th>
									<th className="pb-3 font-medium">Type</th>
									<th className="pb-3 font-medium">Status</th>
									{canEdit && (
										<th className="pb-3 font-medium text-right">Actions</th>
									)}
								</tr>
							</thead>
							<tbody>
								{allowlists.map((entry) => (
									<tr
										key={entry.id}
										className="border-b border-gray-100 dark:border-gray-700"
									>
										<td className="py-3 text-sm text-gray-900 dark:text-white font-mono">
											{entry.cidr}
										</td>
										<td className="py-3 text-sm text-gray-600 dark:text-gray-400">
											{entry.description || '-'}
										</td>
										<td className="py-3">
											<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
												{typeLabels[entry.type]}
											</span>
										</td>
										<td className="py-3">
											<span
												className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
													entry.enabled
														? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
														: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
												}`}
											>
												{entry.enabled ? 'Active' : 'Disabled'}
											</span>
										</td>
										{canEdit && (
											<td className="py-3 text-right">
												<button
													type="button"
													onClick={() => setEditingAllowlist({ ...entry })}
													className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm mr-3"
												>
													Edit
												</button>
												<button
													type="button"
													onClick={() => setDeleteConfirm(entry.id)}
													className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm"
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
							<p className="text-gray-500 dark:text-gray-400">
								No IP ranges configured yet
							</p>
							{canEdit && (
								<button
									type="button"
									onClick={() => setShowAddModal(true)}
									className="mt-4 text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
								>
									Add your first IP range
								</button>
							)}
						</div>
					)}
				</div>
			</div>

			{/* Blocked Attempts */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Recent Blocked Attempts
					</h2>
				</div>
				<div className="p-6">
					{attemptsLoading ? (
						<div className="space-y-4">
							{[1, 2, 3].map((i) => (
								<div
									key={i}
									className="h-12 w-full bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
								/>
							))}
						</div>
					) : blockedAttempts && blockedAttempts.attempts.length > 0 ? (
						<table className="w-full">
							<thead>
								<tr className="text-left text-sm text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
									<th className="pb-3 font-medium">Timestamp</th>
									<th className="pb-3 font-medium">IP Address</th>
									<th className="pb-3 font-medium">Type</th>
									<th className="pb-3 font-medium">Path</th>
									<th className="pb-3 font-medium">Reason</th>
								</tr>
							</thead>
							<tbody>
								{blockedAttempts.attempts.map((attempt) => (
									<tr
										key={attempt.id}
										className="border-b border-gray-100 dark:border-gray-700"
									>
										<td className="py-3 text-sm text-gray-600 dark:text-gray-400">
											{new Date(attempt.created_at).toLocaleString()}
										</td>
										<td className="py-3 text-sm text-gray-900 dark:text-white font-mono">
											{attempt.ip_address}
										</td>
										<td className="py-3">
											<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
												{attempt.request_type}
											</span>
										</td>
										<td className="py-3 text-sm text-gray-600 dark:text-gray-400 font-mono truncate max-w-[200px]">
											{attempt.path || '-'}
										</td>
										<td className="py-3 text-sm text-gray-600 dark:text-gray-400">
											{attempt.reason || '-'}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					) : (
						<div className="text-center py-8">
							<p className="text-gray-500 dark:text-gray-400">
								No blocked attempts recorded
							</p>
						</div>
					)}
				</div>
			</div>

			{/* Add Modal */}
			{showAddModal && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Add IP Range
						</h3>
						<form onSubmit={handleAddAllowlist} className="space-y-4">
							<div>
								<label
									htmlFor="cidr"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									IP Address or CIDR
								</label>
								<input
									type="text"
									id="cidr"
									value={newCIDR}
									onChange={(e) => setNewCIDR(e.target.value)}
									placeholder="e.g., 192.168.1.0/24 or 10.0.0.1"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
									required
								/>
								<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
									Enter a single IP address or CIDR notation (e.g., 10.0.0.0/8)
								</p>
							</div>
							<div>
								<label
									htmlFor="description"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Description (optional)
								</label>
								<input
									type="text"
									id="description"
									value={newDescription}
									onChange={(e) => setNewDescription(e.target.value)}
									placeholder="e.g., Office network"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								/>
							</div>
							<div>
								<label
									htmlFor="type"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Applies To
								</label>
								<select
									id="type"
									value={newType}
									onChange={(e) =>
										setNewType(e.target.value as IPAllowlistType)
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								>
									{typeOptions.map((type) => (
										<option key={type} value={type}>
											{typeLabels[type]}
										</option>
									))}
								</select>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="newEnabled"
									checked={newEnabled}
									onChange={(e) => setNewEnabled(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="newEnabled"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enabled
								</label>
							</div>
							{createAllowlist.isError && (
								<p className="text-sm text-red-600 dark:text-red-400">
									Failed to create entry. Please check the IP format.
								</p>
							)}
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => {
										setShowAddModal(false);
										resetForm();
									}}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={createAllowlist.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{createAllowlist.isPending ? 'Adding...' : 'Add Range'}
								</button>
							</div>
						</form>
					</div>
				</div>
			)}

			{/* Edit Modal */}
			{editingAllowlist && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Edit IP Range
						</h3>
						<form onSubmit={handleUpdateAllowlist} className="space-y-4">
							<div>
								<label
									htmlFor="editCidr"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									IP Address or CIDR
								</label>
								<input
									type="text"
									id="editCidr"
									value={editingAllowlist.cidr}
									onChange={(e) =>
										setEditingAllowlist({
											...editingAllowlist,
											cidr: e.target.value,
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
									required
								/>
							</div>
							<div>
								<label
									htmlFor="editDescription"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Description (optional)
								</label>
								<input
									type="text"
									id="editDescription"
									value={editingAllowlist.description || ''}
									onChange={(e) =>
										setEditingAllowlist({
											...editingAllowlist,
											description: e.target.value,
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								/>
							</div>
							<div>
								<label
									htmlFor="editType"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Applies To
								</label>
								<select
									id="editType"
									value={editingAllowlist.type}
									onChange={(e) =>
										setEditingAllowlist({
											...editingAllowlist,
											type: e.target.value as IPAllowlistType,
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								>
									{typeOptions.map((type) => (
										<option key={type} value={type}>
											{typeLabels[type]}
										</option>
									))}
								</select>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="editEnabled"
									checked={editingAllowlist.enabled}
									onChange={(e) =>
										setEditingAllowlist({
											...editingAllowlist,
											enabled: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="editEnabled"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enabled
								</label>
							</div>
							{updateAllowlist.isError && (
								<p className="text-sm text-red-600 dark:text-red-400">
									Failed to update entry. Please try again.
								</p>
							)}
							<div className="flex justify-end gap-3 pt-2">
								<button
									type="button"
									onClick={() => setEditingAllowlist(null)}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={updateAllowlist.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateAllowlist.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						</form>
					</div>
				</div>
			)}

			{/* Delete Confirmation Modal */}
			{deleteConfirm && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<div className="flex items-center gap-3 mb-4">
							<div className="p-2 bg-red-100 dark:bg-red-900/30 rounded-full">
								<svg
									aria-hidden="true"
									className="w-6 h-6 text-red-600 dark:text-red-400"
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
								Delete IP Range
							</h3>
						</div>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							Are you sure you want to delete this IP range? This action cannot
							be undone and may affect access for users or agents from this IP
							range.
						</p>
						{deleteAllowlist.isError && (
							<p className="text-sm text-red-600 dark:text-red-400 mb-4">
								Failed to delete entry. Please try again.
							</p>
						)}
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setDeleteConfirm(null)}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={() => handleDeleteAllowlist(deleteConfirm)}
								disabled={deleteAllowlist.isPending}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{deleteAllowlist.isPending ? 'Deleting...' : 'Delete'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
