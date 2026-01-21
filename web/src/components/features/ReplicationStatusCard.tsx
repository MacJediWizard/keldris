import type {
	ReplicationStatus,
	ReplicationStatusType,
	Repository,
} from '../../lib/types';

interface ReplicationStatusCardProps {
	statuses: ReplicationStatus[];
	repositories: Repository[];
}

function StatusBadge({ status }: { status: ReplicationStatusType }) {
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
	}[status];

	return (
		<span
			className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.text}`}
		>
			<span className={`w-1.5 h-1.5 rounded-full ${config.dot}`} />
			{config.label}
		</span>
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

export function ReplicationStatusCard({
	statuses,
	repositories,
}: ReplicationStatusCardProps) {
	const getRepoName = (id: string) => {
		const repo = repositories.find((r) => r.id === id);
		return repo?.name ?? 'Unknown';
	};

	if (statuses.length === 0) {
		return (
			<div className="bg-white rounded-lg border border-gray-200 p-4">
				<h3 className="text-sm font-medium text-gray-900 mb-2">
					Replication Status
				</h3>
				<p className="text-sm text-gray-500">
					No replication configured. Add secondary repositories to enable
					replication.
				</p>
			</div>
		);
	}

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-4">
			<h3 className="text-sm font-medium text-gray-900 mb-3">
				Replication Status
			</h3>
			<div className="space-y-3">
				{statuses.map((status) => (
					<div
						key={status.id}
						className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg"
					>
						<div className="flex-shrink-0 mt-0.5">
							<svg
								aria-hidden="true"
								className="w-4 h-4 text-gray-400"
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
							<div className="flex items-center gap-2 text-sm">
								<span className="font-medium text-gray-900 truncate">
									{getRepoName(status.source_repository_id)}
								</span>
								<span className="text-gray-400">→</span>
								<span className="font-medium text-gray-900 truncate">
									{getRepoName(status.target_repository_id)}
								</span>
							</div>
							<div className="flex items-center gap-3 mt-1">
								<StatusBadge status={status.status} />
								{status.last_sync_at && (
									<span className="text-xs text-gray-500">
										Last synced: {formatRelativeTime(status.last_sync_at)}
									</span>
								)}
							</div>
							{status.error_message && (
								<p className="text-xs text-red-600 mt-1.5 bg-red-50 px-2 py-1 rounded">
									{status.error_message}
								</p>
							)}
						</div>
					</div>
				))}
			</div>
		</div>
	);
}
