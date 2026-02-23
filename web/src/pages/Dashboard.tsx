import { Link } from 'react-router-dom';
import { ActivityFeedWidget } from '../components/features/ActivityFeed';
import { MiniBackupCalendar } from '../components/features/BackupCalendar';
import { HelpTooltip } from '../components/ui/HelpTooltip';
import {
	DashboardBackupListSkeleton,
	DashboardFavoritesSkeleton,
	DashboardQueueSkeleton,
} from '../components/ui/PageSkeletons';
import {
	Skeleton,
	TextSkeleton,
	skeletonKeys,
} from '../components/ui/Skeleton';
import { StarButton } from '../components/ui/StarButton';
import { useAgents, useFleetHealth } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import { useFavorites } from '../hooks/useFavorites';
import { useLocale } from '../hooks/useLocale';
import {
	useBackupDurationTrend,
	useDailyBackupStats,
	useDashboardStats,
	useStorageGrowthTrend,
} from '../hooks/useMetrics';
import { useDRStatus } from '../hooks/useDRRunbooks';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import { useStorageStatsSummary } from '../hooks/useStorageStats';
import { dashboardHelp } from '../lib/help-content';
import type { Agent, FleetHealthSummary, Repository, Schedule } from '../lib/types';
import {
	formatBytes,
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
	helpContent?: string;
	helpTitle?: string;
	docsUrl?: string;
}

function StatCard({
	title,
	value,
	subtitle,
	icon,
	isLoading,
	helpContent,
	helpTitle,
	docsUrl,
}: StatCardProps) {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
			<div className="flex items-center justify-between">
				<div>
					<p className="flex items-center gap-1.5 text-sm font-medium text-gray-600 dark:text-gray-400">
						{title}
						{helpContent && (
							<HelpTooltip
								content={helpContent}
								title={helpTitle}
								docsUrl={docsUrl}
							/>
						)}
					</p>
					<p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
						{isLoading ? (
							<TextSkeleton width="xs" size="xl" className="inline-block" />
						) : (
							value
						)}
					</p>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						{subtitle}
					</p>
				</div>
				<div className="p-3 bg-indigo-50 dark:bg-indigo-900/50 rounded-lg text-indigo-600 dark:text-indigo-400">
					{icon}
				</div>
			</div>
		</div>
	);
}

function LoadingRow() {
	return (
		<div className="flex items-center justify-between py-3 border-b border-gray-100 dark:border-gray-700 last:border-0">
			<div className="space-y-2">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
				<div className="h-3 w-24 bg-gray-100 dark:bg-gray-600 rounded animate-pulse" />
			</div>
			<div className="h-6 w-20 bg-gray-200 dark:bg-gray-700 rounded-full animate-pulse" />
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
	const { data: dashboardStats, isLoading: dashboardStatsLoading } =
		useDashboardStats();
	const { data: agents, isLoading: agentsLoading } = useAgents();
	const { data: repositories, isLoading: reposLoading } = useRepositories();
	const { data: schedules, isLoading: schedulesLoading } = useSchedules();
	const { data: backups, isLoading: backupsLoading } = useBackups();
	const { data: dailyStats, isLoading: dailyStatsLoading } =
		useDailyBackupStats(30);
	const { data: storageGrowth, isLoading: storageGrowthLoading } =
		useStorageGrowthTrend(30);
	const { data: durationTrend, isLoading: durationTrendLoading } =
		useBackupDurationTrend(30);
	const { data: storageStats, isLoading: statsLoading } =
		useStorageStatsSummary();
	const { data: drStatus, isLoading: drStatusLoading } = useDRStatus();
	const { data: favorites, isLoading: favoritesLoading } = useFavorites();
	const { data: fleetHealthResponse, isLoading: fleetHealthLoading } =
		useFleetHealth();
	const { t, formatRelativeTime, formatBytes: localFormatBytes, formatPercent: localFormatPercent } = useLocale();

	const activeAgents = dashboardStats?.agent_online ?? 0;
	const enabledSchedules = dashboardStats?.schedule_enabled ?? 0;
	const recentBackups = backups?.slice(0, 5) ?? [];
	const runningBackups = backups?.filter((b) => b.status === 'running') ?? [];

	// Calculate priority queue summary from schedules
	const priorityQueueSummary = {
		high: schedules?.filter((s: Schedule) => s.enabled && s.priority === 1).length ?? 0,
		medium: schedules?.filter((s: Schedule) => s.enabled && s.priority === 2).length ?? 0,
		low: schedules?.filter((s: Schedule) => s.enabled && s.priority === 3).length ?? 0,
		preemptible:
			schedules?.filter((s: Schedule) => s.enabled && s.preemptible).length ?? 0,
	};

	// Get favorite items with details
	const favoriteAgents =
		favorites
			?.filter((f) => f.entity_type === 'agent')
			.map((f) => agents?.find((a: Agent) => a.id === f.entity_id))
			.filter((a): a is Agent => a !== undefined) ?? [];
	const favoriteSchedules =
		favorites
			?.filter((f) => f.entity_type === 'schedule')
			.map((f) => schedules?.find((s: Schedule) => s.id === f.entity_id))
			.filter((s): s is Schedule => s !== undefined) ?? [];
	const favoriteRepos =
		favorites
			?.filter((f) => f.entity_type === 'repository')
			.map((f) => repositories?.find((r: Repository) => r.id === f.entity_id))
			.filter((r): r is Repository => r !== undefined) ?? [];
	const hasFavorites =
		favoriteAgents.length > 0 ||
		favoriteSchedules.length > 0 ||
		favoriteRepos.length > 0;

	const isLoading =
		dashboardStatsLoading || agentsLoading || reposLoading || schedulesLoading || backupsLoading;

	return (
		<div className="space-y-6">
			<div>
				<div className="flex items-center gap-2">
					<h1 className="text-2xl font-bold text-gray-900">
						{t('dashboard.title')}
					</h1>
					<HelpTooltip
						content="Your central hub for monitoring backup status, agent health, and storage efficiency across your infrastructure."
						title="Dashboard Overview"
						docsUrl="/docs/getting-started"
						size="md"
					/>
				</div>
				<p className="text-gray-600 mt-1">{t('dashboard.subtitle')}</p>
			</div>

			{/* Main Stats Row */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-6">
				<StatCard
					title={t('dashboard.activeAgents')}
					value={`${dashboardStats?.agent_online ?? 0}/${dashboardStats?.agent_total ?? 0}`}
					subtitle={`${dashboardStats?.agent_offline ?? 0} offline`}
					isLoading={dashboardStatsLoading}
					helpContent={dashboardHelp.activeAgents.content}
					helpTitle={dashboardHelp.activeAgents.title}
					docsUrl={dashboardHelp.activeAgents.docsUrl}
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
					title={t('dashboard.repositories')}
					value={String(dashboardStats?.repository_count ?? 0)}
					subtitle={t('dashboard.backupDestinations')}
					isLoading={dashboardStatsLoading}
					helpContent={dashboardHelp.repositories.content}
					helpTitle={dashboardHelp.repositories.title}
					docsUrl={dashboardHelp.repositories.docsUrl}
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
								d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
							/>
						</svg>
					}
				/>
				<StatCard
					title={t('dashboard.scheduledJobs')}
					value={String(dashboardStats?.schedule_enabled ?? 0)}
					subtitle={`${dashboardStats?.schedule_count ?? 0} total`}
					isLoading={dashboardStatsLoading}
					helpContent={dashboardHelp.scheduledJobs.content}
					helpTitle={dashboardHelp.scheduledJobs.title}
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
				<StatCard
					title="Failed (24h)"
					value={String(dashboardStats?.backup_failed_24h ?? 0)}
					subtitle={`${dashboardStats?.backup_running ?? 0} running`}
					isLoading={dashboardStatsLoading}
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
					title={t('dashboard.totalBackups')}
					value={String(dashboardStats?.backup_total ?? 0)}
					subtitle={t('dashboard.allTime')}
					isLoading={dashboardStatsLoading}
					helpContent={dashboardHelp.totalBackups.content}
					helpTitle={dashboardHelp.totalBackups.title}
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
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
					isLoading={dashboardStatsLoading}
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
					{dashboardStatsLoading ? (
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

			{/* Backup Queue Status */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Backup Queue Status
					</h2>
					<Link
						to="/schedules"
						className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
						Manage Schedules
					</Link>
				</div>
				{schedulesLoading ? (
					<DashboardQueueSkeleton />
				) : (
					<div className="grid grid-cols-2 md:grid-cols-5 gap-4">
						<div>
							<div className="flex items-center gap-1.5 mb-1">
								<span className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
								<span className="text-sm font-medium text-gray-600 dark:text-gray-400">
									Running
								</span>
							</div>
							<p className="text-2xl font-bold text-blue-600 dark:text-blue-400">
								{runningBackups.length}
							</p>
						</div>
						<div>
							<div className="flex items-center gap-1.5 mb-1">
								<svg
									className="w-3 h-3 text-red-500"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M5 15l7-7 7 7"
									/>
								</svg>
								<span className="text-sm font-medium text-gray-600 dark:text-gray-400">
									High Priority
								</span>
							</div>
							<p className="text-2xl font-bold text-red-600 dark:text-red-400">
								{priorityQueueSummary.high}
							</p>
						</div>
						<div>
							<div className="flex items-center gap-1.5 mb-1">
								<span className="w-2 h-2 bg-yellow-500 rounded-full" />
								<span className="text-sm font-medium text-gray-600 dark:text-gray-400">
									Medium
								</span>
							</div>
							<p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
								{priorityQueueSummary.medium}
							</p>
						</div>
						<div>
							<div className="flex items-center gap-1.5 mb-1">
								<svg
									className="w-3 h-3 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M19 9l-7 7-7-7"
									/>
								</svg>
								<span className="text-sm font-medium text-gray-600 dark:text-gray-400">
									Low Priority
								</span>
							</div>
							<p className="text-2xl font-bold text-gray-600 dark:text-gray-400">
								{priorityQueueSummary.low}
							</p>
						</div>
						<div>
							<div className="flex items-center gap-1.5 mb-1">
								<svg
									className="w-3 h-3 text-amber-500"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
									/>
								</svg>
								<span className="text-sm font-medium text-gray-600 dark:text-gray-400">
									Preemptible
								</span>
							</div>
							<p className="text-2xl font-bold text-amber-600 dark:text-amber-400">
								{priorityQueueSummary.preemptible}
							</p>
						</div>
					</div>
				)}
				{enabledSchedules > 0 && !schedulesLoading && (
					<div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
						<div className="flex items-center justify-between text-sm">
							<span className="text-gray-600 dark:text-gray-400">
								Total enabled schedules
							</span>
							<span className="font-medium text-gray-900 dark:text-white">
								{enabledSchedules}
							</span>
						</div>
						<div className="mt-2 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
							<div className="h-full flex">
								{priorityQueueSummary.high > 0 && (
									<div
										className="bg-red-500 h-full"
										style={{
											width: `${(priorityQueueSummary.high / enabledSchedules) * 100}%`,
										}}
									/>
								)}
								{priorityQueueSummary.medium > 0 && (
									<div
										className="bg-yellow-500 h-full"
										style={{
											width: `${(priorityQueueSummary.medium / enabledSchedules) * 100}%`,
										}}
									/>
								)}
								{priorityQueueSummary.low > 0 && (
									<div
										className="bg-gray-400 h-full"
										style={{
											width: `${(priorityQueueSummary.low / enabledSchedules) * 100}%`,
										}}
									/>
								)}
							</div>
						</div>
						<div className="mt-2 flex gap-4 text-xs text-gray-500 dark:text-gray-400">
							<span className="flex items-center gap-1">
								<span className="w-2 h-2 bg-red-500 rounded" />
								High
							</span>
							<span className="flex items-center gap-1">
								<span className="w-2 h-2 bg-yellow-500 rounded" />
								Medium
							</span>
							<span className="flex items-center gap-1">
								<span className="w-2 h-2 bg-gray-400 rounded" />
								Low
							</span>
						</div>
					</div>
				)}
			</div>

			{/* Favorites Section */}
			{hasFavorites && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-yellow-400 fill-current"
								viewBox="0 0 24 24"
							>
								<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
							</svg>
							Favorites
						</h2>
					</div>
					{favoritesLoading ? (
						<DashboardFavoritesSkeleton />
					) : (
						<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
							{/* Favorite Agents */}
							{favoriteAgents.length > 0 && (
								<div>
									<h3 className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-3 flex items-center gap-2">
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
												d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
											/>
										</svg>
										Agents ({favoriteAgents.length})
									</h3>
									<ul className="space-y-2">
										{favoriteAgents.slice(0, 5).map((agent) => (
											<li
												key={agent.id}
												className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
											>
												<div className="flex items-center gap-2">
													<StarButton
														entityType="agent"
														entityId={agent.id}
														isFavorite={true}
														size="sm"
													/>
													<Link
														to={`/agents/${agent.id}`}
														className="text-sm font-medium text-gray-900 dark:text-white hover:text-indigo-600"
													>
														{agent.hostname}
													</Link>
												</div>
												<span
													className={`w-2 h-2 rounded-full ${agent.status === 'active' ? 'bg-green-500' : 'bg-gray-400'}`}
												/>
											</li>
										))}
									</ul>
								</div>
							)}
							{/* Favorite Schedules */}
							{favoriteSchedules.length > 0 && (
								<div>
									<h3 className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-3 flex items-center gap-2">
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
												d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
											/>
										</svg>
										Schedules ({favoriteSchedules.length})
									</h3>
									<ul className="space-y-2">
										{favoriteSchedules.slice(0, 5).map((schedule) => (
											<li
												key={schedule.id}
												className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
											>
												<div className="flex items-center gap-2">
													<StarButton
														entityType="schedule"
														entityId={schedule.id}
														isFavorite={true}
														size="sm"
													/>
													<Link
														to="/schedules"
														className="text-sm font-medium text-gray-900 dark:text-white hover:text-indigo-600"
													>
														{schedule.name}
													</Link>
												</div>
												<span
													className={`w-2 h-2 rounded-full ${schedule.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
												/>
											</li>
										))}
									</ul>
								</div>
							)}
							{/* Favorite Repositories */}
							{favoriteRepos.length > 0 && (
								<div>
									<h3 className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-3 flex items-center gap-2">
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
												d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
											/>
										</svg>
										Repositories ({favoriteRepos.length})
									</h3>
									<ul className="space-y-2">
										{favoriteRepos.slice(0, 5).map((repo) => (
											<li
												key={repo.id}
												className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
											>
												<div className="flex items-center gap-2">
													<StarButton
														entityType="repository"
														entityId={repo.id}
														isFavorite={true}
														size="sm"
													/>
													<Link
														to="/repositories"
														className="text-sm font-medium text-gray-900 dark:text-white hover:text-indigo-600"
													>
														{repo.name}
													</Link>
												</div>
												<span className="text-xs text-gray-500 dark:text-gray-400">
													{repo.type}
												</span>
											</li>
										))}
									</ul>
								</div>
							)}
						</div>
					)}
				</div>
			)}

			{/* Backup Duration and Recent Backups Row */}
			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<BackupDurationChart
					data={durationTrend ?? []}
					isLoading={durationTrendLoading}
				/>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="flex items-center gap-1.5 text-lg font-semibold text-gray-900 dark:text-white">
							{t('dashboard.recentBackups')}
							<HelpTooltip
								content={dashboardHelp.recentBackups.content}
								title={dashboardHelp.recentBackups.title}
								size="md"
							/>
						</h2>
						<Link
							to="/backups"
							className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
						>
							View All
						</Link>
					</div>
					{backupsLoading ? (
						<DashboardBackupListSkeleton />
					) : recentBackups.length === 0 ? (
						<div className="text-center py-8 text-gray-500 dark:text-gray-400">
							<svg
								aria-hidden="true"
								className="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600"
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
							<p>{t('dashboard.noBackupsYet')}</p>
							<p className="text-sm">{t('dashboard.configureAgentToStart')}</p>
						</div>
					) : (
						<div className="space-y-1">
							{recentBackups.map((backup) => {
								const statusColor = getBackupStatusColor(backup.status);
								return (
									<div
										key={backup.id}
										className="flex items-center justify-between py-3 border-b border-gray-100 dark:border-gray-700 last:border-0"
									>
										<div>
											<p className="text-sm font-medium text-gray-900 dark:text-white">
												{truncateSnapshotId(backup.snapshot_id) ||
													t('dashboard.running')}
											</p>
											<p className="text-xs text-gray-500 dark:text-gray-400">
												{formatRelativeTime(backup.started_at)}
												{backup.size_bytes !== undefined &&
													` - ${localFormatBytes(backup.size_bytes)}`}
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

			{/* System Status, DR Testing, and Calendar Row */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="flex items-center gap-1.5 text-lg font-semibold text-gray-900 mb-4">
						{t('dashboard.systemStatus')}
						<HelpTooltip
							content={dashboardHelp.systemStatus.content}
							title={dashboardHelp.systemStatus.title}
							size="md"
						/>
					</h2>
					<div className="space-y-4">
						<div className="flex items-center justify-between">
							<span className="text-gray-600">{t('dashboard.server')}</span>
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								{t('dashboard.online')}
							</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-gray-600">{t('dashboard.database')}</span>
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								{t('dashboard.connected')}
							</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-gray-600">{t('dashboard.scheduler')}</span>
							{enabledSchedules > 0 ? (
								<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
									<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
									{t('dashboard.activeJobs', { count: enabledSchedules })}
								</span>
							) : (
								<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
									<span className="w-1.5 h-1.5 bg-gray-400 rounded-full" />
									{t('dashboard.idle')}
								</span>
							)}
						</div>
					</div>

					<div className="border-t border-gray-200 mt-4 pt-4">
						<h3 className="text-sm font-medium text-gray-900 mb-3">
							DR Testing
						</h3>
						<div className="space-y-3">
							<div className="flex items-center justify-between">
								<span className="text-gray-600 text-sm">Active Runbooks</span>
								<span className="text-sm font-medium text-gray-900">
									{drStatusLoading ? (
										<span className="inline-block w-6 h-4 bg-gray-200 rounded animate-pulse" />
									) : (
										`${drStatus?.active_runbooks ?? 0} / ${drStatus?.total_runbooks ?? 0}`
									)}
								</span>
							</div>
							<div className="flex items-center justify-between">
								<span className="text-gray-600 text-sm">Last Test</span>
								<span className="text-sm text-gray-900">
									{drStatusLoading ? (
										<span className="inline-block w-16 h-4 bg-gray-200 rounded animate-pulse" />
									) : drStatus?.last_test_at ? (
										formatRelativeTime(drStatus.last_test_at)
									) : (
										<span className="text-gray-400">Never</span>
									)}
								</span>
							</div>
							<div className="flex items-center justify-between">
								<span className="text-gray-600 text-sm">Next Test</span>
								<span className="text-sm text-gray-900">
									{drStatusLoading ? (
										<span className="inline-block w-16 h-4 bg-gray-200 rounded animate-pulse" />
									) : drStatus?.next_test_at ? (
										formatRelativeTime(drStatus.next_test_at)
									) : (
										<span className="text-gray-400">Not scheduled</span>
									)}
								</span>
							</div>
						</div>
					</div>
				</div>

				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="flex items-center gap-1.5 text-lg font-semibold text-gray-900 mb-4">
						{t('dashboard.storageEfficiency')}
						<HelpTooltip
							content={dashboardHelp.storageEfficiency.content}
							title={dashboardHelp.storageEfficiency.title}
							docsUrl={dashboardHelp.storageEfficiency.docsUrl}
							size="md"
						/>
					</h2>
					{statsLoading ? (
						<div className="grid grid-cols-2 gap-4">
							{skeletonKeys(4, 'efficiency').map((key) => (
								<div key={key} className="animate-pulse">
									<TextSkeleton width="md" size="sm" className="mb-2" />
									<Skeleton className="h-8 w-20 mb-1" />
									<TextSkeleton width="lg" size="sm" />
								</div>
							))}
						</div>
					) : storageStats ? (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-600">
									{t('dashboard.avgDedupRatio')}
								</p>
								<p
									className={`text-2xl font-bold mt-1 ${getDedupRatioColor(storageStats.avg_dedup_ratio)}`}
								>
									{formatDedupRatio(storageStats.avg_dedup_ratio)}
								</p>
								<p className="text-sm text-gray-500 mt-1">
									{t('dashboard.repositoriesCount', {
										count: storageStats.repository_count,
									})}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">
									{t('dashboard.spaceSaved')}
								</p>
								<p
									className={`text-2xl font-bold mt-1 ${getSpaceSavedColor(storageStats.total_restore_size > 0 ? (storageStats.total_space_saved / storageStats.total_restore_size) * 100 : 0)}`}
								>
									{localFormatBytes(storageStats.total_space_saved)}
								</p>
								<p className="text-sm text-gray-500 mt-1">
									{localFormatPercent(
										storageStats.total_restore_size > 0
											? (storageStats.total_space_saved /
													storageStats.total_restore_size) *
													100
											: 0,
									)}{' '}
									{t('dashboard.ofOriginal')}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">
									{t('dashboard.actualStorage')}
								</p>
								<p className="text-2xl font-bold text-gray-900 mt-1">
									{localFormatBytes(storageStats.total_raw_size)}
								</p>
								<p className="text-sm text-gray-500 mt-1">
									{t('dashboard.fromOriginal', {
										size: localFormatBytes(storageStats.total_restore_size),
									})}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-600">
									{t('dashboard.totalSnapshots')}
								</p>
								<p className="text-2xl font-bold text-gray-900 mt-1">
									{storageStats.total_snapshots}
								</p>
								<p className="text-sm text-gray-500 mt-1">
									{t('dashboard.acrossRepositories')}
								</p>
							</div>
						</div>
					) : (
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
									d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
								/>
							</svg>
							<p>{t('dashboard.noStorageStats')}</p>
							<p className="text-sm">{t('dashboard.statsCollectedAuto')}</p>
						</div>
					)}
				</div>

				<div>
					<MiniBackupCalendar />
				</div>
			</div>

			{/* Activity Feed */}
			<ActivityFeedWidget limit={5} enableRealtime={true} />
		</div>
	);
}

export default Dashboard;
