import { useState } from 'react';
import { useAgents } from '../hooks/useAgents';
import { useRepositories } from '../hooks/useRepositories';
import {
	useCreateSchedule,
	useDeleteSchedule,
	useRunSchedule,
	useSchedules,
	useUpdateSchedule,
} from '../hooks/useSchedules';
import type { Schedule } from '../lib/types';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 rounded-full" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-24 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

interface CreateScheduleModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateScheduleModal({ isOpen, onClose }: CreateScheduleModalProps) {
	const [name, setName] = useState('');
	const [agentId, setAgentId] = useState('');
	const [repositoryId, setRepositoryId] = useState('');
	const [cronExpression, setCronExpression] = useState('0 2 * * *');
	const [paths, setPaths] = useState('/home');

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const createSchedule = useCreateSchedule();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createSchedule.mutateAsync({
				name,
				agent_id: agentId,
				repository_id: repositoryId,
				cron_expression: cronExpression,
				paths: paths.split('\n').filter((p) => p.trim()),
				enabled: true,
			});
			onClose();
			setName('');
			setAgentId('');
			setRepositoryId('');
			setCronExpression('0 2 * * *');
			setPaths('/home');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Create Schedule
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="schedule-name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="schedule-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Daily Home Backup"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="schedule-agent"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Agent
							</label>
							<select
								id="schedule-agent"
								value={agentId}
								onChange={(e) => setAgentId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							>
								<option value="">Select an agent</option>
								{agents?.map((agent) => (
									<option key={agent.id} value={agent.id}>
										{agent.hostname}
									</option>
								))}
							</select>
						</div>
						<div>
							<label
								htmlFor="schedule-repo"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Repository
							</label>
							<select
								id="schedule-repo"
								value={repositoryId}
								onChange={(e) => setRepositoryId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							>
								<option value="">Select a repository</option>
								{repositories?.map((repo) => (
									<option key={repo.id} value={repo.id}>
										{repo.name} ({repo.type})
									</option>
								))}
							</select>
						</div>
						<div>
							<label
								htmlFor="schedule-cron"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Schedule (Cron Expression)
							</label>
							<input
								type="text"
								id="schedule-cron"
								value={cronExpression}
								onChange={(e) => setCronExpression(e.target.value)}
								placeholder="0 2 * * *"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
								required
							/>
							<p className="text-xs text-gray-500 mt-1">
								Examples: 0 2 * * * (daily at 2 AM), 0 */6 * * * (every 6 hours)
							</p>
						</div>
						<div>
							<label
								htmlFor="schedule-paths"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Paths to Backup (one per line)
							</label>
							<textarea
								id="schedule-paths"
								value={paths}
								onChange={(e) => setPaths(e.target.value)}
								placeholder="/home&#10;/etc&#10;/var/www"
								rows={3}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
								required
							/>
						</div>
					</div>
					{createSchedule.isError && (
						<p className="text-sm text-red-600 mt-4">
							Failed to create schedule. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createSchedule.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createSchedule.isPending ? 'Creating...' : 'Create Schedule'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ScheduleRowProps {
	schedule: Schedule;
	agentName?: string;
	repoName?: string;
	onToggle: (id: string, enabled: boolean) => void;
	onDelete: (id: string) => void;
	onRun: (id: string) => void;
	isUpdating: boolean;
	isDeleting: boolean;
	isRunning: boolean;
}

function ScheduleRow({
	schedule,
	agentName,
	repoName,
	onToggle,
	onDelete,
	onRun,
	isUpdating,
	isDeleting,
	isRunning,
}: ScheduleRowProps) {
	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900">{schedule.name}</div>
				<div className="text-sm text-gray-500">
					{agentName ?? 'Unknown Agent'} → {repoName ?? 'Unknown Repo'}
				</div>
			</td>
			<td className="px-6 py-4">
				<code className="text-sm bg-gray-100 px-2 py-1 rounded font-mono">
					{schedule.cron_expression}
				</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{schedule.paths.length} path{schedule.paths.length !== 1 ? 's' : ''}
			</td>
			<td className="px-6 py-4">
				<button
					type="button"
					onClick={() => onToggle(schedule.id, !schedule.enabled)}
					disabled={isUpdating}
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium transition-colors ${
						schedule.enabled
							? 'bg-green-100 text-green-800 hover:bg-green-200'
							: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
					}`}
				>
					<span
						className={`w-1.5 h-1.5 rounded-full ${
							schedule.enabled ? 'bg-green-500' : 'bg-gray-400'
						}`}
					/>
					{schedule.enabled ? 'Active' : 'Paused'}
				</button>
			</td>
			<td className="px-6 py-4 text-right">
				<div className="flex items-center justify-end gap-2">
					<button
						type="button"
						onClick={() => onRun(schedule.id)}
						disabled={isRunning}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium disabled:opacity-50"
					>
						{isRunning ? 'Running...' : 'Run Now'}
					</button>
					<span className="text-gray-300">|</span>
					<button
						type="button"
						onClick={() => onDelete(schedule.id)}
						disabled={isDeleting}
						className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
					>
						Delete
					</button>
				</div>
			</td>
		</tr>
	);
}

export function Schedules() {
	const [searchQuery, setSearchQuery] = useState('');
	const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'paused'>(
		'all',
	);
	const [showCreateModal, setShowCreateModal] = useState(false);

	const { data: schedules, isLoading, isError } = useSchedules();
	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const updateSchedule = useUpdateSchedule();
	const deleteSchedule = useDeleteSchedule();
	const runSchedule = useRunSchedule();

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));
	const repoMap = new Map(repositories?.map((r) => [r.id, r.name]));

	const filteredSchedules = schedules?.filter((schedule) => {
		const matchesSearch = schedule.name
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesStatus =
			statusFilter === 'all' ||
			(statusFilter === 'active' && schedule.enabled) ||
			(statusFilter === 'paused' && !schedule.enabled);
		return matchesSearch && matchesStatus;
	});

	const handleToggle = (id: string, enabled: boolean) => {
		updateSchedule.mutate({ id, data: { enabled } });
	};

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this schedule?')) {
			deleteSchedule.mutate(id);
		}
	};

	const handleRun = (id: string) => {
		runSchedule.mutate(id);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Schedules</h1>
					<p className="text-gray-600 mt-1">Configure automated backup jobs</p>
				</div>
				<button
					type="button"
					onClick={() => setShowCreateModal(true)}
					className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
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
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Create Schedule
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search schedules..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as 'all' | 'active' | 'paused')
							}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="active">Active</option>
							<option value="paused">Paused</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">Failed to load schedules</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Schedule
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Cron
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Paths
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
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
						</tbody>
					</table>
				) : filteredSchedules && filteredSchedules.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Schedule
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Cron
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Paths
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200">
							{filteredSchedules.map((schedule) => (
								<ScheduleRow
									key={schedule.id}
									schedule={schedule}
									agentName={agentMap.get(schedule.agent_id)}
									repoName={repoMap.get(schedule.repository_id)}
									onToggle={handleToggle}
									onDelete={handleDelete}
									onRun={handleRun}
									isUpdating={updateSchedule.isPending}
									isDeleting={deleteSchedule.isPending}
									isRunning={runSchedule.isPending}
								/>
							))}
						</tbody>
					</table>
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
								d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 mb-2">
							No schedules configured
						</h3>
						<p className="mb-4">Create a schedule to automate your backups</p>
						<div className="bg-gray-50 rounded-lg p-4 max-w-md mx-auto text-left space-y-2">
							<p className="text-sm font-medium text-gray-700">
								Common schedules:
							</p>
							<div className="text-sm text-gray-600 space-y-1">
								<p>
									<span className="font-mono bg-gray-200 px-1 rounded">
										0 2 * * *
									</span>{' '}
									— Daily at 2 AM
								</p>
								<p>
									<span className="font-mono bg-gray-200 px-1 rounded">
										0 */6 * * *
									</span>{' '}
									— Every 6 hours
								</p>
								<p>
									<span className="font-mono bg-gray-200 px-1 rounded">
										0 3 * * 0
									</span>{' '}
									— Weekly on Sunday
								</p>
							</div>
						</div>
					</div>
				)}
			</div>

			<CreateScheduleModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>
		</div>
	);
}
