import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMe } from '../hooks/useAuth';
import { useLegalHolds, useDeleteLegalHold } from '../hooks/useLegalHolds';
import { formatDateTime } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-8 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

export function LegalHolds() {
	const { data: user } = useMe();
	const { data: holds, isLoading, isError, refetch } = useLegalHolds();
	const deleteHold = useDeleteLegalHold();
	const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	// Non-admins should not see this page
	if (!isAdmin) {
		return (
			<div className="space-y-6">
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
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
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
						Access Denied
					</h3>
					<p className="text-gray-500 dark:text-gray-400">
						You must be an administrator to view legal holds.
					</p>
				</div>
			</div>
		);
	}

	const handleDelete = async (snapshotId: string) => {
		try {
			await deleteHold.mutateAsync(snapshotId);
			setDeleteConfirm(null);
		} catch {
			// Error handling is managed by the mutation
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Legal Holds
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Snapshots protected from deletion for legal discovery
					</p>
				</div>
				<button
					type="button"
					onClick={() => refetch()}
					className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
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
				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load legal holds</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Snapshot
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Reason
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Placed By
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
						</tbody>
					</table>
				) : holds && holds.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Snapshot
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Reason
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Placed By
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
								{holds.map((hold) => (
									<tr
										key={hold.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4">
											<div className="flex items-center gap-2">
												<svg
													aria-hidden="true"
													className="w-5 h-5 text-amber-500"
													fill="none"
													stroke="currentColor"
													viewBox="0 0 24 24"
												>
													<path
														strokeLinecap="round"
														strokeLinejoin="round"
														strokeWidth={2}
														d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
													/>
												</svg>
												<Link
													to={`/restore?snapshot=${hold.snapshot_id}`}
													className="font-mono text-sm text-indigo-600 hover:underline"
												>
													{hold.snapshot_id.length > 12
														? `${hold.snapshot_id.substring(0, 12)}...`
														: hold.snapshot_id}
												</Link>
											</div>
										</td>
										<td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100 max-w-md">
											{hold.reason}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											{hold.placed_by_name || 'Unknown'}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
											{formatDateTime(hold.created_at)}
										</td>
										<td className="px-6 py-4">
											{deleteConfirm === hold.snapshot_id ? (
												<div className="flex items-center gap-2">
													<button
														type="button"
														onClick={() => handleDelete(hold.snapshot_id)}
														disabled={deleteHold.isPending}
														className="px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
													>
														{deleteHold.isPending ? 'Removing...' : 'Confirm'}
													</button>
													<button
														type="button"
														onClick={() => setDeleteConfirm(null)}
														className="px-3 py-1 border border-gray-300 text-gray-700 text-sm rounded hover:bg-gray-50"
													>
														Cancel
													</button>
												</div>
											) : (
												<button
													type="button"
													onClick={() => setDeleteConfirm(hold.snapshot_id)}
													className="px-3 py-1 border border-red-300 text-red-700 text-sm rounded hover:bg-red-50"
												>
													Remove Hold
												</button>
											)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
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
								d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No Legal Holds
						</h3>
						<p className="mb-4">
							No snapshots are currently under legal hold.
						</p>
						<p className="text-sm">
							To place a legal hold on a snapshot, navigate to the{' '}
							<Link to="/restore" className="text-indigo-600 hover:underline">
								Restore page
							</Link>{' '}
							and click the lock icon on any snapshot.
						</p>
					</div>
				)}
			</div>

			{/* Info card */}
			<div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 p-4">
				<div className="flex items-start gap-3">
					<svg
						aria-hidden="true"
						className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<div className="text-sm text-blue-800 dark:text-blue-200">
						<p className="font-medium mb-1">About Legal Holds</p>
						<p>
							Snapshots under legal hold cannot be deleted by retention policies
							or manual deletion. All hold actions are recorded in the audit log
							for compliance tracking.
						</p>
					</div>
				</div>
			</div>
		</div>
	);
}
