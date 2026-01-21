import { Link } from 'react-router-dom';
import { useFleetHealth } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import {
	useBackupDurationTrend,
	useDailyBackupStats,
	useDashboardStats,
	useStorageGrowthTrend,
} from '../hooks/useMetrics';
import type { FleetHealthSummary } from '../lib/types';
import {
	formatBytes,
	formatDate,
	formatDedupRatio,
	formatDurationMs,
	formatPercent,
	getBackupStatusColor,
	getDedupRatioColor,
	getSpaceSavedColor,
	getSuccessRateColor,
	truncateSnapshotId,
} from '../lib/utils';

interface StatCardProps {
	title: string;
	value: string;
	subtitle: string;
	icon: React.ReactNode;
	isLoading?: boolean;
	trend?: 'up' | 'down' | 'neutral';
	trendValue?: string;
}

function StatCard({
	title,
	value,
	subtitle,
	icon,
	isLoading,
	trend,
	trendValue,
}: StatCardProps) {
	const trendColors = {
		up: 'text-green-600',
		down: 'text-red-600',
		neutral: 'text-gray-500',
	};

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between">
				<div>
					<p className="text-sm font-medium text-gray-600">{title}</p>
					<p className="text-2xl font-bold text-gray-900 mt-1">
						{isLoading ? (
							<span className="inline-block w-8 h-7 bg-gray-200 rounded animate-pulse" />
						) : (
							value
						)}
					</p>
					<div className="flex items-center gap-2 mt-1">
						<p className="text-sm text-gray-500">{subtitle}</p>
						{trend && trendValue && (
							<span className={`text-xs font-medium ${trendColors[trend]}`}>
								{trend === 'up' ? '+' : trend === 'down' ? '-' : ''}
								{trendValue}
							</span>
						)}
					</div>
				</div>
				<div className="p-3 bg-indigo-50 rounded-lg text-indigo-600">
					{icon}
				</div>
			</div>
		</div>
	);
}

function LoadingRow() {
	return (
		<div className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0">
			<div className="space-y-2">
				<div className="h-4 w-32 bg-gray-200 rounded animate-pulse" />
				<div className="h-3 w-24 bg-gray-100 rounded animate-pulse" />
			</div>
			<div className="h-6 w-20 bg-gray-200 rounded-full animate-pulse" />
		</div>
	);
}

function SuccessRateWidget({
	rate7d,
	rate30d,
	isLoading,
}: {
	rate7d: number;
	rate30d: number;
	isLoading: boolean;
}) {
	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<h3 className="text-lg font-semibold text-gray-900 mb-4">
				Backup Success Rate
			</h3>
			{isLoading ? (
				<div className="space-y-4">
					<div className="animate-pulse h-4 bg-gray-200 rounded w-3/4" />
					<div className="animate-pulse h-4 bg-gray-200 rounded w-2/3" />
				</div>
			) : (
				<div className="space-y-4">
					<div>
						<div className="flex items-center justify-between mb-1">
							<span className="text-sm text-gray-600">Last 7 days</span>
							<span
								className={`text-sm font-semibold ${getSuccessRateColor(rate7d)}`}
							>
								{formatPercent(rate7d)}
							</span>
						</div>
						<div className="w-full bg-gray-200 rounded-full h-2">
							<div
								className={`h-2 rounded-full ${rate7d >= 95 ? 'bg-green-500' : rate7d >= 80 ? 'bg-yellow-500' : 'bg-red-500'}`}
								style={{ width: `${Math.min(rate7d, 100)}%` }}
							/>
						</div>
					</div>
					<div>
						<div className="flex items-center justify-between mb-1">
							<span className="text-sm text-gray-600">Last 30 days</span>
							<span
								className={`text-sm font-semibold ${getSuccessRateColor(rate30d)}`}
							>
								{formatPercent(rate30d)}
							</span>
						</div>
						<div className="w-full bg-gray-200 rounded-full h-2">
							<div
								className={`h-2 rounded-full ${rate30d >= 95 ? 'bg-green-500' : rate30d >= 80 ? 'bg-yellow-500' : 'bg-red-500'}`}
								style={{ width: `${Math.min(rate30d, 100)}%` }}
							/>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}

function StorageGrowthChart({
	data,
	isLoading,
}: {
	data: { date: string; total_size: number; raw_size: number }[];
	isLoading: boolean;
}) {
	const maxSize = Math.max(...data.map((d) => d.total_size), 1);

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between mb-4">
				<h3 className="text-lg font-semibold text-gray-900">Storage Growth</h3>
				<Link
					to="/stats"
					className="text-sm text-indigo-600 hover:text-indigo-800"
				>
					View Details
				</Link>
			</div>
			{isLoading ? (
				<div className="h-40 flex items-center justify-center">
					<div className="animate-pulse h-full w-full bg-gray-100 rounded" />
				</div>
			) : data.length === 0 ? (
				<div className="h-40 flex items-center justify-center text-gray-500">
					No storage data yet
				</div>
			) : (
				<div className="h-40 flex items-end gap-1">
					{data.slice(-14).map((point, i) => (
						<div
							key={point.date}
							className="flex-1 flex flex-col items-center gap-1"
						>
							<div
								className="w-full bg-indigo-500 rounded-t hover:bg-indigo-600 transition-colors"
								style={{
									height: `${(point.total_size / maxSize) * 100}%`,
									minHeight: '4px',
								}}
								title={`${formatBytes(point.total_size)} on ${new Date(point.date).toLocaleDateString()}`}
							/>
							{i % 2 === 0 && (
								<span className="text-[10px] text-gray-400">
									{new Date(point.date).toLocaleDateString('en-US', {
										month: 'short',
										day: 'numeric',
									})}
								</span>
							)}
						</div>
					))}
				</div>
			)}
		</div>
	);
}

function BackupDurationChart({
	data,
	isLoading,
}: {
	data: {
		date: string;
		avg_duration_ms: number;
		max_duration_ms: number;
		backup_count: number;
	}[];
	isLoading: boolean;
}) {
	const maxDuration = Math.max(...data.map((d) => d.avg_duration_ms), 1);

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<h3 className="text-lg font-semibold text-gray-900 mb-4">
				Backup Duration Trends
			</h3>
			{isLoading ? (
				<div className="h-40 flex items-center justify-center">
					<div className="animate-pulse h-full w-full bg-gray-100 rounded" />
				</div>
			) : data.length === 0 ? (
				<div className="h-40 flex items-center justify-center text-gray-500">
					No backup duration data yet
				</div>
			) : (
				<div className="h-40 flex items-end gap-1">
					{data.slice(-14).map((point, i) => (
						<div
							key={point.date}
							className="flex-1 flex flex-col items-center gap-1"
						>
							<div
								className="w-full bg-cyan-500 rounded-t hover:bg-cyan-600 transition-colors"
								style={{
									height: `${(point.avg_duration_ms / maxDuration) * 100}%`,
									minHeight: '4px',
								}}
								title={`Avg: ${formatDurationMs(point.avg_duration_ms)} (${point.backup_count} backups)`}
							/>
							{i % 2 === 0 && (
								<span className="text-[10px] text-gray-400">
									{new Date(point.date).toLocaleDateString('en-US', {
										month: 'short',
										day: 'numeric',
									})}
								</span>
							)}
						</div>
					))}
				</div>
			)}
		</div>
	);
}

function DailyBackupsChart({
	data,
	isLoading,
}: {
	data: {
		date: string;
		total: number;
		successful: number;
		failed: number;
	}[];
	isLoading: boolean;
}) {
	const maxCount = Math.max(...data.map((d) => d.total), 1);

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between mb-4">
				<h3 className="text-lg font-semibold text-gray-900">Daily Backups</h3>
				<div className="flex items-center gap-4 text-xs">
					<span className="flex items-center gap-1">
						<span className="w-3 h-3 bg-green-500 rounded" /> Successful
					</span>
					<span className="flex items-center gap-1">
						<span className="w-3 h-3 bg-red-500 rounded" /> Failed
					</span>
				</div>
			</div>
			{isLoading ? (
				<div className="h-40 flex items-center justify-center">
					<div className="animate-pulse h-full w-full bg-gray-100 rounded" />
				</div>
			) : data.length === 0 ? (
				<div className="h-40 flex items-center justify-center text-gray-500">
					No backup data yet
				</div>
			) : (
				<div className="h-40 flex items-end gap-1">
					{data.slice(-14).map((point, i) => (
						<div
							key={point.date}
							className="flex-1 flex flex-col items-center gap-1"
						>
							<div className="w-full flex flex-col" style={{ height: '100%' }}>
								<div
									className="w-full bg-red-500 rounded-t"
									style={{
										height:
											point.total > 0
												? `${(point.failed / maxCount) * 100}%`
												: '0%',
										minHeight: point.failed > 0 ? '2px' : '0',
									}}
								/>
								<div
									className="w-full bg-green-500"
									style={{
										height:
											point.total > 0
												? `${(point.successful / maxCount) * 100}%`
												: '0%',
										minHeight: point.successful > 0 ? '2px' : '0',
									}}
									title={`${point.successful} successful, ${point.failed} failed`}
								/>
							</div>
							{i % 2 === 0 && (
								<span className="text-[10px] text-gray-400">
									{new Date(point.date).toLocaleDateString('en-US', {
										month: 'short',
										day: 'numeric',
									})}
								</span>
							)}
						</div>
					))}
				</div>
			)}
		</div>
	);
}

function FleetHealthWidget({
	data,
	isLoading,
}: {
	data: FleetHealthSummary | undefined;
	isLoading: boolean;
}) {
	const totalAgents = data?.total_agents ?? 0;
	const healthyPercent =
		totalAgents > 0 ? ((data?.healthy_count ?? 0) / totalAgents) * 100 : 0;

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between mb-4">
				<h3 className="text-lg font-semibold text-gray-900">Fleet Health</h3>
				<Link
					to="/agents"
					className="text-sm text-indigo-600 hover:text-indigo-800"
				>
					View Agents
				</Link>
			</div>
			{isLoading ? (
				<div className="space-y-4">
					<div className="animate-pulse h-20 bg-gray-100 rounded" />
					<div className="animate-pulse h-4 bg-gray-200 rounded w-3/4" />
				</div>
			) : totalAgents === 0 ? (
				<div className="text-center py-8 text-gray-500">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
						/>
					</svg>
					<p>No agents registered</p>
					<p className="text-sm">Register an agent to monitor fleet health</p>
				</div>
			) : (
				<div className="space-y-4">
					{/* Health Distribution */}
					<div className="flex items-center gap-4">
						{/* Health Ring */}
						<div className="relative w-20 h-20">
							<svg
								aria-hidden="true"
								className="w-20 h-20 -rotate-90"
								viewBox="0 0 36 36"
							>
								{/* Background circle */}
								<circle
									cx="18"
									cy="18"
									r="15.5"
									fill="none"
									stroke="#e5e7eb"
									strokeWidth="3"
								/>
								{/* Healthy segment */}
								<circle
									cx="18"
									cy="18"
									r="15.5"
									fill="none"
									stroke="#22c55e"
									strokeWidth="3"
									strokeDasharray={`${healthyPercent} 100`}
									strokeLinecap="round"
								/>
							</svg>
							<div className="absolute inset-0 flex items-center justify-center">
								<span className="text-lg font-bold text-gray-900">
									{Math.round(healthyPercent)}%
								</span>
							</div>
						</div>

						{/* Agent counts */}
						<div className="flex-1 grid grid-cols-2 gap-2">
							<div className="flex items-center gap-2">
								<span className="w-3 h-3 bg-green-500 rounded-full" />
								<div>
									<p className="text-sm font-medium text-gray-900">
										{data?.healthy_count ?? 0}
									</p>
									<p className="text-xs text-gray-500">Healthy</p>
								</div>
							</div>
							<div className="flex items-center gap-2">
								<span className="w-3 h-3 bg-yellow-500 rounded-full" />
								<div>
									<p className="text-sm font-medium text-gray-900">
										{data?.warning_count ?? 0}
									</p>
									<p className="text-xs text-gray-500">Warning</p>
								</div>
							</div>
							<div className="flex items-center gap-2">
								<span className="w-3 h-3 bg-red-500 rounded-full" />
								<div>
									<p className="text-sm font-medium text-gray-900">
										{data?.critical_count ?? 0}
									</p>
									<p className="text-xs text-gray-500">Critical</p>
								</div>
							</div>
							<div className="flex items-center gap-2">
								<span className="w-3 h-3 bg-gray-400 rounded-full" />
								<div>
									<p className="text-sm font-medium text-gray-900">
										{data?.unknown_count ?? 0}
									</p>
									<p className="text-xs text-gray-500">Unknown</p>
								</div>
							</div>
						</div>
					</div>

					{/* Resource Averages */}
					<div className="border-t border-gray-200 pt-4">
						<p className="text-sm font-medium text-gray-600 mb-3">
							Average Resource Usage
						</p>
						<div className="space-y-2">
							<div>
								<div className="flex items-center justify-between text-xs mb-1">
									<span className="text-gray-500">CPU</span>
									<span className="font-medium text-gray-700">
										{formatPercent(data?.avg_cpu_usage)}
									</span>
								</div>
								<div className="w-full bg-gray-200 rounded-full h-1.5">
									<div
										className={`h-1.5 rounded-full ${
											(data?.avg_cpu_usage ?? 0) >= 80
												? 'bg-red-500'
												: (data?.avg_cpu_usage ?? 0) >= 60
													? 'bg-yellow-500'
													: 'bg-green-500'
										}`}
										style={{
											width: `${Math.min(data?.avg_cpu_usage ?? 0, 100)}%`,
										}}
									/>
								</div>
							</div>
							<div>
								<div className="flex items-center justify-between text-xs mb-1">
									<span className="text-gray-500">Memory</span>
									<span className="font-medium text-gray-700">
										{formatPercent(data?.avg_memory_usage)}
									</span>
								</div>
								<div className="w-full bg-gray-200 rounded-full h-1.5">
									<div
										className={`h-1.5 rounded-full ${
											(data?.avg_memory_usage ?? 0) >= 85
												? 'bg-red-500'
												: (data?.avg_memory_usage ?? 0) >= 70
													? 'bg-yellow-500'
													: 'bg-blue-500'
										}`}
										style={{
											width: `${Math.min(data?.avg_memory_usage ?? 0, 100)}%`,
										}}
									/>
								</div>
							</div>
							<div>
								<div className="flex items-center justify-between text-xs mb-1">
									<span className="text-gray-500">Disk</span>
									<span className="font-medium text-gray-700">
										{formatPercent(data?.avg_disk_usage)}
									</span>
								</div>
								<div className="w-full bg-gray-200 rounded-full h-1.5">
									<div
										className={`h-1.5 rounded-full ${
											(data?.avg_disk_usage ?? 0) >= 80
												? 'bg-red-500'
												: (data?.avg_disk_usage ?? 0) >= 60
													? 'bg-yellow-500'
													: 'bg-purple-500'
										}`}
										style={{
											width: `${Math.min(data?.avg_disk_usage ?? 0, 100)}%`,
										}}
									/>
								</div>
							</div>
						</div>
					</div>

					{/* Alerts */}
					{((data?.critical_count ?? 0) > 0 ||
						(data?.warning_count ?? 0) > 0) && (
						<div className="border-t border-gray-200 pt-4">
							{(data?.critical_count ?? 0) > 0 && (
								<div className="flex items-center gap-2 text-red-700 bg-red-50 px-3 py-2 rounded-lg text-sm mb-2">
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
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									{data?.critical_count} agent(s) in critical state
								</div>
							)}
							{(data?.warning_count ?? 0) > 0 && (
								<div className="flex items-center gap-2 text-yellow-700 bg-yellow-50 px-3 py-2 rounded-lg text-sm">
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
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									{data?.warning_count} agent(s) need attention
								</div>
							)}
						</div>
					)}
				</div>
			)}
		</div>
	);
}

export function Dashboard() {
	const { data: dashboardStats, isLoading: statsLoading } = useDashboardStats();
	const { data: fleetHealthResponse, isLoading: fleetHealthLoading } =
		useFleetHealth();
	const { data: backups, isLoading: backupsLoading } = useBackups();
	const { data: dailyStats, isLoading: dailyStatsLoading } =
		useDailyBackupStats(30);
	const { data: storageGrowth, isLoading: storageGrowthLoading } =
		useStorageGrowthTrend(30);
	const { data: durationTrend, isLoading: durationTrendLoading } =
		useBackupDurationTrend(30);

	const recentBackups = backups?.slice(0, 5) ?? [];

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
				<p className="text-gray-600 mt-1">Overview of your backup system</p>
			</div>

			{/* Main Stats Row */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				<StatCard
					title="Total Backup Size"
					value={formatBytes(dashboardStats?.total_backup_size ?? 0)}
					subtitle={`${dashboardStats?.repository_count ?? 0} repositories`}
					isLoading={statsLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
							/>
						</svg>
					}
				/>
				<StatCard
					title="Active Agents"
					value={`${dashboardStats?.agent_online ?? 0}/${dashboardStats?.agent_total ?? 0}`}
					subtitle={`${dashboardStats?.agent_offline ?? 0} offline`}
					isLoading={statsLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
							/>
						</svg>
					}
				/>
				<StatCard
					title="Failed (24h)"
					value={String(dashboardStats?.backup_failed_24h ?? 0)}
					subtitle={`${dashboardStats?.backup_running ?? 0} running`}
					isLoading={statsLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
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
					}
				/>
				<StatCard
					title="Scheduled Jobs"
					value={String(dashboardStats?.schedule_enabled ?? 0)}
					subtitle={`${dashboardStats?.schedule_count ?? 0} total`}
					isLoading={statsLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					}
				/>
			</div>

			{/* Fleet Health, Success Rate and Storage Efficiency Row */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				<FleetHealthWidget
					data={fleetHealthResponse}
					isLoading={fleetHealthLoading}
				/>
				<SuccessRateWidget
					rate7d={dashboardStats?.success_rate_7d ?? 0}
					rate30d={dashboardStats?.success_rate_30d ?? 0}
					isLoading={statsLoading}
				/>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="flex items-center justify-between mb-4">
						<h3 className="text-lg font-semibold text-gray-900">
							Storage Efficiency
						</h3>
						<Link
							to="/stats"
							className="text-sm text-indigo-600 hover:text-indigo-800"
						>
							View Details
						</Link>
					</div>
					{statsLoading ? (
						<div className="grid grid-cols-2 gap-4">
							{[1, 2, 3, 4].map((i) => (
								<div key={i} className="animate-pulse">
									<div className="h-4 w-24 bg-gray-200 rounded mb-2" />
									<div className="h-8 w-20 bg-gray-200 rounded" />
								</div>
							))}
						</div>
					) : dashboardStats ? (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-600">Dedup Ratio</p>
								<p
									className={`text-2xl font-bold mt-1 ${getDedupRatioColor(dashboardStats.avg_dedup_ratio)}`}
								>
									{formatDedupRatio(dashboardStats.avg_dedup_ratio)}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">Space Saved</p>
								<p
									className={`text-2xl font-bold mt-1 ${getSpaceSavedColor(dashboardStats.total_backup_size > 0 ? (dashboardStats.total_space_saved / dashboardStats.total_backup_size) * 100 : 0)}`}
								>
									{formatBytes(dashboardStats.total_space_saved)}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">
									Actual Storage
								</p>
								<p className="text-2xl font-bold text-gray-900 mt-1">
									{formatBytes(dashboardStats.total_raw_size)}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">
									Original Size
								</p>
								<p className="text-2xl font-bold text-gray-900 mt-1">
									{formatBytes(dashboardStats.total_backup_size)}
								</p>
							</div>
						</div>
					) : (
						<div className="text-center py-8 text-gray-500">
							No storage stats available
						</div>
					)}
				</div>
			</div>

			{/* Charts Row */}
			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<StorageGrowthChart
					data={storageGrowth ?? []}
					isLoading={storageGrowthLoading}
				/>
				<DailyBackupsChart
					data={dailyStats ?? []}
					isLoading={dailyStatsLoading}
				/>
			</div>

			{/* Backup Duration and Recent Backups Row */}
			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<BackupDurationChart
					data={durationTrend ?? []}
					isLoading={durationTrendLoading}
				/>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900">
							Recent Backups
						</h2>
						<Link
							to="/backups"
							className="text-sm text-indigo-600 hover:text-indigo-800"
						>
							View All
						</Link>
					</div>
					{backupsLoading ? (
						<div className="space-y-1">
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</div>
					) : recentBackups.length === 0 ? (
						<div className="text-center py-8 text-gray-500">
							<svg
								aria-hidden="true"
								className="w-12 h-12 mx-auto mb-3 text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
								/>
							</svg>
							<p>No backups yet</p>
							<p className="text-sm">Configure an agent to start backing up</p>
						</div>
					) : (
						<div className="space-y-1">
							{recentBackups.map((backup) => {
								const statusColor = getBackupStatusColor(backup.status);
								return (
									<div
										key={backup.id}
										className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0"
									>
										<div>
											<p className="text-sm font-medium text-gray-900">
												{truncateSnapshotId(backup.snapshot_id) || 'Running...'}
											</p>
											<p className="text-xs text-gray-500">
												{formatDate(backup.started_at)}
												{backup.size_bytes !== undefined &&
													` - ${formatBytes(backup.size_bytes)}`}
											</p>
										</div>
										<span
											className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
										>
											<span
												className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`}
											/>
											{backup.status}
										</span>
									</div>
								);
							})}
						</div>
					)}
				</div>
			</div>

			{/* System Status */}
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<h2 className="text-lg font-semibold text-gray-900 mb-4">
					System Status
				</h2>
				<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
					<div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
						<span className="text-gray-600">Server</span>
						<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
							<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
							Online
						</span>
					</div>
					<div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
						<span className="text-gray-600">Database</span>
						<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
							<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
							Connected
						</span>
					</div>
					<div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
						<span className="text-gray-600">Scheduler</span>
						{(dashboardStats?.schedule_enabled ?? 0) > 0 ? (
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								Active ({dashboardStats?.schedule_enabled} jobs)
							</span>
						) : (
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
								<span className="w-1.5 h-1.5 bg-gray-400 rounded-full" />
								Idle
							</span>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}
