import { Link, useLocation } from 'react-router-dom';

export function NotFound() {
	const location = useLocation();

	return (
		<div className="min-h-[60vh] flex items-center justify-center">
			<div className="text-center max-w-md px-4">
				<div className="mx-auto w-24 h-24 bg-indigo-100 rounded-full flex items-center justify-center mb-6">
					<svg
						className="w-12 h-12 text-indigo-600"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>

				<h1 className="text-6xl font-bold text-gray-900 mb-2">404</h1>
				<h2 className="text-xl font-semibold text-gray-700 mb-4">
					Page Not Found
				</h2>
				<p className="text-gray-600 mb-6">
					The page you're looking for doesn't exist or has been moved. Check the
					URL or try searching for what you need.
				</p>

				<div className="bg-gray-50 rounded-lg p-4 mb-6 text-left">
					<p className="text-sm text-gray-500 mb-1">Requested URL:</p>
					<code className="text-sm text-gray-700 break-all">
						{location.pathname}
					</code>
				</div>

				<div className="space-y-3">
					<p className="text-sm text-gray-500">Try one of these instead:</p>
					<div className="flex flex-col sm:flex-row gap-3 justify-center">
						<Link
							to="/"
							className="inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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
									d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
								/>
							</svg>
							Go to Dashboard
						</Link>
						<Link
							to="/file-search"
							className="inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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
									d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
								/>
							</svg>
							Search Files
						</Link>
					</div>
				</div>
			</div>
		</div>
	);
}
