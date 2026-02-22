import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { SnapshotComments } from '../components/features/SnapshotComments';
import { useAgents } from '../hooks/useAgents';
import { useMe } from '../hooks/useAuth';
import {
	useCreateLegalHold,
	useDeleteLegalHold,
	useLegalHolds,
} from '../hooks/useLegalHolds';
import { useRepositories } from '../hooks/useRepositories';
import {
	useCloudRestoreProgress,
	useCreateCloudRestore,
	useCreateRestore,
	useRestorePreview,
	useRestores,
} from '../hooks/useRestore';
import { useSnapshotComments } from '../hooks/useSnapshotComments';
import { useSnapshotFiles, useSnapshots } from '../hooks/useSnapshots';
import type {
	Agent,
	CloudRestoreProgress,
	CloudRestoreTarget,
	CloudRestoreTargetType,
	RestorePreview,
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
		case 'uploading':
			return {
				bg: 'bg-purple-100',
				text: 'text-purple-800',
				dot: 'bg-purple-500',
			};
		case 'verifying':
			return {
				bg: 'bg-cyan-100',
				text: 'text-cyan-800',
				dot: 'bg-cyan-500',
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

function CommentIndicator({ snapshotId }: { snapshotId: string }) {
	const { data: comments } = useSnapshotComments(snapshotId);
	const count = comments?.length ?? 0;

	if (count === 0) return null;

	return (
		<span
			className="inline-flex items-center gap-1 ml-2 text-gray-400"
			title={`${count} note${count !== 1 ? 's' : ''}`}
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
					d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
				/>
			</svg>
			<span className="text-xs">{count}</span>
		</span>
	);
}

function HoldIndicator({ hasHold }: { hasHold: boolean }) {
	if (!hasHold) return null;

	return (
		<span
			className="inline-flex items-center ml-2 text-amber-500"
			title="Legal hold - cannot be deleted"
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
					d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
				/>
			</svg>
		</span>
	);
}

interface SnapshotRowProps {
	snapshot: Snapshot;
	agentName?: string;
	repoName?: string;
	onSelect: (snapshot: Snapshot) => void;
	isSelectedForCompare: boolean;
	onToggleCompare: (snapshotId: string) => void;
	compareSelectionCount: number;
	hasHold: boolean;
	isAdmin: boolean;
	onToggleHold: (snapshotId: string, hasHold: boolean) => void;
	isHoldLoading: boolean;
}

function SnapshotRow({
	snapshot,
	agentName,
	repoName,
	onSelect,
	isSelectedForCompare,
	onToggleCompare,
	compareSelectionCount,
	hasHold,
	isAdmin,
	onToggleHold,
	isHoldLoading,
}: SnapshotRowProps) {
	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<input
					type="checkbox"
					checked={isSelectedForCompare}
					onChange={() => onToggleCompare(snapshot.id)}
					disabled={!isSelectedForCompare && compareSelectionCount >= 2}
					className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
					title={
						!isSelectedForCompare && compareSelectionCount >= 2
							? 'Deselect a snapshot to select another'
							: 'Select for comparison'
					}
				/>
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center">
					<code className="text-sm font-mono text-gray-900">
						{snapshot.short_id}
					</code>
					<HoldIndicator hasHold={hasHold} />
					<CommentIndicator snapshotId={snapshot.id} />
				</div>
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
				<div className="flex items-center justify-end gap-2">
					{isAdmin && (
						<button
							type="button"
							onClick={() => onToggleHold(snapshot.id, hasHold)}
							disabled={isHoldLoading}
							className={`p-1.5 rounded transition-colors ${
								hasHold
									? 'text-amber-600 hover:bg-amber-50'
									: 'text-gray-400 hover:bg-gray-100 hover:text-gray-600'
							} disabled:opacity-50`}
							title={hasHold ? 'Remove legal hold' : 'Place legal hold'}
						>
							<svg
								aria-hidden="true"
								className="w-4 h-4"
								fill={hasHold ? 'currentColor' : 'none'}
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
								/>
							</svg>
						</button>
					)}
					<button
						type="button"
						onClick={() => onSelect(snapshot)}
						className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
					>
						Restore
					</button>
				</div>
			</td>
		</tr>
	);
}

interface RestoreRowProps {
	restore: RestoreType;
	agentName?: string;
	sourceAgentName?: string;
	onViewDetails: (restore: RestoreType) => void;
}

function RestoreRow({
	restore,
	agentName,
	sourceAgentName,
	onViewDetails,
}: RestoreRowProps) {
	const statusColor = getRestoreStatusColor(restore.status);

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<code className="text-sm font-mono text-gray-900">
					{restore.snapshot_id.substring(0, 8)}
				</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-900">
				<div className="flex items-center gap-1">
					{agentName ?? 'Unknown'}
					{restore.is_cross_agent && (
						<span
							className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800"
							title={`Cross-agent restore from ${sourceAgentName ?? 'unknown'}`}
						>
							<svg
								aria-hidden="true"
								className="w-3 h-3 mr-0.5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
								/>
							</svg>
							cross
						</span>
					)}
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-xs truncate">
				{restore.target_path}
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<span
						className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
					>
						<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
						{restore.status}
					</span>
					{restore.progress && restore.status === 'running' && (
						<span className="text-xs text-gray-500">
							{restore.progress.total_bytes
								? `${Math.round((restore.progress.bytes_restored / restore.progress.total_bytes) * 100)}%`
								: `${formatBytes(restore.progress.bytes_restored)}`}
						</span>
					)}
				</div>
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
	sourceAgentName?: string;
	onClose: () => void;
}

function RestoreDetailsModal({
	restore,
	agentName,
	sourceAgentName,
	onClose,
}: RestoreDetailsModalProps) {
	const statusColor = getRestoreStatusColor(restore.status);

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<div className="flex items-center gap-2">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							Restore Details
						</h3>
						{restore.is_cross_agent && (
							<span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
								<svg
									aria-hidden="true"
									className="w-3 h-3 mr-1"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
									/>
								</svg>
								Cross-Agent
							</span>
						)}
					</div>
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

					{restore.is_cross_agent ? (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Source Agent
								</p>
								<p className="text-gray-900">{sourceAgentName ?? 'Unknown'}</p>
							</div>
							<div>
								<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Target Agent
								</p>
								<p className="text-gray-900">{agentName ?? 'Unknown'}</p>
							</div>
						</div>
					) : (
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
					)}

					{restore.is_cross_agent && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Created
							</p>
							<p className="text-gray-900">
								{formatDateTime(restore.created_at)}
							</p>
						</div>
					)}

					<div>
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Target Path
						</p>
						<p className="font-mono text-gray-900 break-all">
							{restore.target_path}
						</p>
					</div>

					{restore.path_mappings && restore.path_mappings.length > 0 && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
								Path Mappings
							</p>
							<div className="mt-1 space-y-1">
								{restore.path_mappings.map((mapping) => (
									<div
										key={`${mapping.source_path}-${mapping.target_path}`}
										className="flex items-center gap-2 text-sm font-mono"
									>
										<span className="text-gray-600">{mapping.source_path}</span>
										<svg
											aria-hidden="true"
											className="w-4 h-4 text-gray-400"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M13 7l5 5m0 0l-5 5m5-5H6"
											/>
										</svg>
										<span className="text-gray-900">{mapping.target_path}</span>
									</div>
								))}
							</div>
						</div>
					)}

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

					{restore.progress && (
						<div>
							<p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">
								Progress
							</p>
							<div className="bg-gray-50 rounded-lg p-3 space-y-2">
								{restore.progress.total_bytes && (
									<div className="w-full bg-gray-200 rounded-full h-2">
										<div
											className="bg-indigo-600 h-2 rounded-full transition-all"
											style={{
												width: `${Math.min(100, Math.round((restore.progress.bytes_restored / restore.progress.total_bytes) * 100))}%`,
											}}
										/>
									</div>
								)}
								<div className="grid grid-cols-2 gap-2 text-sm">
									<div>
										<span className="text-gray-500">Files:</span>{' '}
										<span className="text-gray-900">
											{restore.progress.files_restored}
											{restore.progress.total_files &&
												` / ${restore.progress.total_files}`}
										</span>
									</div>
									<div>
										<span className="text-gray-500">Bytes:</span>{' '}
										<span className="text-gray-900">
											{formatBytes(restore.progress.bytes_restored)}
											{restore.progress.total_bytes &&
												` / ${formatBytes(restore.progress.total_bytes)}`}
										</span>
									</div>
								</div>
								{restore.progress.current_file && (
									<div className="text-xs text-gray-500 truncate">
										Current: {restore.progress.current_file}
									</div>
								)}
							</div>
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

					{restore.is_cloud_restore && (
						<div className="border-t border-gray-200 pt-4 mt-4">
							<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
								Cloud Restore Details
							</h4>
							<div className="space-y-3">
								{restore.cloud_target && (
									<div>
										<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
											Target Type
										</p>
										<p className="text-gray-900 capitalize">
											{restore.cloud_target.type === 's3'
												? 'Amazon S3'
												: restore.cloud_target.type === 'b2'
													? 'Backblaze B2'
													: 'Restic Repository'}
										</p>
									</div>
								)}
								{restore.cloud_target_location && (
									<div>
										<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
											Target Location
										</p>
										<p className="font-mono text-gray-900 break-all text-sm">
											{restore.cloud_target_location}
										</p>
									</div>
								)}
								{restore.cloud_progress && (
									<div className="grid grid-cols-2 gap-4">
										<div>
											<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
												Files Uploaded
											</p>
											<p className="text-gray-900">
												{restore.cloud_progress.uploaded_files} /{' '}
												{restore.cloud_progress.total_files}
											</p>
										</div>
										<div>
											<p className="text-sm font-medium text-gray-500 dark:text-gray-400">
												Data Uploaded
											</p>
											<p className="text-gray-900">
												{formatBytes(restore.cloud_progress.uploaded_bytes)} /{' '}
												{formatBytes(restore.cloud_progress.total_bytes)}
											</p>
										</div>
									</div>
								)}
								{restore.verify_upload && restore.cloud_progress && (
									<div
										className={`flex items-center gap-2 p-2 rounded ${
											restore.cloud_progress.verified_checksum
												? 'bg-green-50 text-green-700'
												: 'bg-yellow-50 text-yellow-700'
										}`}
									>
										{restore.cloud_progress.verified_checksum ? (
											<>
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
														d="M5 13l4 4L19 7"
													/>
												</svg>
												<span className="text-sm">Integrity verified</span>
											</>
										) : (
											<>
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
														d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
													/>
												</svg>
												<span className="text-sm">Verification pending</span>
											</>
										)}
									</div>
								)}
							</div>
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
	isParentSelected?: boolean;
}

function FileTreeItem({
	file,
	selectedPaths,
	onToggle,
	depth,
	isParentSelected = false,
}: FileTreeItemProps) {
	const isSelected = selectedPaths.has(file.path);
	const isEffectivelySelected = isSelected || isParentSelected;

	return (
		<div
			className={`flex items-center py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors ${
				isEffectivelySelected ? 'bg-indigo-50 dark:bg-indigo-900/20' : ''
			}`}
			style={{ paddingLeft: `${depth * 16 + 8}px` }}
		>
			<input
				type="checkbox"
				checked={isEffectivelySelected}
				onChange={() => onToggle(file.path)}
				disabled={isParentSelected}
				className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
				title={isParentSelected ? 'Included via parent folder' : undefined}
			/>
			<span className="ml-2 flex items-center flex-1 min-w-0">
				{file.type === 'dir' ? (
					<svg
						aria-hidden="true"
						className="w-4 h-4 text-yellow-500 mr-1.5 flex-shrink-0"
						fill="currentColor"
						viewBox="0 0 20 20"
					>
						<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
					</svg>
				) : (
					<svg
						aria-hidden="true"
						className="w-4 h-4 text-gray-400 mr-1.5 flex-shrink-0"
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
				<span className="text-sm text-gray-900 dark:text-gray-100 truncate">
					{file.name}
				</span>
			</span>
			<span className="ml-2 text-xs text-gray-500 dark:text-gray-400 tabular-nums flex-shrink-0">
				{file.type === 'file' ? formatBytes(file.size) : ''}
			</span>
		</div>
	);
}

interface FileBrowserProps {
	files: SnapshotFile[];
	selectedPaths: Set<string>;
	onToggle: (path: string) => void;
	onSelectAll: () => void;
	onClearAll: () => void;
	totalSize: number;
	selectedSize: number;
	selectedCount: number;
}

function FileBrowser({
	files,
	selectedPaths,
	onToggle,
	onSelectAll,
	onClearAll,
	totalSize,
	selectedSize,
	selectedCount,
}: FileBrowserProps) {
	// Check if a path's parent is selected
	const isParentSelected = (path: string): boolean => {
		for (const selectedPath of selectedPaths) {
			if (path !== selectedPath && path.startsWith(`${selectedPath}/`)) {
				return true;
			}
		}
		return false;
	};

	return (
		<div className="border border-gray-200 dark:border-gray-600 rounded-lg overflow-hidden">
			<div className="bg-gray-50 dark:bg-gray-700 px-3 py-2 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
				<div className="flex items-center gap-3">
					<span className="text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wide">
						Files
					</span>
					<span className="text-xs text-gray-500 dark:text-gray-400">
						{files.length} items ({formatBytes(totalSize)})
					</span>
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={onSelectAll}
						className="text-xs text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
					>
						Select All
					</button>
					<span className="text-gray-300 dark:text-gray-500">|</span>
					<button
						type="button"
						onClick={onClearAll}
						className="text-xs text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300"
					>
						Clear
					</button>
				</div>
			</div>
			<div className="max-h-64 overflow-y-auto bg-white dark:bg-gray-800">
				{files.length > 0 ? (
					files.map((file) => (
						<FileTreeItem
							key={file.path}
							file={file}
							selectedPaths={selectedPaths}
							onToggle={onToggle}
							depth={0}
							isParentSelected={isParentSelected(file.path)}
						/>
					))
				) : (
					<div className="p-4 text-center text-sm text-gray-500 dark:text-gray-400">
						No files available for selection
					</div>
				)}
			</div>
			{selectedCount > 0 && (
				<div className="bg-indigo-50 dark:bg-indigo-900/30 px-3 py-2 border-t border-indigo-100 dark:border-indigo-800 flex items-center justify-between">
					<div className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-4 h-4 text-indigo-600 dark:text-indigo-400"
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
						<span className="text-sm font-medium text-indigo-700 dark:text-indigo-300">
							{selectedCount} item{selectedCount !== 1 ? 's' : ''} selected
						</span>
					</div>
					<span className="text-sm text-indigo-600 dark:text-indigo-400 tabular-nums">
						{formatBytes(selectedSize)}
					</span>
				</div>
			)}
		</div>
	);
}

type RestoreStep = 'configure' | 'preview' | 'restoring';

interface RestorePreviewDisplayProps {
	preview: RestorePreview;
}

function RestorePreviewDisplay({ preview }: RestorePreviewDisplayProps) {
	const hasSelectedPaths =
		preview.selected_paths && preview.selected_paths.length > 0;

	return (
		<div className="space-y-4">
			<div className="bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
				<h4 className="font-medium text-blue-900 dark:text-blue-100 mb-2">
					Restore Preview
				</h4>
				<p className="text-sm text-blue-700 dark:text-blue-300">
					{hasSelectedPaths
						? `Restoring ${preview.selected_paths?.length} selected item(s) to ${preview.target_path === '/' ? 'original location' : preview.target_path}`
						: `Restoring all files to ${preview.target_path === '/' ? 'original location' : preview.target_path}`}
				</p>
			</div>

			{hasSelectedPaths && (
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
					<p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-2">
						Selected Paths
					</p>
					<ul className="space-y-1">
						{preview.selected_paths?.map((path) => (
							<li
								key={path}
								className="text-sm font-mono text-gray-700 dark:text-gray-300 flex items-center gap-2"
							>
								<svg
									aria-hidden="true"
									className="w-4 h-4 text-indigo-500"
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
								{path}
							</li>
						))}
					</ul>
				</div>
			)}

			<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
					<p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
						Files
					</p>
					<p className="text-lg font-semibold text-gray-900 dark:text-white">
						{preview.total_files}
					</p>
				</div>
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
					<p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
						Directories
					</p>
					<p className="text-lg font-semibold text-gray-900 dark:text-white">
						{preview.total_dirs}
					</p>
				</div>
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
					<p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
						{hasSelectedPaths ? 'Selected Size' : 'Total Size'}
					</p>
					<p className="text-lg font-semibold text-gray-900 dark:text-white">
						{formatBytes(preview.selected_size ?? preview.total_size)}
					</p>
				</div>
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
					<p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
						Disk Space
					</p>
					<p className="text-lg font-semibold text-gray-900 dark:text-white">
						{formatBytes(preview.disk_space_needed)}
					</p>
				</div>
			</div>

			{preview.conflict_count > 0 && (
				<div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
					<div className="flex items-start gap-3">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5"
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
						<div>
							<h4 className="font-medium text-yellow-800">
								{preview.conflict_count} file
								{preview.conflict_count !== 1 ? 's' : ''} will be overwritten
							</h4>
							<p className="text-sm text-yellow-700 mt-1">
								Existing files at the target location will be replaced with the
								backup versions.
							</p>
						</div>
					</div>
				</div>
			)}

			{preview.files.length > 0 && (
				<div>
					<p className="text-sm font-medium text-gray-700 mb-2">
						Files to restore ({preview.files.length})
					</p>
					<div className="border border-gray-200 rounded-lg max-h-48 overflow-y-auto">
						<table className="w-full text-sm">
							<thead className="bg-gray-50 sticky top-0">
								<tr>
									<th className="text-left px-3 py-2 text-xs font-medium text-gray-500 uppercase">
										Path
									</th>
									<th className="text-left px-3 py-2 text-xs font-medium text-gray-500 uppercase w-20">
										Type
									</th>
									<th className="text-right px-3 py-2 text-xs font-medium text-gray-500 uppercase w-24">
										Size
									</th>
									<th className="text-center px-3 py-2 text-xs font-medium text-gray-500 uppercase w-20">
										Status
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-100">
								{preview.files.map((file) => (
									<tr
										key={file.path}
										className={file.has_conflict ? 'bg-yellow-50' : ''}
									>
										<td
											className="px-3 py-2 font-mono text-xs truncate max-w-xs"
											title={file.path}
										>
											{file.path}
										</td>
										<td className="px-3 py-2 text-xs text-gray-500">
											{file.type}
										</td>
										<td className="px-3 py-2 text-xs text-gray-500 text-right">
											{file.type === 'file' ? formatBytes(file.size) : '-'}
										</td>
										<td className="px-3 py-2 text-center">
											{file.has_conflict ? (
												<span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
													Overwrite
												</span>
											) : (
												<span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
													New
												</span>
											)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</div>
			)}

			{preview.files.length === 0 && (
				<div className="text-center py-8 text-gray-500">
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
					<p className="font-medium">Preview not available</p>
					<p className="text-sm">
						File details will be shown after the restore completes.
					</p>
				</div>
			)}
		</div>
	);
}

interface CloudRestoreProgressDisplayProps {
	progress: CloudRestoreProgress;
	status: RestoreStatus;
}

function CloudRestoreProgressDisplay({
	progress,
	status,
}: CloudRestoreProgressDisplayProps) {
	const statusLabel =
		status === 'uploading'
			? 'Uploading to cloud storage...'
			: status === 'verifying'
				? 'Verifying upload integrity...'
				: status === 'completed'
					? 'Upload completed!'
					: 'Processing...';

	return (
		<div className="space-y-4">
			<div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
				<h4 className="font-medium text-blue-900 mb-2">
					Cloud Restore Progress
				</h4>
				<p className="text-sm text-blue-700">{statusLabel}</p>
			</div>

			<div className="space-y-3">
				<div>
					<div className="flex justify-between text-sm text-gray-600 mb-1">
						<span>Upload Progress</span>
						<span>{progress.percent_complete.toFixed(1)}%</span>
					</div>
					<div className="w-full bg-gray-200 rounded-full h-2.5">
						<div
							className="bg-indigo-600 h-2.5 rounded-full transition-all duration-300"
							style={{ width: `${progress.percent_complete}%` }}
						/>
					</div>
				</div>

				<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
					<div className="bg-gray-50 rounded-lg p-3">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Files Uploaded
						</p>
						<p className="text-lg font-semibold text-gray-900">
							{progress.uploaded_files} / {progress.total_files}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-3">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Data Uploaded
						</p>
						<p className="text-lg font-semibold text-gray-900">
							{formatBytes(progress.uploaded_bytes)} /{' '}
							{formatBytes(progress.total_bytes)}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-3 col-span-2">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Current File
						</p>
						<p className="text-sm font-mono text-gray-900 truncate">
							{progress.current_file || '-'}
						</p>
					</div>
				</div>

				{status === 'completed' && (
					<div
						className={`flex items-center gap-2 p-3 rounded-lg ${
							progress.verified_checksum
								? 'bg-green-50 text-green-800'
								: 'bg-yellow-50 text-yellow-800'
						}`}
					>
						{progress.verified_checksum ? (
							<>
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
										d="M5 13l4 4L19 7"
									/>
								</svg>
								<span className="text-sm font-medium">
									Upload verified successfully
								</span>
							</>
						) : (
							<>
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
										d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
									/>
								</svg>
								<span className="text-sm font-medium">
									Upload completed (verification not requested)
								</span>
							</>
						)}
					</div>
				)}
			</div>
		</div>
	);
}

interface CloudRestoreTargetFormProps {
	target: CloudRestoreTarget;
	onChange: (target: CloudRestoreTarget) => void;
	verifyUpload: boolean;
	onVerifyUploadChange: (verify: boolean) => void;
}

function CloudRestoreTargetForm({
	target,
	onChange,
	verifyUpload,
	onVerifyUploadChange,
}: CloudRestoreTargetFormProps) {
	const handleTypeChange = (type: CloudRestoreTargetType) => {
		onChange({ ...target, type });
	};

	return (
		<div className="space-y-4">
			<div>
				<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
					Cloud Target Type
				</p>
				<div className="grid grid-cols-3 gap-3">
					<button
						type="button"
						onClick={() => handleTypeChange('s3')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 's3'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" />
						</svg>
						<span className="text-sm font-medium">Amazon S3</span>
					</button>
					<button
						type="button"
						onClick={() => handleTypeChange('b2')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 'b2'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14H6v-2h6v2zm4-4H6v-2h10v2zm0-4H6V7h10v2z" />
						</svg>
						<span className="text-sm font-medium">Backblaze B2</span>
					</button>
					<button
						type="button"
						onClick={() => handleTypeChange('restic')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 'restic'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
						</svg>
						<span className="text-sm font-medium">Restic Repo</span>
					</button>
				</div>
			</div>

			{(target.type === 's3' || target.type === 'b2') && (
				<>
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="cloud-bucket"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Bucket
							</label>
							<input
								type="text"
								id="cloud-bucket"
								value={target.bucket || ''}
								onChange={(e) =>
									onChange({ ...target, bucket: e.target.value })
								}
								placeholder="my-backup-bucket"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="cloud-prefix"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Prefix (optional)
							</label>
							<input
								type="text"
								id="cloud-prefix"
								value={target.prefix || ''}
								onChange={(e) =>
									onChange({ ...target, prefix: e.target.value })
								}
								placeholder="restores/2024/"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="cloud-region"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Region
							</label>
							<input
								type="text"
								id="cloud-region"
								value={target.region || ''}
								onChange={(e) =>
									onChange({ ...target, region: e.target.value })
								}
								placeholder={target.type === 'b2' ? 'us-west-002' : 'us-east-1'}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						{target.type === 's3' && (
							<div>
								<label
									htmlFor="cloud-endpoint"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Endpoint (optional)
								</label>
								<input
									type="text"
									id="cloud-endpoint"
									value={target.endpoint || ''}
									onChange={(e) =>
										onChange({ ...target, endpoint: e.target.value })
									}
									placeholder="s3.amazonaws.com"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						)}
					</div>

					{target.type === 's3' ? (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="cloud-access-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Access Key ID
								</label>
								<input
									type="text"
									id="cloud-access-key"
									value={target.access_key_id || ''}
									onChange={(e) =>
										onChange({ ...target, access_key_id: e.target.value })
									}
									placeholder="AKIAIOSFODNN7EXAMPLE"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div>
								<label
									htmlFor="cloud-secret-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Secret Access Key
								</label>
								<input
									type="password"
									id="cloud-secret-key"
									value={target.secret_access_key || ''}
									onChange={(e) =>
										onChange({ ...target, secret_access_key: e.target.value })
									}
									placeholder="••••••••••••••••"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						</div>
					) : (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="cloud-account-id"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Account ID
								</label>
								<input
									type="text"
									id="cloud-account-id"
									value={target.account_id || ''}
									onChange={(e) =>
										onChange({ ...target, account_id: e.target.value })
									}
									placeholder="0001234567890ab"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div>
								<label
									htmlFor="cloud-app-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Application Key
								</label>
								<input
									type="password"
									id="cloud-app-key"
									value={target.application_key || ''}
									onChange={(e) =>
										onChange({ ...target, application_key: e.target.value })
									}
									placeholder="••••••••••••••••"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						</div>
					)}
				</>
			)}

			{target.type === 'restic' && (
				<>
					<div>
						<label
							htmlFor="cloud-repository"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Target Repository
						</label>
						<input
							type="text"
							id="cloud-repository"
							value={target.repository || ''}
							onChange={(e) =>
								onChange({ ...target, repository: e.target.value })
							}
							placeholder="s3:s3.amazonaws.com/bucket-name"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<p className="text-xs text-gray-500 mt-1">
							Enter the restic repository URL (e.g., s3:..., b2:..., sftp:...)
						</p>
					</div>
					<div>
						<label
							htmlFor="cloud-repo-password"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Repository Password
						</label>
						<input
							type="password"
							id="cloud-repo-password"
							value={target.repository_password || ''}
							onChange={(e) =>
								onChange({ ...target, repository_password: e.target.value })
							}
							placeholder="••••••••••••••••"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</>
			)}

			<div className="flex items-center">
				<input
					type="checkbox"
					id="verify-upload"
					checked={verifyUpload}
					onChange={(e) => onVerifyUploadChange(e.target.checked)}
					className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
				/>
				<label
					htmlFor="verify-upload"
					className="ml-2 text-sm text-gray-700 dark:text-gray-300"
				>
					Verify upload integrity (recommended)
				</label>
			</div>
		</div>
	);
}

interface CloudRestoreProgressDisplayProps {
	progress: CloudRestoreProgress;
	status: RestoreStatus;
}

function CloudRestoreProgressDisplay({
	progress,
	status,
}: CloudRestoreProgressDisplayProps) {
	const statusLabel =
		status === 'uploading'
			? 'Uploading to cloud storage...'
			: status === 'verifying'
				? 'Verifying upload integrity...'
				: status === 'completed'
					? 'Upload completed!'
					: 'Processing...';

	return (
		<div className="space-y-4">
			<div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
				<h4 className="font-medium text-blue-900 mb-2">
					Cloud Restore Progress
				</h4>
				<p className="text-sm text-blue-700">{statusLabel}</p>
			</div>

			<div className="space-y-3">
				<div>
					<div className="flex justify-between text-sm text-gray-600 mb-1">
						<span>Upload Progress</span>
						<span>{progress.percent_complete.toFixed(1)}%</span>
					</div>
					<div className="w-full bg-gray-200 rounded-full h-2.5">
						<div
							className="bg-indigo-600 h-2.5 rounded-full transition-all duration-300"
							style={{ width: `${progress.percent_complete}%` }}
						/>
					</div>
				</div>

				<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
					<div className="bg-gray-50 rounded-lg p-3">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Files Uploaded
						</p>
						<p className="text-lg font-semibold text-gray-900">
							{progress.uploaded_files} / {progress.total_files}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-3">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Data Uploaded
						</p>
						<p className="text-lg font-semibold text-gray-900">
							{formatBytes(progress.uploaded_bytes)} /{' '}
							{formatBytes(progress.total_bytes)}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-3 col-span-2">
						<p className="text-xs text-gray-500 uppercase tracking-wide">
							Current File
						</p>
						<p className="text-sm font-mono text-gray-900 truncate">
							{progress.current_file || '-'}
						</p>
					</div>
				</div>

				{status === 'completed' && (
					<div
						className={`flex items-center gap-2 p-3 rounded-lg ${
							progress.verified_checksum
								? 'bg-green-50 text-green-800'
								: 'bg-yellow-50 text-yellow-800'
						}`}
					>
						{progress.verified_checksum ? (
							<>
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
										d="M5 13l4 4L19 7"
									/>
								</svg>
								<span className="text-sm font-medium">
									Upload verified successfully
								</span>
							</>
						) : (
							<>
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
										d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
									/>
								</svg>
								<span className="text-sm font-medium">
									Upload completed (verification not requested)
								</span>
							</>
						)}
					</div>
				)}
			</div>
		</div>
	);
}

interface CloudRestoreTargetFormProps {
	target: CloudRestoreTarget;
	onChange: (target: CloudRestoreTarget) => void;
	verifyUpload: boolean;
	onVerifyUploadChange: (verify: boolean) => void;
}

function CloudRestoreTargetForm({
	target,
	onChange,
	verifyUpload,
	onVerifyUploadChange,
}: CloudRestoreTargetFormProps) {
	const handleTypeChange = (type: CloudRestoreTargetType) => {
		onChange({ ...target, type });
	};

	return (
		<div className="space-y-4">
			<div>
				<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
					Cloud Target Type
				</p>
				<div className="grid grid-cols-3 gap-3">
					<button
						type="button"
						onClick={() => handleTypeChange('s3')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 's3'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" />
						</svg>
						<span className="text-sm font-medium">Amazon S3</span>
					</button>
					<button
						type="button"
						onClick={() => handleTypeChange('b2')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 'b2'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14H6v-2h6v2zm4-4H6v-2h10v2zm0-4H6V7h10v2z" />
						</svg>
						<span className="text-sm font-medium">Backblaze B2</span>
					</button>
					<button
						type="button"
						onClick={() => handleTypeChange('restic')}
						className={`p-3 border rounded-lg text-center transition-colors ${
							target.type === 'restic'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6 mx-auto mb-1"
							fill="currentColor"
							viewBox="0 0 24 24"
						>
							<path d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
						</svg>
						<span className="text-sm font-medium">Restic Repo</span>
					</button>
				</div>
			</div>

			{(target.type === 's3' || target.type === 'b2') && (
				<>
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="cloud-bucket"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Bucket
							</label>
							<input
								type="text"
								id="cloud-bucket"
								value={target.bucket || ''}
								onChange={(e) =>
									onChange({ ...target, bucket: e.target.value })
								}
								placeholder="my-backup-bucket"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="cloud-prefix"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Prefix (optional)
							</label>
							<input
								type="text"
								id="cloud-prefix"
								value={target.prefix || ''}
								onChange={(e) =>
									onChange({ ...target, prefix: e.target.value })
								}
								placeholder="restores/2024/"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="cloud-region"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Region
							</label>
							<input
								type="text"
								id="cloud-region"
								value={target.region || ''}
								onChange={(e) =>
									onChange({ ...target, region: e.target.value })
								}
								placeholder={target.type === 'b2' ? 'us-west-002' : 'us-east-1'}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						{target.type === 's3' && (
							<div>
								<label
									htmlFor="cloud-endpoint"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Endpoint (optional)
								</label>
								<input
									type="text"
									id="cloud-endpoint"
									value={target.endpoint || ''}
									onChange={(e) =>
										onChange({ ...target, endpoint: e.target.value })
									}
									placeholder="s3.amazonaws.com"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						)}
					</div>

					{target.type === 's3' ? (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="cloud-access-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Access Key ID
								</label>
								<input
									type="text"
									id="cloud-access-key"
									value={target.access_key_id || ''}
									onChange={(e) =>
										onChange({ ...target, access_key_id: e.target.value })
									}
									placeholder="AKIAIOSFODNN7EXAMPLE"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div>
								<label
									htmlFor="cloud-secret-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Secret Access Key
								</label>
								<input
									type="password"
									id="cloud-secret-key"
									value={target.secret_access_key || ''}
									onChange={(e) =>
										onChange({ ...target, secret_access_key: e.target.value })
									}
									placeholder="••••••••••••••••"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						</div>
					) : (
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="cloud-account-id"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Account ID
								</label>
								<input
									type="text"
									id="cloud-account-id"
									value={target.account_id || ''}
									onChange={(e) =>
										onChange({ ...target, account_id: e.target.value })
									}
									placeholder="0001234567890ab"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div>
								<label
									htmlFor="cloud-app-key"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Application Key
								</label>
								<input
									type="password"
									id="cloud-app-key"
									value={target.application_key || ''}
									onChange={(e) =>
										onChange({ ...target, application_key: e.target.value })
									}
									placeholder="••••••••••••••••"
									className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						</div>
					)}
				</>
			)}

			{target.type === 'restic' && (
				<>
					<div>
						<label
							htmlFor="cloud-repository"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Target Repository
						</label>
						<input
							type="text"
							id="cloud-repository"
							value={target.repository || ''}
							onChange={(e) =>
								onChange({ ...target, repository: e.target.value })
							}
							placeholder="s3:s3.amazonaws.com/bucket-name"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<p className="text-xs text-gray-500 mt-1">
							Enter the restic repository URL (e.g., s3:..., b2:..., sftp:...)
						</p>
					</div>
					<div>
						<label
							htmlFor="cloud-repo-password"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Repository Password
						</label>
						<input
							type="password"
							id="cloud-repo-password"
							value={target.repository_password || ''}
							onChange={(e) =>
								onChange({ ...target, repository_password: e.target.value })
							}
							placeholder="••••••••••••••••"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</>
			)}

			<div className="flex items-center">
				<input
					type="checkbox"
					id="verify-upload"
					checked={verifyUpload}
					onChange={(e) => onVerifyUploadChange(e.target.checked)}
					className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
				/>
				<label
					htmlFor="verify-upload"
					className="ml-2 text-sm text-gray-700 dark:text-gray-300"
				>
					Verify upload integrity (recommended)
				</label>
			</div>
		</div>
	);
}

type RestoreTargetType = 'agent' | 'cloud';

interface RestoreModalProps {
	snapshot: Snapshot;
	agentName?: string;
	repoName?: string;
	agents?: Agent[];
	onClose: () => void;
	onSubmit: (
		targetPath: string,
		includePaths: string[],
		targetAgentId?: string,
		pathMappings?: Array<{ source_path: string; target_path: string }>,
	) => void;
	onSubmit: (targetPath: string, includePaths: string[]) => void;
	onCloudSubmit: (
		includePaths: string[],
		cloudTarget: CloudRestoreTarget,
		verifyUpload: boolean,
	) => void;
	isSubmitting: boolean;
	cloudRestoreId?: string;
}

function RestoreModal({
	snapshot,
	agentName,
	repoName,
	agents,
	onClose,
	onSubmit,
	onCloudSubmit,
	isSubmitting,
	cloudRestoreId,
}: RestoreModalProps) {
	const [step, setStep] = useState<RestoreStep>('configure');
	const [restoreType, setRestoreType] = useState<RestoreTargetType>('agent');
	const [targetPath, setTargetPath] = useState('');
	const [useOriginalPath, setUseOriginalPath] = useState(true);
	const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
	const [preview, setPreview] = useState<RestorePreview | null>(null);
	const [cloudTarget, setCloudTarget] = useState<CloudRestoreTarget>({
		type: 's3',
	});
	const [verifyUpload, setVerifyUpload] = useState(true);
	const [enableCrossAgent, setEnableCrossAgent] = useState(false);
	const [targetAgentId, setTargetAgentId] = useState(snapshot.agent_id);
	const [pathMappings, setPathMappings] = useState<
		Array<{ id: string; source_path: string; target_path: string }>
	>([]);

	const { data: filesData } = useSnapshotFiles(snapshot.id);
	const previewMutation = useRestorePreview();
	const { data: cloudProgress } = useCloudRestoreProgress(
		cloudRestoreId ?? '',
		!!cloudRestoreId && step === 'restoring',
	);

	const isCrossAgent = enableCrossAgent && targetAgentId !== snapshot.agent_id;

	const togglePath = (path: string) => {
		const newSelected = new Set(selectedPaths);
		if (newSelected.has(path)) {
			newSelected.delete(path);
		} else {
			newSelected.add(path);
		}
		setSelectedPaths(newSelected);
	};

	const addPathMapping = () => {
		setPathMappings([
			...pathMappings,
			{ id: crypto.randomUUID(), source_path: '', target_path: '' },
		]);
	};

	const updatePathMapping = (
		id: string,
		field: 'source_path' | 'target_path',
		value: string,
	) => {
		setPathMappings(
			pathMappings.map((m) => (m.id === id ? { ...m, [field]: value } : m)),
		);
	};

	const removePathMapping = (id: string) => {
		setPathMappings(pathMappings.filter((m) => m.id !== id));
	};

	const handlePreview = async (e: React.FormEvent) => {
		e.preventDefault();
		const finalTargetPath = useOriginalPath ? '/' : targetPath;
		const includePaths = Array.from(selectedPaths);
		const validMappings = pathMappings
			.filter((m) => m.source_path && m.target_path)
			.map(({ source_path, target_path }) => ({ source_path, target_path }));

		try {
			const result = await previewMutation.mutateAsync({
				snapshot_id: snapshot.id,
				agent_id: isCrossAgent ? targetAgentId : snapshot.agent_id,
				source_agent_id: isCrossAgent ? snapshot.agent_id : undefined,
				repository_id: snapshot.repository_id,
				target_path: finalTargetPath,
				include_paths: includePaths.length > 0 ? includePaths : undefined,
				path_mappings: validMappings.length > 0 ? validMappings : undefined,
			});
			setPreview(result);
			setStep('preview');
		} catch {
			// Error is handled by the mutation
		}
	};

	const handleRestore = () => {
		const includePaths = Array.from(selectedPaths);
		const validMappings = pathMappings
			.filter((m) => m.source_path && m.target_path)
			.map(({ source_path, target_path }) => ({ source_path, target_path }));
		setStep('restoring');

		if (restoreType === 'cloud') {
			onCloudSubmit(includePaths, cloudTarget, verifyUpload);
		} else {
			const finalTargetPath = useOriginalPath ? '/' : targetPath;
			onSubmit(
				finalTargetPath,
				includePaths,
				isCrossAgent ? targetAgentId : undefined,
				validMappings.length > 0 ? validMappings : undefined,
			);
			onSubmit(finalTargetPath, includePaths);
		}
	};

	const isCloudTargetValid = () => {
		if (cloudTarget.type === 's3') {
			return (
				!!cloudTarget.bucket &&
				!!cloudTarget.access_key_id &&
				!!cloudTarget.secret_access_key
			);
		}
		if (cloudTarget.type === 'b2') {
			return (
				!!cloudTarget.bucket &&
				!!cloudTarget.account_id &&
				!!cloudTarget.application_key
			);
		}
		if (cloudTarget.type === 'restic') {
			return !!cloudTarget.repository && !!cloudTarget.repository_password;
		}
		return false;
	};

	const handleBack = () => {
		setStep('configure');
		setPreview(null);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center justify-between">
						<div>
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
								{step === 'preview' ? 'Restore Preview' : 'Restore Snapshot'}
							</h3>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Snapshot {snapshot.short_id} from{' '}
								{formatDateTime(snapshot.time)}
							</p>
						</div>
						{step === 'configure' && (
							<div className="flex items-center gap-2 text-sm text-gray-500">
								<span className="w-6 h-6 rounded-full bg-indigo-600 text-white flex items-center justify-center text-xs font-medium">
									1
								</span>
								<span>Configure</span>
								<span className="mx-1">→</span>
								<span className="w-6 h-6 rounded-full bg-gray-200 text-gray-500 flex items-center justify-center text-xs font-medium">
									2
								</span>
								<span>Preview</span>
							</div>
						)}
						{step === 'preview' && (
							<div className="flex items-center gap-2 text-sm text-gray-500">
								<span className="w-6 h-6 rounded-full bg-green-500 text-white flex items-center justify-center text-xs">
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
											d="M5 13l4 4L19 7"
										/>
									</svg>
								</span>
								<span>Configure</span>
								<span className="mx-1">→</span>
								<span className="w-6 h-6 rounded-full bg-indigo-600 text-white flex items-center justify-center text-xs font-medium">
									2
								</span>
								<span>Preview</span>
							</div>
						)}
					</div>
				</div>

				{step === 'configure' && (
					<form
						onSubmit={handlePreview}
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

							{/* Cross-Agent Restore Option */}
							{agents && agents.length > 1 && (
								<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
									<label className="flex items-center">
										<input
											type="checkbox"
											checked={enableCrossAgent}
											onChange={(e) => {
												setEnableCrossAgent(e.target.checked);
												if (!e.target.checked) {
													setTargetAgentId(snapshot.agent_id);
													setPathMappings([]);
												}
											}}
											className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
										/>
										<span className="ml-2 text-sm font-medium text-gray-900 dark:text-white">
											Restore to a different agent
										</span>
									</label>
									{enableCrossAgent && (
										<div className="mt-3 space-y-3">
											<div>
												<label
													htmlFor="target-agent-select"
													className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
												>
													Target Agent
												</label>
												<select
													id="target-agent-select"
													value={targetAgentId}
													onChange={(e) => setTargetAgentId(e.target.value)}
													className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
												>
													{agents.map((agent) => (
														<option key={agent.id} value={agent.id}>
															{agent.hostname}
															{agent.id === snapshot.agent_id
																? ' (original)'
																: ''}
														</option>
													))}
												</select>
											</div>

											{isCrossAgent && (
												<div>
													<div className="flex items-center justify-between mb-2">
														<span className="text-sm font-medium text-gray-700 dark:text-gray-300">
															Path Mappings (optional)
														</span>
														<button
															type="button"
															onClick={addPathMapping}
															className="text-sm text-indigo-600 hover:text-indigo-700"
														>
															+ Add mapping
														</button>
													</div>
													<p className="text-xs text-gray-500 dark:text-gray-400 mb-2">
														Map source paths to different target paths
													</p>
													{pathMappings.map((mapping) => (
														<div
															key={mapping.id}
															className="flex items-center gap-2 mb-2"
														>
															<input
																type="text"
																value={mapping.source_path}
																onChange={(e) =>
																	updatePathMapping(
																		mapping.id,
																		'source_path',
																		e.target.value,
																	)
																}
																placeholder="/source/path"
																className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded text-sm font-mono"
															/>
															<svg
																aria-hidden="true"
																className="w-4 h-4 text-gray-400 flex-shrink-0"
																fill="none"
																stroke="currentColor"
																viewBox="0 0 24 24"
															>
																<path
																	strokeLinecap="round"
																	strokeLinejoin="round"
																	strokeWidth={2}
																	d="M13 7l5 5m0 0l-5 5m5-5H6"
																/>
															</svg>
															<input
																type="text"
																value={mapping.target_path}
																onChange={(e) =>
																	updatePathMapping(
																		mapping.id,
																		'target_path',
																		e.target.value,
																	)
																}
																placeholder="/target/path"
																className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded text-sm font-mono"
															/>
															<button
																type="button"
																onClick={() => removePathMapping(mapping.id)}
																className="text-red-500 hover:text-red-700 p-1"
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
																		d="M6 18L18 6M6 6l12 12"
																	/>
																</svg>
															</button>
														</div>
													))}
												</div>
											)}
										</div>
									)}
							<div>
								<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
									Restore Type
								</p>
								<div className="grid grid-cols-2 gap-3">
									<button
										type="button"
										onClick={() => setRestoreType('agent')}
										className={`p-3 border rounded-lg text-left transition-colors ${
											restoreType === 'agent'
												? 'border-indigo-500 bg-indigo-50 text-indigo-700'
												: 'border-gray-200 hover:border-gray-300'
										}`}
									>
										<div className="flex items-center gap-2">
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
													d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
												/>
											</svg>
											<span className="font-medium">Restore to Agent</span>
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Restore files back to the original agent
										</p>
									</button>
									<button
										type="button"
										onClick={() => setRestoreType('cloud')}
										className={`p-3 border rounded-lg text-left transition-colors ${
											restoreType === 'cloud'
												? 'border-indigo-500 bg-indigo-50 text-indigo-700'
												: 'border-gray-200 hover:border-gray-300'
										}`}
									>
										<div className="flex items-center gap-2">
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
													d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
												/>
											</svg>
											<span className="font-medium">Restore to Cloud</span>
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Upload restored files to S3, B2, or another repo
										</p>
									</button>
								</div>
							)}

							<div>
								<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
									Restore Type
								</p>
								<div className="grid grid-cols-2 gap-3">
									<button
										type="button"
										onClick={() => setRestoreType('agent')}
										className={`p-3 border rounded-lg text-left transition-colors ${
											restoreType === 'agent'
												? 'border-indigo-500 bg-indigo-50 text-indigo-700'
												: 'border-gray-200 hover:border-gray-300'
										}`}
									>
										<div className="flex items-center gap-2">
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
													d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
												/>
											</svg>
											<span className="font-medium">Restore to Agent</span>
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Restore files back to the original agent
										</p>
									</button>
									<button
										type="button"
										onClick={() => setRestoreType('cloud')}
										className={`p-3 border rounded-lg text-left transition-colors ${
											restoreType === 'cloud'
												? 'border-indigo-500 bg-indigo-50 text-indigo-700'
												: 'border-gray-200 hover:border-gray-300'
										}`}
									>
										<div className="flex items-center gap-2">
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
													d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
												/>
											</svg>
											<span className="font-medium">Restore to Cloud</span>
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Upload restored files to S3, B2, or another repo
										</p>
									</button>
								</div>
							</div>

							{restoreType === 'cloud' ? (
								<CloudRestoreTargetForm
									target={cloudTarget}
									onChange={setCloudTarget}
									verifyUpload={verifyUpload}
									onVerifyUploadChange={setVerifyUpload}
								/>
							) : (
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
							)}

							<div>
								<p className="text-sm font-medium text-gray-700 dark:text-gray-300">
									Select files to restore (optional)
								</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 mt-1 mb-2">
									Leave empty to restore all files, or select specific
									files/folders
								</p>
								{filesData?.files && filesData.files.length > 0 ? (
									<FileBrowser
										files={filesData.files}
										selectedPaths={selectedPaths}
										onToggle={togglePath}
										onSelectAll={() => {
											const allPaths = new Set(
												filesData.files.map((f) => f.path),
											);
											setSelectedPaths(allPaths);
										}}
										onClearAll={() => setSelectedPaths(new Set())}
										totalSize={filesData.files.reduce(
											(acc, f) => acc + (f.type === 'file' ? f.size : 0),
											0,
										)}
										selectedSize={filesData.files
											.filter((f) => {
												if (selectedPaths.has(f.path)) return true;
												// Check if parent is selected
												for (const path of selectedPaths) {
													if (f.path.startsWith(`${path}/`)) return true;
												}
												return false;
											})
											.reduce(
												(acc, f) => acc + (f.type === 'file' ? f.size : 0),
												0,
											)}
										selectedCount={selectedPaths.size}
									/>
								) : (
									<div className="border border-gray-200 dark:border-gray-600 rounded-lg p-4 bg-gray-50 dark:bg-gray-700">
										<p className="text-sm text-gray-500 dark:text-gray-400 text-center">
											{filesData?.message ||
												'File listing not available. All files will be restored.'}
										</p>
									</div>
								)}
							</div>

							<div className="border-t border-gray-200 pt-6">
								<SnapshotComments snapshotId={snapshot.id} />
							</div>
						</div>

						<div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
							<button
								type="button"
								onClick={onClose}
								disabled={previewMutation.isPending || isSubmitting}
								className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
							>
								Cancel
							</button>
							{restoreType === 'cloud' ? (
								<button
									type="button"
									onClick={handleRestore}
									disabled={isSubmitting || !isCloudTargetValid()}
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
											Starting Cloud Restore...
										</>
									) : (
										<>
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
													d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
												/>
											</svg>
											Start Cloud Restore
										</>
									)}
								</button>
							) : (
								<button
									type="submit"
									disabled={
										previewMutation.isPending ||
										(!useOriginalPath && !targetPath)
									}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 flex items-center gap-2"
								>
									{previewMutation.isPending ? (
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
											Loading Preview...
										</>
									) : (
										<>
											Preview Restore
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
													d="M9 5l7 7-7 7"
												/>
											</svg>
										</>
									)}
								</button>
							)}
						</div>
					</form>
				)}

				{step === 'restoring' && restoreType === 'cloud' && cloudProgress && (
					<div className="flex flex-col flex-1 overflow-hidden">
						<div className="p-6 overflow-y-auto flex-1">
							<CloudRestoreProgressDisplay
								progress={cloudProgress}
								status={isSubmitting ? 'uploading' : 'completed'}
							/>
						</div>
						<div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end">
							<button
								type="button"
								onClick={onClose}
								disabled={isSubmitting}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								{isSubmitting ? 'Uploading...' : 'Close'}
							</button>
						</div>
					</div>
				)}

				{step === 'preview' && preview && (
					<div className="flex flex-col flex-1 overflow-hidden">
						<div className="p-6 overflow-y-auto flex-1">
							<RestorePreviewDisplay preview={preview} />
						</div>

						<div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-between">
							<button
								type="button"
								onClick={handleBack}
								className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors flex items-center gap-2"
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
										d="M15 19l-7-7 7-7"
									/>
								</svg>
								Back
							</button>
							<div className="flex gap-3">
								<button
									type="button"
									onClick={onClose}
									className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleRestore}
									disabled={isSubmitting}
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
											Starting Restore...
										</>
									) : (
										<>
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
													d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
												/>
											</svg>
											Start Restore
										</>
									)}
								</button>
							</div>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}

export function Restore() {
	const navigate = useNavigate();
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
	const [compareSelection, setCompareSelection] = useState<Set<string>>(
		new Set(),
	);
	const [holdModalSnapshot, setHoldModalSnapshot] = useState<string | null>(
		null,
	);
	const [holdReason, setHoldReason] = useState('');

	const { data: user } = useMe();
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
	const createCloudRestore = useCreateCloudRestore();
	const [cloudRestoreId, setCloudRestoreId] = useState<string | undefined>();

	// Legal holds
	const { data: legalHolds } = useLegalHolds();
	const createLegalHold = useCreateLegalHold();
	const deleteLegalHold = useDeleteLegalHold();

	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	// Build a set of snapshot IDs that have legal holds
	const holdsSet = new Set(legalHolds?.map((h) => h.snapshot_id) ?? []);

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));
	const repoMap = new Map(repositories?.map((r) => [r.id, r.name]));

	const handleRestore = (
		targetPath: string,
		includePaths: string[],
		targetAgentId?: string,
		pathMappings?: Array<{ source_path: string; target_path: string }>,
	) => {
		if (!selectedSnapshot) return;

		const isCrossAgent =
			targetAgentId && targetAgentId !== selectedSnapshot.agent_id;

		createRestore.mutate(
			{
				snapshot_id: selectedSnapshot.id,
				agent_id: isCrossAgent ? targetAgentId : selectedSnapshot.agent_id,
				source_agent_id: isCrossAgent ? selectedSnapshot.agent_id : undefined,
				repository_id: selectedSnapshot.repository_id,
				target_path: targetPath,
				include_paths: includePaths.length > 0 ? includePaths : undefined,
				path_mappings:
					pathMappings && pathMappings.length > 0 ? pathMappings : undefined,
			},
			{
				onSuccess: () => {
					setSelectedSnapshot(null);
					setActiveTab('restores');
				},
			},
		);
	};

	const handleCloudRestore = (
		includePaths: string[],
		cloudTarget: CloudRestoreTarget,
		verifyUpload: boolean,
	) => {
		if (!selectedSnapshot) return;

		createCloudRestore.mutate(
			{
				snapshot_id: selectedSnapshot.id,
				agent_id: selectedSnapshot.agent_id,
				repository_id: selectedSnapshot.repository_id,
				include_paths: includePaths.length > 0 ? includePaths : undefined,
				cloud_target: cloudTarget,
				verify_upload: verifyUpload,
			},
			{
				onSuccess: (restore) => {
					setCloudRestoreId(restore.id);
					setActiveTab('restores');
				},
				onSettled: () => {
					setSelectedSnapshot(null);
					setCloudRestoreId(undefined);
				},
			},
		);
	};

	const handleToggleHold = (snapshotId: string, hasHold: boolean) => {
		if (hasHold) {
			// Remove hold
			deleteLegalHold.mutate(snapshotId);
		} else {
			// Show modal to add hold
			setHoldModalSnapshot(snapshotId);
			setHoldReason('');
		}
	};

	const handleCreateHold = () => {
		if (!holdModalSnapshot || !holdReason.trim()) return;
		createLegalHold.mutate(
			{
				snapshotId: holdModalSnapshot,
				data: { reason: holdReason.trim() },
			},
			{
				onSuccess: () => {
					setHoldModalSnapshot(null);
					setHoldReason('');
				},
			},
		);
	};

	const toggleCompareSelection = (snapshotId: string) => {
		const newSelection = new Set(compareSelection);
		if (newSelection.has(snapshotId)) {
			newSelection.delete(snapshotId);
		} else if (newSelection.size < 2) {
			newSelection.add(snapshotId);
		}
		setCompareSelection(newSelection);
	};

	const handleCompare = () => {
		if (compareSelection.size !== 2) return;
		const [snapshot1, snapshot2] = Array.from(compareSelection);
		navigate(
			`/snapshots/compare?snapshot1=${snapshot1}&snapshot2=${snapshot2}`,
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
				<Link
					to="/file-history"
					className="inline-flex items-center px-4 py-2 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors text-sm font-medium text-gray-700"
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
							d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					File History
				</Link>
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
					<div className="flex items-center justify-between">
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
						{activeTab === 'snapshots' && (
							<div className="flex items-center gap-3">
								{compareSelection.size > 0 && (
									<span className="text-sm text-gray-500 dark:text-gray-400">
										{compareSelection.size} selected
									</span>
								)}
								<button
									type="button"
									onClick={handleCompare}
									disabled={compareSelection.size !== 2}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
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
											d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
										/>
									</svg>
									Compare
								</button>
							</div>
						)}
					</div>
				</div>

				<div className="overflow-x-auto">
					{activeTab === 'snapshots' ? (
						snapshotsError ? (
							<div className="p-12 text-center text-red-500 dark:text-red-400">
								<p className="font-medium">Failed to load snapshots</p>
								<p className="text-sm">Please try refreshing the page</p>
							</div>
						) : snapshotsLoading ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-12">
											<span className="sr-only">Compare</span>
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
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-12">
											<span className="sr-only">Compare</span>
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
											isSelectedForCompare={compareSelection.has(snapshot.id)}
											onToggleCompare={toggleCompareSelection}
											compareSelectionCount={compareSelection.size}
											hasHold={holdsSet.has(snapshot.id)}
											isAdmin={isAdmin}
											onToggleHold={handleToggleHold}
											isHoldLoading={
												createLegalHold.isPending || deleteLegalHold.isPending
											}
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
										sourceAgentName={
											restore.source_agent_id
												? agentMap.get(restore.source_agent_id)
												: undefined
										}
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
					agents={agents}
					onClose={() => {
						setSelectedSnapshot(null);
						setCloudRestoreId(undefined);
					}}
					onSubmit={handleRestore}
					onCloudSubmit={handleCloudRestore}
					isSubmitting={createRestore.isPending || createCloudRestore.isPending}
					cloudRestoreId={cloudRestoreId}
				/>
			)}

			{selectedRestore && (
				<RestoreDetailsModal
					restore={selectedRestore}
					agentName={agentMap.get(selectedRestore.agent_id)}
					sourceAgentName={
						selectedRestore.source_agent_id
							? agentMap.get(selectedRestore.source_agent_id)
							: undefined
					}
					onClose={() => setSelectedRestore(null)}
				/>
			)}

			{/* Legal Hold Modal */}
			{holdModalSnapshot && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<div className="flex items-center gap-3 mb-4">
							<div className="p-2 bg-amber-100 rounded-full">
								<svg
									aria-hidden="true"
									className="w-6 h-6 text-amber-600"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
									/>
								</svg>
							</div>
							<div>
								<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
									Place Legal Hold
								</h3>
								<p className="text-sm text-gray-500 dark:text-gray-400">
									Snapshot {holdModalSnapshot.substring(0, 8)}...
								</p>
							</div>
						</div>

						<p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
							This will prevent the snapshot from being deleted by retention
							policies or manual deletion. All hold actions are recorded in the
							audit log.
						</p>

						<div className="mb-4">
							<label
								htmlFor="hold-reason"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Reason for hold
							</label>
							<textarea
								id="hold-reason"
								value={holdReason}
								onChange={(e) => setHoldReason(e.target.value)}
								placeholder="e.g., Legal discovery request #12345"
								rows={3}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>

						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => {
									setHoldModalSnapshot(null);
									setHoldReason('');
								}}
								className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleCreateHold}
								disabled={!holdReason.trim() || createLegalHold.isPending}
								className="px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50"
							>
								{createLegalHold.isPending ? 'Placing Hold...' : 'Place Hold'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
