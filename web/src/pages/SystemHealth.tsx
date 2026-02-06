import { useState } from 'react';
import {
	useSystemHealth,
	useSystemHealthHistory,
} from '../hooks/useSystemHealth';
import type {
	HealthHistoryRecord,
	ServerError,
	SystemHealthStatus,
} from '../lib/types';

function formatDateTime(dateStr: string): string {
	const date = new Date(dateStr);
	return date.toLocaleString();
}

function formatUptime(seconds: number): string {
	const days = Math.floor(seconds / 86400);
	const hours = Math.floor((seconds % 86400) / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	if (days > 0) {
		return `${days}d ${hours}h ${minutes}m`;
	}
	if (hours > 0) {
		return `${hours}h ${minutes}m`;
	}
	return `${minutes}m`;
}

function formatBytes(bytes: number): string {
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let unitIndex = 0;
	let size = bytes;
	while (size >= 1024 && unitIndex < units.length - 1) {
		size /= 1024;
		unitIndex++;
	}
	return `${size.toFixed(1)} ${units[unitIndex]}`;
}

function getStatusColor(status: SystemHealthStatus): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'healthy':
			return {
				bg: 'bg-green-100 dark:bg-green-900',
				text: 'text-green-700 dark:text-green-300',
				dot: 'bg-green-500',
			};
		case 'warning':
			return {
				bg: 'bg-yellow-100 dark:bg-yellow-900',
				text: 'text-yellow-700 dark:text-yellow-300',
				dot: 'bg-yellow-500',
			};
		case 'critical':
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

function StatusBadge({ status }: { status: SystemHealthStatus }) {
	const color = getStatusColor(status);
	return (
		<span
			className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${color.bg} ${color.text}`}
		>
			<span className={`w-1.5 h-1.5 ${color.dot} rounded-full`} />
			{status.charAt(0).toUpperCase() + status.slice(1)}
		</span>
	);
}

function MetricCard({
	title,
	value,
	subtitle,
	status,
}: {
	title: string;
	value: string | number;
	subtitle?: string;
	status?: SystemHealthStatus;
}) {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
			<div className="flex items-center justify-between">
				<h4 className="text-sm font-medium text-gray-500 dark:text-gray-400">
					{title}
				</h4>
				{status && <StatusBadge status={status} />}
			</div>
			<p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
				{value}
			</p>
			{subtitle && (
				<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
					{subtitle}
				</p>
			)}
		</div>
	);
}

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			<div className="mt-2 h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			<div className="mt-1 h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
		</div>
	);
}

function ErrorLogRow({ error }: { error: ServerError }) {
	const levelColor =
		error.level === 'fatal'
			? 'text-red-600 dark:text-red-400'
			: 'text-orange-600 dark:text-orange-400';

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
				{formatDateTime(error.timestamp)}
			</td>
			<td className="px-4 py-3">
				<span className={`text-xs font-medium uppercase ${levelColor}`}>
					{error.level}
				</span>
			</td>
			<td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
				{error.component || '-'}
			</td>
			<td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
				<span className="truncate max-w-md block">{error.message}</span>
			</td>
		</tr>
	);
}

function HistoryChart({ records }: { records: HealthHistoryRecord[] }) {
	if (records.length === 0) {
		return (
			<div className="text-center py-8 text-gray-500 dark:text-gray-400">
				No historical data available
			</div>
		);
	}

	// Get last 24 records (or all if less)
	const chartData = records.slice(0, 24).reverse();
	const maxMemory = Math.max(...chartData.map((r) => r.memory_alloc_mb));
	const maxGoroutines = Math.max(...chartData.map((r) => r.goroutine_count));

	return (
		<div className="space-y-6">
			<div>
				<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
					Memory Usage (MB)
				</h4>
				<div className="flex items-end gap-1 h-24">
					{chartData.map((record) => {
						const height =
							maxMemory > 0 ? (record.memory_alloc_mb / maxMemory) * 100 : 0;
						const statusColor =
							record.status === 'healthy'
								? 'bg-green-500'
								: record.status === 'warning'
									? 'bg-yellow-500'
									: 'bg-red-500';
						return (
							<div
								key={record.id}
								className={`flex-1 ${statusColor} rounded-t transition-all hover:opacity-80`}
								style={{ height: `${height}%`, minHeight: '4px' }}
								title={`${record.memory_alloc_mb.toFixed(1)} MB at ${formatDateTime(record.timestamp)}`}
							/>
						);
					})}
				</div>
			</div>

			<div>
				<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
					Goroutines
				</h4>
				<div className="flex items-end gap-1 h-24">
					{chartData.map((record) => {
						const height =
							maxGoroutines > 0
								? (record.goroutine_count / maxGoroutines) * 100
								: 0;
						return (
							<div
								key={record.id}
								className="flex-1 bg-indigo-500 rounded-t transition-all hover:opacity-80"
								style={{ height: `${height}%`, minHeight: '4px' }}
								title={`${record.goroutine_count} goroutines at ${formatDateTime(record.timestamp)}`}
							/>
						);
					})}
				</div>
			</div>

			<div>
				<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
					Pending Backups
				</h4>
				<div className="flex items-end gap-1 h-24">
					{chartData.map((record) => {
						const maxPending =
							Math.max(...chartData.map((r) => r.pending_backups)) || 1;
						const height = (record.pending_backups / maxPending) * 100;
						return (
							<div
								key={record.id}
								className="flex-1 bg-purple-500 rounded-t transition-all hover:opacity-80"
								style={{ height: `${height}%`, minHeight: '4px' }}
								title={`${record.pending_backups} pending at ${formatDateTime(record.timestamp)}`}
							/>
						);
					})}
				</div>
			</div>
		</div>
	);
}

export function SystemHealth() {
	const [autoRefresh, setAutoRefresh] = useState(true);
	const { data: health, isLoading, isError, refetch } = useSystemHealth();
	const { data: historyData } = useSystemHealthHistory();

	if (isError) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						System Health
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Monitor system status and performance
					</p>
				</div>
				<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6 text-center">
					<p className="text-red-600 dark:text-red-400 font-medium">
						Failed to load system health data
					</p>
					<p className="text-red-500 dark:text-red-300 text-sm mt-1">
						You may not have superuser access to view this page
					</p>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						System Health
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Monitor system status and performance
					</p>
				</div>
				<div className="flex items-center gap-4">
					<label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
						<input
							type="checkbox"
							checked={autoRefresh}
							onChange={(e) => setAutoRefresh(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						Auto-refresh (30s)
					</label>
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
			</div>

			{/* Overall Status */}
			{health && (
				<div
					className={`rounded-lg border p-4 ${
						health.status === 'healthy'
							? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
							: health.status === 'warning'
								? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800'
								: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
					}`}
				>
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-3">
							<div
								className={`w-3 h-3 rounded-full ${
									health.status === 'healthy'
										? 'bg-green-500'
										: health.status === 'warning'
											? 'bg-yellow-500'
											: 'bg-red-500'
								} animate-pulse`}
							/>
							<span className="font-medium text-gray-900 dark:text-white">
								System Status:{' '}
								{health.status.charAt(0).toUpperCase() + health.status.slice(1)}
							</span>
						</div>
						<span className="text-sm text-gray-500 dark:text-gray-400">
							Last updated: {formatDateTime(health.timestamp)}
						</span>
					</div>
					{health.issues && health.issues.length > 0 && (
						<ul className="mt-2 space-y-1">
							{health.issues.map((issue) => (
								<li
									key={issue}
									className="text-sm text-gray-700 dark:text-gray-300 flex items-center gap-2"
								>
									<span className="text-yellow-500">!</span>
									{issue}
								</li>
							))}
						</ul>
					)}
				</div>
			)}

			{/* Server Status */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Server Status
					</h2>
				</div>
				<div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
					{isLoading ? (
						<>
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
						</>
					) : health ? (
						<>
							<MetricCard
								title="Memory Usage"
								value={`${health.server.memory_usage.toFixed(1)}%`}
								subtitle={`${health.server.memory_alloc_mb.toFixed(1)} MB allocated`}
								status={health.server.status}
							/>
							<MetricCard
								title="Goroutines"
								value={health.server.goroutine_count}
								subtitle={`${health.server.num_cpu} CPUs`}
							/>
							<MetricCard
								title="Uptime"
								value={formatUptime(health.server.uptime_seconds)}
								subtitle={health.server.go_version}
							/>
							<MetricCard
								title="System Memory"
								value={`${health.server.memory_sys_mb.toFixed(1)} MB`}
								subtitle="Total system allocation"
							/>
						</>
					) : null}
				</div>
			</div>

			{/* Database Status */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Database Status
					</h2>
				</div>
				<div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
					{isLoading ? (
						<>
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
						</>
					) : health ? (
						<>
							<MetricCard
								title="Connection"
								value={health.database.connected ? 'Connected' : 'Disconnected'}
								subtitle={`Latency: ${health.database.latency}`}
								status={health.database.status}
							/>
							<MetricCard
								title="Active Connections"
								value={health.database.active_connections}
								subtitle={
									health.database.max_connections > 0
										? `Max: ${health.database.max_connections}`
										: undefined
								}
							/>
							<MetricCard
								title="Database Size"
								value={
									health.database.size_formatted ||
									formatBytes(health.database.size_bytes)
								}
							/>
							<MetricCard
								title="Connection Pool"
								value={
									health.database.max_connections > 0
										? `${((health.database.active_connections / health.database.max_connections) * 100).toFixed(0)}%`
										: 'N/A'
								}
								subtitle="Pool utilization"
							/>
						</>
					) : null}
				</div>
			</div>

			{/* Queue Status */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Backup Queue Status
					</h2>
				</div>
				<div className="p-6 grid grid-cols-1 md:grid-cols-3 gap-4">
					{isLoading ? (
						<>
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
						</>
					) : health ? (
						<>
							<MetricCard
								title="Pending Backups"
								value={health.queue.pending_backups}
								status={health.queue.status}
							/>
							<MetricCard
								title="Running Backups"
								value={health.queue.running_backups}
							/>
							<MetricCard
								title="Total Queued"
								value={health.queue.total_queued}
							/>
						</>
					) : null}
				</div>
			</div>

			{/* Background Jobs */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Background Jobs
					</h2>
				</div>
				<div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-4">
					{isLoading ? (
						<>
							<LoadingCard />
							<LoadingCard />
						</>
					) : health ? (
						<>
							<MetricCard
								title="Goroutine Count"
								value={health.background_jobs.goroutine_count}
								status={health.background_jobs.status}
							/>
							<MetricCard
								title="Active Jobs"
								value={health.background_jobs.active_jobs}
								subtitle="Estimated active background tasks"
							/>
						</>
					) : null}
				</div>
			</div>

			{/* Historical Data */}
			{historyData && historyData.records.length > 0 && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Historical Data (24h)
						</h2>
					</div>
					<div className="p-6">
						<HistoryChart records={historyData.records} />
					</div>
				</div>
			)}

			{/* Recent Errors */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Recent Errors
					</h2>
				</div>
				{isLoading ? (
					<div className="p-6 space-y-4 animate-pulse">
						<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full" />
						<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full" />
						<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full" />
					</div>
				) : health && health.recent_errors.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Timestamp
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Level
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Component
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Message
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{health.recent_errors.map((error) => (
									<ErrorLogRow key={error.id} error={error} />
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<svg
							aria-hidden="true"
							className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600"
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
						<p>No recent errors</p>
					</div>
				)}
			</div>
		</div>
	);
}
