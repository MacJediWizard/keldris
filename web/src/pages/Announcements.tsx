import { useState } from 'react';
import {
	useAnnouncements,
	useCreateAnnouncement,
	useDeleteAnnouncement,
	useUpdateAnnouncement,
} from '../hooks/useAnnouncements';
import { useMe } from '../hooks/useAuth';
import type {
	Announcement,
	AnnouncementType,
	CreateAnnouncementRequest,
	OrgRole,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	return new Date(dateStr).toLocaleString();
}

function getAnnouncementStatus(
	announcement: Announcement,
): 'active' | 'scheduled' | 'ended' | 'inactive' {
	if (!announcement.active) return 'inactive';

	const now = new Date();

	if (announcement.starts_at && new Date(announcement.starts_at) > now) {
		return 'scheduled';
	}

	if (announcement.ends_at && new Date(announcement.ends_at) < now) {
		return 'ended';
	}

	return 'active';
}

function StatusBadge({
	status,
}: {
	status: 'active' | 'scheduled' | 'ended' | 'inactive';
}) {
	const colors = {
		active: 'bg-green-100 text-green-800',
		scheduled: 'bg-blue-100 text-blue-800',
		ended: 'bg-gray-100 text-gray-800',
		inactive: 'bg-yellow-100 text-yellow-800',
	};

	const labels = {
		active: 'Active',
		scheduled: 'Scheduled',
		ended: 'Ended',
		inactive: 'Inactive',
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status]}`}
		>
			{labels[status]}
		</span>
	);
}

function TypeBadge({ type }: { type: AnnouncementType }) {
	const colors = {
		info: 'bg-blue-100 text-blue-800',
		warning: 'bg-amber-100 text-amber-800',
		critical: 'bg-red-100 text-red-800',
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[type]}`}
		>
			{type}
		</span>
	);
}

export function Announcements() {
	const { data: user } = useMe();
	const { data: announcements, isLoading, isError } = useAnnouncements();
	const createAnnouncement = useCreateAnnouncement();
	const updateAnnouncement = useUpdateAnnouncement();
	const deleteAnnouncement = useDeleteAnnouncement();

	const [showForm, setShowForm] = useState(false);
	const [editingId, setEditingId] = useState<string | null>(null);
	const [formData, setFormData] = useState<CreateAnnouncementRequest>({
		title: '',
		message: '',
		type: 'info',
		dismissible: true,
		starts_at: '',
		ends_at: '',
		active: true,
	});

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isAdmin = currentUserRole === 'owner' || currentUserRole === 'admin';

	const handleCreate = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const data = {
				...formData,
				starts_at: formData.starts_at || undefined,
				ends_at: formData.ends_at || undefined,
			};
			await createAnnouncement.mutateAsync(data);
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
			const data = {
				...formData,
				starts_at: formData.starts_at || undefined,
				ends_at: formData.ends_at || undefined,
			};
			await updateAnnouncement.mutateAsync({
				id: editingId,
				data,
			});
			setEditingId(null);
			resetForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleDelete = async (id: string) => {
		if (!window.confirm('Are you sure you want to delete this announcement?')) {
			return;
		}
		try {
			await deleteAnnouncement.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	const startEdit = (a: Announcement) => {
		setEditingId(a.id);
		setFormData({
			title: a.title,
			message: a.message ?? '',
			type: a.type,
			dismissible: a.dismissible,
			starts_at: a.starts_at ? a.starts_at.slice(0, 16) : '',
			ends_at: a.ends_at ? a.ends_at.slice(0, 16) : '',
			active: a.active,
		});
		setShowForm(false);
	};

	const resetForm = () => {
		setFormData({
			title: '',
			message: '',
			type: 'info',
			dismissible: true,
			starts_at: '',
			ends_at: '',
			active: true,
		});
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
					<h1 className="text-2xl font-bold text-gray-900">Announcements</h1>
					<p className="text-gray-600 mt-1">
						Manage system-wide announcements for your organization
					</p>
				</div>
				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<p className="text-amber-800">
						Only administrators can manage announcements.
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
					<h1 className="text-2xl font-bold text-gray-900">Announcements</h1>
				</div>
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<p className="text-red-800">Failed to load announcements</p>
				</div>
			</div>
		);
	}

	const sortedAnnouncements = [...(announcements ?? [])].sort(
		(a, b) =>
			new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
	);

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Announcements</h1>
					<p className="text-gray-600 mt-1">
						Manage system-wide announcements for your organization
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
						Create Announcement
					</button>
				)}
			</div>

			{(showForm || editingId) && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-medium text-gray-900 mb-4">
						{editingId ? 'Edit Announcement' : 'Create Announcement'}
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
									placeholder="Important system update"
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
									placeholder="Additional details about the announcement..."
								/>
							</div>

							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="type"
										className="block text-sm font-medium text-gray-700"
									>
										Type
									</label>
									<select
										id="type"
										value={formData.type}
										onChange={(e) =>
											setFormData({
												...formData,
												type: e.target.value as AnnouncementType,
											})
										}
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									>
										<option value="info">Info</option>
										<option value="warning">Warning</option>
										<option value="critical">Critical</option>
									</select>
								</div>

								<div className="flex items-center">
									<div className="mt-6">
										<label className="inline-flex items-center">
											<input
												type="checkbox"
												checked={formData.dismissible}
												onChange={(e) =>
													setFormData({
														...formData,
														dismissible: e.target.checked,
													})
												}
												className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
											/>
											<span className="ml-2 text-sm text-gray-700">
												Allow users to dismiss
											</span>
										</label>
									</div>
								</div>
							</div>

							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="starts_at"
										className="block text-sm font-medium text-gray-700"
									>
										Start Time (optional)
									</label>
									<input
										type="datetime-local"
										id="starts_at"
										value={formData.starts_at}
										onChange={(e) =>
											setFormData({ ...formData, starts_at: e.target.value })
										}
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Leave empty to show immediately
									</p>
								</div>

								<div>
									<label
										htmlFor="ends_at"
										className="block text-sm font-medium text-gray-700"
									>
										End Time (optional)
									</label>
									<input
										type="datetime-local"
										id="ends_at"
										value={formData.ends_at}
										onChange={(e) =>
											setFormData({ ...formData, ends_at: e.target.value })
										}
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Leave empty to show indefinitely
									</p>
								</div>
							</div>

							<div>
								<label className="inline-flex items-center">
									<input
										type="checkbox"
										checked={formData.active}
										onChange={(e) =>
											setFormData({
												...formData,
												active: e.target.checked,
											})
										}
										className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="ml-2 text-sm text-gray-700">
										Active (show to users)
									</span>
								</label>
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
										createAnnouncement.isPending || updateAnnouncement.isPending
									}
									className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
								>
									{createAnnouncement.isPending || updateAnnouncement.isPending
										? 'Saving...'
										: editingId
											? 'Update'
											: 'Create'}
								</button>
							</div>
						</div>
					</form>
				</div>
			)}

			<div className="bg-white rounded-lg border border-gray-200">
				{sortedAnnouncements.length === 0 ? (
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
								d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z"
							/>
						</svg>
						<h3 className="mt-2 text-sm font-medium text-gray-900">
							No announcements
						</h3>
						<p className="mt-1 text-sm text-gray-500">
							Create an announcement to notify users of important information.
						</p>
					</div>
				) : (
					<ul className="divide-y divide-gray-200">
						{sortedAnnouncements.map((a) => {
							const status = getAnnouncementStatus(a);
							return (
								<li key={a.id} className="p-4 hover:bg-gray-50">
									<div className="flex items-center justify-between">
										<div className="flex-1 min-w-0">
											<div className="flex items-center gap-2 flex-wrap">
												<h3 className="text-sm font-medium text-gray-900 truncate">
													{a.title}
												</h3>
												<TypeBadge type={a.type} />
												<StatusBadge status={status} />
												{!a.dismissible && (
													<span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
														Non-dismissible
													</span>
												)}
											</div>
											{a.message && (
												<p className="mt-1 text-sm text-gray-500 truncate">
													{a.message}
												</p>
											)}
											<div className="mt-1 text-xs text-gray-500 space-x-4">
												<span>Created: {formatDateTime(a.created_at)}</span>
												{a.starts_at && (
													<span>Starts: {formatDateTime(a.starts_at)}</span>
												)}
												{a.ends_at && (
													<span>Ends: {formatDateTime(a.ends_at)}</span>
												)}
											</div>
										</div>
										<div className="flex items-center gap-2 ml-4">
											<button
												type="button"
												onClick={() => startEdit(a)}
												className="text-indigo-600 hover:text-indigo-900 text-sm font-medium"
											>
												Edit
											</button>
											<button
												type="button"
												onClick={() => handleDelete(a.id)}
												disabled={deleteAnnouncement.isPending}
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
