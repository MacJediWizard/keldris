export function Repositories() {
	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Repositories</h1>
					<p className="text-gray-600 mt-1">
						Configure backup storage destinations
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
					Add Repository
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search repositories..."
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500">
							<option value="all">All Types</option>
							<option value="local">Local</option>
							<option value="s3">Amazon S3</option>
							<option value="b2">Backblaze B2</option>
							<option value="sftp">SFTP</option>
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
							d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 mb-2">
						No repositories configured
					</h3>
					<p className="mb-4">Add a repository to store your backup data</p>
					<div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-2xl mx-auto">
						<div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors">
							<p className="font-medium text-gray-900">Local</p>
							<p className="text-xs text-gray-500">Filesystem path</p>
						</div>
						<div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors">
							<p className="font-medium text-gray-900">S3</p>
							<p className="text-xs text-gray-500">AWS / MinIO</p>
						</div>
						<div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors">
							<p className="font-medium text-gray-900">B2</p>
							<p className="text-xs text-gray-500">Backblaze</p>
						</div>
						<div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors">
							<p className="font-medium text-gray-900">SFTP</p>
							<p className="text-xs text-gray-500">Remote server</p>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
