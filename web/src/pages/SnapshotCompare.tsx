import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAgents } from '../hooks/useAgents';
import { useRepositories } from '../hooks/useRepositories';
import { useSnapshotCompare, useSnapshots } from '../hooks/useSnapshots';
import type {
	Snapshot,
	SnapshotDiffChangeType,
	SnapshotDiffEntry,
} from '../lib/types';
import { formatBytes, formatDate, formatDateTime } from '../lib/utils';

function getChangeTypeColor(changeType: SnapshotDiffChangeType): {
	bg: string;
	text: string;
	icon: string;
} {
	switch (changeType) {
		case 'added':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				icon: 'text-green-600',
			};
		case 'removed':
			return { bg: 'bg-red-100', text: 'text-red-800', icon: 'text-red-600' };
		case 'modified':
			return {
				bg: 'bg-blue-100',
				text: 'text-blue-800',
				icon: 'text-blue-600',
			};
		default:
			return {
				bg: 'bg-gray-100',
				text: 'text-gray-800',
				icon: 'text-gray-600',
			};
	}
}

function ChangeIcon({ changeType }: { changeType: SnapshotDiffChangeType }) {
	const color = getChangeTypeColor(changeType);

	if (changeType === 'added') {
		return (
			<svg
				aria-hidden="true"
				className={`w-4 h-4 ${color.icon}`}
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M12 4v16m8-8H4"
				/>
			</svg>
		);
	}
	if (changeType === 'removed') {
		return (
			<svg
				aria-hidden="true"
				className={`w-4 h-4 ${color.icon}`}
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M20 12H4"
				/>
			</svg>
		);
	}
	return (
		<svg
			aria-hidden="true"
			className={`w-4 h-4 ${color.icon}`}
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
			/>
		</svg>
	);
}

interface SnapshotSelectorProps {
	id: string;
	label: string;
	value: string;
	onChange: (id: string) => void;
	snapshots: Snapshot[];
	agentMap: Map<string, string>;
	excludeId?: string;
}

function SnapshotSelector({
	id,
	label,
	value,
	onChange,
	snapshots,
	agentMap,
	excludeId,
}: SnapshotSelectorProps) {
	const filteredSnapshots = excludeId
		? snapshots.filter((s) => s.id !== excludeId)
		: snapshots;

	return (
		<div>
			<label
				htmlFor={id}
				className="block text-sm font-medium text-gray-700 mb-1"
			>
				{label}
			</label>
			<select
				id={id}
				value={value}
				onChange={(e) => onChange(e.target.value)}
				className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
			>
				<option value="">Select a snapshot...</option>
				{filteredSnapshots.map((snapshot) => (
					<option key={snapshot.id} value={snapshot.id}>
						{snapshot.short_id} - {agentMap.get(snapshot.agent_id) ?? 'Unknown'}{' '}
						- {formatDate(snapshot.time)}
					</option>
				))}
			</select>
		</div>
	);
}

interface DiffEntryRowProps {
	entry: SnapshotDiffEntry;
	snapshot1Id: string;
	snapshot2Id: string;
}

function DiffEntryRow({ entry, snapshot1Id, snapshot2Id }: DiffEntryRowProps) {
	const color = getChangeTypeColor(entry.change_type);
	const isFile = entry.type === 'file';

	const diffUrl = isFile
		? `/snapshots/file-diff?snapshot1=${snapshot1Id}&snapshot2=${snapshot2Id}&path=${encodeURIComponent(entry.path)}`
		: undefined;

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-3">
				<div className="flex items-center gap-2">
					<ChangeIcon changeType={entry.change_type} />
					<span
						className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${color.bg} ${color.text}`}
					>
						{entry.change_type}
					</span>
				</div>
			</td>
			<td className="px-6 py-3">
				<div className="flex items-center gap-2">
					{entry.type === 'dir' ? (
						<svg
							aria-hidden="true"
							className="w-4 h-4 text-yellow-500"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
						</svg>
					) : (
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
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
					)}
					{diffUrl ? (
						<Link
							to={diffUrl}
							className="font-mono text-sm text-indigo-600 hover:text-indigo-800 hover:underline break-all"
						>
							{entry.path}
						</Link>
					) : (
						<span className="font-mono text-sm text-gray-900 break-all">
							{entry.path}
						</span>
					)}
				</div>
			</td>
			<td className="px-6 py-3 text-sm text-gray-500">
				{entry.old_size !== undefined && entry.old_size > 0
					? formatBytes(entry.old_size)
					: '-'}
			</td>
			<td className="px-6 py-3 text-sm text-gray-500">
				{entry.new_size !== undefined && entry.new_size > 0
					? formatBytes(entry.new_size)
					: '-'}
			</td>
			<td className="px-6 py-3 text-sm">
				{entry.size_change !== undefined && entry.size_change !== 0 ? (
					<span
						className={
							entry.size_change > 0 ? 'text-green-600' : 'text-red-600'
						}
					>
						{entry.size_change > 0 ? '+' : ''}
						{formatBytes(Math.abs(entry.size_change))}
					</span>
				) : (
					'-'
				)}
			</td>
			<td className="px-6 py-3 text-sm">
				{diffUrl && (
					<Link
						to={diffUrl}
						className="inline-flex items-center gap-1 text-indigo-600 hover:text-indigo-800"
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
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/>
						</svg>
						View Diff
					</Link>
				)}
			</td>
		</tr>
	);
}

export function SnapshotCompare() {
	const [searchParams, setSearchParams] = useSearchParams();
	const [agentFilter, setAgentFilter] = useState<string>('all');
	const [repoFilter, setRepoFilter] = useState<string>('all');
	const [changeTypeFilter, setChangeTypeFilter] = useState<string>('all');

	const snapshot1Id = searchParams.get('snapshot1') ?? '';
	const snapshot2Id = searchParams.get('snapshot2') ?? '';

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const { data: snapshots, isLoading: snapshotsLoading } = useSnapshots({
		agent_id: agentFilter !== 'all' ? agentFilter : undefined,
		repository_id: repoFilter !== 'all' ? repoFilter : undefined,
	});

	const {
		data: compareResult,
		isLoading: compareLoading,
		isError: compareError,
	} = useSnapshotCompare(snapshot1Id, snapshot2Id);

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));

	const setSnapshot1 = (id: string) => {
		const newParams = new URLSearchParams(searchParams);
		if (id) {
			newParams.set('snapshot1', id);
		} else {
			newParams.delete('snapshot1');
		}
		setSearchParams(newParams);
	};

	const setSnapshot2 = (id: string) => {
		const newParams = new URLSearchParams(searchParams);
		if (id) {
			newParams.set('snapshot2', id);
		} else {
			newParams.delete('snapshot2');
		}
		setSearchParams(newParams);
	};

	const filteredChanges =
		compareResult?.changes.filter(
			(change) =>
				changeTypeFilter === 'all' || change.change_type === changeTypeFilter,
		) ?? [];

	const canCompare = snapshot1Id && snapshot2Id && snapshot1Id !== snapshot2Id;

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">Compare Snapshots</h1>
				<p className="text-gray-600 mt-1">
					Select two snapshots to see what changed between them
				</p>
			</div>

			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<h2 className="text-lg font-semibold text-gray-900 mb-4">
					Select Snapshots
				</h2>

				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
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
						value={repoFilter}
						onChange={(e) => setRepoFilter(e.target.value)}
						className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="all">All Repositories</option>
						{repositories?.map((repo) => (
							<option key={repo.id} value={repo.id}>
								{repo.name}
							</option>
						))}
					</select>
				</div>

				{snapshotsLoading ? (
					<div className="text-center py-8 text-gray-500">
						Loading snapshots...
					</div>
				) : snapshots && snapshots.length > 0 ? (
					<div className="grid grid-cols-1 md:grid-cols-2 gap-6">
						<SnapshotSelector
							id="snapshot1-select"
							label="First Snapshot (older)"
							value={snapshot1Id}
							onChange={setSnapshot1}
							snapshots={snapshots}
							agentMap={agentMap}
							excludeId={snapshot2Id}
						/>
						<SnapshotSelector
							id="snapshot2-select"
							label="Second Snapshot (newer)"
							value={snapshot2Id}
							onChange={setSnapshot2}
							snapshots={snapshots}
							agentMap={agentMap}
							excludeId={snapshot1Id}
						/>
					</div>
				) : (
					<div className="text-center py-8 text-gray-500">
						No snapshots available. Snapshots will appear here once backups
						complete.
					</div>
				)}
			</div>

			{canCompare && compareLoading && (
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
					<p className="text-gray-600">Comparing snapshots...</p>
				</div>
			)}

			{canCompare && !compareLoading && compareError && (
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
						Failed to compare snapshots
					</p>
					<p className="text-gray-500 text-sm mt-1">
						Please check that both snapshots exist and try again
					</p>
				</div>
			)}

			{canCompare && !compareLoading && !compareError && compareResult && (
				<>
					<div className="grid grid-cols-1 md:grid-cols-2 gap-6">
						{compareResult.snapshot_1 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<div className="flex items-center justify-between mb-2">
									<h3 className="font-medium text-gray-900">First Snapshot</h3>
									<code className="text-sm font-mono bg-gray-100 px-2 py-1 rounded">
										{compareResult.snapshot_1.short_id}
									</code>
								</div>
								<div className="text-sm text-gray-600 space-y-1">
									<p>
										<span className="text-gray-500">Agent:</span>{' '}
										{agentMap.get(compareResult.snapshot_1.agent_id) ??
											'Unknown'}
									</p>
									<p>
										<span className="text-gray-500">Date:</span>{' '}
										{formatDateTime(compareResult.snapshot_1.time)}
									</p>
									<p>
										<span className="text-gray-500">Size:</span>{' '}
										{formatBytes(compareResult.snapshot_1.size_bytes)}
									</p>
								</div>
							</div>
						)}
						{compareResult.snapshot_2 && (
							<div className="bg-white rounded-lg border border-gray-200 p-4">
								<div className="flex items-center justify-between mb-2">
									<h3 className="font-medium text-gray-900">Second Snapshot</h3>
									<code className="text-sm font-mono bg-gray-100 px-2 py-1 rounded">
										{compareResult.snapshot_2.short_id}
									</code>
								</div>
								<div className="text-sm text-gray-600 space-y-1">
									<p>
										<span className="text-gray-500">Agent:</span>{' '}
										{agentMap.get(compareResult.snapshot_2.agent_id) ??
											'Unknown'}
									</p>
									<p>
										<span className="text-gray-500">Date:</span>{' '}
										{formatDateTime(compareResult.snapshot_2.time)}
									</p>
									<p>
										<span className="text-gray-500">Size:</span>{' '}
										{formatBytes(compareResult.snapshot_2.size_bytes)}
									</p>
								</div>
							</div>
						)}
					</div>

					<div className="bg-white rounded-lg border border-gray-200 p-6">
						<h2 className="text-lg font-semibold text-gray-900 mb-4">
							Summary
						</h2>
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
							<div className="bg-green-50 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-green-600">
									{compareResult.stats.files_added}
								</div>
								<div className="text-sm text-green-700">Files Added</div>
							</div>
							<div className="bg-red-50 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-red-600">
									{compareResult.stats.files_removed}
								</div>
								<div className="text-sm text-red-700">Files Removed</div>
							</div>
							<div className="bg-blue-50 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-blue-600">
									{compareResult.stats.files_modified}
								</div>
								<div className="text-sm text-blue-700">Files Modified</div>
							</div>
							<div className="bg-gray-50 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-gray-600">
									{compareResult.stats.dirs_added +
										compareResult.stats.dirs_removed}
								</div>
								<div className="text-sm text-gray-700">Dirs Changed</div>
							</div>
						</div>
						<div className="mt-4 grid grid-cols-2 gap-4 text-sm">
							<div className="flex justify-between py-2 border-t border-gray-200">
								<span className="text-gray-600">Total Size Added:</span>
								<span className="text-green-600 font-medium">
									+{formatBytes(compareResult.stats.total_size_added)}
								</span>
							</div>
							<div className="flex justify-between py-2 border-t border-gray-200">
								<span className="text-gray-600">Total Size Removed:</span>
								<span className="text-red-600 font-medium">
									-{formatBytes(compareResult.stats.total_size_removed)}
								</span>
							</div>
						</div>
					</div>

					<div className="bg-white rounded-lg border border-gray-200">
						<div className="p-4 border-b border-gray-200 flex items-center justify-between">
							<h2 className="text-lg font-semibold text-gray-900">
								Changed Files
							</h2>
							<select
								value={changeTypeFilter}
								onChange={(e) => setChangeTypeFilter(e.target.value)}
								className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="all">All Changes</option>
								<option value="added">Added</option>
								<option value="removed">Removed</option>
								<option value="modified">Modified</option>
							</select>
						</div>
						{filteredChanges.length > 0 ? (
							<div className="overflow-x-auto">
								<table className="w-full">
									<thead className="bg-gray-50 border-b border-gray-200">
										<tr>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-28">
												Change
											</th>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
												Path
											</th>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
												Old Size
											</th>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
												New Size
											</th>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
												Difference
											</th>
											<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
												Actions
											</th>
										</tr>
									</thead>
									<tbody className="divide-y divide-gray-200">
										{filteredChanges.map((entry, index) => (
											<DiffEntryRow
												key={`${entry.path}-${index}`}
												entry={entry}
												snapshot1Id={snapshot1Id}
												snapshot2Id={snapshot2Id}
											/>
										))}
									</tbody>
								</table>
							</div>
						) : (
							<div className="p-12 text-center text-gray-500">
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
										d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
									/>
								</svg>
								<p className="font-medium text-gray-900">
									{changeTypeFilter === 'all'
										? 'No changes found'
										: `No ${changeTypeFilter} files`}
								</p>
								<p className="text-sm">
									{changeTypeFilter === 'all'
										? 'The snapshots appear to be identical'
										: 'Try selecting a different filter'}
								</p>
							</div>
						)}
					</div>
				</>
			)}

			{!canCompare && snapshot1Id && snapshot2Id && (
				<div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-center">
					<p className="text-yellow-800">
						Please select two different snapshots to compare
					</p>
				</div>
			)}
		</div>
	);
}
