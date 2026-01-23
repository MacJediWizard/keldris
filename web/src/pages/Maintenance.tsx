import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useCreateMaintenanceWindow,
	useDeleteMaintenanceWindow,
	useEmergencyOverride,
	useMaintenanceWindows,
	useUpdateMaintenanceWindow,
} from '../hooks/useMaintenance';
import type {
	CreateMaintenanceWindowRequest,
	MaintenanceWindow,
	OrgRole,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	return new Date(dateStr).toLocaleString();
}

function getWindowStatus(
	window: MaintenanceWindow,
): 'active' | 'upcoming' | 'past' {
	const now = new Date();
	const starts = new Date(window.starts_at);
	const ends = new Date(window.ends_at);

	if (now >= starts && now < ends) return 'active';
	if (now < starts) return 'upcoming';
	return 'past';
}

function StatusBadge({ status }: { status: 'active' | 'upcoming' | 'past' }) {
	const colors = {
		active: 'bg-amber-100 text-amber-800',
		upcoming: 'bg-blue-100 text-blue-800',
		past: 'bg-gray-100 text-gray-800',
	};

	const labels = {
		active: 'Active',
		upcoming: 'Upcoming',
		past: 'Completed',
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status]}`}
		>
			{labels[status]}
		</span>
	);
}

export function Maintenance() {
	const { data: user } = useMe();
	const { data: windows, isLoading, isError } = useMaintenanceWindows();
	const createMaintenance = useCreateMaintenanceWindow();
	const updateMaintenance = useUpdateMaintenanceWindow();
	const deleteMaintenance = useDeleteMaintenanceWindow();

	const emergencyOverride = useEmergencyOverride();
	const [showForm, setShowForm] = useState(false);
	const [editingId, setEditingId] = useState<string | null>(null);
	const [formData, setFormData] = useState<CreateMaintenanceWindowRequest>({
		title: '',
		message: '',
		starts_at: '',
		ends_at: '',
		notify_before_minutes: 60,
		read_only: false,
		countdown_start_minutes: 30,
	});

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isAdmin = currentUserRole === 'owner' || currentUserRole === 'admin';

	const handleCreate = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createMaintenance.mutateAsync(formData);
			setShowForm(false);
			resetForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleUpdate = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!editingId) return;
		try {
			await updateMaintenance.mutateAsync({
				id: editingId,
				data: formData,
			});
			setEditingId(null);
			resetForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleDelete = async (id: string) => {
		if (
			!window.confirm(
				'Are you sure you want to delete this maintenance window?',
			)
		) {
			return;
		}
		try {
			await deleteMaintenance.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	const startEdit = (w: MaintenanceWindow) => {
		setEditingId(w.id);
		setFormData({
			title: w.title,
			message: w.message ?? '',
			starts_at: w.starts_at.slice(0, 16), // Format for datetime-local
			ends_at: w.ends_at.slice(0, 16),
			notify_before_minutes: w.notify_before_minutes,
			read_only: w.read_only,
			countdown_start_minutes: w.countdown_start_minutes,
		});
		setShowForm(false);
	};

	const resetForm = () => {
		setFormData({
			title: '',
			message: '',
			starts_at: '',
			ends_at: '',
			notify_before_minutes: 60,
			read_only: false,
			countdown_start_minutes: 30,
		});
	};

	const handleEmergencyOverride = (id: string, currentOverride: boolean) => {
		emergencyOverride.mutate({ id, override: !currentOverride });
	};

	const cancelEdit = () => {
		setEditingId(null);
		setShowForm(false);
		resetForm();
	};

	if (!isAdmin) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Maintenance</h1>
					<p className="text-gray-600 mt-1">
						Schedule maintenance windows to pause backups
					</p>
				</div>
				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<p className="text-amber-800">
						Only administrators can manage maintenance windows.
					</p>
				</div>
			</div>
		);
	}

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div key={i} className="h-16 bg-gray-200 rounded animate-pulse" />
						))}
					</div>
				</div>
			</div>
		);
	}

	if (isError) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Maintenance</h1>
				</div>
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<p className="text-red-800">Failed to load maintenance windows</p>
				</div>
			</div>
		);
	}

	const sortedWindows = [...(windows ?? [])].sort(
		(a, b) => new Date(b.starts_at).getTime() - new Date(a.starts_at).getTime(),
	);

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Maintenance</h1>
					<p className="text-gray-600 mt-1">
						Schedule maintenance windows to pause backups
					</p>
				</div>
				{!showForm && !editingId && (
					<button
						type="button"
						onClick={() => setShowForm(true)}
						className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
					>
						<svg
							className="w-4 h-4 mr-2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 4v16m8-8H4"
							/>
						</svg>
						Schedule Maintenance
					</button>
				)}
			</div>

			{(showForm || editingId) && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-medium text-gray-900 mb-4">
						{editingId
							? 'Edit Maintenance Window'
							: 'Schedule Maintenance Window'}
					</h2>
					<form onSubmit={editingId ? handleUpdate : handleCreate}>
						<div className="space-y-4">
							<div>
								<label
									htmlFor="title"
									className="block text-sm font-medium text-gray-700"
								>
									Title
								</label>
								<input
									type="text"
									id="title"
									value={formData.title}
									onChange={(e) =>
										setFormData({ ...formData, title: e.target.value })
									}
									required
									className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									placeholder="Scheduled system maintenance"
								/>
							</div>

							<div>
								<label
									htmlFor="message"
									className="block text-sm font-medium text-gray-700"
								>
									Message (optional)
								</label>
								<textarea
									id="message"
									value={formData.message}
									onChange={(e) =>
										setFormData({ ...formData, message: e.target.value })
									}
									rows={2}
									className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									placeholder="Additional details about the maintenance..."
								/>
							</div>

							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="starts_at"
										className="block text-sm font-medium text-gray-700"
									>
										Start Time
									</label>
									<input
										type="datetime-local"
										id="starts_at"
										value={formData.starts_at}
										onChange={(e) =>
											setFormData({ ...formData, starts_at: e.target.value })
										}
										required
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>

								<div>
									<label
										htmlFor="ends_at"
										className="block text-sm font-medium text-gray-700"
									>
										End Time
									</label>
									<input
										type="datetime-local"
										id="ends_at"
										value={formData.ends_at}
										onChange={(e) =>
											setFormData({ ...formData, ends_at: e.target.value })
										}
										required
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
							</div>

							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="notify_before"
										className="block text-sm font-medium text-gray-700"
									>
										Notify Before (minutes)
									</label>
									<input
										type="number"
										id="notify_before"
										value={formData.notify_before_minutes}
										onChange={(e) =>
											setFormData({
												...formData,
												notify_before_minutes: Number.parseInt(
													e.target.value,
													10,
												),
											})
										}
										min={0}
										max={1440}
										className="mt-1 block w-32 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Send notification before maintenance
									</p>
								</div>

								<div>
									<label
										htmlFor="countdown_start"
										className="block text-sm font-medium text-gray-700"
									>
										Countdown Start (minutes)
									</label>
									<input
										type="number"
										id="countdown_start"
										value={formData.countdown_start_minutes}
										onChange={(e) =>
											setFormData({
												...formData,
												countdown_start_minutes: Number.parseInt(
													e.target.value,
													10,
												),
											})
										}
										min={0}
										max={1440}
										className="mt-1 block w-32 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Show countdown timer before maintenance
									</p>
								</div>
							</div>

							<div className="flex items-center">
								<input
									type="checkbox"
									id="read_only"
									checked={formData.read_only}
									onChange={(e) =>
										setFormData({
											...formData,
											read_only: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="read_only"
									className="ml-2 block text-sm text-gray-900"
								>
									Enable read-only mode
								</label>
								<p className="ml-6 text-xs text-gray-500">
									Block write operations during this maintenance window
								</p>
							</div>

							<div className="flex justify-end gap-3 pt-4">
								<button
									type="button"
									onClick={cancelEdit}
									className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={
										createMaintenance.isPending || updateMaintenance.isPending
									}
									className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
								>
									{createMaintenance.isPending || updateMaintenance.isPending
										? 'Saving...'
										: editingId
											? 'Update'
											: 'Schedule'}
								</button>
							</div>
						</div>
					</form>
				</div>
			)}

			<div className="bg-white rounded-lg border border-gray-200">
				{sortedWindows.length === 0 ? (
					<div className="p-8 text-center">
						<svg
							className="mx-auto h-12 w-12 text-gray-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
							/>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/>
						</svg>
						<h3 className="mt-2 text-sm font-medium text-gray-900">
							No maintenance windows
						</h3>
						<p className="mt-1 text-sm text-gray-500">
							Schedule a maintenance window to pause backups during maintenance.
						</p>
					</div>
				) : (
					<ul className="divide-y divide-gray-200">
						{sortedWindows.map((w) => {
							const status = getWindowStatus(w);
							const isActiveReadOnly =
								status === 'active' && w.read_only && !w.emergency_override;
							return (
								<li key={w.id} className="p-4 hover:bg-gray-50">
									<div className="flex items-center justify-between">
										<div className="flex-1 min-w-0">
											<div className="flex items-center gap-2 flex-wrap">
												<h3 className="text-sm font-medium text-gray-900 truncate">
													{w.title}
												</h3>
												<StatusBadge status={status} />
												{w.read_only && (
													<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
														Read-only
													</span>
												)}
												{w.emergency_override && (
													<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800">
														Override Active
													</span>
												)}
											</div>
											{w.message && (
												<p className="mt-1 text-sm text-gray-500 truncate">
													{w.message}
												</p>
											)}
											<div className="mt-1 text-xs text-gray-500">
												<span>{formatDateTime(w.starts_at)}</span>
												<span className="mx-2">-</span>
												<span>{formatDateTime(w.ends_at)}</span>
												{w.countdown_start_minutes > 0 && (
													<span className="ml-2 text-gray-400">
														(Countdown: {w.countdown_start_minutes}m before)
													</span>
												)}
											</div>
										</div>
										<div className="flex items-center gap-2 ml-4">
											{isActiveReadOnly && (
												<button
													type="button"
													onClick={() =>
														handleEmergencyOverride(w.id, w.emergency_override)
													}
													disabled={emergencyOverride.isPending}
													className="text-amber-600 hover:text-amber-900 text-sm font-medium disabled:opacity-50"
												>
													Emergency Override
												</button>
											)}
											{status === 'active' &&
												w.read_only &&
												w.emergency_override && (
													<button
														type="button"
														onClick={() =>
															handleEmergencyOverride(
																w.id,
																w.emergency_override,
															)
														}
														disabled={emergencyOverride.isPending}
														className="text-green-600 hover:text-green-900 text-sm font-medium disabled:opacity-50"
													>
														Re-enable Read-only
													</button>
												)}
											<button
												type="button"
												onClick={() => startEdit(w)}
												className="text-indigo-600 hover:text-indigo-900 text-sm font-medium"
											>
												Edit
											</button>
											<button
												type="button"
												onClick={() => handleDelete(w.id)}
												disabled={deleteMaintenance.isPending}
												className="text-red-600 hover:text-red-900 text-sm font-medium disabled:opacity-50"
											>
												Delete
											</button>
										</div>
									</div>
								</li>
							);
						})}
					</ul>
				)}
			</div>
		</div>
	);
}
