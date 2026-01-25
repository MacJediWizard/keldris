import { useRateLimitDashboard } from '../hooks/useRateLimits';
import type { EndpointRateLimitInfo, RateLimitClientStats } from '../lib/types';

function formatDateTime(dateStr: string): string {
	const date = new Date(dateStr);
	return date.toLocaleString();
}

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
			<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
		</div>
	);
}

function StatCard({
	title,
	value,
	subtitle,
	color = 'text-gray-900',
}: {
	title: string;
	value: string | number;
	subtitle?: string;
	color?: string;
}) {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
			<div className="text-sm font-medium text-gray-500 dark:text-gray-400">
				{title}
			</div>
			<div className={`text-2xl font-bold mt-1 ${color} dark:text-white`}>
				{value}
			</div>
			{subtitle && (
				<div className="text-xs text-gray-400 dark:text-gray-500 mt-1">
					{subtitle}
				</div>
			)}
		</div>
	);
}

export function RateLimitDashboard() {
	const { stats, isLoading, error, refresh } = useRateLimitDashboard();

	if (error) {
		return (
			<div className="p-12 text-center text-red-500 dark:text-red-400">
				<p className="font-medium">Failed to load rate limit data</p>
				<p className="text-sm">
					You may not have admin access to view this page
				</p>
			</div>
		);
	}

	const rejectionRate =
		stats && stats.total_requests > 0
			? ((stats.total_rejected / stats.total_requests) * 100).toFixed(2)
			: '0.00';

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Rate Limit Dashboard
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Monitor API rate limiting and client statistics
					</p>
				</div>
				<button
					type="button"
					onClick={() => refresh()}
					className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
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

			{/* Summary Stats */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				{isLoading ? (
					<>
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
					</>
				) : stats ? (
					<>
						<StatCard
							title="Default Limit"
							value={`${stats.default_limit} / ${stats.default_period}`}
						/>
						<StatCard
							title="Total Requests"
							value={stats.total_requests.toLocaleString()}
						/>
						<StatCard
							title="Rejected Requests"
							value={stats.total_rejected.toLocaleString()}
							color="text-red-600"
						/>
						<StatCard
							title="Rejection Rate"
							value={`${rejectionRate}%`}
							color={
								Number.parseFloat(rejectionRate) > 5
									? 'text-red-600'
									: 'text-green-600'
							}
						/>
					</>
				) : null}
			</div>

			{/* Endpoint Configurations */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Endpoint Rate Limits
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Custom rate limits configured for specific endpoints
					</p>
				</div>
				{isLoading ? (
					<div className="p-6">
						<div className="animate-pulse space-y-4">
							<div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded" />
							<div className="h-4 w-3/4 bg-gray-200 dark:bg-gray-700 rounded" />
						</div>
					</div>
				) : stats?.endpoint_configs && stats.endpoint_configs.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Pattern
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Limit
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Period
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{stats.endpoint_configs.map((config: EndpointRateLimitInfo) => (
									<tr
										key={config.pattern}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4 font-mono text-sm text-gray-900 dark:text-white">
											{config.pattern}
										</td>
										<td className="px-6 py-4 text-gray-700 dark:text-gray-300">
											{config.limit.toLocaleString()} requests
										</td>
										<td className="px-6 py-4 text-gray-700 dark:text-gray-300">
											{config.period}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div className="p-6 text-center text-gray-500 dark:text-gray-400">
						<p>No custom endpoint limits configured</p>
						<p className="text-sm mt-1">
							All endpoints use the default rate limit
						</p>
					</div>
				)}
			</div>

			{/* Client Statistics */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Client Statistics
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Rate limit statistics by client IP address
					</p>
				</div>
				{isLoading ? (
					<div className="p-6">
						<div className="animate-pulse space-y-4">
							<div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded" />
							<div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded" />
							<div className="h-4 w-3/4 bg-gray-200 dark:bg-gray-700 rounded" />
						</div>
					</div>
				) : stats?.client_stats && stats.client_stats.length > 0 ? (
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Client IP
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Total Requests
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Rejected
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Rejection Rate
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Last Request
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{stats.client_stats.map((client: RateLimitClientStats) => {
									const clientRejectionRate =
										client.total_requests > 0
											? (
													(client.rejected_count / client.total_requests) *
													100
												).toFixed(2)
											: '0.00';
									return (
										<tr
											key={client.client_ip}
											className="hover:bg-gray-50 dark:hover:bg-gray-700"
										>
											<td className="px-6 py-4 font-mono text-sm text-gray-900 dark:text-white">
												{client.client_ip}
											</td>
											<td className="px-6 py-4 text-gray-700 dark:text-gray-300">
												{client.total_requests.toLocaleString()}
											</td>
											<td className="px-6 py-4">
												<span
													className={
														client.rejected_count > 0
															? 'text-red-600 dark:text-red-400'
															: 'text-gray-700 dark:text-gray-300'
													}
												>
													{client.rejected_count.toLocaleString()}
												</span>
											</td>
											<td className="px-6 py-4">
												<span
													className={
														Number.parseFloat(clientRejectionRate) > 10
															? 'text-red-600 dark:text-red-400 font-medium'
															: Number.parseFloat(clientRejectionRate) > 5
																? 'text-yellow-600 dark:text-yellow-400'
																: 'text-green-600 dark:text-green-400'
													}
												>
													{clientRejectionRate}%
												</span>
											</td>
											<td className="px-6 py-4 text-gray-500 dark:text-gray-400">
												{client.last_request
													? formatDateTime(client.last_request)
													: '-'}
											</td>
										</tr>
									);
								})}
							</tbody>
						</table>
					</div>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<svg
							aria-hidden="true"
							className="w-16 h-16 mx-auto mb-4 text-gray-300 dark:text-gray-600"
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
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No client statistics yet
						</h3>
						<p>Statistics will appear as clients make API requests</p>
					</div>
				)}
			</div>

			{/* Rate Limit Headers Info */}
			<div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 p-6">
				<h3 className="text-lg font-semibold text-blue-900 dark:text-blue-100 mb-2">
					Rate Limit Headers
				</h3>
				<p className="text-blue-700 dark:text-blue-300 mb-4">
					All API responses include rate limit information in the following
					headers:
				</p>
				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
					<div className="bg-white dark:bg-gray-800 rounded p-4 border border-blue-200 dark:border-blue-800">
						<code className="font-mono text-blue-800 dark:text-blue-200">
							X-RateLimit-Limit
						</code>
						<p className="text-gray-600 dark:text-gray-400 mt-1">
							Maximum number of requests allowed in the time window
						</p>
					</div>
					<div className="bg-white dark:bg-gray-800 rounded p-4 border border-blue-200 dark:border-blue-800">
						<code className="font-mono text-blue-800 dark:text-blue-200">
							X-RateLimit-Remaining
						</code>
						<p className="text-gray-600 dark:text-gray-400 mt-1">
							Number of requests remaining in the current window
						</p>
					</div>
					<div className="bg-white dark:bg-gray-800 rounded p-4 border border-blue-200 dark:border-blue-800">
						<code className="font-mono text-blue-800 dark:text-blue-200">
							X-RateLimit-Reset
						</code>
						<p className="text-gray-600 dark:text-gray-400 mt-1">
							Unix timestamp when the rate limit window resets
						</p>
					</div>
					<div className="bg-white dark:bg-gray-800 rounded p-4 border border-blue-200 dark:border-blue-800">
						<code className="font-mono text-blue-800 dark:text-blue-200">
							Retry-After
						</code>
						<p className="text-gray-600 dark:text-gray-400 mt-1">
							Seconds to wait before retrying (only on 429 responses)
						</p>
					</div>
				</div>
			</div>
		</div>
	);
}
