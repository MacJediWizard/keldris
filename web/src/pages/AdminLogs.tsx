import { useState } from 'react';
import {
	useClearServerLogs,
	useExportServerLogsCsv,
	useExportServerLogsJson,
	useServerLogComponents,
	useServerLogs,
} from '../hooks/useServerLogs';
import type { ServerLogFilter, ServerLogLevel } from '../lib/types';

function formatDateTime(dateStr: string): string {
	const date = new Date(dateStr);
	return date.toLocaleString();
}

function getLogLevelColor(level: ServerLogLevel): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (level) {
		case 'debug':
			return {
				bg: 'bg-gray-100 dark:bg-gray-800',
				text: 'text-gray-700 dark:text-gray-300',
				dot: 'bg-gray-400',
			};
		case 'info':
			return {
				bg: 'bg-blue-100 dark:bg-blue-900',
				text: 'text-blue-700 dark:text-blue-300',
				dot: 'bg-blue-500',
			};
		case 'warn':
			return {
				bg: 'bg-yellow-100 dark:bg-yellow-900',
				text: 'text-yellow-700 dark:text-yellow-300',
				dot: 'bg-yellow-500',
			};
		case 'error':
			return {
				bg: 'bg-red-100 dark:bg-red-900',
				text: 'text-red-700 dark:text-red-300',
				dot: 'bg-red-500',
			};
		case 'fatal':
			return {
				bg: 'bg-red-200 dark:bg-red-800',
				text: 'text-red-800 dark:text-red-200',
				dot: 'bg-red-600',
			};
		default:
			return {
				bg: 'bg-gray-100 dark:bg-gray-800',
				text: 'text-gray-700 dark:text-gray-300',
				dot: 'bg-gray-400',
			};
	}
}

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
				<div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

export function AdminLogs() {
	const [filter, setFilter] = useState<ServerLogFilter>({
		limit: 100,
		offset: 0,
	});
	const [searchInput, setSearchInput] = useState('');
	const [autoRefresh, setAutoRefresh] = useState(true);
	const [showClearConfirm, setShowClearConfirm] = useState(false);

	const { data, isLoading, isError, refetch } = useServerLogs(filter);
	const { data: components } = useServerLogComponents();
	const exportCsv = useExportServerLogsCsv();
	const exportJson = useExportServerLogsJson();
	const clearLogs = useClearServerLogs();

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
		key: keyof ServerLogFilter,
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

	const handleClearLogs = () => {
		clearLogs.mutate(undefined, {
			onSuccess: () => {
				setShowClearConfirm(false);
			},
		});
	};

	const totalPages = data
		? Math.ceil(data.total_count / (filter.limit || 100))
		: 0;
	const currentPage = data
		? Math.floor((filter.offset || 0) / (filter.limit || 100)) + 1
		: 1;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Server Logs
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						View server logs for debugging and monitoring
					</p>
				</div>
				<div className="flex items-center gap-2">
					<label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
						<input
							type="checkbox"
							checked={autoRefresh}
							onChange={(e) => setAutoRefresh(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						Auto-refresh
					</label>
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
						{exportCsv.isPending ? 'Exporting...' : 'CSV'}
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
						{exportJson.isPending ? 'Exporting...' : 'JSON'}
					</button>
					<button
						type="button"
						onClick={() => setShowClearConfirm(true)}
						className="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
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
								d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
							/>
						</svg>
						Clear
					</button>
				</div>
			</div>

			{/* Clear Confirmation Modal */}
			{showClearConfirm && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Clear All Logs?
						</h3>
						<p className="text-gray-600 dark:text-gray-400 mb-6">
							This will permanently delete all server logs from the buffer. This
							action cannot be undone.
						</p>
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setShowClearConfirm(false)}
								className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleClearLogs}
								disabled={clearLogs.isPending}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{clearLogs.isPending ? 'Clearing...' : 'Clear Logs'}
							</button>
						</div>
					</div>
				</div>
			)}

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
							value={filter.level || ''}
							onChange={(e) =>
								handleFilterChange(
									'level',
									e.target.value as ServerLogLevel | undefined,
								)
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="">All Levels</option>
							<option value="debug">Debug</option>
							<option value="info">Info</option>
							<option value="warn">Warning</option>
							<option value="error">Error</option>
							<option value="fatal">Fatal</option>
						</select>
						<select
							value={filter.component || ''}
							onChange={(e) => handleFilterChange('component', e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="">All Components</option>
							{components?.map((comp) => (
								<option key={comp} value={comp}>
									{comp}
								</option>
							))}
						</select>
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
						<p className="font-medium">Failed to load server logs</p>
						<p className="text-sm">
							You may not have admin access to view this page
						</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Timestamp
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Level
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Component
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Message
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
				) : data && data.logs.length > 0 ? (
					<>
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Timestamp
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Level
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Component
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Message
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700 font-mono text-sm">
									{data.logs.map((log, index) => {
										const levelColor = getLogLevelColor(log.level);
										return (
											<tr
												key={`${log.timestamp}-${index}`}
												className="hover:bg-gray-50 dark:hover:bg-gray-700"
											>
												<td className="px-6 py-3 text-gray-500 dark:text-gray-400 whitespace-nowrap">
													{formatDateTime(log.timestamp)}
												</td>
												<td className="px-6 py-3">
													<span
														className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${levelColor.bg} ${levelColor.text}`}
													>
														<span
															className={`w-1.5 h-1.5 ${levelColor.dot} rounded-full`}
														/>
														{log.level.toUpperCase()}
													</span>
												</td>
												<td className="px-6 py-3 text-gray-600 dark:text-gray-400">
													{log.component || '-'}
												</td>
												<td className="px-6 py-3 text-gray-900 dark:text-white">
													<div className="max-w-xl">
														<span className="break-words">{log.message}</span>
														{log.fields &&
															Object.keys(log.fields).length > 0 && (
																<details className="mt-1">
																	<summary className="text-xs text-indigo-600 dark:text-indigo-400 cursor-pointer">
																		Show fields
																	</summary>
																	<pre className="mt-2 text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-auto">
																		{JSON.stringify(log.fields, null, 2)}
																	</pre>
																</details>
															)}
													</div>
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
									(filter.offset || 0) + (filter.limit || 100),
									data.total_count,
								)}{' '}
								of {data.total_count} entries
							</div>
							<div className="flex items-center gap-2">
								<button
									type="button"
									onClick={() =>
										handlePageChange(
											Math.max(0, (filter.offset || 0) - (filter.limit || 100)),
										)
									}
									disabled={currentPage === 1}
									className="px-3 py-1 border border-gray-300 rounded text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									Previous
								</button>
								<span className="text-sm text-gray-700 dark:text-gray-300">
									Page {currentPage} of {totalPages || 1}
								</span>
								<button
									type="button"
									onClick={() =>
										handlePageChange(
											(filter.offset || 0) + (filter.limit || 100),
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
							No server logs found
						</h3>
						<p>
							{filter.search || filter.level || filter.component
								? 'Try adjusting your filters'
								: 'Server logs will appear here as the server runs'}
						</p>
					</div>
				)}
			</div>
		</div>
	);
}
