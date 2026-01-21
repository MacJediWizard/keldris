import { useState } from 'react';
import { useAgents } from '../hooks/useAgents';
import { useRepositories } from '../hooks/useRepositories';
import { useCreateRestore, useRestores } from '../hooks/useRestore';
import { useSnapshotFiles, useSnapshots } from '../hooks/useSnapshots';
import type {
	RestoreStatus,
	Restore as RestoreType,
	Snapshot,
	SnapshotFile,
} from '../lib/types';
import { formatBytes, formatDate, formatDateTime } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

function getRestoreStatusColor(status: RestoreStatus): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'completed':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'running':
			return {
				bg: 'bg-blue-100',
				text: 'text-blue-800',
				dot: 'bg-blue-500',
			};
		case 'pending':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		case 'failed':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'canceled':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

interface SnapshotRowProps {
	snapshot: Snapshot;
	agentName?: string;
	repoName?: string;
	onSelect: (snapshot: Snapshot) => void;
}

function SnapshotRow({
	snapshot,
	agentName,
	repoName,
	onSelect,
}: SnapshotRowProps) {
	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<code className="text-sm font-mono text-gray-900">
					{snapshot.short_id}
				</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				{agentName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				{repoName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatBytes(snapshot.size_bytes)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(snapshot.time)}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onSelect(snapshot)}
					className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
				>
					Restore
				</button>
			</td>
		</tr>
	);
}

interface RestoreRowProps {
	restore: RestoreType;
	agentName?: string;
	onViewDetails: (restore: RestoreType) => void;
}

function RestoreRow({ restore, agentName, onViewDetails }: RestoreRowProps) {
	const statusColor = getRestoreStatusColor(restore.status);

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<code className="text-sm font-mono text-gray-900">
					{restore.snapshot_id.substring(0, 8)}
				</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				{agentName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-xs truncate">
				{restore.target_path}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{restore.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(restore.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onViewDetails(restore)}
					className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
				>
					Details
				</button>
			</td>
		</tr>
	);
}

interface RestoreDetailsModalProps {
	restore: RestoreType;
	agentName?: string;
	onClose: () => void;
}

function RestoreDetailsModal({
	restore,
	agentName,
	onClose,
}: RestoreDetailsModalProps) {
	const statusColor = getRestoreStatusColor(restore.status);

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Restore Details
					</h3>
					<span
						className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
					>
						<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
						{restore.status}
					</span>
				</div>

				<div className="space-y-4">
					<div>
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Snapshot ID
						</p>
						<p className="font-mono text-gray-900">{restore.snapshot_id}</p>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Agent
							</p>
							<p className="text-gray-900">{agentName ?? 'Unknown'}</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Created
							</p>
							<p className="text-gray-900">
								{formatDateTime(restore.created_at)}
							</p>
						</div>
					</div>

					<div>
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Target Path
						</p>
						<p className="font-mono text-gray-900 break-all">
							{restore.target_path}
						</p>
					</div>

					{restore.include_paths && restore.include_paths.length > 0 && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Included Paths
							</p>
							<ul className="list-disc list-inside text-sm text-gray-900">
								{restore.include_paths.map((path) => (
									<li key={path} className="font-mono truncate">
										{path}
									</li>
								))}
							</ul>
						</div>
					)}

					{restore.started_at && (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Started
								</p>
								<p className="text-gray-900">
									{formatDateTime(restore.started_at)}
								</p>
							</div>
							{restore.completed_at && (
								<div>
									<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Completed
									</p>
									<p className="text-gray-900">
										{formatDateTime(restore.completed_at)}
									</p>
								</div>
							)}
						</div>
					)}

					{restore.error_message && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Error
							</p>
							<p className="text-red-600 bg-red-50 p-3 rounded-lg text-sm">
								{restore.error_message}
							</p>
						</div>
					)}
				</div>

				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

interface FileTreeItemProps {
	file: SnapshotFile;
	selectedPaths: Set<string>;
	onToggle: (path: string) => void;
	depth: number;
}

function FileTreeItem({
	file,
	selectedPaths,
	onToggle,
	depth,
}: FileTreeItemProps) {
	const isSelected = selectedPaths.has(file.path);

	return (
		<div
			className="flex items-center py-1 hover:bg-gray-50 rounded"
			style={{ paddingLeft: `${depth * 16 + 8}px` }}
		>
			<input
				type="checkbox"
				checked={isSelected}
				onChange={() => onToggle(file.path)}
				className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
			/>
			<span className="ml-2 flex items-center">
				{file.type === 'dir' ? (
					<svg
						aria-hidden="true"
						className="w-4 h-4 text-yellow-500 mr-1"
						fill="currentColor"
						viewBox="0 0 20 20"
					>
						<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
					</svg>
				) : (
					<svg
						aria-hidden="true"
						className="w-4 h-4 text-gray-400 mr-1"
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
				<span className="text-sm text-gray-900">{file.name}</span>
			</span>
			<span className="ml-auto text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
				{file.type === 'file' ? formatBytes(file.size) : ''}
			</span>
		</div>
	);
}

interface RestoreModalProps {
	snapshot: Snapshot;
	agentName?: string;
	repoName?: string;
	onClose: () => void;
	onSubmit: (targetPath: string, includePaths: string[]) => void;
	isSubmitting: boolean;
}

function RestoreModal({
	snapshot,
	agentName,
	repoName,
	onClose,
	onSubmit,
	isSubmitting,
}: RestoreModalProps) {
	const [targetPath, setTargetPath] = useState('');
	const [useOriginalPath, setUseOriginalPath] = useState(true);
	const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());

	const { data: filesData } = useSnapshotFiles(snapshot.id);

	const togglePath = (path: string) => {
		const newSelected = new Set(selectedPaths);
		if (newSelected.has(path)) {
			newSelected.delete(path);
		} else {
			newSelected.add(path);
		}
		setSelectedPaths(newSelected);
	};

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		const finalTargetPath = useOriginalPath ? '/' : targetPath;
		const includePaths = Array.from(selectedPaths);
		onSubmit(finalTargetPath, includePaths);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Restore Snapshot
					</h3>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Snapshot {snapshot.short_id} from {formatDateTime(snapshot.time)}
					</p>
				</div>

				<form
					onSubmit={handleSubmit}
					className="flex flex-col flex-1 overflow-hidden"
				>
					<div className="p-6 space-y-6 overflow-y-auto flex-1">
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Agent
								</p>
								<p className="text-gray-900">{agentName ?? 'Unknown'}</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Repository
								</p>
								<p className="text-gray-900">{repoName ?? 'Unknown'}</p>
							</div>
						</div>

						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Backed up paths
							</p>
							<ul className="list-disc list-inside text-sm text-gray-900">
								{snapshot.paths.map((path) => (
									<li key={path} className="font-mono">
										{path}
									</li>
								))}
							</ul>
						</div>

						<div>
							<p className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300 dark:text-gray-600">
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
										className="mt-2 w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
									/>
								)}
							</div>
						</div>

						<div>
							<p className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300 dark:text-gray-600">
								Select files to restore (optional)
							</p>
							<p className="text-xs text-gray-500 dark:text-gray-400 mt-1 mb-2">
								Leave empty to restore all files
							</p>
							<div className="border border-gray-200 rounded-lg p-2 max-h-48 overflow-y-auto bg-gray-50">
								{filesData?.files && filesData.files.length > 0 ? (
									filesData.files.map((file) => (
										<FileTreeItem
											key={file.path}
											file={file}
											selectedPaths={selectedPaths}
											onToggle={togglePath}
											depth={0}
										/>
									))
								) : (
									<p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
										{filesData?.message ||
											'File listing not available. All files will be restored.'}
									</p>
								)}
							</div>
							{selectedPaths.size > 0 && (
								<p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
									{selectedPaths.size} item(s) selected
								</p>
							)}
						</div>
					</div>

					<div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							disabled={isSubmitting}
							className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
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
									Starting...
								</>
							) : (
								'Start Restore'
							)}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

export function Restore() {
	const [activeTab, setActiveTab] = useState<'snapshots' | 'restores'>(
		'snapshots',
	);
	const [agentFilter, setAgentFilter] = useState<string>('all');
	const [repoFilter, setRepoFilter] = useState<string>('all');
	const [selectedSnapshot, setSelectedSnapshot] = useState<Snapshot | null>(
		null,
	);
	const [selectedRestore, setSelectedRestore] = useState<RestoreType | null>(
		null,
	);

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const {
		data: snapshots,
		isLoading: snapshotsLoading,
		isError: snapshotsError,
	} = useSnapshots({
		agent_id: agentFilter !== 'all' ? agentFilter : undefined,
		repository_id: repoFilter !== 'all' ? repoFilter : undefined,
	});
	const {
		data: restores,
		isLoading: restoresLoading,
		isError: restoresError,
	} = useRestores({
		agent_id: agentFilter !== 'all' ? agentFilter : undefined,
	});
	const createRestore = useCreateRestore();

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));
	const repoMap = new Map(repositories?.map((r) => [r.id, r.name]));

	const handleRestore = (targetPath: string, includePaths: string[]) => {
		if (!selectedSnapshot) return;

		createRestore.mutate(
			{
				snapshot_id: selectedSnapshot.id,
				agent_id: selectedSnapshot.agent_id,
				repository_id: selectedSnapshot.repository_id,
				target_path: targetPath,
				include_paths: includePaths.length > 0 ? includePaths : undefined,
			},
			{
				onSuccess: () => {
					setSelectedSnapshot(null);
					setActiveTab('restores');
				},
			},
		);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Restore
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Browse snapshots and restore files
					</p>
				</div>
			</div>

			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('snapshots')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'snapshots'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Snapshots
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('restores')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'restores'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Restore Jobs
					</button>
				</nav>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<select
							value={agentFilter}
							onChange={(e) => setAgentFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Agents</option>
							{agents?.map((agent) => (
								<option key={agent.id} value={agent.id}>
									{agent.hostname}
								</option>
							))}
						</select>
						{activeTab === 'snapshots' && (
							<select
								value={repoFilter}
								onChange={(e) => setRepoFilter(e.target.value)}
								className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="all">All Repositories</option>
								{repositories?.map((repo) => (
									<option key={repo.id} value={repo.id}>
										{repo.name}
									</option>
								))}
							</select>
						)}
					</div>
				</div>

				<div className="overflow-x-auto">
					{activeTab === 'snapshots' ? (
						snapshotsError ? (
							<div className="p-12 text-center text-red-500 dark:text-red-400 dark:text-red-400">
								<p className="font-medium">Failed to load snapshots</p>
								<p className="text-sm">Please try refreshing the page</p>
							</div>
						) : snapshotsLoading ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Snapshot ID
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Agent
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Repository
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Size
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Date
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									<LoadingRow />
									<LoadingRow />
									<LoadingRow />
								</tbody>
							</table>
						) : snapshots && snapshots.length > 0 ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Snapshot ID
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Agent
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Repository
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Size
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Date
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{snapshots.map((snapshot) => (
										<SnapshotRow
											key={snapshot.id}
											snapshot={snapshot}
											agentName={agentMap.get(snapshot.agent_id)}
											repoName={repoMap.get(snapshot.repository_id)}
											onSelect={setSelectedSnapshot}
										/>
									))}
								</tbody>
							</table>
						) : (
							<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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
										d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
									/>
								</svg>
								<p className="font-medium text-gray-900 dark:text-white">
									No snapshots found
								</p>
								<p className="text-sm">
									Snapshots will appear here once backups complete
								</p>
							</div>
						)
					) : restoresError ? (
						<div className="p-12 text-center text-red-500 dark:text-red-400 dark:text-red-400">
							<p className="font-medium">Failed to load restore jobs</p>
							<p className="text-sm">Please try refreshing the page</p>
						</div>
					) : restoresLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Snapshot ID
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Agent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Target Path
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Date
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : restores && restores.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Snapshot ID
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Agent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Target Path
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Date
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{restores.map((restore) => (
									<RestoreRow
										key={restore.id}
										restore={restore}
										agentName={agentMap.get(restore.agent_id)}
										onViewDetails={setSelectedRestore}
									/>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
								/>
							</svg>
							<p className="font-medium text-gray-900 dark:text-white">
								No restore jobs
							</p>
							<p className="text-sm">
								Select a snapshot to start a restore operation
							</p>
						</div>
					)}
				</div>
			</div>

			{selectedSnapshot && (
				<RestoreModal
					snapshot={selectedSnapshot}
					agentName={agentMap.get(selectedSnapshot.agent_id)}
					repoName={repoMap.get(selectedSnapshot.repository_id)}
					onClose={() => setSelectedSnapshot(null)}
					onSubmit={handleRestore}
					isSubmitting={createRestore.isPending}
				/>
			)}

			{selectedRestore && (
				<RestoreDetailsModal
					restore={selectedRestore}
					agentName={agentMap.get(selectedRestore.agent_id)}
					onClose={() => setSelectedRestore(null)}
				/>
			)}
		</div>
	);
}
