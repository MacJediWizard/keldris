import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAgents } from '../hooks/useAgents';
import { useFileSearch } from '../hooks/useFileSearch';
import { useRepositories } from '../hooks/useRepositories';
import { useCreateRestore } from '../hooks/useRestore';
import type {
	FileSearchParams,
	FileSearchResult,
	SnapshotFileGroup,
} from '../lib/types';
import { formatBytes, formatDateTime } from '../lib/utils';

function LoadingSpinner() {
	return (
		<div className="flex items-center justify-center py-12">
			<svg
				aria-hidden="true"
				className="animate-spin h-8 w-8 text-indigo-600"
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
		</div>
	);
}

interface FileResultRowProps {
	file: FileSearchResult;
	onRestore: (file: FileSearchResult) => void;
}

function FileResultRow({ file, onRestore }: FileResultRowProps) {
	const isDirectory = file.file_type === 'dir';

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-4 py-3">
				<div className="flex items-center gap-2">
					{isDirectory ? (
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-yellow-500 flex-shrink-0"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
						</svg>
					) : (
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-gray-400 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
					)}
					<div className="min-w-0">
						<p className="font-medium text-gray-900 dark:text-white truncate">
							{file.file_name}
						</p>
						<p className="text-xs text-gray-500 dark:text-gray-400 font-mono truncate">
							{file.file_path}
						</p>
					</div>
				</div>
			</td>
			<td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 text-right">
				{isDirectory ? '-' : formatBytes(file.file_size)}
			</td>
			<td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
				{formatDateTime(file.mod_time)}
			</td>
			<td className="px-4 py-3 text-right">
				{!isDirectory && (
					<button
						type="button"
						onClick={() => onRestore(file)}
						className="inline-flex items-center px-2 py-1 text-xs font-medium text-indigo-600 hover:text-indigo-800 hover:bg-indigo-50 rounded transition-colors"
					>
						<svg
							aria-hidden="true"
							className="w-3.5 h-3.5 mr-1"
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
						Restore
					</button>
				)}
			</td>
		</tr>
	);
}

interface SnapshotGroupCardProps {
	group: SnapshotFileGroup;
	isExpanded: boolean;
	onToggle: () => void;
	onRestore: (file: FileSearchResult) => void;
}

function SnapshotGroupCard({
	group,
	isExpanded,
	onToggle,
	onRestore,
}: SnapshotGroupCardProps) {
	const shortId =
		group.snapshot_id.length > 8
			? group.snapshot_id.slice(0, 8)
			: group.snapshot_id;

	return (
		<div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
			<button
				type="button"
				onClick={onToggle}
				className="w-full px-4 py-3 bg-gray-50 dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between transition-colors"
			>
				<div className="flex items-center gap-4">
					<span className="font-mono text-sm font-medium text-indigo-600 bg-indigo-50 px-2 py-1 rounded">
						{shortId}
					</span>
					<div className="text-left">
						<p className="font-medium text-gray-900 dark:text-white">
							{formatDateTime(group.snapshot_time)}
						</p>
						<p className="text-sm text-gray-500 dark:text-gray-400">{group.hostname}</p>
					</div>
				</div>
				<div className="flex items-center gap-4">
					<span className="text-sm text-gray-500 dark:text-gray-400">
						{group.file_count} file{group.file_count !== 1 ? 's' : ''}
					</span>
					<svg
						aria-hidden="true"
						className={`w-5 h-5 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M19 9l-7 7-7-7"
						/>
					</svg>
				</div>
			</button>

			{isExpanded && (
				<div className="overflow-x-auto">
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									File
								</th>
								<th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Size
								</th>
								<th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Modified
								</th>
								<th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{group.files.map((file, index) => (
								<FileResultRow
									key={`${file.file_path}-${index}`}
									file={file}
									onRestore={onRestore}
								/>
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}

interface RestoreFileModalProps {
	file: FileSearchResult;
	agentId: string;
	repositoryId: string;
	onClose: () => void;
	onSubmit: (targetPath: string) => void;
	isSubmitting: boolean;
}

function RestoreFileModal({
	file,
	onClose,
	onSubmit,
	isSubmitting,
}: RestoreFileModalProps) {
	const [targetPath, setTargetPath] = useState('');
	const [useOriginalPath, setUseOriginalPath] = useState(true);
	const shortId =
		file.snapshot_id.length > 8
			? file.snapshot_id.slice(0, 8)
			: file.snapshot_id;

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		const finalPath = useOriginalPath ? '/' : targetPath;
		onSubmit(finalPath);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg max-w-lg w-full mx-4">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">Restore File</h3>
					<p className="text-sm text-gray-500 mt-1">
						Restore file from snapshot {shortId}
					</p>
				</div>

				<form onSubmit={handleSubmit} className="p-6 space-y-4">
					<div>
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">File</p>
						<p className="font-mono text-gray-900 dark:text-white break-all">
							{file.file_path}
						</p>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">Snapshot Time</p>
							<p className="text-gray-900">
								{formatDateTime(file.snapshot_time)}
							</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">Size</p>
							<p className="text-gray-900 dark:text-white">{formatBytes(file.file_size)}</p>
						</div>
					</div>

					<div>
						<p className="text-sm font-medium text-gray-700 dark:text-gray-300">
							Restore Destination
						</p>
						<div className="mt-2 space-y-2">
							<label className="flex items-center">
								<input
									type="radio"
									checked={useOriginalPath}
									onChange={() => setUseOriginalPath(true)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300"
								/>
								<span className="ml-2 text-sm text-gray-900 dark:text-white">
									Original location
								</span>
							</label>
							<label className="flex items-center">
								<input
									type="radio"
									checked={!useOriginalPath}
									onChange={() => setUseOriginalPath(false)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300"
								/>
								<span className="ml-2 text-sm text-gray-900 dark:text-white">
									Custom location
								</span>
							</label>
							{!useOriginalPath && (
								<input
									type="text"
									value={targetPath}
									onChange={(e) => setTargetPath(e.target.value)}
									placeholder="/path/to/restore"
									className="mt-2 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
								/>
							)}
						</div>
					</div>

					<div className="flex justify-end gap-3 pt-4">
						<button
							type="button"
							onClick={onClose}
							disabled={isSubmitting}
							className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={isSubmitting || (!useOriginalPath && !targetPath)}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 flex items-center gap-2"
						>
							{isSubmitting ? (
								<>
									<svg
										aria-hidden="true"
										className="animate-spin h-4 w-4"
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
									Restoring...
								</>
							) : (
								'Restore File'
							)}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

export function FileSearch() {
	const [searchParams, setSearchParams] = useSearchParams();

	const initialQuery = searchParams.get('q') || '';
	const initialAgentId = searchParams.get('agent_id') || '';
	const initialRepoId = searchParams.get('repository_id') || '';
	const initialPath = searchParams.get('path') || '';

	const [searchQuery, setSearchQuery] = useState(initialQuery);
	const [selectedAgentId, setSelectedAgentId] = useState(initialAgentId);
	const [selectedRepoId, setSelectedRepoId] = useState(initialRepoId);
	const [pathFilter, setPathFilter] = useState(initialPath);
	const [expandedSnapshots, setExpandedSnapshots] = useState<Set<string>>(
		new Set(),
	);
	const [selectedFile, setSelectedFile] = useState<FileSearchResult | null>(
		null,
	);

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const createRestore = useCreateRestore();

	const queryParams: FileSearchParams | null =
		searchQuery && selectedAgentId && selectedRepoId
			? {
					q: searchQuery,
					agent_id: selectedAgentId,
					repository_id: selectedRepoId,
					path: pathFilter || undefined,
					limit: 100,
				}
			: null;

	const {
		data: searchData,
		isLoading,
		isError,
		refetch,
	} = useFileSearch(queryParams);

	const handleSearch = (e: React.FormEvent) => {
		e.preventDefault();
		if (searchQuery && selectedAgentId && selectedRepoId) {
			const params = new URLSearchParams();
			params.set('q', searchQuery);
			params.set('agent_id', selectedAgentId);
			params.set('repository_id', selectedRepoId);
			if (pathFilter) params.set('path', pathFilter);
			setSearchParams(params);
			refetch();
		}
	};

	const toggleSnapshot = (snapshotId: string) => {
		setExpandedSnapshots((prev) => {
			const next = new Set(prev);
			if (next.has(snapshotId)) {
				next.delete(snapshotId);
			} else {
				next.add(snapshotId);
			}
			return next;
		});
	};

	const expandAll = () => {
		if (searchData?.snapshots) {
			setExpandedSnapshots(
				new Set(searchData.snapshots.map((g) => g.snapshot_id)),
			);
		}
	};

	const collapseAll = () => {
		setExpandedSnapshots(new Set());
	};

	const handleRestoreFile = (file: FileSearchResult) => {
		setSelectedFile(file);
	};

	const handleSubmitRestore = (targetPath: string) => {
		if (!selectedFile || !selectedAgentId || !selectedRepoId) return;

		createRestore.mutate(
			{
				snapshot_id: selectedFile.snapshot_id,
				agent_id: selectedAgentId,
				repository_id: selectedRepoId,
				target_path: targetPath,
				include_paths: [selectedFile.file_path],
			},
			{
				onSuccess: () => {
					setSelectedFile(null);
				},
			},
		);
	};

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">File Search</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
					Search for files by name across all backup snapshots
				</p>
			</div>

			{/* Search form */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 p-6">
				<form onSubmit={handleSearch} className="space-y-4">
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="agent"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Agent
							</label>
							<select
								id="agent"
								value={selectedAgentId}
								onChange={(e) => setSelectedAgentId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="">Select agent...</option>
								{agents?.map((agent) => (
									<option key={agent.id} value={agent.id}>
										{agent.hostname}
									</option>
								))}
							</select>
						</div>

						<div>
							<label
								htmlFor="repository"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Repository
							</label>
							<select
								id="repository"
								value={selectedRepoId}
								onChange={(e) => setSelectedRepoId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="">Select repository...</option>
								{repositories?.map((repo) => (
									<option key={repo.id} value={repo.id}>
										{repo.name}
									</option>
								))}
							</select>
						</div>
					</div>

					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="searchQuery"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								File Name
							</label>
							<input
								id="searchQuery"
								type="text"
								value={searchQuery}
								onChange={(e) => setSearchQuery(e.target.value)}
								placeholder="config.json, *.log, my-file*"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Supports wildcards: * matches any characters
							</p>
						</div>

						<div>
							<label
								htmlFor="pathFilter"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Path Filter (Optional)
							</label>
							<input
								id="pathFilter"
								type="text"
								value={pathFilter}
								onChange={(e) => setPathFilter(e.target.value)}
								placeholder="/home/user/documents"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
							/>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Limit search to files under this path
							</p>
						</div>
					</div>

					<div className="flex justify-end">
						<button
							type="submit"
							disabled={!searchQuery || !selectedAgentId || !selectedRepoId}
							className="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
						>
							<svg
								aria-hidden="true"
								className="w-4 h-4 mr-2"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
								/>
							</svg>
							Search Files
						</button>
					</div>
				</form>
			</div>

			{/* Results */}
			{queryParams && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200">
					{isLoading ? (
						<LoadingSpinner />
					) : isError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">Failed to search files</p>
							<p className="text-sm">Please try again</p>
						</div>
					) : searchData ? (
						<>
							{/* Header */}
							<div className="p-6 border-b border-gray-200 dark:border-gray-700">
								<div className="flex items-start justify-between">
									<div>
										<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
											Search Results
										</h2>
										<p className="text-sm text-gray-500 mt-1">
											Found{' '}
											<span className="font-medium">
												{searchData.total_count}
											</span>{' '}
											file
											{searchData.total_count !== 1 ? 's' : ''} in{' '}
											<span className="font-medium">
												{searchData.snapshot_count}
											</span>{' '}
											snapshot
											{searchData.snapshot_count !== 1 ? 's' : ''} matching "
											{searchData.query}"
										</p>
									</div>
									{searchData.snapshot_count > 0 && (
										<div className="flex gap-2">
											<button
												type="button"
												onClick={expandAll}
												className="text-sm text-indigo-600 hover:text-indigo-800"
											>
												Expand all
											</button>
											<span className="text-gray-300">|</span>
											<button
												type="button"
												onClick={collapseAll}
												className="text-sm text-indigo-600 hover:text-indigo-800"
											>
												Collapse all
											</button>
										</div>
									)}
								</div>
							</div>

							{/* Results grouped by snapshot */}
							<div className="p-6">
								{searchData.message && (
									<div className="mb-6 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
										<div className="flex">
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-yellow-400"
												fill="currentColor"
												viewBox="0 0 20 20"
											>
												<path
													fillRule="evenodd"
													d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
													clipRule="evenodd"
												/>
											</svg>
											<p className="ml-3 text-sm text-yellow-700 dark:text-yellow-300">
												{searchData.message}
											</p>
										</div>
									</div>
								)}

								{searchData.snapshots.length > 0 ? (
									<div className="space-y-4">
										{searchData.snapshots.map((group) => (
											<SnapshotGroupCard
												key={group.snapshot_id}
												group={group}
												isExpanded={expandedSnapshots.has(group.snapshot_id)}
												onToggle={() => toggleSnapshot(group.snapshot_id)}
												onRestore={handleRestoreFile}
											/>
										))}
									</div>
								) : (
									<div className="text-center py-12 text-gray-500">
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
												d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
											/>
										</svg>
										<p className="font-medium text-gray-900 dark:text-white">No files found</p>
										<p className="text-sm">
											No files matching "{searchData.query}" were found in any
											snapshot
										</p>
									</div>
								)}
							</div>
						</>
					) : null}
				</div>
			)}

			{/* Empty state before search */}
			{!queryParams && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 p-12 text-center text-gray-500">
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
							d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
						/>
					</svg>
					<p className="font-medium text-gray-900 dark:text-white">
						Search for files across snapshots
					</p>
					<p className="text-sm">
						Select an agent, repository, and enter a filename pattern to search
					</p>
				</div>
			)}

			{/* Restore modal */}
			{selectedFile && (
				<RestoreFileModal
					file={selectedFile}
					agentId={selectedAgentId}
					repositoryId={selectedRepoId}
					onClose={() => setSelectedFile(null)}
					onSubmit={handleSubmitRestore}
					isSubmitting={createRestore.isPending}
				/>
			)}
		</div>
	);
}
