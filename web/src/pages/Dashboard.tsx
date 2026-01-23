import { Link } from 'react-router-dom';
import { MiniBackupCalendar } from '../components/features/BackupCalendar';
import { useAgents } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import { useLocale } from '../hooks/useLocale';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import { useStorageStatsSummary } from '../hooks/useStorageStats';
import {
	formatDedupRatio,
	getBackupStatusColor,
	getDedupRatioColor,
	getSpaceSavedColor,
	truncateSnapshotId,
} from '../lib/utils';

interface StatCardProps {
	title: string;
	value: string;
	subtitle: string;
	icon: React.ReactNode;
	isLoading?: boolean;
}

function StatCard({ title, value, subtitle, icon, isLoading }: StatCardProps) {
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
					<p className="text-sm text-gray-500 mt-1">{subtitle}</p>
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

export function Dashboard() {
	const { data: agents, isLoading: agentsLoading } = useAgents();
	const { data: repositories, isLoading: reposLoading } = useRepositories();
	const { data: schedules, isLoading: schedulesLoading } = useSchedules();
	const { data: backups, isLoading: backupsLoading } = useBackups();
	const { data: storageStats, isLoading: statsLoading } =
		useStorageStatsSummary();
	const { t, formatRelativeTime, formatBytes, formatPercent } = useLocale();

	const activeAgents = agents?.filter((a) => a.status === 'active').length ?? 0;
	const enabledSchedules = schedules?.filter((s) => s.enabled).length ?? 0;
	const recentBackups = backups?.slice(0, 5) ?? [];

	const isLoading =
		agentsLoading || reposLoading || schedulesLoading || backupsLoading;

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">
					{t('dashboard.title')}
				</h1>
				<p className="text-gray-600 mt-1">{t('dashboard.subtitle')}</p>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				<StatCard
					title={t('dashboard.activeAgents')}
					value={String(activeAgents)}
					subtitle={t('dashboard.connectedAgents')}
					isLoading={agentsLoading}
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
					value={String(repositories?.length ?? 0)}
					subtitle={t('dashboard.backupDestinations')}
					isLoading={reposLoading}
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
					value={String(enabledSchedules)}
					subtitle={t('dashboard.activeSchedules')}
					isLoading={schedulesLoading}
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
					title={t('dashboard.totalBackups')}
					value={String(backups?.length ?? 0)}
					subtitle={t('dashboard.allTime')}
					isLoading={backupsLoading}
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

			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						{t('dashboard.recentBackups')}
					</h2>
					{isLoading ? (
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
										className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0"
									>
										<div>
											<p className="text-sm font-medium text-gray-900">
												{truncateSnapshotId(backup.snapshot_id) ||
													t('dashboard.running')}
											</p>
											<p className="text-xs text-gray-500">
												{formatRelativeTime(backup.started_at)}
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

				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						{t('dashboard.systemStatus')}
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
				</div>

				<div>
					<MiniBackupCalendar />
				</div>
			</div>

			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900">
						{t('dashboard.storageEfficiency')}
					</h2>
					<Link
						to="/stats"
						className="text-sm text-indigo-600 hover:text-indigo-800"
					>
						{t('common.viewDetails')}
					</Link>
				</div>
				{statsLoading ? (
					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
						{[1, 2, 3, 4].map((i) => (
							<div key={i} className="animate-pulse">
								<div className="h-4 w-24 bg-gray-200 rounded mb-2" />
								<div className="h-8 w-20 bg-gray-200 rounded" />
							</div>
						))}
					</div>
				) : storageStats ? (
					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
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
								{formatBytes(storageStats.total_space_saved)}
							</p>
							<p className="text-sm text-gray-500 mt-1">
								{formatPercent(
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
								{formatBytes(storageStats.total_raw_size)}
							</p>
							<p className="text-sm text-gray-500 mt-1">
								{t('dashboard.fromOriginal', {
									size: formatBytes(storageStats.total_restore_size),
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
		</div>
	);
}
