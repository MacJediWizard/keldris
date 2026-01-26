import {
	useBackupQueue,
	useBackupQueueSummary,
	useCancelQueuedBackup,
} from '../../hooks/useBackupQueue';
import { formatDateTime } from '../../lib/utils';

export function BackupQueuePanel() {
	const { data: queue, isLoading } = useBackupQueue();
	const { data: summary } = useBackupQueueSummary();
	const cancelBackup = useCancelQueuedBackup();

	const handleCancel = async (id: string) => {
		if (confirm('Are you sure you want to cancel this queued backup?')) {
			await cancelBackup.mutateAsync(id);
		}
	};

	if (isLoading) {
		return (
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="animate-pulse">
					<div className="h-6 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-4" />
					<div className="space-y-3">
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
					</div>
				</div>
			</div>
		);
	}

	if (!queue || queue.length === 0) {
		return (
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Backup Queue
				</h2>
				<div className="text-center py-8">
					<svg
						aria-hidden="true"
						className="mx-auto h-12 w-12 text-gray-400 dark:text-gray-500"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"
						/>
					</svg>
					<h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
						No queued backups
					</h3>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						All backup slots are available
					</p>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
			<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
				<div className="flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Backup Queue
					</h2>
					<div className="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400">
						<span>
							<span className="font-medium text-gray-900 dark:text-white">
								{summary?.total_running ?? 0}
							</span>{' '}
							running
						</span>
						<span>
							<span className="font-medium text-yellow-600 dark:text-yellow-400">
								{summary?.total_queued ?? 0}
							</span>{' '}
							queued
						</span>
					</div>
				</div>
				{summary && summary.avg_wait_minutes > 0 && (
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Average wait time: {Math.round(summary.avg_wait_minutes)} minutes
					</p>
				)}
			</div>

			<div className="overflow-x-auto">
				<table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
					<thead className="bg-gray-50 dark:bg-gray-900">
						<tr>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Position
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Schedule
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Agent
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Queued At
							</th>
							<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Actions
							</th>
						</tr>
					</thead>
					<tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
						{queue.map((entry) => (
							<tr
								key={entry.id}
								className="hover:bg-gray-50 dark:hover:bg-gray-700"
							>
								<td className="px-6 py-4 whitespace-nowrap">
									<span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 font-medium text-sm">
										{entry.queue_position}
									</span>
								</td>
								<td className="px-6 py-4 whitespace-nowrap">
									<p className="text-sm font-medium text-gray-900 dark:text-white">
										{entry.schedule_name}
									</p>
								</td>
								<td className="px-6 py-4 whitespace-nowrap">
									<p className="text-sm text-gray-600 dark:text-gray-400">
										{entry.agent_hostname}
									</p>
								</td>
								<td className="px-6 py-4 whitespace-nowrap">
									<p className="text-sm text-gray-600 dark:text-gray-400">
										{formatDateTime(entry.queued_at)}
									</p>
								</td>
								<td className="px-6 py-4 whitespace-nowrap text-right">
									<button
										type="button"
										onClick={() => handleCancel(entry.id)}
										disabled={cancelBackup.isPending}
										className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm font-medium disabled:opacity-50"
									>
										Cancel
									</button>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			</div>
		</div>
	);
}
