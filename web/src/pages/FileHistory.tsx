import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAgents } from '../hooks/useAgents';
import { useFileHistory } from '../hooks/useFileHistory';
import { useRepositories } from '../hooks/useRepositories';
import { useCreateRestore } from '../hooks/useRestore';
import type { FileHistoryParams, FileVersion } from '../lib/types';
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

interface VersionRowProps {
	version: FileVersion;
	index: number;
	total: number;
	onRestore: (version: FileVersion) => void;
	isLatest: boolean;
}

function VersionRow({
	version,
	index,
	total,
	onRestore,
	isLatest,
}: VersionRowProps) {
	return (
		<div className="relative">
			{/* Timeline line */}
			{index < total - 1 && (
				<div className="absolute left-4 top-10 bottom-0 w-0.5 bg-gray-200" />
			)}

			<div className="flex items-start gap-4 py-4">
				{/* Timeline dot */}
				<div
					className={`relative z-10 w-8 h-8 rounded-full flex items-center justify-center ${
						isLatest ? 'bg-indigo-600 text-white' : 'bg-gray-200 text-gray-600'
					}`}
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
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
						/>
					</svg>
				</div>

				{/* Version info */}
				<div className="flex-1 min-w-0">
					<div className="flex items-center gap-2">
						<span className="font-medium text-gray-900">
							{formatDateTime(version.snapshot_time)}
						</span>
						{isLatest && (
							<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-800">
								Latest
							</span>
						)}
					</div>
					<div className="mt-1 flex items-center gap-4 text-sm text-gray-500">
						<span className="font-mono">{version.short_id}</span>
						<span>{formatBytes(version.size)}</span>
						{version.mod_time && (
							<span>Modified: {formatDateTime(version.mod_time)}</span>
						)}
					</div>
				</div>

				{/* Actions */}
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => onRestore(version)}
						className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-800 hover:bg-indigo-50 rounded-lg transition-colors"
					>
						<svg
							aria-hidden="true"
							className="w-4 h-4 mr-1"
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
				</div>
			</div>
		</div>
	);
}

interface RestoreVersionModalProps {
	version: FileVersion;
	filePath: string;
	agentId: string;
	repositoryId: string;
	onClose: () => void;
	onSubmit: (targetPath: string) => void;
	isSubmitting: boolean;
}

function RestoreVersionModal({
	version,
	filePath,
	onClose,
	onSubmit,
	isSubmitting,
}: RestoreVersionModalProps) {
	const [targetPath, setTargetPath] = useState('');
	const [useOriginalPath, setUseOriginalPath] = useState(true);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		const finalPath = useOriginalPath ? '/' : targetPath;
		onSubmit(finalPath);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg max-w-lg w-full mx-4">
				<div className="p-6 border-b border-gray-200">
					<h3 className="text-lg font-semibold text-gray-900">
						Restore File Version
					</h3>
					<p className="text-sm text-gray-500 mt-1">
						Restore file from snapshot {version.short_id}
					</p>
				</div>

				<form onSubmit={handleSubmit} className="p-6 space-y-4">
					<div>
						<p className="text-sm font-medium text-gray-500">File</p>
						<p className="font-mono text-gray-900 break-all">{filePath}</p>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500">Snapshot Time</p>
							<p className="text-gray-900">
								{formatDateTime(version.snapshot_time)}
							</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500">Size</p>
							<p className="text-gray-900">{formatBytes(version.size)}</p>
						</div>
					</div>

					<div>
						<p className="text-sm font-medium text-gray-700">
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
								<span className="ml-2 text-sm text-gray-900">
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
								<span className="ml-2 text-sm text-gray-900">
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
							className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
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
								'Restore Version'
							)}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

export function FileHistory() {
	const [searchParams, setSearchParams] = useSearchParams();

	const initialPath = searchParams.get('path') || '';
	const initialAgentId = searchParams.get('agent_id') || '';
	const initialRepoId = searchParams.get('repository_id') || '';

	const [filePath, setFilePath] = useState(initialPath);
	const [selectedAgentId, setSelectedAgentId] = useState(initialAgentId);
	const [selectedRepoId, setSelectedRepoId] = useState(initialRepoId);
	const [selectedVersion, setSelectedVersion] = useState<FileVersion | null>(
		null,
	);

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const createRestore = useCreateRestore();

	const queryParams: FileHistoryParams | null =
		filePath && selectedAgentId && selectedRepoId
			? {
					path: filePath,
					agent_id: selectedAgentId,
					repository_id: selectedRepoId,
				}
			: null;

	const {
		data: historyData,
		isLoading,
		isError,
		refetch,
	} = useFileHistory(queryParams);

	const handleSearch = (e: React.FormEvent) => {
		e.preventDefault();
		if (filePath && selectedAgentId && selectedRepoId) {
			const params = new URLSearchParams();
			params.set('path', filePath);
			params.set('agent_id', selectedAgentId);
			params.set('repository_id', selectedRepoId);
			setSearchParams(params);
			refetch();
		}
	};

	const handleRestoreVersion = (version: FileVersion) => {
		setSelectedVersion(version);
	};

	const handleSubmitRestore = (targetPath: string) => {
		if (!selectedVersion || !selectedAgentId || !selectedRepoId) return;

		createRestore.mutate(
			{
				snapshot_id: selectedVersion.snapshot_id,
				agent_id: selectedAgentId,
				repository_id: selectedRepoId,
				target_path: targetPath,
				include_paths: [filePath],
			},
			{
				onSuccess: () => {
					setSelectedVersion(null);
				},
			},
		);
	};

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">File History</h1>
				<p className="text-gray-600 mt-1">
					Browse all versions of a file across backup snapshots
				</p>
			</div>

			{/* Search form */}
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<form onSubmit={handleSearch} className="space-y-4">
					<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
						<div>
							<label
								htmlFor="agent"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Agent
							</label>
							<select
								id="agent"
								value={selectedAgentId}
								onChange={(e) => setSelectedAgentId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
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
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Repository
							</label>
							<select
								id="repository"
								value={selectedRepoId}
								onChange={(e) => setSelectedRepoId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="">Select repository...</option>
								{repositories?.map((repo) => (
									<option key={repo.id} value={repo.id}>
										{repo.name}
									</option>
								))}
							</select>
						</div>

						<div>
							<label
								htmlFor="filePath"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								File Path
							</label>
							<input
								id="filePath"
								type="text"
								value={filePath}
								onChange={(e) => setFilePath(e.target.value)}
								placeholder="/path/to/file"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
							/>
						</div>
					</div>

					<div className="flex justify-end">
						<button
							type="submit"
							disabled={!filePath || !selectedAgentId || !selectedRepoId}
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
							Search History
						</button>
					</div>
				</form>
			</div>

			{/* Results */}
			{queryParams && (
				<div className="bg-white rounded-lg border border-gray-200">
					{isLoading ? (
						<LoadingSpinner />
					) : isError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">Failed to load file history</p>
							<p className="text-sm">Please try again</p>
						</div>
					) : historyData ? (
						<>
							{/* Header */}
							<div className="p-6 border-b border-gray-200">
								<div className="flex items-start justify-between">
									<div>
										<h2 className="text-lg font-semibold text-gray-900">
											Version History
										</h2>
										<p className="font-mono text-sm text-gray-500 mt-1 break-all">
											{historyData.file_path}
										</p>
										<div className="flex items-center gap-4 mt-2 text-sm text-gray-500">
											<span>Agent: {historyData.agent_name}</span>
											<span>Repository: {historyData.repo_name}</span>
										</div>
									</div>
									<span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800">
										{historyData.versions.length} version
										{historyData.versions.length !== 1 ? 's' : ''}
									</span>
								</div>
							</div>

							{/* Timeline */}
							<div className="p-6">
								{historyData.message && (
									<div className="mb-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
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
											<p className="ml-3 text-sm text-yellow-700">
												{historyData.message}
											</p>
										</div>
									</div>
								)}

								{historyData.versions.length > 0 ? (
									<div className="space-y-0">
										{historyData.versions.map((version, index) => (
											<VersionRow
												key={`${version.snapshot_id}-${index}`}
												version={version}
												index={index}
												total={historyData.versions.length}
												onRestore={handleRestoreVersion}
												isLatest={index === 0}
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
												d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
											/>
										</svg>
										<p className="font-medium text-gray-900">
											No versions found
										</p>
										<p className="text-sm">
											No backup snapshots contain this file path
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
				<div className="bg-white rounded-lg border border-gray-200 p-12 text-center text-gray-500">
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
							d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<p className="font-medium text-gray-900">Search for file history</p>
					<p className="text-sm">
						Select an agent, repository, and enter a file path to view all
						versions
					</p>
				</div>
			)}

			{/* Restore modal */}
			{selectedVersion && (
				<RestoreVersionModal
					version={selectedVersion}
					filePath={filePath}
					agentId={selectedAgentId}
					repositoryId={selectedRepoId}
					onClose={() => setSelectedVersion(null)}
					onSubmit={handleSubmitRestore}
					isSubmitting={createRestore.isPending}
				/>
			)}
		</div>
	);
}

export default FileHistory;
