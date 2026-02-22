import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
	useRepositoryStatsList,
	useStorageGrowth,
	useStorageStatsSummary,
} from '../hooks/useStorageStats';
import {
	formatBytes,
	formatChartDate,
	formatDate,
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

function LoadingTable() {
	return (
		<div className="animate-pulse">
			{[1, 2, 3].map((i) => (
				<div
					key={i}
					className="flex items-center justify-between py-4 border-b border-gray-100"
				>
					<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="h-4 w-32 bg-gray-200 rounded" />
					<div className="h-4 w-20 bg-gray-200 rounded" />
					<div className="h-4 w-16 bg-gray-200 rounded" />
					<div className="h-4 w-24 bg-gray-200 rounded" />
				</div>
			))}
		</div>
	);
}

export function StorageStats() {
	const [growthDays, setGrowthDays] = useState(30);
	const { data: summary, isLoading: summaryLoading } = useStorageStatsSummary();
	const { data: repoStats, isLoading: repoStatsLoading } =
		useRepositoryStatsList();
	const { data: growth, isLoading: growthLoading } =
		useStorageGrowth(growthDays);

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					Storage Statistics
				</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
				<h1 className="text-2xl font-bold text-gray-900">Storage Statistics</h1>
				<p className="text-gray-600 mt-1">
					Monitor storage efficiency and deduplication across your repositories
				</p>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				{summaryLoading ? (
					<>
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
					</>
				) : summary ? (
					<>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">
								Average Dedup Ratio
							</p>
							<p
								className={`text-3xl font-bold mt-1 ${getDedupRatioColor(summary.avg_dedup_ratio)}`}
							>
								{formatDedupRatio(summary.avg_dedup_ratio)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Across {summary.repository_count} repositories
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm text-gray-500 mt-1">
								Across {summary.repository_count} repositories
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">
								Total Space Saved
							</p>
							<p
								className={`text-3xl font-bold mt-1 ${getSpaceSavedColor(summary.total_restore_size > 0 ? (summary.total_space_saved / summary.total_restore_size) * 100 : 0)}`}
							>
								{formatBytes(summary.total_space_saved)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							<p className="text-sm text-gray-500 mt-1">
								{formatPercent(
									summary.total_restore_size > 0
										? (summary.total_space_saved / summary.total_restore_size) *
												100
										: 0,
								)}{' '}
								savings
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">
								Actual Storage Used
							</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{formatBytes(summary.total_raw_size)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								From {formatBytes(summary.total_restore_size)} original
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm text-gray-500 mt-1">
								From {formatBytes(summary.total_restore_size)} original
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">
								Total Snapshots
							</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{summary.total_snapshots}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							<p className="text-sm text-gray-500 mt-1">
								{summary.repository_count} repositories
							</p>
						</div>
					</>
				) : (
					<div className="col-span-4 text-center py-8 text-gray-500 dark:text-gray-400">
					<div className="col-span-4 text-center py-8 text-gray-500">
						No storage statistics available yet
					</div>
				)}
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900">
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
					<div className="h-64 bg-gray-100 rounded animate-pulse" />
				) : growth && growth.length > 0 ? (
					<div className="h-64">
						<div className="flex items-end justify-between h-full gap-1">
							{growth.map((point) => {
								const maxSize = Math.max(...growth.map((p) => p.restore_size));
								const rawHeight =
									maxSize > 0 ? (point.raw_data_size / maxSize) * 100 : 0;
								const restoreHeight =
									maxSize > 0 ? (point.restore_size / maxSize) * 100 : 0;
								return (
									<div
										key={point.date}
										className="flex-1 flex flex-col items-center justify-end h-full"
									>
										<div className="w-full flex flex-col items-center justify-end relative h-52">
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
						<div className="flex items-center justify-center gap-6 mt-4">
							<div className="flex items-center gap-2">
								<div className="w-3 h-3 bg-indigo-500 rounded" />
								<span className="text-sm text-gray-600 dark:text-gray-400">
									Actual storage
								</span>
							</div>
							<div className="flex items-center gap-2">
								<div className="w-3 h-3 bg-gray-200 dark:bg-gray-700 rounded" />
								<span className="text-sm text-gray-600 dark:text-gray-400">
									Original data
								</span>
								<span className="text-sm text-gray-600">Actual storage</span>
							</div>
							<div className="flex items-center gap-2">
								<div className="w-3 h-3 bg-gray-200 rounded" />
								<span className="text-sm text-gray-600">Original data</span>
							</div>
						</div>
					</div>
				) : (
					<div className="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
					<div className="h-64 flex items-center justify-center text-gray-500">
						<div className="text-center">
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
							<p>No growth data available</p>
							<p className="text-sm">
								Data will appear after stats are collected
							</p>
						</div>
					</div>
				)}
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
			<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div className="px-6 py-4 border-b border-gray-200">
					<h2 className="text-lg font-semibold text-gray-900">
						Repository Statistics
					</h2>
				</div>
				{repoStatsLoading ? (
					<div className="p-6">
						<LoadingTable />
					</div>
				) : repoStats && repoStats.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Repository
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
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Repository
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Dedup Ratio
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Space Saved
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actual Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Original Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Snapshots
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Last Updated
									</th>
								</tr>
							</thead>
							<tbody className="bg-white divide-y divide-gray-200 dark:divide-gray-700">
								{repoStats.map((stats) => (
									<tr
										key={stats.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4 whitespace-nowrap">
											<Link
												to={`/stats/${stats.repository_id}`}
												className="text-sm font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
							<tbody className="bg-white divide-y divide-gray-200">
								{repoStats.map((stats) => (
									<tr key={stats.id} className="hover:bg-gray-50">
										<td className="px-6 py-4 whitespace-nowrap">
											<Link
												to={`/stats/${stats.repository_id}`}
												className="text-sm font-medium text-indigo-600 hover:text-indigo-800"
											>
												{stats.repository_name}
											</Link>
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<span
												className={`text-sm font-medium ${getDedupRatioColor(stats.dedup_ratio)}`}
											>
												{formatDedupRatio(stats.dedup_ratio)}
											</span>
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<div>
												<span className="text-sm font-medium text-gray-900 dark:text-white">
												<span className="text-sm font-medium text-gray-900">
													{formatBytes(stats.space_saved)}
												</span>
												<span
													className={`ml-2 text-xs ${getSpaceSavedColor(stats.space_saved_pct)}`}
												>
													({formatPercent(stats.space_saved_pct)})
												</span>
											</div>
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
											{formatBytes(stats.raw_data_size)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{formatBytes(stats.restore_size)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
											{stats.snapshot_count}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{formatDate(stats.collected_at)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
					<div className="p-12 text-center text-gray-500">
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
								d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
							/>
						</svg>
						<p>No repository statistics yet</p>
						<p className="text-sm">
							Statistics will be collected automatically once backups run
						</p>
					</div>
				)}
			</div>
		</div>
	);
}

export default StorageStats;
