import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
	useAgent,
	useAgentBackups,
	useAgentCommands,
	useAgentHealthHistory,
	useAgentSchedules,
	useAgentStats,
	useCancelAgentCommand,
	useCreateAgentCommand,
	useDeleteAgent,
	useRevokeAgentApiKey,
	useRotateAgentApiKey,
	useRunSchedule,
} from '../hooks/useAgents';
import type {
	AgentCommand,
	AgentHealthHistory,
	Backup,
	CommandStatus,
	CommandType,
	Schedule,
} from '../lib/types';
import {
	formatBytes,
	formatDate,
	formatDateTime,
	formatDuration,
	formatPercent,
	formatUptime,
	getAgentStatusColor,
	getBackupStatusColor,
	getHealthStatusColor,
	getHealthStatusLabel,
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

function getCommandStatusColor(status: CommandStatus) {
	switch (status) {
		case 'pending':
			return { bg: 'bg-gray-100', text: 'text-gray-800', dot: 'bg-gray-400' };
		case 'acknowledged':
			return { bg: 'bg-blue-100', text: 'text-blue-800', dot: 'bg-blue-400' };
		case 'running':
			return { bg: 'bg-yellow-100', text: 'text-yellow-800', dot: 'bg-yellow-400' };
		case 'completed':
			return { bg: 'bg-green-100', text: 'text-green-800', dot: 'bg-green-400' };
		case 'failed':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-400' };
		case 'timed_out':
			return { bg: 'bg-orange-100', text: 'text-orange-800', dot: 'bg-orange-400' };
		case 'canceled':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-800', dot: 'bg-gray-400' };
	}
}

function getCommandTypeLabel(type: CommandType) {
	switch (type) {
		case 'backup_now':
			return 'Backup Now';
		case 'update':
			return 'Update';
		case 'restart':
			return 'Restart';
		case 'diagnostics':
			return 'Diagnostics';
		default:
			return type;
	}
}

interface CommandRowProps {
	command: AgentCommand;
	onCancel: (id: string) => void;
	isCanceling: boolean;
}

function CommandRow({ command, onCancel, isCanceling }: CommandRowProps) {
	const statusColor = getCommandStatusColor(command.status);
	const canCancel = ['pending', 'acknowledged', 'running'].includes(command.status);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
				{formatDateTime(command.created_at)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
				{getCommandTypeLabel(command.type)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{command.status}
				</span>
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{command.created_by_name || '-'}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{command.result?.error ? (
					<span className="text-red-600">{command.result.error}</span>
				) : command.result?.output ? (
					<span className="truncate max-w-xs">{command.result.output}</span>
				) : (
					'-'
				)}
			</td>
			<td className="px-6 py-4 text-right">
				{canCancel && (
					<button
						type="button"
						onClick={() => onCancel(command.id)}
						disabled={isCanceling}
						className="text-sm text-red-600 hover:text-red-700 disabled:opacity-50"
					>
						Cancel
					</button>
				)}
			</td>
		</tr>
	);
}

export function AgentDetails() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const [newApiKey, setNewApiKey] = useState<string | null>(null);
	const [activeTab, setActiveTab] = useState<
		'overview' | 'backups' | 'schedules' | 'health' | 'commands'
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
	const { data: healthHistoryResponse, isLoading: healthLoading } =
		useAgentHealthHistory(id ?? '', 50);
	const { data: commandsResponse, isLoading: commandsLoading } =
		useAgentCommands(id ?? '', 50);

	const deleteAgent = useDeleteAgent();
	const rotateApiKey = useRotateAgentApiKey();
	const revokeApiKey = useRevokeAgentApiKey();
	const runSchedule = useRunSchedule();
	const createCommand = useCreateAgentCommand();
	const cancelCommand = useCancelAgentCommand();

	const stats = statsResponse?.stats;
	const backups = backupsResponse?.backups ?? [];
	const schedules = schedulesResponse?.schedules ?? [];
	const healthHistory = healthHistoryResponse?.history ?? [];
	const commands = commandsResponse?.commands ?? [];

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

	const handleSendCommand = (type: CommandType) => {
		const typeLabels: Record<CommandType, string> = {
			backup_now: 'trigger an immediate backup',
			update: 'update the agent',
			restart: 'restart the agent',
			diagnostics: 'run diagnostics',
		};
		if (confirm(`Are you sure you want to ${typeLabels[type]}?`)) {
			createCommand.mutate({
				agentId: id ?? '',
				data: { type },
			});
		}
	};

	const handleCancelCommand = (commandId: string) => {
		if (confirm('Are you sure you want to cancel this command?')) {
			cancelCommand.mutate({
				agentId: id ?? '',
				commandId,
			});
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
	const healthColor = getHealthStatusColor(agent.health_status || 'unknown');

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
							<span
								className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${healthColor.bg} ${healthColor.text}`}
								title={
									agent.health_metrics
										? `CPU: ${agent.health_metrics.cpu_usage?.toFixed(1)}% | Memory: ${agent.health_metrics.memory_usage?.toFixed(1)}% | Disk: ${agent.health_metrics.disk_usage?.toFixed(1)}%`
										: 'No health data'
								}
							>
								<span
									className={`w-1.5 h-1.5 ${healthColor.dot} rounded-full`}
								/>
								{getHealthStatusLabel(agent.health_status || 'unknown')}
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
					{/* Command Buttons */}
					<div className="flex items-center gap-1 mr-2 pr-2 border-r border-gray-200">
						<button
							type="button"
							onClick={() => handleSendCommand('backup_now')}
							disabled={createCommand.isPending || agent.status !== 'active'}
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-indigo-700 bg-indigo-50 border border-indigo-200 rounded-lg hover:bg-indigo-100 transition-colors disabled:opacity-50"
							title="Trigger immediate backup"
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
									d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
								/>
							</svg>
							Backup
						</button>
						<button
							type="button"
							onClick={() => handleSendCommand('diagnostics')}
							disabled={createCommand.isPending || agent.status !== 'active'}
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
							title="Run diagnostics"
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
									d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
								/>
							</svg>
							Diagnostics
						</button>
						<button
							type="button"
							onClick={() => handleSendCommand('restart')}
							disabled={createCommand.isPending || agent.status !== 'active'}
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-orange-700 bg-orange-50 border border-orange-200 rounded-lg hover:bg-orange-100 transition-colors disabled:opacity-50"
							title="Restart agent"
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
							Restart
						</button>
					</div>
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
					<button
						type="button"
						onClick={() => setActiveTab('health')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'health'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Health
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('commands')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'commands'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Commands ({commands.length})
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

			{activeTab === 'health' && (
				<div className="space-y-6">
					{/* Current Health Status */}
					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<div className="flex items-center gap-2 mb-2">
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
									/>
								</svg>
								<p className="text-sm font-medium text-gray-600">CPU Usage</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.cpu_usage ?? 0) >= 95
										? 'text-red-600'
										: (agent.health_metrics?.cpu_usage ?? 0) >= 80
											? 'text-yellow-600'
											: 'text-gray-900'
								}`}
							>
								{formatPercent(agent.health_metrics?.cpu_usage)}
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<div className="flex items-center gap-2 mb-2">
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
									/>
								</svg>
								<p className="text-sm font-medium text-gray-600">
									Memory Usage
								</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.memory_usage ?? 0) >= 95
										? 'text-red-600'
										: (agent.health_metrics?.memory_usage ?? 0) >= 85
											? 'text-yellow-600'
											: 'text-gray-900'
								}`}
							>
								{formatPercent(agent.health_metrics?.memory_usage)}
							</p>
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<div className="flex items-center gap-2 mb-2">
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"
									/>
								</svg>
								<p className="text-sm font-medium text-gray-600">Disk Usage</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.disk_usage ?? 0) >= 90
										? 'text-red-600'
										: (agent.health_metrics?.disk_usage ?? 0) >= 80
											? 'text-yellow-600'
											: 'text-gray-900'
								}`}
							>
								{formatPercent(agent.health_metrics?.disk_usage)}
							</p>
							{agent.health_metrics && (
								<p className="text-sm text-gray-500 mt-1">
									{formatBytes(agent.health_metrics.disk_free_bytes)} free of{' '}
									{formatBytes(agent.health_metrics.disk_total_bytes)}
								</p>
							)}
						</div>
						<div className="bg-white rounded-lg border border-gray-200 p-6">
							<div className="flex items-center gap-2 mb-2">
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-gray-400"
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
								<p className="text-sm font-medium text-gray-600">Uptime</p>
							</div>
							<p className="text-3xl font-bold text-gray-900">
								{formatUptime(agent.health_metrics?.uptime_seconds)}
							</p>
						</div>
					</div>

					{/* Health Issues */}
					{agent.health_metrics?.issues &&
						agent.health_metrics.issues.length > 0 && (
							<div className="bg-white rounded-lg border border-gray-200 p-6">
								<h3 className="text-lg font-semibold text-gray-900 mb-4">
									Health Issues
								</h3>
								<div className="space-y-3">
									{agent.health_metrics.issues.map((issue) => {
										const severityColors =
											issue.severity === 'critical'
												? 'bg-red-50 border-red-200 text-red-800'
												: 'bg-yellow-50 border-yellow-200 text-yellow-800';
										return (
											<div
												key={`${issue.component}-${issue.severity}-${issue.message}`}
												className={`p-4 rounded-lg border ${severityColors}`}
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
															d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
														/>
													</svg>
													<span className="font-medium capitalize">
														{issue.severity}
													</span>
												</div>
												<p className="mt-1">{issue.message}</p>
											</div>
										);
									})}
								</div>
							</div>
						)}

					{/* Restic Info */}
					<div className="bg-white rounded-lg border border-gray-200 p-6">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Restic Information
						</h3>
						<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
							<div>
								<p className="text-sm text-gray-600">Version</p>
								<p className="font-medium text-gray-900">
									{agent.health_metrics?.restic_version || 'Unknown'}
								</p>
							</div>
							<div>
								<p className="text-sm text-gray-600">Available</p>
								<span
									className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
										agent.health_metrics?.restic_available
											? 'bg-green-100 text-green-800'
											: 'bg-red-100 text-red-800'
									}`}
								>
									{agent.health_metrics?.restic_available ? 'Yes' : 'No'}
								</span>
							</div>
							<div>
								<p className="text-sm text-gray-600">Network</p>
								<span
									className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
										agent.health_metrics?.network_up
											? 'bg-green-100 text-green-800'
											: 'bg-red-100 text-red-800'
									}`}
								>
									{agent.health_metrics?.network_up ? 'Online' : 'Offline'}
								</span>
							</div>
						</div>
					</div>

					{/* Health History Chart */}
					<div className="bg-white rounded-lg border border-gray-200 p-6">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Health History
						</h3>
						{healthLoading ? (
							<div className="h-64 flex items-center justify-center">
								<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
							</div>
						) : healthHistory.length > 0 ? (
							<div className="space-y-6">
								{/* Simple bar chart for CPU, Memory, Disk */}
								<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
									{/* CPU History */}
									<div>
										<p className="text-sm font-medium text-gray-600 mb-2">
											CPU Usage
										</p>
										<div className="h-32 flex items-end gap-1">
											{healthHistory
												.slice(-24)
												.reverse()
												.map((h: AgentHealthHistory) => {
													const value = h.cpu_usage ?? 0;
													const color =
														value >= 95
															? 'bg-red-500'
															: value >= 80
																? 'bg-yellow-500'
																: 'bg-green-500';
													return (
														<div
															key={h.id}
															className={`flex-1 ${color} rounded-t transition-all`}
															style={{ height: `${Math.max(value, 2)}%` }}
															title={`${value.toFixed(1)}%`}
														/>
													);
												})}
										</div>
									</div>
									{/* Memory History */}
									<div>
										<p className="text-sm font-medium text-gray-600 mb-2">
											Memory Usage
										</p>
										<div className="h-32 flex items-end gap-1">
											{healthHistory
												.slice(-24)
												.reverse()
												.map((h: AgentHealthHistory) => {
													const value = h.memory_usage ?? 0;
													const color =
														value >= 95
															? 'bg-red-500'
															: value >= 85
																? 'bg-yellow-500'
																: 'bg-blue-500';
													return (
														<div
															key={h.id}
															className={`flex-1 ${color} rounded-t transition-all`}
															style={{ height: `${Math.max(value, 2)}%` }}
															title={`${value.toFixed(1)}%`}
														/>
													);
												})}
										</div>
									</div>
									{/* Disk History */}
									<div>
										<p className="text-sm font-medium text-gray-600 mb-2">
											Disk Usage
										</p>
										<div className="h-32 flex items-end gap-1">
											{healthHistory
												.slice(-24)
												.reverse()
												.map((h: AgentHealthHistory) => {
													const value = h.disk_usage ?? 0;
													const color =
														value >= 90
															? 'bg-red-500'
															: value >= 80
																? 'bg-yellow-500'
																: 'bg-purple-500';
													return (
														<div
															key={h.id}
															className={`flex-1 ${color} rounded-t transition-all`}
															style={{ height: `${Math.max(value, 2)}%` }}
															title={`${value.toFixed(1)}%`}
														/>
													);
												})}
										</div>
									</div>
								</div>

								{/* Health History Table */}
								<div className="overflow-x-auto">
									<table className="w-full text-sm">
										<thead className="bg-gray-50 border-b border-gray-200">
											<tr>
												<th className="px-4 py-2 text-left font-medium text-gray-500">
													Time
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500">
													Status
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500">
													CPU
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500">
													Memory
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500">
													Disk
												</th>
											</tr>
										</thead>
										<tbody className="divide-y divide-gray-200">
											{healthHistory
												.slice(0, 10)
												.map((h: AgentHealthHistory) => {
													const hColor = getHealthStatusColor(h.health_status);
													return (
														<tr key={h.id} className="hover:bg-gray-50">
															<td className="px-4 py-2 text-gray-900">
																{formatDateTime(h.recorded_at)}
															</td>
															<td className="px-4 py-2">
																<span
																	className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${hColor.bg} ${hColor.text}`}
																>
																	<span
																		className={`w-1.5 h-1.5 ${hColor.dot} rounded-full`}
																	/>
																	{getHealthStatusLabel(h.health_status)}
																</span>
															</td>
															<td className="px-4 py-2 text-gray-500">
																{formatPercent(h.cpu_usage)}
															</td>
															<td className="px-4 py-2 text-gray-500">
																{formatPercent(h.memory_usage)}
															</td>
															<td className="px-4 py-2 text-gray-500">
																{formatPercent(h.disk_usage)}
															</td>
														</tr>
													);
												})}
										</tbody>
									</table>
								</div>
							</div>
						) : (
							<div className="h-64 flex items-center justify-center text-gray-500">
								<div className="text-center">
									<svg
										aria-hidden="true"
										className="w-12 h-12 mx-auto mb-4 text-gray-300"
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
									<p>No health history available</p>
									<p className="text-sm mt-1">
										Health metrics will appear after the agent reports data
									</p>
								</div>
							</div>
						)}
					</div>
				</div>
			)}

			{/* Commands Tab */}
			{activeTab === 'commands' && (
				<div className="bg-white rounded-lg border border-gray-200">
					<div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
						<h3 className="font-semibold text-gray-900">Command History</h3>
						<span className="text-sm text-gray-500">
							Showing last 50 commands
						</span>
					</div>
					{commandsLoading ? (
						<div className="p-12 flex items-center justify-center">
							<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
						</div>
					) : commands.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Created
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Type
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Created By
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
											Result
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200">
									{commands.map((cmd: AgentCommand) => (
										<CommandRow
											key={cmd.id}
											command={cmd}
											onCancel={handleCancelCommand}
											isCanceling={cancelCommand.isPending}
										/>
									))}
								</tbody>
							</table>
						</div>
					) : (
						<div className="p-12 text-center text-gray-500">
							<svg
								aria-hidden="true"
								className="w-12 h-12 mx-auto mb-4 text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
								/>
							</svg>
							<p>No commands have been sent to this agent</p>
							<p className="text-sm mt-1">
								Use the command buttons above to send a command
							</p>
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
