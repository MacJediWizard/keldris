import { useState } from 'react';
import {
	useSLAs,
	useCreateSLA,
	useUpdateSLA,
	useDeleteSLA,
	useSLADashboard,
	useActiveSLABreaches,
	useAcknowledgeBreach,
	useResolveBreach,
} from '../hooks/useSLA';
import { useMe } from '../hooks/useAuth';
import type {
	SLAWithAssignments,
	SLAScope,
	SLABreach,
	CreateSLADefinitionRequest,
	OrgRole,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	return new Date(dateStr).toLocaleString();
}

function formatDuration(minutes: number): string {
	if (minutes < 60) {
		return `${minutes}m`;
	}
	const hours = Math.floor(minutes / 60);
	const mins = minutes % 60;
	if (hours < 24) {
		return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
	}
	const days = Math.floor(hours / 24);
	const remainingHours = hours % 24;
	return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
}

function ScopeBadge({ scope }: { scope: SLAScope }) {
	const colors = {
		agent: 'bg-blue-100 text-blue-800',
		repository: 'bg-purple-100 text-purple-800',
		organization: 'bg-green-100 text-green-800',
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[scope]}`}
		>
			{scope}
		</span>
	);
}

function StatusBadge({ active }: { active: boolean }) {
	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
				active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
			}`}
		>
			{active ? 'Active' : 'Inactive'}
		</span>
	);
}

function BreachTypeBadge({ type }: { type: string }) {
	const colors: Record<string, string> = {
		rpo: 'bg-red-100 text-red-800',
		rto: 'bg-orange-100 text-orange-800',
		uptime: 'bg-yellow-100 text-yellow-800',
	};

	const labels: Record<string, string> = {
		rpo: 'RPO',
		rto: 'RTO',
		uptime: 'Uptime',
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[type] ?? 'bg-gray-100 text-gray-800'}`}
		>
			{labels[type] ?? type}
		</span>
	);
}

function DashboardCard({
	title,
	value,
	subtitle,
	color,
}: {
	title: string;
	value: string | number;
	subtitle?: string;
	color?: 'green' | 'red' | 'yellow' | 'blue';
}) {
	const colors = {
		green: 'text-green-600',
		red: 'text-red-600',
		yellow: 'text-yellow-600',
		blue: 'text-blue-600',
	};

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-4">
			<p className="text-sm font-medium text-gray-500">{title}</p>
			<p className={`text-2xl font-bold ${color ? colors[color] : 'text-gray-900'}`}>
				{value}
			</p>
			{subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
		</div>
	);
}

export function SLA() {
	const { data: user } = useMe();
	const { data: slas, isLoading, isError } = useSLAs();
	const { data: dashboard } = useSLADashboard();
	const { data: activeBreaches } = useActiveSLABreaches();
	const createSLA = useCreateSLA();
	const updateSLA = useUpdateSLA();
	const deleteSLA = useDeleteSLA();
	const acknowledgeBreach = useAcknowledgeBreach();
	const resolveBreach = useResolveBreach();

	const [showForm, setShowForm] = useState(false);
	const [editingId, setEditingId] = useState<string | null>(null);
	const [formData, setFormData] = useState<CreateSLADefinitionRequest>({
		name: '',
		description: '',
		rpo_minutes: undefined,
		rto_minutes: undefined,
		uptime_percentage: undefined,
		scope: 'agent',
		active: true,
	});

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isAdmin = currentUserRole === 'owner' || currentUserRole === 'admin';

	const handleCreate = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createSLA.mutateAsync(formData);
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
			await updateSLA.mutateAsync({
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
		if (!window.confirm('Are you sure you want to delete this SLA?')) {
			return;
		}
		try {
			await deleteSLA.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	const handleAcknowledge = async (breach: SLABreach) => {
		try {
			await acknowledgeBreach.mutateAsync({ id: breach.id });
		} catch {
			// Error handled by mutation
		}
	};

	const handleResolve = async (breach: SLABreach) => {
		try {
			await resolveBreach.mutateAsync(breach.id);
		} catch {
			// Error handled by mutation
		}
	};

	const startEdit = (sla: SLAWithAssignments) => {
		setEditingId(sla.id);
		setFormData({
			name: sla.name,
			description: sla.description ?? '',
			rpo_minutes: sla.rpo_minutes,
			rto_minutes: sla.rto_minutes,
			uptime_percentage: sla.uptime_percentage,
			scope: sla.scope,
			active: sla.active,
		});
		setShowForm(false);
	};

	const resetForm = () => {
		setFormData({
			name: '',
			description: '',
			rpo_minutes: undefined,
			rto_minutes: undefined,
			uptime_percentage: undefined,
			scope: 'agent',
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
					<h1 className="text-2xl font-bold text-gray-900">
						Service Level Agreements
					</h1>
					<p className="text-gray-600 mt-1">
						Define and track SLA compliance for your organization
					</p>
				</div>
				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<p className="text-amber-800">
						Only administrators can manage SLAs.
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
				<div className="grid grid-cols-4 gap-4">
					{[1, 2, 3, 4].map((i) => (
						<div key={i} className="h-24 bg-gray-200 rounded animate-pulse" />
					))}
				</div>
			</div>
		);
	}

	if (isError) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">
						Service Level Agreements
					</h1>
				</div>
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<p className="text-red-800">Failed to load SLAs</p>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">
						Service Level Agreements
					</h1>
					<p className="text-gray-600 mt-1">
						Define and track SLA compliance for your organization
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
						Create SLA
					</button>
				)}
			</div>

			{/* Dashboard Stats */}
			{dashboard && (
				<div className="grid grid-cols-4 gap-4">
					<DashboardCard
						title="Total SLAs"
						value={dashboard.total_slas}
						subtitle={`${dashboard.active_slas} active`}
						color="blue"
					/>
					<DashboardCard
						title="Overall Compliance"
						value={`${dashboard.overall_compliance.toFixed(1)}%`}
						color={dashboard.overall_compliance >= 95 ? 'green' : dashboard.overall_compliance >= 90 ? 'yellow' : 'red'}
					/>
					<DashboardCard
						title="Active Breaches"
						value={dashboard.active_breaches}
						color={dashboard.active_breaches > 0 ? 'red' : 'green'}
					/>
					<DashboardCard
						title="Unacknowledged"
						value={dashboard.unacknowledged_count}
						subtitle="Requires attention"
						color={dashboard.unacknowledged_count > 0 ? 'yellow' : 'green'}
					/>
				</div>
			)}

			{/* Active Breaches */}
			{activeBreaches && activeBreaches.length > 0 && (
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<h2 className="text-lg font-medium text-red-900 mb-3">
						Active Breaches ({activeBreaches.length})
					</h2>
					<div className="space-y-2">
						{activeBreaches.slice(0, 5).map((breach) => (
							<div
								key={breach.id}
								className="flex items-center justify-between bg-white rounded-md p-3 border border-red-100"
							>
								<div className="flex items-center gap-3">
									<BreachTypeBadge type={breach.breach_type} />
									<span className="text-sm text-gray-700">
										{breach.description ?? `${breach.breach_type.toUpperCase()} breach`}
									</span>
									<span className="text-xs text-gray-500">
										Started: {formatDateTime(breach.breach_start)}
									</span>
								</div>
								<div className="flex items-center gap-2">
									{!breach.acknowledged && (
										<button
											type="button"
											onClick={() => handleAcknowledge(breach)}
											disabled={acknowledgeBreach.isPending}
											className="text-sm text-amber-600 hover:text-amber-800 font-medium disabled:opacity-50"
										>
											Acknowledge
										</button>
									)}
									<button
										type="button"
										onClick={() => handleResolve(breach)}
										disabled={resolveBreach.isPending}
										className="text-sm text-green-600 hover:text-green-800 font-medium disabled:opacity-50"
									>
										Resolve
									</button>
								</div>
							</div>
						))}
					</div>
				</div>
			)}

			{/* Create/Edit Form */}
			{(showForm || editingId) && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-medium text-gray-900 mb-4">
						{editingId ? 'Edit SLA' : 'Create SLA'}
					</h2>
					<form onSubmit={editingId ? handleUpdate : handleCreate}>
						<div className="space-y-4">
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="name"
										className="block text-sm font-medium text-gray-700"
									>
										Name
									</label>
									<input
										type="text"
										id="name"
										value={formData.name}
										onChange={(e) =>
											setFormData({ ...formData, name: e.target.value })
										}
										required
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
										placeholder="Production Backup SLA"
									/>
								</div>
								<div>
									<label
										htmlFor="scope"
										className="block text-sm font-medium text-gray-700"
									>
										Scope
									</label>
									<select
										id="scope"
										value={formData.scope}
										onChange={(e) =>
											setFormData({
												...formData,
												scope: e.target.value as SLAScope,
											})
										}
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									>
										<option value="agent">Agent</option>
										<option value="repository">Repository</option>
										<option value="organization">Organization</option>
									</select>
								</div>
							</div>

							<div>
								<label
									htmlFor="description"
									className="block text-sm font-medium text-gray-700"
								>
									Description (optional)
								</label>
								<textarea
									id="description"
									value={formData.description}
									onChange={(e) =>
										setFormData({ ...formData, description: e.target.value })
									}
									rows={2}
									className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									placeholder="SLA for critical production systems..."
								/>
							</div>

							<div className="grid grid-cols-3 gap-4">
								<div>
									<label
										htmlFor="rpo"
										className="block text-sm font-medium text-gray-700"
									>
										RPO (minutes)
									</label>
									<input
										type="number"
										id="rpo"
										value={formData.rpo_minutes ?? ''}
										onChange={(e) =>
											setFormData({
												...formData,
												rpo_minutes: e.target.value ? Number(e.target.value) : undefined,
											})
										}
										min="1"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
										placeholder="60"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Max time between backups
									</p>
								</div>
								<div>
									<label
										htmlFor="rto"
										className="block text-sm font-medium text-gray-700"
									>
										RTO (minutes)
									</label>
									<input
										type="number"
										id="rto"
										value={formData.rto_minutes ?? ''}
										onChange={(e) =>
											setFormData({
												...formData,
												rto_minutes: e.target.value ? Number(e.target.value) : undefined,
											})
										}
										min="1"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
										placeholder="30"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Max restore time
									</p>
								</div>
								<div>
									<label
										htmlFor="uptime"
										className="block text-sm font-medium text-gray-700"
									>
										Uptime (%)
									</label>
									<input
										type="number"
										id="uptime"
										value={formData.uptime_percentage ?? ''}
										onChange={(e) =>
											setFormData({
												...formData,
												uptime_percentage: e.target.value ? Number(e.target.value) : undefined,
											})
										}
										min="0"
										max="100"
										step="0.01"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
										placeholder="99.9"
									/>
									<p className="mt-1 text-xs text-gray-500">
										Target uptime percentage
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
										Active (enforce this SLA)
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
									disabled={createSLA.isPending || updateSLA.isPending}
									className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
								>
									{createSLA.isPending || updateSLA.isPending
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

			{/* SLA List */}
			<div className="bg-white rounded-lg border border-gray-200">
				{(slas ?? []).length === 0 ? (
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
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<h3 className="mt-2 text-sm font-medium text-gray-900">No SLAs</h3>
						<p className="mt-1 text-sm text-gray-500">
							Create an SLA to start tracking compliance.
						</p>
					</div>
				) : (
					<ul className="divide-y divide-gray-200">
						{(slas ?? []).map((sla) => (
							<li key={sla.id} className="p-4 hover:bg-gray-50">
								<div className="flex items-center justify-between">
									<div className="flex-1 min-w-0">
										<div className="flex items-center gap-2 flex-wrap">
											<h3 className="text-sm font-medium text-gray-900">
												{sla.name}
											</h3>
											<ScopeBadge scope={sla.scope} />
											<StatusBadge active={sla.active} />
										</div>
										{sla.description && (
											<p className="mt-1 text-sm text-gray-500 truncate">
												{sla.description}
											</p>
										)}
										<div className="mt-2 flex flex-wrap gap-4 text-xs text-gray-500">
											{sla.rpo_minutes && (
												<span>RPO: {formatDuration(sla.rpo_minutes)}</span>
											)}
											{sla.rto_minutes && (
												<span>RTO: {formatDuration(sla.rto_minutes)}</span>
											)}
											{sla.uptime_percentage && (
												<span>Uptime: {sla.uptime_percentage}%</span>
											)}
											<span>
												Assigned: {sla.agent_count} agents, {sla.repository_count}{' '}
												repos
											</span>
										</div>
									</div>
									<div className="flex items-center gap-2 ml-4">
										<button
											type="button"
											onClick={() => startEdit(sla)}
											className="text-indigo-600 hover:text-indigo-900 text-sm font-medium"
										>
											Edit
										</button>
										<button
											type="button"
											onClick={() => handleDelete(sla.id)}
											disabled={deleteSLA.isPending}
											className="text-red-600 hover:text-red-900 text-sm font-medium disabled:opacity-50"
										>
											Delete
										</button>
									</div>
								</div>
							</li>
						))}
					</ul>
				)}
			</div>
		</div>
	);
}
