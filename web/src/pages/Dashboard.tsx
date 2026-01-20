import { useAgents } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import { useDRStatus } from '../hooks/useDRRunbooks';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import {
	formatBytes,
	formatDate,
	formatRelativeTime,
	getBackupStatusColor,
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
	const { data: drStatus, isLoading: drStatusLoading } = useDRStatus();

	const activeAgents = agents?.filter((a) => a.status === 'active').length ?? 0;
	const enabledSchedules = schedules?.filter((s) => s.enabled).length ?? 0;
	const recentBackups = backups?.slice(0, 5) ?? [];

	const isLoading =
		agentsLoading || reposLoading || schedulesLoading || backupsLoading;

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
				<p className="text-gray-600 mt-1">Overview of your backup system</p>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-6">
				<StatCard
					title="Active Agents"
					value={String(activeAgents)}
					subtitle="Connected agents"
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
					title="Repositories"
					value={String(repositories?.length ?? 0)}
					subtitle="Backup destinations"
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
					title="Scheduled Jobs"
					value={String(enabledSchedules)}
					subtitle="Active schedules"
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
					title="Total Backups"
					value={String(backups?.length ?? 0)}
					subtitle="All time"
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
				<StatCard
					title="DR Pass Rate"
					value={drStatus ? `${Math.round(drStatus.pass_rate)}%` : '0%'}
					subtitle={`${drStatus?.tests_last_30_days ?? 0} tests (30 days)`}
					isLoading={drStatusLoading}
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
								d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
							/>
						</svg>
					}
				/>
			</div>

			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						Recent Backups
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

				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						System Status
					</h2>
					<div className="space-y-4">
						<div className="flex items-center justify-between">
							<span className="text-gray-600">Server</span>
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								Online
							</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-gray-600">Database</span>
							<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								Connected
							</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-gray-600">Scheduler</span>
							{enabledSchedules > 0 ? (
								<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
									<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
									Active ({enabledSchedules} jobs)
								</span>
							) : (
								<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
									<span className="w-1.5 h-1.5 bg-gray-400 rounded-full" />
									Idle
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
			</div>
		</div>
	);
}
