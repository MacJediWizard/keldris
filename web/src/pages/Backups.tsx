import { useState } from 'react';
import { ClassificationBadge } from '../components/ClassificationBadge';
import { BackupCalendar } from '../components/features/BackupCalendar';
import { type BulkAction, BulkActions } from '../components/ui/BulkActions';
import {
	BulkOperationProgress,
	useBulkOperation,
} from '../components/ui/BulkOperationProgress';
import {
	BulkSelectCheckbox,
	BulkSelectHeader,
	BulkSelectToolbar,
} from '../components/ui/BulkSelect';
import { HelpTooltip } from '../components/ui/HelpTooltip';
import { useAgents } from '../hooks/useAgents';
import { useBackups } from '../hooks/useBackups';
import { useBulkSelect } from '../hooks/useBulkSelect';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import { useBackupTags, useSetBackupTags, useTags } from '../hooks/useTags';
import { statusHelp } from '../lib/help-content';
import type {
	Backup,
	BackupStatus,
	ClassificationLevel,
	Tag,
} from '../lib/types';
import type { Backup, BackupStatus, Tag } from '../lib/types';
import {
	formatBytes,
	formatDate,
	formatDateTime,
	formatDuration,
	getBackupStatusColor,
	truncateSnapshotId,
} from '../lib/utils';

type ViewMode = 'list' | 'calendar';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4 w-12">
				<div className="h-4 w-4 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
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
				<div className="h-6 w-20 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-4 w-12 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
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
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
			<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Backup Details
					</h3>
					<span className="inline-flex items-center gap-1.5">
						<span
							className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
						>
							<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
							{backup.status}
						</span>
						<HelpTooltip
							content={
								backup.status === 'completed'
									? statusHelp.backupCompleted.content
									: backup.status === 'running'
										? statusHelp.backupRunning.content
										: backup.status === 'failed'
											? statusHelp.backupFailed.content
											: statusHelp.backupCanceled.content
							}
							title={
								backup.status === 'completed'
									? statusHelp.backupCompleted.title
									: backup.status === 'running'
										? statusHelp.backupRunning.title
										: backup.status === 'failed'
											? statusHelp.backupFailed.title
											: statusHelp.backupCanceled.title
							}
							position="left"
						/>
					</span>
				</div>

				<div className="space-y-4">
					{backup.resumed && (
						<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
							<div className="flex items-center gap-2">
								<svg
									className="w-5 h-5 text-blue-500"
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
								<p className="text-sm font-medium text-blue-700 dark:text-blue-400">
									Resumed Backup
								</p>
							</div>
							<p className="text-sm text-blue-600 dark:text-blue-300 mt-1">
								This backup was resumed from an interrupted backup.
								{backup.original_backup_id && (
									<span className="ml-1">
										Original backup ID:{' '}
										<code className="font-mono text-xs">
											{backup.original_backup_id.slice(0, 8)}...
										</code>
									</span>
								)}
							</p>
						</div>
					)}

					{backup.snapshot_id && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Snapshot ID
							</p>
							<p className="font-mono text-gray-900 dark:text-white">
								{backup.snapshot_id}
							</p>
						</div>
					)}

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Agent
							</p>
							<p className="text-gray-900 dark:text-white">
								{agentName ?? 'Unknown'}
							</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Repository
							</p>
							<p className="text-gray-900 dark:text-white">
								{repoName ?? 'Unknown'}
							</p>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Started
							</p>
							<p className="text-gray-900 dark:text-white">
								{formatDateTime(backup.started_at)}
							</p>
						</div>
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Duration
							</p>
							<p className="text-gray-900 dark:text-white">
								{formatDuration(backup.started_at, backup.completed_at)}
							</p>
						</div>
					</div>

					{backup.status === 'completed' && (
						<div className="grid grid-cols-3 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Size
								</p>
								<p className="text-gray-900 dark:text-white">
									{formatBytes(backup.size_bytes)}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									New Files
								</p>
								<p className="text-gray-900 dark:text-white">
									{backup.files_new ?? 0}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Changed
								</p>
								<p className="text-gray-900 dark:text-white">
									{backup.files_changed ?? 0}
								</p>
							</div>
						</div>
					)}

					{(backup.classification_level ||
						backup.classification_data_types) && (
						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Classification
							</p>
							<ClassificationBadge
								level={backup.classification_level || 'public'}
								dataTypes={backup.classification_data_types}
								showDataTypes
								size="md"
							/>
						</div>
					)}

					<BackupTagsEditor backupId={backup.id} allTags={allTags} />

					{backup.error_message && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Error
							</p>
							<p className="text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/30 p-3 rounded-lg text-sm">
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

					{(backup.container_pre_hook_output ||
						backup.container_pre_hook_error) && (
						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Container Pre-Backup Hook
							</p>
							{backup.container_pre_hook_output && (
								<div className="mb-2">
									<p className="text-xs font-medium text-gray-400 mb-1">
										Output
									</p>
									<pre className="bg-gray-100 p-3 rounded-lg text-xs text-gray-800 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.container_pre_hook_output}
									</pre>
								</div>
							)}
							{backup.container_pre_hook_error && (
								<div>
									<p className="text-xs font-medium text-red-400 mb-1">Error</p>
									<pre className="bg-red-50 p-3 rounded-lg text-xs text-red-700 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.container_pre_hook_error}
									</pre>
								</div>
							)}
						</div>
					)}

					{(backup.container_post_hook_output ||
						backup.container_post_hook_error) && (
						<div>
							<p className="text-sm font-medium text-gray-500 mb-2">
								Container Post-Backup Hook
							</p>
							{backup.container_post_hook_output && (
								<div className="mb-2">
									<p className="text-xs font-medium text-gray-400 mb-1">
										Output
									</p>
									<pre className="bg-gray-100 p-3 rounded-lg text-xs text-gray-800 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.container_post_hook_output}
									</pre>
								</div>
							)}
							{backup.container_post_hook_error && (
								<div>
									<p className="text-xs font-medium text-red-400 mb-1">Error</p>
									<pre className="bg-red-50 p-3 rounded-lg text-xs text-red-700 overflow-x-auto max-h-32 whitespace-pre-wrap">
										{backup.container_post_hook_error}
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
						className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
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
	isSelected: boolean;
	onToggleSelect: () => void;
}

function BackupRow({
	backup,
	agentName,
	repoName,
	onViewDetails,
	isSelected,
	onToggleSelect,
}: BackupRowProps) {
	const statusColor = getBackupStatusColor(backup.status);

	return (
		<tr
			className={`hover:bg-gray-50 dark:hover:bg-gray-700 ${isSelected ? 'bg-indigo-50 dark:bg-indigo-900/20' : ''}`}
		>
			<td className="px-6 py-4 w-12">
				<BulkSelectCheckbox checked={isSelected} onChange={onToggleSelect} />
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<code className="text-sm font-mono text-gray-900 dark:text-white">
						{truncateSnapshotId(backup.snapshot_id)}
					</code>
					{backup.resumed && (
						<span
							className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
							title="This backup was resumed from an interrupted backup"
						>
							Resumed
						</span>
					)}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
				{agentName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
				{repoName ?? 'Unknown'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatBytes(backup.size_bytes)}
			</td>
			<td className="px-6 py-4">
				<div className="flex flex-col gap-1">
					<span className="inline-flex items-center gap-1.5">
						<span
							className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
						>
							<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
							{backup.status}
						</span>
						<HelpTooltip
							content={
								backup.status === 'completed'
									? statusHelp.backupCompleted.content
									: backup.status === 'running'
										? statusHelp.backupRunning.content
										: backup.status === 'failed'
											? statusHelp.backupFailed.content
											: statusHelp.backupCanceled.content
							}
							title={
								backup.status === 'completed'
									? statusHelp.backupCompleted.title
									: backup.status === 'running'
										? statusHelp.backupRunning.title
										: backup.status === 'failed'
											? statusHelp.backupFailed.title
											: statusHelp.backupCanceled.title
							}
							position="right"
						/>
					</span>
					{backup.classification_level &&
						backup.classification_level !== 'public' && (
							<ClassificationBadge
								level={backup.classification_level}
								size="sm"
							/>
						)}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(backup.started_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onViewDetails(backup)}
					className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
				>
					Details
				</button>
			</td>
		</tr>
	);
}

export function Backups() {
	const [viewMode, setViewMode] = useState<ViewMode>('list');
	const [searchQuery, setSearchQuery] = useState('');
	const [agentFilter, setAgentFilter] = useState<string>('all');
	const [statusFilter, setStatusFilter] = useState<BackupStatus | 'all'>('all');
	const [classificationFilter, setClassificationFilter] = useState<
		ClassificationLevel | 'all'
	>('all');
	const [selectedTagFilters, setSelectedTagFilters] = useState<Set<string>>(
		new Set(),
	);
	const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null);
	const [showAddTagModal, setShowAddTagModal] = useState(false);
	const [selectedTagId, setSelectedTagId] = useState('');

	const { data: backups, isLoading, isError } = useBackups();
	const { data: agents } = useAgents();
	const { data: schedules } = useSchedules();
	const { data: repositories } = useRepositories();
	const { data: allTags } = useTags();
	const setBackupTags = useSetBackupTags();

	const bulkOperation = useBulkOperation();

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
		const matchesClassification =
			classificationFilter === 'all' ||
			(backup.classification_level || 'public') === classificationFilter;
		// Note: Tag filtering would require loading backup tags for each backup,
		// which is expensive. For a more complete implementation, you'd want to
		// fetch this data on the server side with proper filtering.
		return (
			matchesSearch && matchesAgent && matchesStatus && matchesClassification
		);
		// Note: Tag filtering would require loading backup tags for each backup,
		// which is expensive. For a more complete implementation, you'd want to
		// fetch this data on the server side with proper filtering.
		return matchesSearch && matchesAgent && matchesStatus;
	});

	const backupIds = filteredBackups?.map((b) => b.id) ?? [];
	const bulkSelect = useBulkSelect(backupIds);

	const bulkActions: BulkAction[] = [
		{
			id: 'add-tag',
			label: 'Add Tag',
			icon: (
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
						d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z"
					/>
				</svg>
			),
		},
	];

	const handleBulkAction = (actionId: string) => {
		switch (actionId) {
			case 'add-tag':
				setShowAddTagModal(true);
				break;
		}
	};

	const handleBulkAddTag = async () => {
		if (!selectedTagId) return;
		setShowAddTagModal(false);

		await bulkOperation.start(
			[...bulkSelect.selectedIds],
			async (backupId: string) => {
				// Get current tags and add the new one
				await setBackupTags.mutateAsync({
					backupId,
					data: { tag_ids: [selectedTagId] },
				});
			},
		);
		bulkSelect.clear();
		setSelectedTagId('');
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Backups
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						View and manage backup snapshots
					</p>
				</div>
				<div className="flex items-center gap-2 bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
					<button
						type="button"
						onClick={() => setViewMode('list')}
						className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
							viewMode === 'list'
								? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
								: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
						}`}
					>
						<span className="flex items-center gap-1.5">
							<svg
								className="w-4 h-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M4 6h16M4 10h16M4 14h16M4 18h16"
								/>
							</svg>
							List
						</span>
					</button>
					<button
						type="button"
						onClick={() => setViewMode('calendar')}
						className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
							viewMode === 'calendar'
								? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
								: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
						}`}
					>
						<span className="flex items-center gap-1.5">
							<svg
								className="w-4 h-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
								/>
							</svg>
							Calendar
						</span>
					</button>
				</div>
			</div>

			{/* Bulk Selection Toolbar - only show in list view */}
			{viewMode === 'list' && bulkSelect.selectedCount > 0 && (
				<BulkSelectToolbar
					selectedCount={bulkSelect.selectedCount}
					totalCount={backupIds.length}
					onSelectAll={() => bulkSelect.selectAll(backupIds)}
					onDeselectAll={bulkSelect.deselectAll}
					itemLabel="backup"
				>
					<BulkActions
						actions={bulkActions}
						onAction={handleBulkAction}
						label="Actions"
					/>
				</BulkSelectToolbar>
			)}
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

			{viewMode === 'calendar' ? (
				<BackupCalendar onSelectBackup={setSelectedBackup} />
			) : (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="p-6 border-b border-gray-200 dark:border-gray-700">
						<div className="flex items-center gap-4 mb-4">
							<input
								type="text"
								placeholder="Search by snapshot ID..."
								value={searchQuery}
								onChange={(e) => setSearchQuery(e.target.value)}
								className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
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
							<select
								value={statusFilter}
								onChange={(e) =>
									setStatusFilter(e.target.value as BackupStatus | 'all')
								}
								className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="all">All Status</option>
								<option value="completed">Completed</option>
								<option value="running">Running</option>
								<option value="failed">Failed</option>
								<option value="canceled">Canceled</option>
							</select>
							<select
								value={classificationFilter}
								onChange={(e) =>
									setClassificationFilter(
										e.target.value as ClassificationLevel | 'all',
									)
								}
								className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="all">All Classifications</option>
								<option value="public">Public</option>
								<option value="internal">Internal</option>
								<option value="confidential">Confidential</option>
								<option value="restricted">Restricted</option>
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
							<div className="p-12 text-center text-red-500 dark:text-red-400">
								<p className="font-medium">Failed to load backups</p>
								<p className="text-sm">Please try refreshing the page</p>
							</div>
						) : isLoading ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 w-12" />
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
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created
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
									<LoadingRow />
									<LoadingRow />
								</tbody>
							</table>
						) : filteredBackups && filteredBackups.length > 0 ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 w-12">
											<BulkSelectHeader
												isAllSelected={bulkSelect.isAllSelected}
												isPartiallySelected={bulkSelect.isPartiallySelected}
												onToggleAll={() => bulkSelect.toggleAll(backupIds)}
												selectedCount={bulkSelect.selectedCount}
												totalCount={backupIds.length}
											/>
										</th>
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
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{filteredBackups.map((backup) => (
										<BackupRow
											key={backup.id}
											backup={backup}
											agentName={agentMap.get(backup.agent_id)}
											repoName={getRepoNameForBackup(backup)}
											onViewDetails={setSelectedBackup}
											isSelected={bulkSelect.isSelected(backup.id)}
											onToggleSelect={() => bulkSelect.toggle(backup.id)}
										/>
									))}
								</tbody>
							</table>
						) : (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 w-12" />
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
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									<tr>
										<td
											colSpan={8}
											className="px-6 py-12 text-center text-gray-500 dark:text-gray-400"
										>
											<svg
												aria-hidden="true"
												className="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600"
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
											<p className="font-medium text-gray-900 dark:text-white">
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
			)}

			{selectedBackup && (
				<BackupDetailsModal
					backup={selectedBackup}
					agentName={agentMap.get(selectedBackup.agent_id)}
					repoName={getRepoNameForBackup(selectedBackup)}
					allTags={allTags ?? []}
					onClose={() => setSelectedBackup(null)}
				/>
			)}

			{/* Add Tag Modal */}
			{showAddTagModal && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Add Tag
						</h3>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							Add a tag to {bulkSelect.selectedCount} backup
							{bulkSelect.selectedCount !== 1 ? 's' : ''}.
						</p>
						<div className="mb-4">
							<label
								htmlFor="bulk-tag-select"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Tag
							</label>
							<select
								id="bulk-tag-select"
								value={selectedTagId}
								onChange={(e) => setSelectedTagId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="">Select a tag</option>
								{allTags?.map((tag) => (
									<option key={tag.id} value={tag.id}>
										{tag.name}
									</option>
								))}
							</select>
						</div>
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => {
									setShowAddTagModal(false);
									setSelectedTagId('');
								}}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleBulkAddTag}
								disabled={!selectedTagId}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								Add Tag
							</button>
						</div>
					</div>
				</div>
			)}

			{/* Bulk Operation Progress */}
			<BulkOperationProgress
				isOpen={bulkOperation.isRunning || bulkOperation.isComplete}
				onClose={bulkOperation.reset}
				title="Bulk Operation"
				total={bulkOperation.total}
				completed={bulkOperation.completed}
				results={bulkOperation.results}
				isComplete={bulkOperation.isComplete}
			/>
		</div>
	);
}

export default Backups;
