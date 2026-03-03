import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { AgentLogViewer } from '../components/features/AgentLogViewer';
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
	useSetAgentDebugMode,
} from '../hooks/useAgents';
import {
	useAgentConcurrency,
	useUpdateAgentConcurrency,
} from '../hooks/useBackupQueue';
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
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse">
			<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
			<div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-1" />
			<div className="h-3 w-20 bg-gray-100 dark:bg-gray-700 rounded" />
		</div>
	);
}

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
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
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-full">
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
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						API Key Regenerated
					</h3>
				</div>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					Save this API key now. You won't be able to see it again!
				</p>
				<div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-2">
						<code className="text-sm font-mono text-gray-800 dark:text-gray-200 break-all">
							{apiKey}
						</code>
						<button
							type="button"
							onClick={copyToClipboard}
							className="flex-shrink-0 p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
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
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
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
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
				{formatDuration(backup.started_at, backup.completed_at)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
				{formatBytes(backup.size_bytes)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
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
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">{schedule.name}</div>
				<div className="text-sm text-gray-500 dark:text-gray-400">{schedule.cron_expression}</div>
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
						schedule.enabled
							? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
							: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
					}`}
				>
					{schedule.enabled ? 'Enabled' : 'Disabled'}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
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
			return { bg: 'bg-gray-100 dark:bg-gray-700', text: 'text-gray-800 dark:text-gray-200', dot: 'bg-gray-400' };
		case 'acknowledged':
			return { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-800 dark:text-blue-400', dot: 'bg-blue-400' };
		case 'running':
			return {
				bg: 'bg-yellow-100 dark:bg-yellow-900/30',
				text: 'text-yellow-800 dark:text-yellow-400',
				dot: 'bg-yellow-400',
			};
		case 'completed':
			return {
				bg: 'bg-green-100 dark:bg-green-900/30',
				text: 'text-green-800 dark:text-green-400',
				dot: 'bg-green-400',
			};
		case 'failed':
			return { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-800 dark:text-red-400', dot: 'bg-red-400' };
		case 'timed_out':
			return {
				bg: 'bg-orange-100 dark:bg-orange-900/30',
				text: 'text-orange-800 dark:text-orange-400',
				dot: 'bg-orange-400',
			};
		case 'canceled':
			return { bg: 'bg-gray-100 dark:bg-gray-700', text: 'text-gray-600 dark:text-gray-400', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100 dark:bg-gray-700', text: 'text-gray-800 dark:text-gray-200', dot: 'bg-gray-400' };
	}
}

function getCommandTypeLabel(type: CommandType) {
	switch (type) {
		case 'backup_now':
			return 'Backup Now';
		case 'update':
			return 'Update Agent';
		case 'update_restic':
			return 'Update Restic';
		case 'restart':
			return 'Restart';
		case 'diagnostics':
			return 'Diagnostics';
		case 'uninstall':
			return 'Uninstall';
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
	const canCancel = ['pending', 'acknowledged', 'running'].includes(
		command.status,
	);

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
				{formatDateTime(command.created_at)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
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
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
				{command.created_by_name || '-'}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
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
		'overview' | 'backups' | 'schedules' | 'health' | 'logs' | 'commands'
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
	const setDebugMode = useSetAgentDebugMode();
	const createCommand = useCreateAgentCommand();
	const cancelCommand = useCancelAgentCommand();

	// Concurrency hooks
	const { data: concurrencyData, isLoading: concurrencyLoading } =
		useAgentConcurrency(id ?? '');
	const updateConcurrency = useUpdateAgentConcurrency();
	const [isEditingConcurrency, setIsEditingConcurrency] = useState(false);
	const [concurrencyLimit, setConcurrencyLimit] = useState<string>('');

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

	const handleToggleDebugMode = () => {
		if (!agent) return;

		if (agent.debug_mode) {
			if (confirm('Disable debug mode on this agent?')) {
				setDebugMode.mutate({
					id: agent.id,
					data: { enabled: false },
				});
			}
		} else {
			const hours = prompt(
				'Enable debug mode for how many hours? (default: 4)',
				'4',
			);
			if (hours !== null) {
				const durationHours = Number.parseInt(hours, 10) || 4;
				setDebugMode.mutate({
					id: agent.id,
					data: { enabled: true, duration_hours: durationHours },
				});
			}
		}
	};

	const handleSendCommand = (type: CommandType) => {
		const typeLabels: Record<CommandType, string> = {
			backup_now: 'trigger an immediate backup',
			update: 'update the agent',
			update_restic: 'update the restic binary',
			restart: 'restart the agent',
			diagnostics: 'run diagnostics',
			uninstall: 'uninstall the agent',
		};
		if (confirm(`Are you sure you want to ${typeLabels[type]}?`)) {
			createCommand.mutate({
				agentId: id ?? '',
				data: { type },
			});
		}
	};

	const handleUninstall = (purge: boolean) => {
		const message = purge
			? 'This will completely uninstall the agent and remove all its data, config, and managed restic binary from the host. This action cannot be undone.\n\nAre you sure?'
			: 'This will uninstall the agent service and binary from the host, but keep configuration files.\n\nAre you sure?';
		if (confirm(message)) {
			createCommand.mutate({
				agentId: id ?? '',
				data: { type: 'uninstall', payload: { purge } },
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
					<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-4 w-32 bg-gray-100 dark:bg-gray-700 rounded" />
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
				<h2 className="text-xl font-semibold text-gray-900 dark:text-white">Agent not found</h2>
				<p className="text-gray-500 dark:text-gray-400 mt-2">
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
						className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
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
							<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
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
						<p className="text-gray-600 dark:text-gray-400 mt-1">
							{agent.os_info
								? `${agent.os_info.os} ${agent.os_info.arch}${agent.os_info.version ? ` (${agent.os_info.version})` : ''}`
								: 'OS information not available'}
							{agent.agent_version && (
								<span className="ml-3 text-gray-400">
									Agent {agent.agent_version}
								</span>
							)}
						</p>
					</div>
				</div>

				{/* Actions */}
				<div className="flex items-center gap-2">
					{/* Command Buttons */}
					<div className="flex items-center gap-1 mr-2 pr-2 border-r border-gray-200 dark:border-gray-700">
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
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
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
						<button
							type="button"
							onClick={() => handleSendCommand('update')}
							disabled={createCommand.isPending || agent.status !== 'active'}
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg hover:bg-emerald-100 transition-colors disabled:opacity-50"
							title="Update agent binary"
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
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
								/>
							</svg>
							Update Agent
						</button>
						<button
							type="button"
							onClick={() => handleSendCommand('update_restic')}
							disabled={createCommand.isPending || agent.status !== 'active'}
							className="inline-flex items-center gap-1.5 px-3 py-2 text-sm text-teal-700 bg-teal-50 border border-teal-200 rounded-lg hover:bg-teal-100 transition-colors disabled:opacity-50"
							title="Update restic binary"
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
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
								/>
							</svg>
							Update Restic
						</button>
					</div>
					<button
						type="button"
						onClick={handleToggleDebugMode}
						disabled={setDebugMode.isPending}
						className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg transition-colors disabled:opacity-50 ${
							agent.debug_mode
								? 'text-orange-700 bg-orange-50 border border-orange-200 hover:bg-orange-100 dark:text-orange-400 dark:bg-orange-900/30 dark:border-orange-800 dark:hover:bg-orange-900/50'
								: 'text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700'
						}`}
						title={
							agent.debug_mode
								? `Debug mode active until ${agent.debug_mode_expires_at ? new Date(agent.debug_mode_expires_at).toLocaleString() : 'indefinitely'}`
								: 'Enable verbose logging on this agent'
						}
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
								d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"
							/>
						</svg>
						{setDebugMode.isPending
							? 'Updating...'
							: agent.debug_mode
								? 'Debug On'
								: 'Debug Mode'}
					</button>
					<button
						type="button"
						onClick={handleRotateKey}
						disabled={rotateApiKey.isPending}
						className="inline-flex items-center gap-2 px-4 py-2 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
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

			{/* Debug Mode Warning Banner */}
			{agent.debug_mode && (
				<div className="bg-orange-50 dark:bg-orange-900/30 border border-orange-200 dark:border-orange-800 rounded-lg p-4">
					<div className="flex items-start gap-3">
						<div className="flex-shrink-0">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-orange-600"
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
						</div>
						<div className="flex-1">
							<h3 className="text-sm font-medium text-orange-800 dark:text-orange-400">
								Debug Mode Active
							</h3>
							<p className="text-sm text-orange-700 dark:text-orange-300 mt-1">
								This agent is running in debug mode with verbose logging
								enabled. Detailed restic output and file operations are being
								logged.
								{agent.debug_mode_expires_at && (
									<>
										{' '}
										Debug mode will auto-disable on{' '}
										<strong>
											{new Date(agent.debug_mode_expires_at).toLocaleString()}
										</strong>
										.
									</>
								)}
							</p>
						</div>
						<button
							type="button"
							onClick={handleToggleDebugMode}
							disabled={setDebugMode.isPending}
							className="flex-shrink-0 text-sm font-medium text-orange-700 hover:text-orange-800 disabled:opacity-50"
						>
							{setDebugMode.isPending ? 'Disabling...' : 'Disable'}
						</button>
					</div>
				</div>
			)}

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
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Backups</p>
							<p className="text-3xl font-bold text-gray-900 dark:text-white mt-1">
								{stats?.total_backups ?? 0}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{stats?.successful_backups ?? 0} successful,{' '}
								{stats?.failed_backups ?? 0} failed
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600 dark:text-gray-400">Success Rate</p>
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
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{stats?.schedule_count ?? 0} active schedules
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600 dark:text-gray-400">
								Total Backup Size
							</p>
							<p className="text-3xl font-bold text-gray-900 dark:text-white mt-1">
								{formatBytes(stats?.total_size_bytes)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Across all backups</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							<p className="text-sm font-medium text-gray-600 dark:text-gray-400">Last Backup</p>
							<p className="text-3xl font-bold text-gray-900 dark:text-white mt-1">
								{formatDate(stats?.last_backup_at)}
							</p>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Last seen: {formatDate(agent.last_seen)}
							</p>
						</div>
					</>
				)}
			</div>

			{/* Agent Info Card */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Agent Information
				</h2>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">Hostname</p>
						<p className="font-medium text-gray-900 dark:text-white">{agent.hostname}</p>
					</div>
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">Operating System</p>
						<p className="font-medium text-gray-900 dark:text-white">
							{agent.os_info?.os ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">Architecture</p>
						<p className="font-medium text-gray-900 dark:text-white">
							{agent.os_info?.arch ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">OS Version</p>
						<p className="font-medium text-gray-900 dark:text-white">
							{agent.os_info?.version ?? 'Unknown'}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">Registered</p>
						<p className="font-medium text-gray-900 dark:text-white">
							{formatDateTime(agent.created_at)}
						</p>
					</div>
					<div>
						<p className="text-sm text-gray-600 dark:text-gray-400">Last Seen</p>
						<p className="font-medium text-gray-900 dark:text-white">
							{formatDateTime(agent.last_seen)}
						</p>
					</div>
				</div>
			</div>

			{/* Docker Information */}
			{agent.docker_info && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="flex items-center gap-3 mb-4">
						<div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
							<svg
								className="w-5 h-5 text-blue-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
								/>
							</svg>
						</div>
						<div>
							<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
								Docker Information
							</h2>
							{agent.docker_info.available ? (
								<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400">
									Available
								</span>
							) : (
								<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
									Not Available
								</span>
							)}
						</div>
					</div>

					{agent.docker_info.available ? (
						<>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
								<div>
									<p className="text-sm text-gray-600 dark:text-gray-400">Docker Version</p>
									<p className="font-medium text-gray-900 dark:text-white">
										{agent.docker_info.version ?? 'Unknown'}
									</p>
								</div>
								<div>
									<p className="text-sm text-gray-600 dark:text-gray-400">Containers</p>
									<p className="font-medium text-gray-900 dark:text-white">
										{agent.docker_info.container_count}
										<span className="text-sm text-gray-500 dark:text-gray-400 ml-1">
											({agent.docker_info.running_count} running)
										</span>
									</p>
								</div>
								<div>
									<p className="text-sm text-gray-600 dark:text-gray-400">Volumes</p>
									<p className="font-medium text-gray-900 dark:text-white">
										{agent.docker_info.volume_count}
									</p>
								</div>
								{agent.docker_info.detected_at && (
									<div>
										<p className="text-sm text-gray-600 dark:text-gray-400">Last Detected</p>
										<p className="font-medium text-gray-900 dark:text-white">
											{formatDateTime(agent.docker_info.detected_at)}
										</p>
									</div>
								)}
							</div>

							{/* Containers List */}
							{agent.docker_info.containers &&
								agent.docker_info.containers.length > 0 && (
									<div className="mb-6">
										<h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
											Containers
										</h3>
										<div className="overflow-x-auto">
											<table className="min-w-full text-sm">
												<thead className="bg-gray-50 dark:bg-gray-900">
													<tr>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Name
														</th>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Image
														</th>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Status
														</th>
													</tr>
												</thead>
												<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
													{agent.docker_info.containers.map((container) => (
														<tr key={container.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
															<td className="px-3 py-2 font-medium text-gray-900 dark:text-white">
																{container.name}
															</td>
															<td className="px-3 py-2 text-gray-500 dark:text-gray-400 font-mono text-xs">
																{container.image}
															</td>
															<td className="px-3 py-2">
																<span
																	className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
																		container.state === 'running'
																			? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
																			: container.state === 'paused'
																				? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400'
																				: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
																	}`}
																>
																	{container.state}
																</span>
															</td>
														</tr>
													))}
												</tbody>
											</table>
										</div>
									</div>
								)}

							{/* Volumes List */}
							{agent.docker_info.volumes &&
								agent.docker_info.volumes.length > 0 && (
									<div>
										<h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
											Volumes
										</h3>
										<div className="overflow-x-auto">
											<table className="min-w-full text-sm">
												<thead className="bg-gray-50 dark:bg-gray-900">
													<tr>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Name
														</th>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Driver
														</th>
														<th className="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
															Mount Point
														</th>
													</tr>
												</thead>
												<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
													{agent.docker_info.volumes.map((volume) => (
														<tr key={volume.name} className="hover:bg-gray-50 dark:hover:bg-gray-700">
															<td className="px-3 py-2 font-medium text-gray-900 dark:text-white">
																{volume.name}
															</td>
															<td className="px-3 py-2 text-gray-500 dark:text-gray-400">
																{volume.driver}
															</td>
															<td className="px-3 py-2 text-gray-500 dark:text-gray-400 font-mono text-xs">
																{volume.mountpoint || '-'}
															</td>
														</tr>
													))}
												</tbody>
											</table>
										</div>
									</div>
								)}
						</>
					) : (
						<div className="text-sm text-gray-500 dark:text-gray-400">
							{agent.docker_info.error ? (
								<p>
									<span className="font-medium text-red-600">Error:</span>{' '}
									{agent.docker_info.error}
								</p>
							) : (
								<p>
									Docker is not installed or not running on this agent. Install
									Docker to enable Docker volume backups.
								</p>
							)}
						</div>
					)}
				</div>
			)}

			{/* Backup Concurrency Settings */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Backup Concurrency
					</h2>
					{!isEditingConcurrency && (
						<button
							type="button"
							onClick={() => {
								setConcurrencyLimit(
									concurrencyData?.max_concurrent_backups?.toString() ?? '',
								);
								setIsEditingConcurrency(true);
							}}
							className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
						>
							Edit
						</button>
					)}
				</div>
				{isEditingConcurrency ? (
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							const limit =
								concurrencyLimit === ''
									? null
									: Number.parseInt(concurrencyLimit, 10);
							await updateConcurrency.mutateAsync({
								agentId: id ?? '',
								data: {
									max_concurrent_backups: limit === null ? undefined : limit,
								},
							});
							setIsEditingConcurrency(false);
						}}
						className="space-y-4"
					>
						<div>
							<label
								htmlFor="concurrencyLimit"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Maximum Concurrent Backups
							</label>
							<input
								type="number"
								id="concurrencyLimit"
								value={concurrencyLimit}
								onChange={(e) => setConcurrencyLimit(e.target.value)}
								min="0"
								placeholder="Use organization default"
								className="w-full max-w-xs px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Leave empty to use organization default. When the limit is
								reached, new backups will be queued.
							</p>
						</div>
						{updateConcurrency.isError && (
							<p className="text-sm text-red-600">
								Failed to update concurrency limit. Please try again.
							</p>
						)}
						<div className="flex gap-3">
							<button
								type="submit"
								disabled={updateConcurrency.isPending}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								{updateConcurrency.isPending ? 'Saving...' : 'Save'}
							</button>
							<button
								type="button"
								onClick={() => setIsEditingConcurrency(false)}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
						</div>
					</form>
				) : (
					<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
						<div>
							<p className="text-sm text-gray-600 dark:text-gray-400">Max Concurrent Backups</p>
							<p className="font-medium text-gray-900 dark:text-white">
								{concurrencyLoading ? (
									<span className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded animate-pulse inline-block" />
								) : concurrencyData?.max_concurrent_backups != null ? (
									concurrencyData.max_concurrent_backups
								) : (
									<span className="text-gray-500 dark:text-gray-400">Use org default</span>
								)}
							</p>
						</div>
						<div>
							<p className="text-sm text-gray-600 dark:text-gray-400">Currently Running</p>
							<p className="font-medium text-gray-900 dark:text-white">
								{concurrencyData?.running_count ?? 0}
							</p>
						</div>
						<div>
							<p className="text-sm text-gray-600 dark:text-gray-400">Queued</p>
							<p className="font-medium text-gray-900 dark:text-white">
								{concurrencyData?.queued_count ?? 0}
								{(concurrencyData?.queued_count ?? 0) > 0 && (
									<span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400">
										Waiting
									</span>
								)}
							</p>
						</div>
					</div>
				)}
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('overview')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'overview'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
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
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
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
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
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
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
						}`}
					>
						Health
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('logs')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'logs'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
						}`}
					>
						Logs
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('commands')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'commands'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
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
					<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
						<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
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
										<div key={i} className="h-12 bg-gray-100 dark:bg-gray-700 rounded" />
									))}
								</div>
							</div>
						) : backups.length > 0 ? (
							<div className="divide-y divide-gray-200 dark:divide-gray-700">
								{backups.slice(0, 5).map((backup) => {
									const statusColor = getBackupStatusColor(backup.status);
									return (
										<div
											key={backup.id}
											className="px-6 py-4 flex items-center justify-between"
										>
											<div>
												<p className="text-sm font-medium text-gray-900 dark:text-white">
													{formatDateTime(backup.started_at)}
												</p>
												<p className="text-sm text-gray-500 dark:text-gray-400">
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
							<div className="p-12 text-center text-gray-500 dark:text-gray-400">
								<p>No backups yet</p>
							</div>
						)}
					</div>

					{/* Schedules */}
					<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
						<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
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
										<div key={i} className="h-12 bg-gray-100 dark:bg-gray-700 rounded" />
									))}
								</div>
							</div>
						) : schedules.length > 0 ? (
							<div className="divide-y divide-gray-200 dark:divide-gray-700">
								{schedules
									.filter((s) => s.enabled)
									.slice(0, 5)
									.map((schedule) => (
										<div
											key={schedule.id}
											className="px-6 py-4 flex items-center justify-between"
										>
											<div>
												<p className="text-sm font-medium text-gray-900 dark:text-white">
													{schedule.name}
												</p>
												<p className="text-sm text-gray-500 dark:text-gray-400">
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
							<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							Backup History
						</h3>
					</div>
					{backupsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Started
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Duration
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Files
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : backups.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Started
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Duration
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Size
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Files
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{backups.map((backup) => (
										<BackupRow key={backup.id} backup={backup} />
									))}
								</tbody>
							</table>
						</div>
					) : (
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
							<p>No backup history</p>
						</div>
					)}
				</div>
			)}

			{activeTab === 'schedules' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
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
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Name
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Paths
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : schedules.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Name
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Paths
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
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
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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
							<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
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
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
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
								<p className="text-sm font-medium text-gray-600 dark:text-gray-400">CPU Usage</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.cpu_usage ?? 0) >= 95
										? 'text-red-600'
										: (agent.health_metrics?.cpu_usage ?? 0) >= 80
											? 'text-yellow-600'
											: 'text-gray-900 dark:text-white'
								}`}
							>
								{formatPercent(agent.health_metrics?.cpu_usage)}
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
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
								<p className="text-sm font-medium text-gray-600 dark:text-gray-400">
									Memory Usage
								</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.memory_usage ?? 0) >= 95
										? 'text-red-600'
										: (agent.health_metrics?.memory_usage ?? 0) >= 85
											? 'text-yellow-600'
											: 'text-gray-900 dark:text-white'
								}`}
							>
								{formatPercent(agent.health_metrics?.memory_usage)}
							</p>
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
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
								<p className="text-sm font-medium text-gray-600 dark:text-gray-400">Disk Usage</p>
							</div>
							<p
								className={`text-3xl font-bold ${
									(agent.health_metrics?.disk_usage ?? 0) >= 90
										? 'text-red-600'
										: (agent.health_metrics?.disk_usage ?? 0) >= 80
											? 'text-yellow-600'
											: 'text-gray-900 dark:text-white'
								}`}
							>
								{formatPercent(agent.health_metrics?.disk_usage)}
							</p>
							{agent.health_metrics && (
								<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
									{formatBytes(agent.health_metrics.disk_free_bytes)} free of{' '}
									{formatBytes(agent.health_metrics.disk_total_bytes)}
								</p>
							)}
						</div>
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
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
								<p className="text-sm font-medium text-gray-600 dark:text-gray-400">Uptime</p>
							</div>
							<p className="text-3xl font-bold text-gray-900 dark:text-white">
								{formatUptime(agent.health_metrics?.uptime_seconds)}
							</p>
						</div>
					</div>

					{/* Health Issues */}
					{agent.health_metrics?.issues &&
						agent.health_metrics.issues.length > 0 && (
							<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
								<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
									Health Issues
								</h3>
								<div className="space-y-3">
									{agent.health_metrics.issues.map((issue) => {
										const severityColors =
											issue.severity === 'critical'
												? 'bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-800 dark:text-red-400'
												: 'bg-yellow-50 dark:bg-yellow-900/30 border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-400';
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
					<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Restic Information
						</h3>
						<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
							<div>
								<p className="text-sm text-gray-600 dark:text-gray-400">Version</p>
								<p className="font-medium text-gray-900 dark:text-white">
									{agent.health_metrics?.restic_version || 'Unknown'}
								</p>
							</div>
							<div>
								<p className="text-sm text-gray-600 dark:text-gray-400">Available</p>
								<span
									className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
										agent.health_metrics?.restic_available
											? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
											: 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400'
									}`}
								>
									{agent.health_metrics?.restic_available ? 'Yes' : 'No'}
								</span>
							</div>
							<div>
								<p className="text-sm text-gray-600 dark:text-gray-400">Network</p>
								<span
									className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
										agent.health_metrics?.network_up
											? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
											: 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400'
									}`}
								>
									{agent.health_metrics?.network_up ? 'Online' : 'Offline'}
								</span>
							</div>
						</div>
					</div>

					{/* Health History Chart */}
					<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
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
										<p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-2">
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
										<p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-2">
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
										<p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-2">
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
										<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
											<tr>
												<th className="px-4 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
													Time
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
													Status
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
													CPU
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
													Memory
												</th>
												<th className="px-4 py-2 text-left font-medium text-gray-500 dark:text-gray-400">
													Disk
												</th>
											</tr>
										</thead>
										<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
											{healthHistory
												.slice(0, 10)
												.map((h: AgentHealthHistory) => {
													const hColor = getHealthStatusColor(h.health_status);
													return (
														<tr key={h.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
															<td className="px-4 py-2 text-gray-900 dark:text-white">
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
															<td className="px-4 py-2 text-gray-500 dark:text-gray-400">
																{formatPercent(h.cpu_usage)}
															</td>
															<td className="px-4 py-2 text-gray-500 dark:text-gray-400">
																{formatPercent(h.memory_usage)}
															</td>
															<td className="px-4 py-2 text-gray-500 dark:text-gray-400">
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
							<div className="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
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

			{activeTab === 'logs' && <AgentLogViewer agentId={id ?? ''} />}

			{/* Commands Tab */}
			{activeTab === 'commands' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<h3 className="font-semibold text-gray-900 dark:text-white">Command History</h3>
						<span className="text-sm text-gray-500 dark:text-gray-400">
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
								<thead className="bg-gray-50 dark:bg-gray-900">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Type
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Created By
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Result
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
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
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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

			{/* Uninstall Agent */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-red-200 dark:border-red-800 p-6">
				<h3 className="text-lg font-semibold text-red-900 dark:text-red-400 mb-3">Uninstall Agent</h3>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					Remotely uninstall the agent from <strong>{agent.hostname}</strong>. The agent must be online to receive the uninstall command.
					Deleting the agent from the dashboard only removes it from the database — use these buttons to also remove the binary and service from the host.
				</p>
				<div className="flex items-center gap-3">
					<button
						type="button"
						onClick={() => handleUninstall(false)}
						disabled={createCommand.isPending || agent.status !== 'active'}
						className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-red-700 bg-red-50 border border-red-200 rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50 dark:text-red-400 dark:bg-red-900/30 dark:border-red-800 dark:hover:bg-red-900/50"
					>
						<svg aria-hidden="true" className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
						Uninstall
					</button>
					<button
						type="button"
						onClick={() => handleUninstall(true)}
						disabled={createCommand.isPending || agent.status !== 'active'}
						className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-red-600 border border-red-600 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 dark:bg-red-700 dark:border-red-700 dark:hover:bg-red-800"
					>
						<svg aria-hidden="true" className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
						Uninstall &amp; Purge Data
					</button>
				</div>
				<p className="text-xs text-gray-500 dark:text-gray-400 mt-3">
					<strong>Uninstall</strong> removes the service and binary. <strong>Uninstall &amp; Purge Data</strong> also removes config, data, and managed restic binary.
					The agent must be online to process the command.
				</p>
			</div>

			{/* API Key Modal */}
			{newApiKey && (
				<ApiKeyModal apiKey={newApiKey} onClose={() => setNewApiKey(null)} />
			)}
		</div>
	);
}

export default AgentDetails;
