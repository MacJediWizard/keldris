import { Link, useSearchParams } from 'react-router-dom';

import { DiffViewer } from '../components/features/DiffViewer';
import { useAgents } from '../hooks/useAgents';
import { useFileDiff, useSnapshot } from '../hooks/useSnapshots';
import { formatDateTime } from '../lib/utils';

export function FileDiff() {
	const [searchParams] = useSearchParams();

	const snapshot1Id = searchParams.get('snapshot1') ?? '';
	const snapshot2Id = searchParams.get('snapshot2') ?? '';
	const filePath = searchParams.get('path') ?? '';

	const { data: agents } = useAgents();
	const { data: snapshot1 } = useSnapshot(snapshot1Id);
	const { data: snapshot2 } = useSnapshot(snapshot2Id);
	const {
		data: fileDiff,
		isLoading,
		isError,
		error,
	} = useFileDiff(snapshot1Id, snapshot2Id, filePath);

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));

	const missingParams = !snapshot1Id || !snapshot2Id || !filePath;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<nav className="flex items-center space-x-2 text-sm text-gray-500 mb-2">
						<Link to="/snapshots/compare" className="hover:text-indigo-600">
							Snapshot Compare
						</Link>
						{snapshot1Id && snapshot2Id && (
							<>
								<span>/</span>
								<Link
									to={`/snapshots/compare?snapshot1=${snapshot1Id}&snapshot2=${snapshot2Id}`}
									className="hover:text-indigo-600"
								>
									{snapshot1Id.slice(0, 8)} vs {snapshot2Id.slice(0, 8)}
								</Link>
							</>
						)}
						<span>/</span>
						<span className="text-gray-900 font-medium">File Diff</span>
					</nav>
					<h1 className="text-2xl font-bold text-gray-900">File Diff</h1>
					{filePath && (
						<p className="text-gray-600 mt-1 font-mono text-sm">{filePath}</p>
					)}
				</div>
				{snapshot1Id && snapshot2Id && (
					<Link
						to={`/snapshots/compare?snapshot1=${snapshot1Id}&snapshot2=${snapshot2Id}`}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
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
								d="M10 19l-7-7m0 0l7-7m-7 7h18"
							/>
						</svg>
						Back to Comparison
					</Link>
				)}
			</div>

			{missingParams && (
				<div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-yellow-500"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
					<p className="text-yellow-800 font-medium">Missing parameters</p>
					<p className="text-yellow-700 text-sm mt-1">
						Please provide snapshot1, snapshot2, and path query parameters.
					</p>
					<Link
						to="/snapshots/compare"
						className="inline-flex items-center gap-2 mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
					>
						Go to Snapshot Compare
					</Link>
				</div>
			)}

			{!missingParams && (
				<>
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						{snapshot1 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<div className="flex items-center justify-between mb-2">
									<h3 className="font-medium text-gray-900">First Snapshot</h3>
									<code className="text-sm font-mono bg-gray-100 px-2 py-1 rounded">
										{snapshot1.short_id}
									</code>
								</div>
								<div className="text-sm text-gray-600 space-y-1">
									<p>
										<span className="text-gray-500">Agent:</span>{' '}
										{agentMap.get(snapshot1.agent_id) ?? 'Unknown'}
									</p>
									<p>
										<span className="text-gray-500">Date:</span>{' '}
										{formatDateTime(snapshot1.time)}
									</p>
								</div>
							</div>
						)}
						{snapshot2 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<div className="flex items-center justify-between mb-2">
									<h3 className="font-medium text-gray-900">Second Snapshot</h3>
									<code className="text-sm font-mono bg-gray-100 px-2 py-1 rounded">
										{snapshot2.short_id}
									</code>
								</div>
								<div className="text-sm text-gray-600 space-y-1">
									<p>
										<span className="text-gray-500">Agent:</span>{' '}
										{agentMap.get(snapshot2.agent_id) ?? 'Unknown'}
									</p>
									<p>
										<span className="text-gray-500">Date:</span>{' '}
										{formatDateTime(snapshot2.time)}
									</p>
								</div>
							</div>
						)}
					</div>

					{isLoading && (
						<div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
							<svg
								aria-hidden="true"
								className="animate-spin h-8 w-8 mx-auto text-indigo-600 mb-4"
								fill="none"
								viewBox="0 0 24 24"
							>
								<circle
									className="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									strokeWidth="4"
								/>
								<path
									className="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								/>
							</svg>
							<p className="text-gray-600">Loading file diff...</p>
						</div>
					)}

					{isError && (
						<div className="bg-white rounded-lg border border-red-200 p-12 text-center">
							<svg
								aria-hidden="true"
								className="w-12 h-12 mx-auto text-red-400 mb-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<p className="text-red-600 font-medium">
								Failed to load file diff
							</p>
							<p className="text-gray-500 text-sm mt-1">
								{error instanceof Error
									? error.message
									: 'Please check that the file exists in both snapshots'}
							</p>
						</div>
					)}

					{!isLoading && !isError && fileDiff && <DiffViewer diff={fileDiff} />}
				</>
			)}
		</div>
	);
}
