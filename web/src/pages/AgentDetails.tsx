import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
	useAgent,
	useAgentBackups,
	useAgentSchedules,
	useAgentStats,
	useDeleteAgent,
	useRevokeAgentApiKey,
	useRotateAgentApiKey,
	useRunSchedule,
} from '../hooks/useAgents';
import type { Backup, Schedule } from '../lib/types';
import {
	formatBytes,
	formatDate,
	formatDateTime,
	formatDuration,
	getAgentStatusColor,
	getBackupStatusColor,
} from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 rounded mb-2" />
			<div className="h-8 w-32 bg-gray-200 rounded mb-1" />
			<div className="h-3 w-20 bg-gray-100 rounded" />
		</div>
	);
}

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
		</tr>
	);
}

interface ApiKeyModalProps {
	apiKey: string;
	onClose: () => void;
}

function ApiKeyModal({ apiKey, onClose }: ApiKeyModalProps) {
	const [copied, setCopied] = useState(false);

	const copyToClipboard = async () => {
		await navigator.clipboard.writeText(apiKey);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-green-100 rounded-full">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-green-600"
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
					</div>
					<h3 className="text-lg font-semibold text-gray-900">
						API Key Regenerated
					</h3>
				</div>
				<p className="text-sm text-gray-600 mb-4">
					Save this API key now. You won't be able to see it again!
				</p>
				<div className="bg-gray-50 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-2">
						<code className="text-sm font-mono text-gray-800 break-all">
							{apiKey}
						</code>
						<button
							type="button"
							onClick={copyToClipboard}
							className="flex-shrink-0 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded transition-colors"
						>
							{copied ? (
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
							) : (
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
										d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
									/>
								</svg>
							)}
						</button>
					</div>
				</div>
				<div className="flex justify-end">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Done
					</button>
				</div>
			</div>
		</div>
	);
}

interface BackupRowProps {
	backup: Backup;
}

function BackupRow({ backup }: BackupRowProps) {
	const statusColor = getBackupStatusColor(backup.status);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
				{formatDateTime(backup.started_at)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{backup.status}
				</span>
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{formatDuration(backup.started_at, backup.completed_at)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{formatBytes(backup.size_bytes)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{backup.files_new !== undefined && backup.files_changed !== undefined
					? `${backup.files_new} new, ${backup.files_changed} changed`
					: '-'}
			</td>
		</tr>
	);
}

interface ScheduleRowProps {
	schedule: Schedule;
	onRun: (id: string) => void;
	isRunning: boolean;
}

function ScheduleRow({ schedule, onRun, isRunning }: ScheduleRowProps) {
	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900">{schedule.name}</div>
				<div className="text-sm text-gray-500">{schedule.cron_expression}</div>
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
						schedule.enabled
							? 'bg-green-100 text-green-800'
							: 'bg-gray-100 text-gray-600'
					}`}
				>
					{schedule.enabled ? 'Enabled' : 'Disabled'}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{schedule.paths.join(', ')}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onRun(schedule.id)}
					disabled={isRunning || !schedule.enabled}
					className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
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
							d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
						/>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					{isRunning ? 'Running...' : 'Run Now'}
				</button>
			</td>
		</tr>
	);
}

export function AgentDetails() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const [newApiKey, setNewApiKey] = useState<string | null>(null);
	const [activeTab, setActiveTab] = useState<
		'overview' | 'backups' | 'schedules'
	>('overview');

	const { data: agent, isLoading: agentLoading } = useAgent(id ?? '');
	const { data: statsResponse, isLoading: statsLoading } = useAgentStats(
		id ?? '',
	);
	const { data: backupsResponse, isLoading: backupsLoading } = useAgentBackups(
		id ?? '',
	);
	const { data: schedulesResponse, isLoading: schedulesLoading } =
		useAgentSchedules(id ?? '');

	const deleteAgent = useDeleteAgent();
	const rotateApiKey = useRotateAgentApiKey();
	const revokeApiKey = useRevokeAgentApiKey();
	const runSchedule = useRunSchedule();

	const stats = statsResponse?.stats;
	const backups = backupsResponse?.backups ?? [];
	const schedules = schedulesResponse?.schedules ?? [];

	const handleDelete = () => {
		if (confirm('Are you sure you want to delete this agent?')) {
			deleteAgent.mutate(id ?? '', {
				onSuccess: () => navigate('/agents'),
			});
		}
	};

	const handleRotateKey = async () => {
		if (
			confirm(
				'Are you sure you want to regenerate this API key? The old key will be invalidated immediately.',
			)
		) {
			try {
				const result = await rotateApiKey.mutateAsync(id ?? '');
				setNewApiKey(result.api_key);
			} catch {
				// Error handled by mutation
			}
		}
	};

	const handleRevokeKey = () => {
		if (
			confirm(
				'Are you sure you want to revoke this API key? The agent will no longer be able to authenticate.',
			)
		) {
			revokeApiKey.mutate(id ?? '');
		}
	};

	const handleRunSchedule = (scheduleId: string) => {
		if (confirm('Run this backup schedule now?')) {
			runSchedule.mutate(scheduleId);
		}
	};

	if (agentLoading) {
		return (
			<div className="space-y-6">
				<div className="animate-pulse">
					<div className="h-8 w-48 bg-gray-200 rounded mb-2" />
					<div className="h-4 w-32 bg-gray-100 rounded" />
				</div>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
					<LoadingCard />
					<LoadingCard />
					<LoadingCard />
					<LoadingCard />
				</div>
			</div>
		);
	}

	if (!agent) {
		return (
			<div className="text-center py-12">
				<h2 className="text-xl font-semibold text-gray-900">Agent not found</h2>
				<p className="text-gray-500 mt-2">
					The agent you're looking for doesn't exist.
				</p>
				<Link
					to="/agents"
					className="mt-4 inline-flex items-center text-indigo-600 hover:text-indigo-700"
				>
					Back to Agents
				</Link>
			</div>
		);
	}

	const statusColor = getAgentStatusColor(agent.status);

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-start justify-between">
				<div className="flex items-center gap-4">
					<Link
						to="/agents"
						className="text-gray-500 hover:text-gray-700 transition-colors"
					>
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
								d="M10 19l-7-7m0 0l7-7m-7 7h18"
							/>
						</svg>
					</Link>
					<div>
						<div className="flex items-center gap-3">
							<h1 className="text-2xl font-bold text-gray-900">
								{agent.hostname}
							</h1>
							<span
								className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
							>
								<span
									className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`}
								/>
								{agent.status}
							</span>
						</div>
						<p className="text-gray-600 mt-1">
							{agent.os_info
								? `${agent.os_info.os} ${agent.os_info.arch}${agent.os_info.version ? ` (${agent.os_info.version})` : ''}`
								: 'OS information not available'}
						</p>
					</div>
				</div>

				{/* Actions */}
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={handleRotateKey}
						disabled={rotateApiKey.isPending}
						className="inline-flex items-center gap-2 px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
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
								d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
							/>
						</svg>
						{rotateApiKey.isPending ? 'Rotating...' : 'Regenerate Key'}
					</button>
					<button
						type="button"
						onClick={handleRevokeKey}
						disabled={revokeApiKey.isPending || agent.status === 'pending'}
						className="inline-flex items-center gap-2 px-4 py-2 text-yellow-700 bg-yellow-50 border border-yellow-200 rounded-lg hover:bg-yellow-100 transition-colors disabled:opacity-50"
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
								d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
							/>
						</svg>
						{revokeApiKey.isPending ? 'Revoking...' : 'Revoke Key'}
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={deleteAgent.isPending}
						className="inline-flex items-center gap-2 px-4 py-2 text-red-700 bg-red-50 border border-red-200 rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
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
								d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
							/>
						</svg>
						{deleteAgent.isPending ? 'Deleting...' : 'Delete Agent'}
					</button>
				</div>
			</div>

			{/* Stats Cards */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				{statsLoading ? (
					<>
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
					</>
				) : (
					<>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">Total Backups</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{stats?.total_backups ?? 0}
							</p>
							<p className="text-sm text-gray-500 mt-1">
								{stats?.successful_backups ?? 0} successful,{' '}
								{stats?.failed_backups ?? 0} failed
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">Success Rate</p>
							<p
								className={`text-3xl font-bold mt-1 ${
									(stats?.success_rate ?? 0) >= 90
										? 'text-green-600'
										: (stats?.success_rate ?? 0) >= 70
											? 'text-yellow-600'
											: 'text-red-600'
								}`}
							>
								{stats?.success_rate?.toFixed(1) ?? '0'}%
							</p>
							<p className="text-sm text-gray-500 mt-1">
								{stats?.schedule_count ?? 0} active schedules
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">
								Total Backup Size
							</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{formatBytes(stats?.total_size_bytes)}
							</p>
							<p className="text-sm text-gray-500 mt-1">Across all backups</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<p className="text-sm font-medium text-gray-600">Last Backup</p>
							<p className="text-3xl font-bold text-gray-900 mt-1">
								{formatDate(stats?.last_backup_at)}
							</p>
							<p className="text-sm text-gray-500 mt-1">
								Last seen: {formatDate(agent.last_seen)}
							</p>
						</div>
					</>
				)}
			</div>

			{/* Agent Info Card */}
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<h2 className="text-lg font-semibold text-gray-900 mb-4">
					Agent Information
				</h2>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					<div>
						<p className="text-sm text-gray-600">Hostname</p>
						<p className="font-medium text-gray-900">{agent.hostname}</p>
					</div>
					<div>
						<p className="text-sm text-gray-600">Operating System</p>
						<p className="font-medium text-gray-900">
							{agent.os_info?.os ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600">Architecture</p>
						<p className="font-medium text-gray-900">
							{agent.os_info?.arch ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600">OS Version</p>
						<p className="font-medium text-gray-900">
							{agent.os_info?.version ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600">Registered</p>
						<p className="font-medium text-gray-900">
							{formatDateTime(agent.created_at)}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600">Last Seen</p>
						<p className="font-medium text-gray-900">
							{formatDateTime(agent.last_seen)}
						</p>
					</div>
				</div>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('overview')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'overview'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Overview
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('backups')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'backups'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Backup History ({backups.length})
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('schedules')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'schedules'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Schedules ({schedules.length})
					</button>
				</nav>
			</div>

			{/* Tab Content */}
			{activeTab === 'overview' && (
				<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
					{/* Recent Backups */}
					<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
						<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
							<h3 className="text-lg font-semibold text-gray-900">
								Recent Backups
							</h3>
							<button
								type="button"
								onClick={() => setActiveTab('backups')}
								className="text-sm text-indigo-600 hover:text-indigo-700"
							>
								View all
							</button>
						</div>
						{backupsLoading ? (
							<div className="p-6">
								<div className="animate-pulse space-y-4">
									{[1, 2, 3].map((i) => (
										<div key={i} className="h-12 bg-gray-100 rounded" />
									))}
								</div>
							</div>
						) : backups.length > 0 ? (
							<div className="divide-y divide-gray-200">
								{backups.slice(0, 5).map((backup) => {
									const statusColor = getBackupStatusColor(backup.status);
									return (
										<div
											key={backup.id}
											className="px-6 py-4 flex items-center justify-between"
										>
											<div>
												<p className="text-sm font-medium text-gray-900">
													{formatDateTime(backup.started_at)}
												</p>
												<p className="text-sm text-gray-500">
													{formatBytes(backup.size_bytes)} -{' '}
													{formatDuration(
														backup.started_at,
														backup.completed_at,
													)}
												</p>
											</div>
											<span
												className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
											>
												<span
													className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`}
												/>
												{backup.status}
											</span>
										</div>
									);
								})}
							</div>
						) : (
							<div className="p-12 text-center text-gray-500">
								<p>No backups yet</p>
							</div>
						)}
					</div>

					{/* Schedules */}
					<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
						<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
							<h3 className="text-lg font-semibold text-gray-900">
								Active Schedules
							</h3>
							<button
								type="button"
								onClick={() => setActiveTab('schedules')}
								className="text-sm text-indigo-600 hover:text-indigo-700"
							>
								View all
							</button>
						</div>
						{schedulesLoading ? (
							<div className="p-6">
								<div className="animate-pulse space-y-4">
									{[1, 2, 3].map((i) => (
										<div key={i} className="h-12 bg-gray-100 rounded" />
									))}
								</div>
							</div>
						) : schedules.length > 0 ? (
							<div className="divide-y divide-gray-200">
								{schedules
									.filter((s) => s.enabled)
									.slice(0, 5)
									.map((schedule) => (
										<div
											key={schedule.id}
											className="px-6 py-4 flex items-center justify-between"
										>
											<div>
												<p className="text-sm font-medium text-gray-900">
													{schedule.name}
												</p>
												<p className="text-sm text-gray-500">
													{schedule.cron_expression}
												</p>
											</div>
											<button
												type="button"
												onClick={() => handleRunSchedule(schedule.id)}
												disabled={runSchedule.isPending}
												className="text-sm text-indigo-600 hover:text-indigo-700 disabled:opacity-50"
											>
												Run Now
											</button>
										</div>
									))}
							</div>
						) : (
							<div className="p-12 text-center text-gray-500">
								<p>No schedules configured</p>
								<Link
									to="/schedules"
									className="mt-2 inline-block text-sm text-indigo-600 hover:text-indigo-700"
								>
									Create a schedule
								</Link>
							</div>
						)}
					</div>
				</div>
			)}

			{activeTab === 'backups' && (
				<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
					<div className="px-6 py-4 border-b border-gray-200">
						<h3 className="text-lg font-semibold text-gray-900">
							Backup History
						</h3>
					</div>
					{backupsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Started
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Duration
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Files
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : backups.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 border-b border-gray-200">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Started
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Duration
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Size
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Files
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200">
									{backups.map((backup) => (
										<BackupRow key={backup.id} backup={backup} />
									))}
								</tbody>
							</table>
						</div>
					) : (
						<div className="p-12 text-center text-gray-500">
							<p>No backup history</p>
						</div>
					)}
				</div>
			)}

			{activeTab === 'schedules' && (
				<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
					<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
						<h3 className="text-lg font-semibold text-gray-900">
							Backup Schedules
						</h3>
						<Link
							to="/schedules"
							className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 transition-colors"
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
									d="M12 4v16m8-8H4"
								/>
							</svg>
							Add Schedule
						</Link>
					</div>
					{schedulesLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Name
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Paths
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : schedules.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 border-b border-gray-200">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Name
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Paths
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200">
									{schedules.map((schedule) => (
										<ScheduleRow
											key={schedule.id}
											schedule={schedule}
											onRun={handleRunSchedule}
											isRunning={runSchedule.isPending}
										/>
									))}
								</tbody>
							</table>
						</div>
					) : (
						<div className="p-12 text-center text-gray-500">
							<svg
								aria-hidden="true"
								className="w-16 h-16 mx-auto mb-4 text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
								/>
							</svg>
							<h3 className="text-lg font-medium text-gray-900 mb-2">
								No schedules configured
							</h3>
							<p className="mb-6">
								Create a backup schedule to automate backups for this agent
							</p>
							<Link
								to="/schedules"
								className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
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
										d="M12 4v16m8-8H4"
									/>
								</svg>
								Create Schedule
							</Link>
						</div>
					)}
				</div>
			)}

			{/* API Key Modal */}
			{newApiKey && (
				<ApiKeyModal apiKey={newApiKey} onClose={() => setNewApiKey(null)} />
			)}
		</div>
	);
}
