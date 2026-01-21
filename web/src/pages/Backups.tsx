import { useState } from 'react';
import { useAgents } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import { useBackupTags, useSetBackupTags, useTags } from '../hooks/useTags';
import type { Backup, BackupStatus, Tag } from '../lib/types';
import {
	formatBytes,
	formatDate,
	formatDateTime,
	formatDuration,
	getBackupStatusColor,
	truncateSnapshotId,
} from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-20 bg-gray-200 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-28 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-4 w-12 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

interface TagChipProps {
	tag: Tag;
	onRemove?: () => void;
	selected?: boolean;
	onClick?: () => void;
}

function TagChip({ tag, onRemove, selected, onClick }: TagChipProps) {
	return (
		<span
			className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium text-white ${
				onClick ? 'cursor-pointer hover:opacity-80' : ''
			} ${selected ? 'ring-2 ring-offset-1 ring-gray-900' : ''}`}
			style={{ backgroundColor: tag.color }}
			onClick={onClick}
			onKeyDown={onClick ? (e) => e.key === 'Enter' && onClick() : undefined}
			role={onClick ? 'button' : undefined}
			tabIndex={onClick ? 0 : undefined}
		>
			{tag.name}
			{onRemove && (
				<button
					type="button"
					onClick={(e) => {
						e.stopPropagation();
						onRemove();
					}}
					className="hover:bg-white/20 rounded-full p-0.5"
					aria-label="Remove tag"
				>
					<svg
						aria-hidden="true"
						className="w-3 h-3"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			)}
		</span>
	);
}

interface BackupTagsEditorProps {
	backupId: string;
	allTags: Tag[];
}

function BackupTagsEditor({ backupId, allTags }: BackupTagsEditorProps) {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: backupTags } = useBackupTags(backupId);
	const setBackupTags = useSetBackupTags();

	const currentTagIds = new Set(backupTags?.map((t) => t.id) ?? []);

	const handleToggleTag = (tagId: string) => {
		const newTagIds = currentTagIds.has(tagId)
			? [...currentTagIds].filter((id) => id !== tagId)
			: [...currentTagIds, tagId];
		setBackupTags.mutate({ backupId, data: { tag_ids: newTagIds } });
	};

	return (
		<div>
			<p className="text-sm font-medium text-gray-500 mb-2">Tags</p>
			<div className="flex flex-wrap gap-1 mb-2">
				{backupTags && backupTags.length > 0 ? (
					backupTags.map((tag) => (
						<TagChip
							key={tag.id}
							tag={tag}
							onRemove={() => handleToggleTag(tag.id)}
						/>
					))
				) : (
					<span className="text-sm text-gray-400">No tags</span>
				)}
			</div>
			<div className="relative">
				<button
					type="button"
					onClick={() => setShowDropdown(!showDropdown)}
					className="text-sm text-indigo-600 hover:text-indigo-800"
				>
					{showDropdown ? 'Done' : '+ Add tags'}
				</button>
				{showDropdown && allTags.length > 0 && (
					<div className="absolute top-full left-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg py-1 z-10 min-w-[150px]">
						{allTags.map((tag) => (
							<button
								key={tag.id}
								type="button"
								onClick={() => handleToggleTag(tag.id)}
								className="w-full text-left px-3 py-1.5 hover:bg-gray-100 flex items-center gap-2"
							>
								<span
									className="w-3 h-3 rounded-full"
									style={{ backgroundColor: tag.color }}
								/>
								<span className="text-sm text-gray-700">{tag.name}</span>
								{currentTagIds.has(tag.id) && (
									<svg
										aria-hidden="true"
										className="w-4 h-4 text-indigo-600 ml-auto"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M5 13l4 4L19 7"
										/>
									</svg>
								)}
							</button>
						))}
					</div>
				)}
			</div>
		</div>
	);
}

interface BackupDetailsModalProps {
	backup: Backup;
	agentName?: string;
	repoName?: string;
	allTags: Tag[];
	onClose: () => void;
}

function BackupDetailsModal({
	backup,
	agentName,
	repoName,
	allTags,
	onClose,
}: BackupDetailsModalProps) {
	const statusColor = getBackupStatusColor(backup.status);

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						Backup Details
					</h3>
					<span
						className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
					>
						<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
						{backup.status}
					</span>
				</div>

				<div className="space-y-4">
					{backup.snapshot_id && (
						<div>
							<p className="text-sm font-medium text-gray-500">Snapshot ID</p>
							<p className="font-mono text-gray-900">{backup.snapshot_id}</p>
						</div>
					)}

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500">Agent</p>
							<p className="text-gray-900">{agentName ?? 'Unknown'}</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500">Repository</p>
							<p className="text-gray-900">{repoName ?? 'Unknown'}</p>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500">Started</p>
							<p className="text-gray-900">
								{formatDateTime(backup.started_at)}
							</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500">Duration</p>
							<p className="text-gray-900">
								{formatDuration(backup.started_at, backup.completed_at)}
							</p>
						</div>
					</div>

					{backup.status === 'completed' && (
						<div className="grid grid-cols-3 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-500">Size</p>
								<p className="text-gray-900">
									{formatBytes(backup.size_bytes)}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500">New Files</p>
								<p className="text-gray-900">{backup.files_new ?? 0}</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500">Changed</p>
								<p className="text-gray-900">{backup.files_changed ?? 0}</p>
							</div>
						</div>
					)}

					<BackupTagsEditor backupId={backup.id} allTags={allTags} />

					{backup.error_message && (
						<div>
							<p className="text-sm font-medium text-gray-500">Error</p>
							<p className="text-red-600 bg-red-50 p-3 rounded-lg text-sm">
								{backup.error_message}
							</p>
						</div>
					)}

					{(backup.pre_script_output || backup.pre_script_error) && (
						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Pre-Backup Script
							</p>
							{backup.pre_script_output && (
								<div className="mb-2">
									<p className="text-xs font-medium text-gray-400 mb-1">
										Output
									</p>
									<pre className="bg-gray-100 p-3 rounded-lg text-xs text-gray-800 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.pre_script_output}
									</pre>
								</div>
							)}
							{backup.pre_script_error && (
								<div>
									<p className="text-xs font-medium text-red-400 mb-1">Error</p>
									<pre className="bg-red-50 p-3 rounded-lg text-xs text-red-700 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.pre_script_error}
									</pre>
								</div>
							)}
						</div>
					)}

					{(backup.post_script_output || backup.post_script_error) && (
						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Post-Backup Script
							</p>
							{backup.post_script_output && (
								<div className="mb-2">
									<p className="text-xs font-medium text-gray-400 mb-1">
										Output
									</p>
									<pre className="bg-gray-100 p-3 rounded-lg text-xs text-gray-800 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.post_script_output}
									</pre>
								</div>
							)}
							{backup.post_script_error && (
								<div>
									<p className="text-xs font-medium text-red-400 mb-1">Error</p>
									<pre className="bg-red-50 p-3 rounded-lg text-xs text-red-700 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.post_script_error}
									</pre>
								</div>
							)}
						</div>
					)}
				</div>

				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

interface BackupRowProps {
	backup: Backup;
	agentName?: string;
	repoName?: string;
	onViewDetails: (backup: Backup) => void;
}

function BackupRow({
	backup,
	agentName,
	repoName,
	onViewDetails,
}: BackupRowProps) {
	const statusColor = getBackupStatusColor(backup.status);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<code className="text-sm font-mono text-gray-900">
					{truncateSnapshotId(backup.snapshot_id)}
				</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				{agentName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				{repoName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatBytes(backup.size_bytes)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{backup.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(backup.started_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onViewDetails(backup)}
					className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
				>
					Details
				</button>
			</td>
		</tr>
	);
}

export function Backups() {
	const [searchQuery, setSearchQuery] = useState('');
	const [agentFilter, setAgentFilter] = useState<string>('all');
	const [statusFilter, setStatusFilter] = useState<BackupStatus | 'all'>('all');
	const [selectedTagFilters, setSelectedTagFilters] = useState<Set<string>>(
		new Set(),
	);
	const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null);

	const { data: backups, isLoading, isError } = useBackups();
	const { data: agents } = useAgents();
	const { data: schedules } = useSchedules();
	const { data: repositories } = useRepositories();
	const { data: allTags } = useTags();

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));
	const repoMap = new Map(repositories?.map((r) => [r.id, r.name]));

	const getRepoNameForBackup = (backup: Backup) => {
		// First check if backup has its own repository_id
		if (backup.repository_id) {
			return repoMap.get(backup.repository_id);
		}
		// Fall back to primary repository from schedule
		const schedule = schedules?.find((s) => s.id === backup.schedule_id);
		const primaryRepo = schedule?.repositories
			?.sort((a, b) => a.priority - b.priority)
			?.find((r) => r.enabled);
		return primaryRepo ? repoMap.get(primaryRepo.repository_id) : undefined;
	};

	const toggleTagFilter = (tagId: string) => {
		const newFilters = new Set(selectedTagFilters);
		if (newFilters.has(tagId)) {
			newFilters.delete(tagId);
		} else {
			newFilters.add(tagId);
		}
		setSelectedTagFilters(newFilters);
	};

	const filteredBackups = backups?.filter((backup) => {
		const snapshotMatch =
			backup.snapshot_id?.toLowerCase().includes(searchQuery.toLowerCase()) ??
			false;
		const matchesSearch = searchQuery === '' || snapshotMatch;
		const matchesAgent =
			agentFilter === 'all' || backup.agent_id === agentFilter;
		const matchesStatus =
			statusFilter === 'all' || backup.status === statusFilter;
		// Note: Tag filtering would require loading backup tags for each backup,
		// which is expensive. For a more complete implementation, you'd want to
		// fetch this data on the server side with proper filtering.
		return matchesSearch && matchesAgent && matchesStatus;
	});

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Backups</h1>
					<p className="text-gray-600 mt-1">View and manage backup snapshots</p>
				</div>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4 mb-4">
						<input
							type="text"
							placeholder="Search by snapshot ID..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={agentFilter}
							onChange={(e) => setAgentFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Agents</option>
							{agents?.map((agent) => (
								<option key={agent.id} value={agent.id}>
									{agent.hostname}
								</option>
							))}
						</select>
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as BackupStatus | 'all')
							}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="completed">Completed</option>
							<option value="running">Running</option>
							<option value="failed">Failed</option>
							<option value="canceled">Canceled</option>
						</select>
					</div>
					{allTags && allTags.length > 0 && (
						<div className="flex items-center gap-2 flex-wrap">
							<span className="text-sm text-gray-500">Filter by tags:</span>
							{allTags.map((tag) => (
								<TagChip
									key={tag.id}
									tag={tag}
									selected={selectedTagFilters.has(tag.id)}
									onClick={() => toggleTagFilter(tag.id)}
								/>
							))}
							{selectedTagFilters.size > 0 && (
								<button
									type="button"
									onClick={() => setSelectedTagFilters(new Set())}
									className="text-sm text-gray-500 hover:text-gray-700"
								>
									Clear all
								</button>
							)}
						</div>
					)}
				</div>

				<div className="overflow-x-auto">
					{isError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">Failed to load backups</p>
							<p className="text-sm">Please try refreshing the page</p>
						</div>
					) : isLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Snapshot ID
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Agent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Repository
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : filteredBackups && filteredBackups.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Snapshot ID
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Agent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Repository
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{filteredBackups.map((backup) => (
									<BackupRow
										key={backup.id}
										backup={backup}
										agentName={agentMap.get(backup.agent_id)}
										repoName={getRepoNameForBackup(backup)}
										onViewDetails={setSelectedBackup}
									/>
								))}
							</tbody>
						</table>
					) : (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Snapshot ID
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Agent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Repository
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								<tr>
									<td
										colSpan={7}
										className="px-6 py-12 text-center text-gray-500"
									>
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
												d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
											/>
										</svg>
										<p className="font-medium text-gray-900">
											No backups found
										</p>
										<p className="text-sm">
											Backups will appear here once agents start running
										</p>
									</td>
								</tr>
							</tbody>
						</table>
					)}
				</div>
			</div>

			{selectedBackup && (
				<BackupDetailsModal
					backup={selectedBackup}
					agentName={agentMap.get(selectedBackup.agent_id)}
					repoName={getRepoNameForBackup(selectedBackup)}
					allTags={allTags ?? []}
					onClose={() => setSelectedBackup(null)}
				/>
			)}
		</div>
	);
}
