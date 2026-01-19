export function Schedules() {
	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Schedules</h1>
					<p className="text-gray-600 mt-1">Configure automated backup jobs</p>
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
					Create Schedule
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search schedules..."
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500">
							<option value="all">All Status</option>
							<option value="active">Active</option>
							<option value="paused">Paused</option>
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
							d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 mb-2">
						No schedules configured
					</h3>
					<p className="mb-4">Create a schedule to automate your backups</p>
					<div className="bg-gray-50 rounded-lg p-4 max-w-md mx-auto text-left space-y-2">
						<p className="text-sm font-medium text-gray-700">
							Common schedules:
						</p>
						<div className="text-sm text-gray-600 space-y-1">
							<p>
								<span className="font-mono bg-gray-200 px-1 rounded">
									0 2 * * *
								</span>{' '}
								— Daily at 2 AM
							</p>
							<p>
								<span className="font-mono bg-gray-200 px-1 rounded">
									0 */6 * * *
								</span>{' '}
								— Every 6 hours
							</p>
							<p>
								<span className="font-mono bg-gray-200 px-1 rounded">
									0 3 * * 0
								</span>{' '}
								— Weekly on Sunday
							</p>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
