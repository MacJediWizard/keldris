import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
	useRepositoryGrowth,
	useRepositoryHistory,
	useRepositoryStats,
} from '../hooks/useStorageStats';
import {
	formatBytes,
	formatChartDate,
	formatDateTime,
	formatDedupRatio,
	formatPercent,
	getDedupRatioColor,
	getSpaceSavedColor,
} from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
			<div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-1" />
		<div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 rounded mb-2" />
			<div className="h-8 w-32 bg-gray-200 rounded mb-1" />
			<div className="h-3 w-20 bg-gray-100 rounded" />
		</div>
	);
}

export function RepositoryStatsDetail() {
	const { id } = useParams<{ id: string }>();
	const [growthDays, setGrowthDays] = useState(30);
	const [historyLimit, setHistoryLimit] = useState(30);

	const { data: statsResponse, isLoading: statsLoading } = useRepositoryStats(
		id ?? '',
	);
	const { data: growthResponse, isLoading: growthLoading } =
		useRepositoryGrowth(id ?? '', growthDays);
	const { data: historyResponse, isLoading: historyLoading } =
		useRepositoryHistory(id ?? '', historyLimit);

	const stats = statsResponse?.stats;
	const repositoryName = statsResponse?.repository_name;
	const growth = growthResponse?.growth ?? [];
	const history = historyResponse?.history ?? [];

	return (
		<div className="space-y-6">
			<div className="flex items-center gap-4">
				<Link
					to="/stats"
					className="text-gray-500 hover:text-gray-700 transition-colors"
				>
					<svg
						aria-hidden="true"
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M10 19l-7-7m0 0l7-7m-7 7h18"
						/>
					</svg>
				</Link>
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						{repositoryName ?? 'Repository Statistics'}
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
					<h1 className="text-2xl font-bold text-gray-900">
						{repositoryName ?? 'Repository Statistics'}
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Detailed storage efficiency metrics
					</p>
				</div>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				{statsLoading ? (
					<>
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
					</>
				) : stats ? (
					<>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">Dedup Ratio</p>
							<p
								className={`text-3xl font-bold mt-1 ${getDedupRatioColor(stats.dedup_ratio)}`}
							>
								{formatDedupRatio(stats.dedup_ratio)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Compression factor
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm text-gray-500 mt-1">Compression factor</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600">Space Saved</p>
							<p
								className={`text-3xl font-bold mt-1 ${getSpaceSavedColor(stats.space_saved_pct)}`}
							>
								{formatBytes(stats.space_saved)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{formatPercent(stats.space_saved_pct)} of original
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm text-gray-500 mt-1">
								{formatPercent(stats.space_saved_pct)} of original
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600">
								Actual Storage
							</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{formatBytes(stats.raw_data_size)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								On disk
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm text-gray-500 mt-1">On disk</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600">Original Size</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{formatBytes(stats.restore_size)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							<p className="text-sm text-gray-500 mt-1">
								{stats.snapshot_count} snapshots
							</p>
						</div>
					</>
				) : (
					<div className="col-span-4 text-center py-8 text-gray-500 dark:text-gray-400">
					<div className="col-span-4 text-center py-8 text-gray-500">
						No statistics available for this repository
					</div>
				)}
			</div>

			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Storage Growth
						</h2>
						<select
							value={growthDays}
							onChange={(e) => setGrowthDays(Number(e.target.value))}
							className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
							className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
						>
							<option value={7}>Last 7 days</option>
							<option value={30}>Last 30 days</option>
							<option value={90}>Last 90 days</option>
							<option value={365}>Last year</option>
						</select>
					</div>
					{growthLoading ? (
						<div className="h-48 bg-gray-100 rounded animate-pulse" />
					) : growth.length > 0 ? (
						<div className="h-48">
							<div className="flex items-end justify-between h-full gap-1">
								{growth.map((point) => {
									const maxSize = Math.max(
										...growth.map((p) => p.restore_size),
									);
									const rawHeight =
										maxSize > 0 ? (point.raw_data_size / maxSize) * 100 : 0;
									const restoreHeight =
										maxSize > 0 ? (point.restore_size / maxSize) * 100 : 0;
									return (
										<div
											key={point.date}
											className="flex-1 flex flex-col items-center justify-end h-full"
										>
											<div className="w-full flex flex-col items-center justify-end relative h-36">
												<div
													className="w-full bg-gray-200 dark:bg-gray-700 rounded-t absolute bottom-0"
													className="w-full bg-gray-200 rounded-t absolute bottom-0"
													style={{ height: `${restoreHeight}%` }}
													title={`Original: ${formatBytes(point.restore_size)}`}
												/>
												<div
													className="w-full bg-indigo-500 rounded-t absolute bottom-0"
													style={{ height: `${rawHeight}%` }}
													title={`Stored: ${formatBytes(point.raw_data_size)}`}
												/>
											</div>
											<span className="text-xs text-gray-500 dark:text-gray-400 mt-2 whitespace-nowrap">
											<span className="text-xs text-gray-500 mt-2 whitespace-nowrap">
												{formatChartDate(point.date)}
											</span>
										</div>
									);
								})}
							</div>
							<div className="flex items-center justify-center gap-4 mt-3">
								<div className="flex items-center gap-1">
									<div className="w-2 h-2 bg-indigo-500 rounded" />
									<span className="text-xs text-gray-600">Stored</span>
								</div>
								<div className="flex items-center gap-1">
									<div className="w-2 h-2 bg-gray-200 dark:bg-gray-700 rounded" />
									<div className="w-2 h-2 bg-gray-200 rounded" />
									<span className="text-xs text-gray-600">Original</span>
								</div>
							</div>
						</div>
					) : (
						<div className="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
						<div className="h-48 flex items-center justify-center text-gray-500">
							<p className="text-sm">No growth data available</p>
						</div>
					)}
				</div>

				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Dedup Efficiency
						</h2>
					</div>
					{stats ? (
						<div className="space-y-4">
							<div>
								<div className="flex items-center justify-between mb-1">
									<span className="text-sm text-gray-600 dark:text-gray-400">
										Storage used
									</span>
									<span className="text-sm font-medium text-gray-900 dark:text-white">
									<span className="text-sm text-gray-600">Storage used</span>
									<span className="text-sm font-medium text-gray-900">
										{formatPercent(
											stats.restore_size > 0
												? (stats.raw_data_size / stats.restore_size) * 100
												: 0,
										)}
									</span>
								</div>
								<div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
								<div className="w-full bg-gray-200 rounded-full h-3">
									<div
										className="bg-indigo-500 h-3 rounded-full transition-all"
										style={{
											width: `${stats.restore_size > 0 ? Math.min((stats.raw_data_size / stats.restore_size) * 100, 100) : 0}%`,
										}}
									/>
								</div>
								<div className="flex items-center justify-between mt-1">
									<span className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
										{formatBytes(stats.raw_data_size)} stored
									</span>
									<span className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									<span className="text-xs text-gray-500">
										{formatBytes(stats.raw_data_size)} stored
									</span>
									<span className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
										{formatBytes(stats.restore_size)} original
									</span>
								</div>
							</div>
							<div className="grid grid-cols-2 gap-4 pt-4 border-t border-gray-100">
								<div>
									<p className="text-sm text-gray-600 dark:text-gray-400">
										Total Files
									</p>
									<p className="text-sm text-gray-600">Total Files</p>
									<p className="text-xl font-semibold text-gray-900">
										{stats.total_file_count.toLocaleString()}
									</p>
								</div>
								<div>
									<p className="text-sm text-gray-600 dark:text-gray-400">
										Snapshots
									</p>
									<p className="text-sm text-gray-600">Snapshots</p>
									<p className="text-xl font-semibold text-gray-900">
										{stats.snapshot_count}
									</p>
								</div>
							</div>
						</div>
					) : (
						<div className="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
						<div className="h-48 flex items-center justify-center text-gray-500">
							<p className="text-sm">No data available</p>
						</div>
					)}
				</div>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
			<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900">
						Collection History
					</h2>
					<select
						value={historyLimit}
						onChange={(e) => setHistoryLimit(Number(e.target.value))}
						className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
						className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
					>
						<option value={10}>Last 10</option>
						<option value={30}>Last 30</option>
						<option value={90}>Last 90</option>
						<option value={365}>Last year</option>
					</select>
				</div>
				{historyLoading ? (
					<div className="p-6">
						<div className="animate-pulse space-y-4">
							{[1, 2, 3].map((i) => (
								<div key={i} className="h-12 bg-gray-100 rounded" />
							))}
						</div>
					</div>
				) : history.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Collected At
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Dedup Ratio
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Space Saved
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actual Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Original Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Collected At
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Dedup Ratio
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Space Saved
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actual Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Original Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Snapshots
									</th>
								</tr>
							</thead>
							<tbody className="bg-white divide-y divide-gray-200 dark:divide-gray-700">
								{history.map((record) => (
									<tr
										key={record.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
							<tbody className="bg-white divide-y divide-gray-200">
								{history.map((record) => (
									<tr
										key={record.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
											{formatDateTime(record.collected_at)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<span
												className={`text-sm font-medium ${getDedupRatioColor(record.dedup_ratio)}`}
											>
												{formatDedupRatio(record.dedup_ratio)}
											</span>
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<div>
												<span className="text-sm font-medium text-gray-900 dark:text-white">
												<span className="text-sm font-medium text-gray-900">
													{formatBytes(record.space_saved)}
												</span>
												<span
													className={`ml-2 text-xs ${getSpaceSavedColor(record.space_saved_pct)}`}
												>
													({formatPercent(record.space_saved_pct)})
												</span>
											</div>
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
											{formatBytes(record.raw_data_size)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{formatBytes(record.restore_size)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
											{record.snapshot_count}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
					<div className="p-12 text-center text-gray-500">
						<p>No collection history available</p>
					</div>
				)}
			</div>
		</div>
	);
}

export default RepositoryStatsDetail;
