import type { GeoReplicationConfig, Repository } from '../../lib/types';

interface GeoReplicationStatusCardProps {
	configs: GeoReplicationConfig[];
	repositories: Repository[];
	onTriggerReplication?: (configId: string) => void;
	onToggleEnabled?: (configId: string, enabled: boolean) => void;
}

function StatusBadge({ status }: { status: string }) {
	const config = {
		pending: {
			bg: 'bg-gray-100',
			text: 'text-gray-700',
			dot: 'bg-gray-400',
			label: 'Pending',
		},
		syncing: {
			bg: 'bg-blue-100',
			text: 'text-blue-700',
			dot: 'bg-blue-500 animate-pulse',
			label: 'Syncing',
		},
		synced: {
			bg: 'bg-green-100',
			text: 'text-green-700',
			dot: 'bg-green-500',
			label: 'Synced',
		},
		failed: {
			bg: 'bg-red-100',
			text: 'text-red-700',
			dot: 'bg-red-500',
			label: 'Failed',
		},
		disabled: {
			bg: 'bg-gray-100',
			text: 'text-gray-500',
			dot: 'bg-gray-300',
			label: 'Disabled',
		},
	}[status] ?? {
		bg: 'bg-gray-100',
		text: 'text-gray-700',
		dot: 'bg-gray-400',
		label: status,
	};

	return (
		<span
			className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.text}`}
		>
			<span className={`w-1.5 h-1.5 rounded-full ${config.dot}`} />
			{config.label}
		</span>
	);
}

function LagIndicator({
	lag,
	maxSnapshots: _maxSnapshots,
	maxHours: _maxHours,
}: {
	lag?: {
		snapshots_behind: number;
		time_behind_hours: number;
		is_healthy: boolean;
	};
	maxSnapshots: number;
	maxHours: number;
}) {
	// Note: maxSnapshots and maxHours are available for future use in displaying thresholds
	void _maxSnapshots;
	void _maxHours;
	if (!lag) return null;

	const isHealthy = lag.is_healthy;

	return (
		<div
			className={`text-xs px-2 py-1 rounded ${
				isHealthy
					? 'bg-green-50 text-green-700'
					: 'bg-yellow-50 text-yellow-700'
			}`}
		>
			{lag.snapshots_behind > 0 && (
				<span className="mr-2">
					{lag.snapshots_behind} snapshot{lag.snapshots_behind !== 1 ? 's' : ''}{' '}
					behind
				</span>
			)}
			{lag.time_behind_hours > 0 && (
				<span>{lag.time_behind_hours}h since last sync</span>
			)}
			{lag.snapshots_behind === 0 && lag.time_behind_hours === 0 && (
				<span>Up to date</span>
			)}
		</div>
	);
}

function formatRelativeTime(dateString: string | undefined): string {
	if (!dateString) return 'Never';

	const date = new Date(dateString);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return 'Just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	if (diffDays < 7) return `${diffDays}d ago`;

	return date.toLocaleDateString();
}

export function GeoReplicationStatusCard({
	configs,
	repositories,
	onTriggerReplication,
	onToggleEnabled,
}: GeoReplicationStatusCardProps) {
	const getRepoName = (id: string) => {
		const repo = repositories.find((r) => r.id === id);
		return repo?.name ?? 'Unknown';
	};

	if (configs.length === 0) {
		return (
			<div className="bg-white rounded-lg border border-gray-200 p-4">
				<h3 className="text-sm font-medium text-gray-900 mb-2">
					Geo-Replication Status
				</h3>
				<p className="text-sm text-gray-500">
					No geo-replication configured. Set up replication to automatically
					copy backups to secondary regions for disaster recovery.
				</p>
			</div>
		);
	}

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-4">
			<div className="flex items-center justify-between mb-3">
				<h3 className="text-sm font-medium text-gray-900">
					Geo-Replication Status
				</h3>
				<div className="text-xs text-gray-500">
					{configs.filter((c) => c.enabled).length} of {configs.length} active
				</div>
			</div>
			<div className="space-y-3">
				{configs.map((config) => (
					<div
						key={config.id}
						className={`p-3 rounded-lg ${
							config.enabled ? 'bg-gray-50' : 'bg-gray-100 opacity-60'
						}`}
					>
						<div className="flex items-start gap-3">
							{/* Replication icon */}
							<div className="flex-shrink-0 mt-0.5">
								<svg
									aria-hidden="true"
									className={`w-4 h-4 ${
										config.enabled ? 'text-gray-400' : 'text-gray-300'
									}`}
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
									/>
								</svg>
							</div>

							<div className="flex-1 min-w-0">
								{/* Source -> Target */}
								<div className="flex items-center gap-2 text-sm">
									<span className="font-medium text-gray-900 truncate">
										{getRepoName(config.source_repository_id)}
									</span>
									<span className="text-gray-400">→</span>
									<span className="font-medium text-gray-900 truncate">
										{getRepoName(config.target_repository_id)}
									</span>
								</div>

								{/* Regions */}
								<div className="text-xs text-gray-500 mt-0.5">
									{config.source_region.display_name} →{' '}
									{config.target_region.display_name}
								</div>

								{/* Status and metrics */}
								<div className="flex flex-wrap items-center gap-2 mt-2">
									<StatusBadge status={config.status} />
									{config.last_sync_at && (
										<span className="text-xs text-gray-500">
											Last synced: {formatRelativeTime(config.last_sync_at)}
										</span>
									)}
								</div>

								{/* Replication lag */}
								{config.replication_lag && (
									<div className="mt-2">
										<LagIndicator
											lag={config.replication_lag}
											maxSnapshots={config.max_lag_snapshots}
											maxHours={config.max_lag_duration_hours}
										/>
									</div>
								)}

								{/* Error message */}
								{config.last_error && (
									<p className="text-xs text-red-600 mt-2 bg-red-50 px-2 py-1 rounded">
										{config.last_error}
									</p>
								)}
							</div>

							{/* Actions */}
							<div className="flex-shrink-0 flex items-center gap-2">
								{onTriggerReplication &&
									config.enabled &&
									config.status !== 'syncing' && (
										<button
											type="button"
											onClick={() => onTriggerReplication(config.id)}
											className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-200 rounded"
											title="Trigger sync now"
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
										</button>
									)}
								{onToggleEnabled && (
									<button
										type="button"
										onClick={() => onToggleEnabled(config.id, !config.enabled)}
										className={`p-1.5 rounded ${
											config.enabled
												? 'text-green-600 hover:bg-green-50'
												: 'text-gray-400 hover:bg-gray-200'
										}`}
										title={
											config.enabled
												? 'Disable replication'
												: 'Enable replication'
										}
									>
										{config.enabled ? (
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
													d="M5 13l4 4L19 7"
												/>
											</svg>
										) : (
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
													d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
												/>
											</svg>
										)}
									</button>
								)}
							</div>
						</div>
					</div>
				))}
			</div>
		</div>
	);
}
