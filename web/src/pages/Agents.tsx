export function Agents() {
	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Agents</h1>
					<p className="text-gray-600 mt-1">
						Manage backup agents across your infrastructure
					</p>
				</div>
				<button
					type="button"
					className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
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
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Register Agent
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search agents..."
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500">
							<option value="all">All Status</option>
							<option value="online">Online</option>
							<option value="offline">Offline</option>
						</select>
					</div>
				</div>

				<div className="p-12 text-center text-gray-500">
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
							d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 mb-2">
						No agents registered
					</h3>
					<p className="mb-4">
						Install and register an agent to start backing up your systems
					</p>
					<div className="bg-gray-50 rounded-lg p-4 max-w-md mx-auto text-left">
						<p className="text-sm font-medium text-gray-700 mb-2">
							Quick start:
						</p>
						<code className="text-sm text-gray-600 block">
							keldris-agent register --server https://your-server
						</code>
					</div>
				</div>
			</div>
		</div>
	);
}
