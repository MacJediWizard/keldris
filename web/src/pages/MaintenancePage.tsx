interface MaintenancePageProps {
	expectedEndTime?: string;
	message?: string;
}

export function MaintenancePage({
	expectedEndTime,
	message,
}: MaintenancePageProps) {
	const formatExpectedTime = (dateStr: string) => {
		try {
			const date = new Date(dateStr);
			return date.toLocaleString(undefined, {
				weekday: 'short',
				month: 'short',
				day: 'numeric',
				hour: 'numeric',
				minute: '2-digit',
				timeZoneName: 'short',
			});
		} catch {
			return dateStr;
		}
	};

	return (
		<div className="min-h-screen bg-gray-50 flex items-center justify-center">
			<div className="text-center max-w-lg px-4">
				<div className="mx-auto w-24 h-24 bg-amber-100 rounded-full flex items-center justify-center mb-6">
					<svg
						className="w-12 h-12 text-amber-600"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
						/>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
						/>
					</svg>
				</div>

				<h1 className="text-3xl font-bold text-gray-900 mb-4">
					Scheduled Maintenance
				</h1>
				<p className="text-gray-600 mb-6">
					{message ||
						"We're currently performing scheduled maintenance to improve your experience. The system will be back online shortly."}
				</p>

				{expectedEndTime && (
					<div className="bg-white rounded-lg border border-gray-200 p-4 mb-6">
						<p className="text-sm text-gray-500 mb-1">Expected to be back:</p>
						<p className="text-lg font-semibold text-gray-900">
							{formatExpectedTime(expectedEndTime)}
						</p>
					</div>
				)}

				<div className="bg-indigo-50 rounded-lg p-4 mb-6">
					<div className="flex items-start gap-3">
						<svg
							className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<div className="text-left">
							<p className="text-sm font-medium text-indigo-900">
								Your backups are safe
							</p>
							<p className="text-sm text-indigo-700">
								All scheduled backups will resume automatically once maintenance
								is complete.
							</p>
						</div>
					</div>
				</div>

				<div className="space-y-4">
					<a
						href="https://status.keldris.io"
						target="_blank"
						rel="noopener noreferrer"
						className="inline-flex items-center justify-center w-full px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
					>
						<svg
							className="w-4 h-4 mr-2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
							/>
						</svg>
						View Status Page
					</a>

					<button
						type="button"
						onClick={() => window.location.reload()}
						className="inline-flex items-center justify-center w-full px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
					>
						<svg
							className="w-4 h-4 mr-2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
							/>
						</svg>
						Refresh Page
					</button>
				</div>

				<div className="mt-8 pt-6 border-t border-gray-200">
					<p className="text-sm text-gray-500">
						Questions? Contact us at{' '}
						<a
							href="mailto:support@keldris.io"
							className="text-indigo-600 hover:text-indigo-800"
						>
							support@keldris.io
						</a>
					</p>
				</div>
			</div>
		</div>
	);
}
