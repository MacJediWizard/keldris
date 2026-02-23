import { useState } from 'react';
import { UpgradePrompt } from '../components/features/UpgradePrompt';
import {
	useAuditLogs,
	useExportAuditLogsCsv,
	useExportAuditLogsJson,
} from '../hooks/useAuditLogs';
import { usePlanLimits } from '../hooks/usePlanLimits';
import type { AuditLogFilter } from '../lib/types';
import {
	formatAuditAction,
	formatDateTime,
	formatResourceType,
	getAuditActionColor,
	getAuditResultColor,
} from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

export function AuditLogs() {
	const [filter, setFilter] = useState<AuditLogFilter>({
		limit: 50,
		offset: 0,
	});
	const [searchInput, setSearchInput] = useState('');

	const { data, isLoading, isError } = useAuditLogs(filter);
	const exportCsv = useExportAuditLogsCsv();
	const exportJson = useExportAuditLogsJson();
	const { hasFeature } = usePlanLimits();
	const hasAuditLogs = hasFeature('audit_logs');

	const handleSearch = () => {
		setFilter((prev) => ({
			...prev,
			search: searchInput || undefined,
			offset: 0,
		}));
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter') {
			handleSearch();
		}
	};

	const handleFilterChange = (
		key: keyof AuditLogFilter,
		value: string | undefined,
	) => {
		setFilter((prev) => ({
			...prev,
			[key]: value || undefined,
			offset: 0,
		}));
	};

	const handlePageChange = (newOffset: number) => {
		setFilter((prev) => ({
			...prev,
			offset: newOffset,
		}));
	};

	const handleExportCsv = () => {
		exportCsv.mutate(filter);
	};

	const handleExportJson = () => {
		exportJson.mutate(filter);
	};

	const totalPages = data
		? Math.ceil(data.total_count / (filter.limit || 50))
		: 0;
	const currentPage = data
		? Math.floor((filter.offset || 0) / (filter.limit || 50)) + 1
		: 1;

	if (!hasAuditLogs) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Audit Logs
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Track all user and system actions for compliance
					</p>
				</div>
				<UpgradePrompt
					feature="audit_logs"
					variant="card"
					source="audit-logs-page"
					showBenefits={true}
				/>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Audit Logs
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Track all user and system actions for compliance
					</p>
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={handleExportCsv}
						disabled={exportCsv.isPending}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
							/>
						</svg>
						{exportCsv.isPending ? 'Exporting...' : 'Export CSV'}
					</button>
					<button
						type="button"
						onClick={handleExportJson}
						disabled={exportJson.isPending}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
							/>
						</svg>
						{exportJson.isPending ? 'Exporting...' : 'Export JSON'}
					</button>
				</div>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex flex-wrap items-center gap-4">
						<div className="flex-1 min-w-64">
							<input
								type="text"
								placeholder="Search logs..."
								value={searchInput}
								onChange={(e) => setSearchInput(e.target.value)}
								onKeyDown={handleKeyDown}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<select
							value={filter.action || ''}
							onChange={(e) => handleFilterChange('action', e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="">All Actions</option>
							<option value="login">Login</option>
							<option value="logout">Logout</option>
							<option value="create">Create</option>
							<option value="read">Read</option>
							<option value="update">Update</option>
							<option value="delete">Delete</option>
							<option value="backup">Backup</option>
							<option value="restore">Restore</option>
						</select>
						<select
							value={filter.resource_type || ''}
							onChange={(e) =>
								handleFilterChange('resource_type', e.target.value)
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="">All Resources</option>
							<option value="agent">Agent</option>
							<option value="repository">Repository</option>
							<option value="schedule">Schedule</option>
							<option value="backup">Backup</option>
							<option value="session">Session</option>
						</select>
						<select
							value={filter.result || ''}
							onChange={(e) => handleFilterChange('result', e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="">All Results</option>
							<option value="success">Success</option>
							<option value="failure">Failure</option>
							<option value="denied">Denied</option>
						</select>
						<input
							type="date"
							value={filter.start_date?.split('T')[0] || ''}
							onChange={(e) =>
								handleFilterChange('start_date', e.target.value || undefined)
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<input
							type="date"
							value={filter.end_date?.split('T')[0] || ''}
							onChange={(e) =>
								handleFilterChange('end_date', e.target.value || undefined)
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<button
							type="button"
							onClick={handleSearch}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Search
						</button>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load audit logs</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Timestamp
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Action
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Resource
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									IP Address
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Result
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Details
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
				) : data && data.audit_logs.length > 0 ? (
					<>
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Timestamp
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Action
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Resource
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											IP Address
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Result
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Details
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{data.audit_logs.map((log) => {
										const actionColor = getAuditActionColor(log.action);
										const resultColor = getAuditResultColor(log.result);
										return (
											<tr
												key={log.id}
												className="hover:bg-gray-50 dark:hover:bg-gray-700"
											>
												<td className="px-6 py-4 text-sm text-gray-900 whitespace-nowrap">
													{formatDateTime(log.created_at)}
												</td>
												<td className="px-6 py-4">
													<span
														className={`inline-flex px-2.5 py-0.5 rounded-full text-xs font-medium ${actionColor.bg} ${actionColor.text}`}
													>
														{formatAuditAction(log.action)}
													</span>
												</td>
												<td className="px-6 py-4 text-sm text-gray-900">
													<div>{formatResourceType(log.resource_type)}</div>
													{log.resource_id && (
														<div className="text-xs text-gray-500 dark:text-gray-400 font-mono">
															{log.resource_id.substring(0, 8)}...
														</div>
													)}
												</td>
												<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 font-mono">
													{log.ip_address || '-'}
												</td>
												<td className="px-6 py-4">
													<span
														className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${resultColor.bg} ${resultColor.text}`}
													>
														<span
															className={`w-1.5 h-1.5 ${resultColor.dot} rounded-full`}
														/>
														{log.result}
													</span>
												</td>
												<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-xs truncate">
													{log.details || '-'}
												</td>
											</tr>
										);
									})}
								</tbody>
							</table>
						</div>

						<div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Showing {(filter.offset || 0) + 1} to{' '}
								{Math.min(
									(filter.offset || 0) + (filter.limit || 50),
									data.total_count,
								)}{' '}
								of {data.total_count} results
							</div>
							<div className="flex items-center gap-2">
								<button
									type="button"
									onClick={() =>
										handlePageChange(
											Math.max(0, (filter.offset || 0) - (filter.limit || 50)),
										)
									}
									disabled={currentPage === 1}
									className="px-3 py-1 border border-gray-300 rounded text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									Previous
								</button>
								<span className="text-sm text-gray-700 dark:text-gray-300">
									Page {currentPage} of {totalPages}
								</span>
								<button
									type="button"
									onClick={() =>
										handlePageChange(
											(filter.offset || 0) + (filter.limit || 50),
										)
									}
									disabled={currentPage >= totalPages}
									className="px-3 py-1 border border-gray-300 rounded text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									Next
								</button>
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
							No audit logs found
						</h3>
						<p>
							{filter.search ||
							filter.action ||
							filter.resource_type ||
							filter.result
								? 'Try adjusting your filters'
								: 'Audit logs will appear here as actions are performed'}
						</p>
					</div>
				)}
			</div>
		</div>
	);
}

export default AuditLogs;
