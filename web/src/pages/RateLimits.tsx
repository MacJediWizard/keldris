import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useBlockedRequests,
	useCreateIPBan,
	useCreateRateLimitConfig,
	useDeleteIPBan,
	useDeleteRateLimitConfig,
	useIPBans,
	useRateLimitConfigs,
	useRateLimitStats,
	useUpdateRateLimitConfig,
} from '../hooks/useRateLimits';
import type {
	CreateIPBanRequest,
	CreateRateLimitConfigRequest,
	IPBan,
	OrgRole,
	RateLimitConfig,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	return new Date(dateStr).toLocaleString();
}

function formatDuration(seconds: number): string {
	if (seconds < 60) return `${seconds}s`;
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
	return `${Math.floor(seconds / 3600)}h`;
}

function isBanActive(ban: IPBan): boolean {
	if (!ban.expires_at) return true;
	return new Date(ban.expires_at) > new Date();
}

export function RateLimits() {
	const { data: user } = useMe();
	const { data: configs, isLoading: configsLoading } = useRateLimitConfigs();
	const { data: statsResponse, isLoading: statsLoading } = useRateLimitStats();
	const { data: blockedResponse } = useBlockedRequests();
	const { data: bans } = useIPBans();

	const createConfig = useCreateRateLimitConfig();
	const updateConfig = useUpdateRateLimitConfig();
	const deleteConfig = useDeleteRateLimitConfig();
	const createBan = useCreateIPBan();
	const deleteBan = useDeleteIPBan();

	const [showConfigForm, setShowConfigForm] = useState(false);
	const [showBanForm, setShowBanForm] = useState(false);
	const [editingId, setEditingId] = useState<string | null>(null);
	const [configFormData, setConfigFormData] =
		useState<CreateRateLimitConfigRequest>({
			endpoint: '',
			requests_per_period: 100,
			period_seconds: 60,
			enabled: true,
		});
	const [banFormData, setBanFormData] = useState<CreateIPBanRequest>({
		ip_address: '',
		reason: '',
		duration_minutes: 60,
	});

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isAdmin = currentUserRole === 'owner' || currentUserRole === 'admin';

	const stats = statsResponse?.stats;
	const blockedRequests = blockedResponse?.blocked_requests ?? [];

	const handleCreateConfig = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createConfig.mutateAsync(configFormData);
			setShowConfigForm(false);
			resetConfigForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleUpdateConfig = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!editingId) return;
		try {
			await updateConfig.mutateAsync({
				id: editingId,
				data: {
					requests_per_period: configFormData.requests_per_period,
					period_seconds: configFormData.period_seconds,
					enabled: configFormData.enabled,
				},
			});
			setEditingId(null);
			resetConfigForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleDeleteConfig = async (id: string) => {
		if (
			!window.confirm('Are you sure you want to delete this rate limit config?')
		) {
			return;
		}
		try {
			await deleteConfig.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	const handleCreateBan = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createBan.mutateAsync(banFormData);
			setShowBanForm(false);
			resetBanForm();
		} catch {
			// Error handled by mutation
		}
	};

	const handleDeleteBan = async (id: string) => {
		if (!window.confirm('Are you sure you want to remove this IP ban?')) {
			return;
		}
		try {
			await deleteBan.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	const handleQuickBan = (ipAddress: string) => {
		setBanFormData({
			ip_address: ipAddress,
			reason: 'Repeated rate limit violations',
			duration_minutes: 60,
		});
		setShowBanForm(true);
	};

	const startEditConfig = (config: RateLimitConfig) => {
		setEditingId(config.id);
		setConfigFormData({
			endpoint: config.endpoint,
			requests_per_period: config.requests_per_period,
			period_seconds: config.period_seconds,
			enabled: config.enabled,
		});
		setShowConfigForm(false);
	};

	const resetConfigForm = () => {
		setConfigFormData({
			endpoint: '',
			requests_per_period: 100,
			period_seconds: 60,
			enabled: true,
		});
	};

	const resetBanForm = () => {
		setBanFormData({
			ip_address: '',
			reason: '',
			duration_minutes: 60,
		});
	};

	const cancelEdit = () => {
		setEditingId(null);
		setShowConfigForm(false);
		resetConfigForm();
	};

	if (!isAdmin) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Rate Limits</h1>
					<p className="text-gray-600 mt-1">
						Configure rate limiting for API endpoints
					</p>
				</div>
				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<p className="text-amber-800">
						Only administrators can manage rate limits.
					</p>
				</div>
			</div>
		);
	}

	if (configsLoading || statsLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 rounded animate-pulse mt-2" />
				</div>
				<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
					{[1, 2, 3].map((i) => (
						<div
							key={i}
							className="h-24 bg-gray-200 rounded-lg animate-pulse"
						/>
					))}
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Rate Limits</h1>
					<p className="text-gray-600 mt-1">
						Configure rate limiting for API endpoints
					</p>
				</div>
			</div>

			{/* Stats Cards */}
			<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
				<div className="bg-white rounded-lg border border-gray-200 p-4">
					<div className="text-sm font-medium text-gray-500">Blocked Today</div>
					<div className="mt-1 text-3xl font-bold text-red-600">
						{stats?.blocked_today ?? 0}
					</div>
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-4">
					<div className="text-sm font-medium text-gray-500">
						Active Configs
					</div>
					<div className="mt-1 text-3xl font-bold text-gray-900">
						{configs?.filter((c) => c.enabled).length ?? 0}
					</div>
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-4">
					<div className="text-sm font-medium text-gray-500">Active Bans</div>
					<div className="mt-1 text-3xl font-bold text-amber-600">
						{bans?.filter(isBanActive).length ?? 0}
					</div>
				</div>
			</div>

			{/* Top Blocked Stats */}
			{stats &&
				(stats.top_blocked_ips?.length > 0 ||
					stats.top_blocked_endpoints?.length > 0) && (
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						{stats.top_blocked_ips?.length > 0 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<h3 className="text-sm font-medium text-gray-900 mb-3">
									Top Blocked IPs (7 days)
								</h3>
								<ul className="space-y-2">
									{stats.top_blocked_ips.slice(0, 5).map((ip) => (
										<li
											key={ip.ip_address}
											className="flex items-center justify-between text-sm"
										>
											<code className="bg-gray-100 px-2 py-0.5 rounded text-gray-700">
												{ip.ip_address}
											</code>
											<div className="flex items-center gap-2">
												<span className="text-red-600 font-medium">
													{ip.count} blocks
												</span>
												<button
													type="button"
													onClick={() => handleQuickBan(ip.ip_address)}
													className="text-xs text-amber-600 hover:text-amber-800"
												>
													Ban
												</button>
											</div>
										</li>
									))}
								</ul>
							</div>
						)}
						{stats.top_blocked_endpoints?.length > 0 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<h3 className="text-sm font-medium text-gray-900 mb-3">
									Top Blocked Endpoints (7 days)
								</h3>
								<ul className="space-y-2">
									{stats.top_blocked_endpoints.slice(0, 5).map((route) => (
										<li
											key={route.endpoint}
											className="flex items-center justify-between text-sm"
										>
											<code className="bg-gray-100 px-2 py-0.5 rounded text-gray-700 truncate max-w-xs">
												{route.endpoint}
											</code>
											<span className="text-red-600 font-medium">
												{route.count} blocks
											</span>
										</li>
									))}
								</ul>
							</div>
						)}
					</div>
				)}

			{/* Rate Limit Configs Section */}
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-medium text-gray-900">
						Rate Limit Configurations
					</h2>
					{!showConfigForm && !editingId && (
						<button
							type="button"
							onClick={() => setShowConfigForm(true)}
							className="inline-flex items-center px-3 py-1.5 text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
						>
							<svg
								className="w-4 h-4 mr-1"
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
							Add Config
						</button>
					)}
				</div>

				{(showConfigForm || editingId) && (
					<div className="p-4 border-b border-gray-200 bg-gray-50">
						<form
							onSubmit={editingId ? handleUpdateConfig : handleCreateConfig}
						>
							<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
								<div>
									<label
										htmlFor="endpoint"
										className="block text-sm font-medium text-gray-700"
									>
										Endpoint
									</label>
									<input
										type="text"
										id="endpoint"
										value={configFormData.endpoint}
										onChange={(e) =>
											setConfigFormData({
												...configFormData,
												endpoint: e.target.value,
											})
										}
										disabled={!!editingId}
										required
										placeholder="/api/v1/agents"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm disabled:bg-gray-100"
									/>
								</div>
								<div>
									<label
										htmlFor="requests"
										className="block text-sm font-medium text-gray-700"
									>
										Requests
									</label>
									<input
										type="number"
										id="requests"
										value={configFormData.requests_per_period}
										onChange={(e) =>
											setConfigFormData({
												...configFormData,
												requests_per_period: Number.parseInt(
													e.target.value,
													10,
												),
											})
										}
										min={1}
										required
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
								<div>
									<label
										htmlFor="period"
										className="block text-sm font-medium text-gray-700"
									>
										Period (seconds)
									</label>
									<input
										type="number"
										id="period"
										value={configFormData.period_seconds}
										onChange={(e) =>
											setConfigFormData({
												...configFormData,
												period_seconds: Number.parseInt(e.target.value, 10),
											})
										}
										min={1}
										required
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
								<div className="flex items-end gap-2">
									<label className="flex items-center">
										<input
											type="checkbox"
											checked={configFormData.enabled}
											onChange={(e) =>
												setConfigFormData({
													...configFormData,
													enabled: e.target.checked,
												})
											}
											className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
										/>
										<span className="ml-2 text-sm text-gray-700">Enabled</span>
									</label>
								</div>
							</div>
							<div className="flex justify-end gap-2 mt-4">
								<button
									type="button"
									onClick={cancelEdit}
									className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={createConfig.isPending || updateConfig.isPending}
									className="px-3 py-1.5 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
								>
									{createConfig.isPending || updateConfig.isPending
										? 'Saving...'
										: editingId
											? 'Update'
											: 'Create'}
								</button>
							</div>
						</form>
					</div>
				)}

				{configs?.length === 0 ? (
					<div className="p-8 text-center text-gray-500">
						No custom rate limit configurations. Using default global limits.
					</div>
				) : (
					<ul className="divide-y divide-gray-200">
						{configs?.map((config) => (
							<li key={config.id} className="px-4 py-3 hover:bg-gray-50">
								<div className="flex items-center justify-between">
									<div className="flex-1 min-w-0">
										<div className="flex items-center gap-2">
											<code className="text-sm font-medium text-gray-900 bg-gray-100 px-2 py-0.5 rounded">
												{config.endpoint}
											</code>
											{config.enabled ? (
												<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
													Active
												</span>
											) : (
												<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
													Disabled
												</span>
											)}
										</div>
										<p className="mt-1 text-sm text-gray-500">
											{config.requests_per_period} requests per{' '}
											{formatDuration(config.period_seconds)}
										</p>
									</div>
									<div className="flex items-center gap-2 ml-4">
										<button
											type="button"
											onClick={() => startEditConfig(config)}
											className="text-indigo-600 hover:text-indigo-900 text-sm font-medium"
										>
											Edit
										</button>
										<button
											type="button"
											onClick={() => handleDeleteConfig(config.id)}
											disabled={deleteConfig.isPending}
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

			{/* IP Bans Section */}
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-medium text-gray-900">IP Bans</h2>
					{!showBanForm && (
						<button
							type="button"
							onClick={() => setShowBanForm(true)}
							className="inline-flex items-center px-3 py-1.5 text-sm font-medium rounded-md text-white bg-amber-600 hover:bg-amber-700"
						>
							<svg
								className="w-4 h-4 mr-1"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
								/>
							</svg>
							Ban IP
						</button>
					)}
				</div>

				{showBanForm && (
					<div className="p-4 border-b border-gray-200 bg-gray-50">
						<form onSubmit={handleCreateBan}>
							<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
								<div>
									<label
										htmlFor="ip_address"
										className="block text-sm font-medium text-gray-700"
									>
										IP Address
									</label>
									<input
										type="text"
										id="ip_address"
										value={banFormData.ip_address}
										onChange={(e) =>
											setBanFormData({
												...banFormData,
												ip_address: e.target.value,
											})
										}
										required
										placeholder="192.168.1.1"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
								<div className="md:col-span-2">
									<label
										htmlFor="reason"
										className="block text-sm font-medium text-gray-700"
									>
										Reason
									</label>
									<input
										type="text"
										id="reason"
										value={banFormData.reason}
										onChange={(e) =>
											setBanFormData({
												...banFormData,
												reason: e.target.value,
											})
										}
										required
										placeholder="Repeated rate limit violations"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
								<div>
									<label
										htmlFor="duration"
										className="block text-sm font-medium text-gray-700"
									>
										Duration (minutes)
									</label>
									<input
										type="number"
										id="duration"
										value={banFormData.duration_minutes ?? ''}
										onChange={(e) =>
											setBanFormData({
												...banFormData,
												duration_minutes: e.target.value
													? Number.parseInt(e.target.value, 10)
													: undefined,
											})
										}
										min={1}
										placeholder="Empty = permanent"
										className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
									/>
								</div>
							</div>
							<div className="flex justify-end gap-2 mt-4">
								<button
									type="button"
									onClick={() => {
										setShowBanForm(false);
										resetBanForm();
									}}
									className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={createBan.isPending}
									className="px-3 py-1.5 text-sm font-medium text-white bg-amber-600 rounded-md hover:bg-amber-700 disabled:opacity-50"
								>
									{createBan.isPending ? 'Banning...' : 'Ban IP'}
								</button>
							</div>
						</form>
					</div>
				)}

				{!bans?.length ? (
					<div className="p-8 text-center text-gray-500">
						No IP bans active.
					</div>
				) : (
					<ul className="divide-y divide-gray-200">
						{bans?.map((ban) => (
							<li key={ban.id} className="px-4 py-3 hover:bg-gray-50">
								<div className="flex items-center justify-between">
									<div className="flex-1 min-w-0">
										<div className="flex items-center gap-2">
											<code className="text-sm font-medium text-gray-900 bg-gray-100 px-2 py-0.5 rounded">
												{ban.ip_address}
											</code>
											{isBanActive(ban) ? (
												<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
													Active
												</span>
											) : (
												<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
													Expired
												</span>
											)}
											{!ban.expires_at && (
												<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800">
													Permanent
												</span>
											)}
										</div>
										<p className="mt-1 text-sm text-gray-500">{ban.reason}</p>
										<p className="text-xs text-gray-400">
											Banned: {formatDateTime(ban.banned_at)}
											{ban.expires_at &&
												` | Expires: ${formatDateTime(ban.expires_at)}`}
										</p>
									</div>
									<div className="flex items-center gap-2 ml-4">
										<button
											type="button"
											onClick={() => handleDeleteBan(ban.id)}
											disabled={deleteBan.isPending}
											className="text-red-600 hover:text-red-900 text-sm font-medium disabled:opacity-50"
										>
											Remove
										</button>
									</div>
								</div>
							</li>
						))}
					</ul>
				)}
			</div>

			{/* Recent Blocked Requests */}
			{blockedRequests.length > 0 && (
				<div className="bg-white rounded-lg border border-gray-200">
					<div className="px-4 py-3 border-b border-gray-200">
						<h2 className="text-lg font-medium text-gray-900">
							Recent Blocked Requests
						</h2>
					</div>
					<div className="overflow-x-auto">
						<table className="min-w-full divide-y divide-gray-200">
							<thead className="bg-gray-50">
								<tr>
									<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Time
									</th>
									<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										IP Address
									</th>
									<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Endpoint
									</th>
									<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Reason
									</th>
									<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Action
									</th>
								</tr>
							</thead>
							<tbody className="bg-white divide-y divide-gray-200">
								{blockedRequests.slice(0, 20).map((req) => (
									<tr key={req.id} className="hover:bg-gray-50">
										<td className="px-4 py-2 text-sm text-gray-500 whitespace-nowrap">
											{formatDateTime(req.blocked_at)}
										</td>
										<td className="px-4 py-2 text-sm whitespace-nowrap">
											<code className="bg-gray-100 px-1 py-0.5 rounded text-gray-700">
												{req.ip_address}
											</code>
										</td>
										<td className="px-4 py-2 text-sm text-gray-900 max-w-xs truncate">
											{req.endpoint}
										</td>
										<td className="px-4 py-2 text-sm text-gray-500">
											{req.reason}
										</td>
										<td className="px-4 py-2 text-sm whitespace-nowrap">
											<button
												type="button"
												onClick={() => handleQuickBan(req.ip_address)}
												className="text-amber-600 hover:text-amber-800 text-xs"
											>
												Ban IP
											</button>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</div>
			)}
		</div>
	);
}
