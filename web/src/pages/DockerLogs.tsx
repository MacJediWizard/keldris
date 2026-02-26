import { useState } from 'react';
import { useAgents } from '../hooks/useAgents';
import {
	useApplyDockerLogRetention,
	useDeleteDockerLogBackup,
	useDockerLogBackups,
	useDockerLogDownload,
	useDockerLogSettings,
	useDockerLogStorageStats,
	useDockerLogView,
	useUpdateDockerLogSettings,
} from '../hooks/useDockerLogs';
import type {
	DockerLogBackup,
	DockerLogBackupStatus,
	DockerLogEntry,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	if (!dateStr) return '-';
	const date = new Date(dateStr);
	return date.toLocaleString();
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

function getStatusColor(status: DockerLogBackupStatus): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'pending':
			return {
				bg: 'bg-gray-100 dark:bg-gray-800',
				text: 'text-gray-700 dark:text-gray-300',
				dot: 'bg-gray-400',
			};
		case 'running':
			return {
				bg: 'bg-blue-100 dark:bg-blue-900',
				text: 'text-blue-700 dark:text-blue-300',
				dot: 'bg-blue-500',
			};
		case 'completed':
			return {
				bg: 'bg-green-100 dark:bg-green-900',
				text: 'text-green-700 dark:text-green-300',
				dot: 'bg-green-500',
			};
		case 'failed':
			return {
				bg: 'bg-red-100 dark:bg-red-900',
				text: 'text-red-700 dark:text-red-300',
				dot: 'bg-red-500',
			};
		default:
			return {
				bg: 'bg-gray-100 dark:bg-gray-800',
				text: 'text-gray-700 dark:text-gray-300',
				dot: 'bg-gray-400',
			};
	}
}

function getStreamColor(stream: string): string {
	switch (stream) {
		case 'stdout':
			return 'text-gray-200';
		case 'stderr':
			return 'text-red-400';
		default:
			return 'text-gray-400';
	}
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
				<div className="h-6 w-20 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-8 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

interface LogViewerModalProps {
	backup: DockerLogBackup;
	onClose: () => void;
}

function LogViewerModal({ backup, onClose }: LogViewerModalProps) {
	const [offset, setOffset] = useState(0);
	const limit = 500;

	const { data, isLoading } = useDockerLogView(backup.id, offset, limit);
	const download = useDockerLogDownload();

	const handleDownload = (format: 'json' | 'csv' | 'raw') => {
		download.mutate({ id: backup.id, format });
	};

	const handlePrevPage = () => {
		setOffset(Math.max(0, offset - limit));
	};

	const handleNextPage = () => {
		if (data && offset + limit < data.total_lines) {
			setOffset(offset + limit);
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-6xl mx-4 max-h-[90vh] flex flex-col">
				<div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<div>
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Container Logs: {backup.container_name}
						</h2>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{formatDateTime(backup.start_time)} -{' '}
							{formatDateTime(backup.end_time)}
						</p>
					</div>
					<div className="flex items-center gap-2">
						<div className="flex items-center gap-1 border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden">
							<button
								type="button"
								onClick={() => handleDownload('json')}
								disabled={download.isPending}
								className="px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
							>
								JSON
							</button>
							<button
								type="button"
								onClick={() => handleDownload('csv')}
								disabled={download.isPending}
								className="px-3 py-1.5 text-sm border-l border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
							>
								CSV
							</button>
							<button
								type="button"
								onClick={() => handleDownload('raw')}
								disabled={download.isPending}
								className="px-3 py-1.5 text-sm border-l border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
							>
								RAW
							</button>
						</div>
						<button
							type="button"
							onClick={onClose}
							className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
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
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					</div>
				</div>

				<div className="flex-1 overflow-auto bg-gray-900 p-4 font-mono text-sm">
					{isLoading ? (
						<div className="flex items-center justify-center h-full">
							<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white" />
						</div>
					) : data?.entries.length === 0 ? (
						<div className="flex items-center justify-center h-full text-gray-400">
							No log entries found
						</div>
					) : (
						<div className="space-y-0.5">
							{data?.entries.map((entry: DockerLogEntry) => (
								<div
									key={entry.line_num}
									className="flex gap-2 hover:bg-gray-800 px-2 py-0.5 rounded"
								>
									<span className="text-gray-500 select-none w-12 text-right shrink-0">
										{entry.line_num}
									</span>
									<span className="text-gray-500 shrink-0 w-20">
										{entry.timestamp
											? new Date(entry.timestamp).toLocaleTimeString()
											: '-'}
									</span>
									<span
										className={`shrink-0 w-12 ${getStreamColor(entry.stream)}`}
									>
										{entry.stream}
									</span>
									<span className="text-gray-200 break-all">
										{entry.message}
									</span>
								</div>
							))}
						</div>
					)}
				</div>

				<div className="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<div className="text-sm text-gray-500 dark:text-gray-400">
						{data ? (
							<>
								Showing {offset + 1} to{' '}
								{Math.min(offset + limit, data.total_lines)} of{' '}
								{data.total_lines} lines
							</>
						) : (
							'Loading...'
						)}
					</div>
					<div className="flex items-center gap-2">
						<button
							type="button"
							onClick={handlePrevPage}
							disabled={offset === 0}
							className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
						>
							Previous
						</button>
						<button
							type="button"
							onClick={handleNextPage}
							disabled={!data || offset + limit >= data.total_lines}
							className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
						>
							Next
						</button>
					</div>
				</div>
			</div>
		</div>
	);
}

interface SettingsModalProps {
	agentId: string;
	agentName: string;
	onClose: () => void;
}

function SettingsModal({ agentId, agentName, onClose }: SettingsModalProps) {
	const { data: settings, isLoading } = useDockerLogSettings(agentId);
	const { data: stats } = useDockerLogStorageStats(agentId);
	const updateSettings = useUpdateDockerLogSettings();
	const applyRetention = useApplyDockerLogRetention();

	const [enabled, setEnabled] = useState(settings?.enabled ?? false);
	const [cronExpression, setCronExpression] = useState(
		settings?.cron_expression ?? '0 * * * *',
	);
	const [maxAgeDays, setMaxAgeDays] = useState(
		settings?.retention_policy.max_age_days ?? 30,
	);
	const [compressEnabled, setCompressEnabled] = useState(
		settings?.retention_policy.compress_enabled ?? true,
	);

	// Update local state when settings load
	if (settings && !updateSettings.isPending) {
		if (enabled !== settings.enabled) setEnabled(settings.enabled);
	}

	const handleSave = () => {
		updateSettings.mutate(
			{
				agentId,
				settings: {
					enabled,
					cron_expression: cronExpression,
					retention_policy: {
						max_age_days: maxAgeDays,
						max_size_bytes:
							settings?.retention_policy.max_size_bytes ?? 1073741824,
						max_files_per_day:
							settings?.retention_policy.max_files_per_day ?? 24,
						compress_enabled: compressEnabled,
						compress_level: settings?.retention_policy.compress_level ?? 6,
					},
				},
			},
			{
				onSuccess: () => {
					onClose();
				},
			},
		);
	};

	const handleApplyRetention = () => {
		applyRetention.mutate({ agentId });
	};

	if (isLoading) {
		return (
			<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
				<div className="bg-white dark:bg-gray-800 rounded-lg p-6">
					<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
				</div>
			</div>
		);
	}

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-lg mx-4">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Docker Log Settings: {agentName}
					</h2>
				</div>

				<div className="p-6 space-y-6">
					{stats && (
						<div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4">
							<h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
								Storage Usage
							</h3>
							<div className="grid grid-cols-2 gap-4 text-sm">
								<div>
									<span className="text-gray-500 dark:text-gray-400">
										Total Size:
									</span>
									<span className="ml-2 font-medium text-gray-900 dark:text-white">
										{formatBytes(stats.total_size)}
									</span>
								</div>
								<div>
									<span className="text-gray-500 dark:text-gray-400">
										Backup Files:
									</span>
									<span className="ml-2 font-medium text-gray-900 dark:text-white">
										{stats.total_files}
									</span>
								</div>
							</div>
						</div>
					)}

					<div className="flex items-center justify-between">
						<label
							htmlFor="enabled"
							className="text-sm font-medium text-gray-700 dark:text-gray-300"
						>
							Enable Docker Log Backups
						</label>
						<button
							type="button"
							id="enabled"
							onClick={() => setEnabled(!enabled)}
							className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
								enabled ? 'bg-indigo-600' : 'bg-gray-300 dark:bg-gray-600'
							}`}
						>
							<span
								className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
									enabled ? 'translate-x-6' : 'translate-x-1'
								}`}
							/>
						</button>
					</div>

					<div>
						<label
							htmlFor="cron"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Backup Schedule (Cron Expression)
						</label>
						<input
							type="text"
							id="cron"
							value={cronExpression}
							onChange={(e) => setCronExpression(e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500"
							placeholder="0 * * * *"
						/>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Default: 0 * * * * (every hour)
						</p>
					</div>

					<div>
						<label
							htmlFor="maxAge"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Retention Period (Days)
						</label>
						<input
							type="number"
							id="maxAge"
							value={maxAgeDays}
							onChange={(e) =>
								setMaxAgeDays(
									Math.max(1, Number.parseInt(e.target.value, 10) || 1),
								)
							}
							min={1}
							max={365}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500"
						/>
					</div>

					<div className="flex items-center justify-between">
						<label
							htmlFor="compress"
							className="text-sm font-medium text-gray-700 dark:text-gray-300"
						>
							Compress Logs (gzip)
						</label>
						<button
							type="button"
							id="compress"
							onClick={() => setCompressEnabled(!compressEnabled)}
							className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
								compressEnabled
									? 'bg-indigo-600'
									: 'bg-gray-300 dark:bg-gray-600'
							}`}
						>
							<span
								className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
									compressEnabled ? 'translate-x-6' : 'translate-x-1'
								}`}
							/>
						</button>
					</div>
				</div>

				<div className="p-6 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<button
						type="button"
						onClick={handleApplyRetention}
						disabled={applyRetention.isPending}
						className="px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors disabled:opacity-50"
					>
						{applyRetention.isPending ? 'Cleaning...' : 'Apply Retention Now'}
					</button>
					<div className="flex items-center gap-2">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
						>
							Cancel
						</button>
						<button
							type="button"
							onClick={handleSave}
							disabled={updateSettings.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{updateSettings.isPending ? 'Saving...' : 'Save Settings'}
						</button>
					</div>
				</div>
			</div>
		</div>
	);
}

export function DockerLogs() {
	const [selectedAgent, setSelectedAgent] = useState<string>('');
	const [statusFilter, setStatusFilter] = useState<DockerLogBackupStatus | ''>(
		'',
	);
	const [viewingBackup, setViewingBackup] = useState<DockerLogBackup | null>(
		null,
	);
	const [settingsAgent, setSettingsAgent] = useState<{
		id: string;
		name: string;
	} | null>(null);
	const [deleteConfirm, setDeleteConfirm] = useState<DockerLogBackup | null>(
		null,
	);

	const { data: agentsData } = useAgents();
	const { data, isLoading, isError, refetch } = useDockerLogBackups(
		statusFilter || undefined,
	);
	const deleteBackup = useDeleteDockerLogBackup();

	const handleDelete = (backup: DockerLogBackup) => {
		deleteBackup.mutate(backup.id, {
			onSuccess: () => {
				setDeleteConfirm(null);
			},
		});
	};

	// Filter backups by selected agent
	const filteredBackups = (data?.backups ?? []).filter((backup) =>
		selectedAgent ? backup.agent_id === selectedAgent : true,
	);

	// Get unique containers for the container column
	const getAgentName = (agentId: string): string => {
		const agent = agentsData?.find((a) => a.id === agentId);
		return agent?.hostname ?? agentId.substring(0, 8);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Docker Container Logs
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						View and manage backed up container logs
					</p>
				</div>
				<button
					type="button"
					onClick={() => refetch()}
					className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
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
							d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
						/>
					</svg>
					Refresh
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex flex-wrap items-center gap-4">
						<select
							value={selectedAgent}
							onChange={(e) => setSelectedAgent(e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500"
						>
							<option value="">All Agents</option>
							{agentsData?.map((agent) => (
								<option key={agent.id} value={agent.id}>
									{agent.hostname}
								</option>
							))}
						</select>
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as DockerLogBackupStatus | '')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500"
						>
							<option value="">All Statuses</option>
							<option value="pending">Pending</option>
							<option value="running">Running</option>
							<option value="completed">Completed</option>
							<option value="failed">Failed</option>
						</select>
						{selectedAgent && (
							<button
								type="button"
								onClick={() => {
									const agent = agentsData?.find((a) => a.id === selectedAgent);
									if (agent) {
										setSettingsAgent({ id: agent.id, name: agent.hostname });
									}
								}}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Settings
							</button>
						)}
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load docker log backups</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Container
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Agent
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Size
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
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : filteredBackups && filteredBackups.length > 0 ? (
					<>
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Container
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Agent
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Size
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Lines
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
									{filteredBackups.map((backup) => {
										const statusColor = getStatusColor(backup.status);
										return (
											<tr
												key={backup.id}
												className="hover:bg-gray-50 dark:hover:bg-gray-700"
											>
												<td className="px-6 py-4">
													<div>
														<div className="font-medium text-gray-900 dark:text-white">
															{backup.container_name}
														</div>
														<div className="text-xs text-gray-500 dark:text-gray-400">
															{backup.container_id.substring(0, 12)}
														</div>
													</div>
												</td>
												<td className="px-6 py-4 text-gray-600 dark:text-gray-400">
													{getAgentName(backup.agent_id)}
												</td>
												<td className="px-6 py-4">
													<span
														className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
													>
														<span
															className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`}
														/>
														{backup.status}
													</span>
												</td>
												<td className="px-6 py-4 text-gray-600 dark:text-gray-400">
													{backup.compressed ? (
														<span
															title={`Original: ${formatBytes(backup.original_size)}`}
														>
															{formatBytes(backup.compressed_size)}
														</span>
													) : (
														formatBytes(backup.original_size)
													)}
												</td>
												<td className="px-6 py-4 text-gray-600 dark:text-gray-400">
													{backup.line_count.toLocaleString()}
												</td>
												<td className="px-6 py-4 text-gray-500 dark:text-gray-400 whitespace-nowrap">
													{formatDateTime(backup.created_at)}
												</td>
												<td className="px-6 py-4">
													<div className="flex items-center gap-2">
														{backup.status === 'completed' && (
															<button
																type="button"
																onClick={() => setViewingBackup(backup)}
																className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors"
															>
																View
															</button>
														)}
														<button
															type="button"
															onClick={() => setDeleteConfirm(backup)}
															className="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
														>
															Delete
														</button>
													</div>
												</td>
											</tr>
										);
									})}
								</tbody>
							</table>
						</div>

						<div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700">
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Showing {filteredBackups.length} of {data?.total_count ?? 0}{' '}
								backups
							</div>
						</div>
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
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No docker log backups found
						</h3>
						<p>
							{selectedAgent || statusFilter
								? 'Try adjusting your filters'
								: 'Container logs will appear here when backups are created'}
						</p>
					</div>
				)}
			</div>

			{/* Log Viewer Modal */}
			{viewingBackup && (
				<LogViewerModal
					backup={viewingBackup}
					onClose={() => setViewingBackup(null)}
				/>
			)}

			{/* Settings Modal */}
			{settingsAgent && (
				<SettingsModal
					agentId={settingsAgent.id}
					agentName={settingsAgent.name}
					onClose={() => setSettingsAgent(null)}
				/>
			)}

			{/* Delete Confirmation Modal */}
			{deleteConfirm && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Delete Backup?
						</h3>
						<p className="text-gray-600 dark:text-gray-400 mb-6">
							This will permanently delete the backup for container{' '}
							<strong>{deleteConfirm.container_name}</strong>. This action
							cannot be undone.
						</p>
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setDeleteConfirm(null)}
								className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={() => handleDelete(deleteConfirm)}
								disabled={deleteBackup.isPending}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{deleteBackup.isPending ? 'Deleting...' : 'Delete'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
