import { Link } from 'react-router-dom';

interface ServerErrorProps {
	error?: Error;
	resetError?: () => void;
}

export function ServerError({ error, resetError }: ServerErrorProps) {
	const handleRetry = () => {
		if (resetError) {
			resetError();
		} else {
			window.location.reload();
		}
	};

	return (
		<div className="min-h-[60vh] flex items-center justify-center">
			<div className="text-center max-w-md px-4">
				<div className="mx-auto w-24 h-24 bg-red-100 rounded-full flex items-center justify-center mb-6">
					<svg
						className="w-12 h-12 text-red-600"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				</div>

				<h1 className="text-6xl font-bold text-gray-900 mb-2">500</h1>
				<h2 className="text-xl font-semibold text-gray-700 mb-4">
					Something Went Wrong
				</h2>
				<p className="text-gray-600 mb-6">
					We encountered an unexpected error. Our team has been notified and is
					working to resolve the issue.
				</p>

				{error && (
					<div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 text-left">
						<p className="text-sm font-medium text-red-800 mb-1">
							Error Details:
						</p>
						<code className="text-sm text-red-700 break-all">
							{error.message || 'An unexpected error occurred'}
						</code>
					</div>
				)}

				<div className="space-y-4">
					<div className="flex flex-col sm:flex-row gap-3 justify-center">
						<button
							type="button"
							onClick={handleRetry}
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
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
								/>
							</svg>
							Try Again
						</button>
						<Link
							to="/"
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
									d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
								/>
							</svg>
							Go to Dashboard
						</Link>
					</div>

					<div className="pt-4 border-t border-gray-200">
						<p className="text-sm text-gray-500 mb-2">Need help?</p>
						<a
							href="mailto:support@keldris.io"
							className="inline-flex items-center text-sm text-indigo-600 hover:text-indigo-800"
						>
							<svg
								className="w-4 h-4 mr-1"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
								/>
							</svg>
							Contact Support
						</a>
					</div>
				</div>
			</div>
		</div>
	);
}
