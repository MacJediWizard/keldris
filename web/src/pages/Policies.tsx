import { useState } from 'react';
import { useAgents } from '../hooks/useAgents';
import {
	useApplyPolicy,
	useCreatePolicy,
	useDeletePolicy,
	usePolicies,
	usePolicySchedules,
} from '../hooks/usePolicies';
import { useRepositories } from '../hooks/useRepositories';
import type {
	BackupWindow,
	CreatePolicyRequest,
	Policy,
	RetentionPolicy,
} from '../lib/types';
import { formatDate } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="h-4 bg-gray-200 rounded w-32" />
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="h-4 bg-gray-200 rounded w-48" />
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="h-4 bg-gray-200 rounded w-24" />
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="h-4 bg-gray-200 rounded w-20" />
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="h-4 bg-gray-200 rounded w-16" />
			</td>
		</tr>
	);
}

interface CreatePolicyModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreatePolicyModal({ isOpen, onClose }: CreatePolicyModalProps) {
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [paths, setPaths] = useState('/home');
	const [excludes, setExcludes] = useState('');
	const [cronExpression, setCronExpression] = useState('0 2 * * *');
	const [showRetention, setShowRetention] = useState(false);
	const [keepLast, setKeepLast] = useState(5);
	const [keepDaily, setKeepDaily] = useState(7);
	const [keepWeekly, setKeepWeekly] = useState(4);
	const [keepMonthly, setKeepMonthly] = useState(6);
	const [showAdvanced, setShowAdvanced] = useState(false);
	const [bandwidthLimit, setBandwidthLimit] = useState('');
	const [windowStart, setWindowStart] = useState('');
	const [windowEnd, setWindowEnd] = useState('');

	const createPolicy = useCreatePolicy();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const retentionPolicy: RetentionPolicy | undefined = showRetention
				? {
						keep_last: keepLast,
						keep_daily: keepDaily,
						keep_weekly: keepWeekly,
						keep_monthly: keepMonthly,
					}
				: undefined;

			const backupWindow: BackupWindow | undefined =
				windowStart || windowEnd
					? { start: windowStart || undefined, end: windowEnd || undefined }
					: undefined;

			const data: CreatePolicyRequest = {
				name,
				description: description || undefined,
				paths: paths
					.split('\n')
					.map((p) => p.trim())
					.filter(Boolean),
				excludes: excludes
					? excludes
							.split('\n')
							.map((e) => e.trim())
							.filter(Boolean)
					: undefined,
				cron_expression: cronExpression || undefined,
				retention_policy: retentionPolicy,
				bandwidth_limit_kb: bandwidthLimit
					? Number.parseInt(bandwidthLimit, 10)
					: undefined,
				backup_window: backupWindow,
			};

			await createPolicy.mutateAsync(data);
			onClose();
			// Reset form
			setName('');
			setDescription('');
			setPaths('/home');
			setExcludes('');
			setCronExpression('0 2 * * *');
			setShowRetention(false);
			setShowAdvanced(false);
			setBandwidthLimit('');
			setWindowStart('');
			setWindowEnd('');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Create Policy
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="policy-name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="policy-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Standard Backup Policy"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>

						<div>
							<label
								htmlFor="policy-description"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Description
							</label>
							<textarea
								id="policy-description"
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								placeholder="Optional description of this policy"
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>

						<div>
							<label
								htmlFor="policy-paths"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Default Paths
							</label>
							<textarea
								id="policy-paths"
								value={paths}
								onChange={(e) => setPaths(e.target.value)}
								placeholder="One path per line, e.g.,&#10;/home&#10;/var/www"
								rows={3}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
							/>
						</div>

						<div>
							<label
								htmlFor="policy-excludes"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Default Excludes
							</label>
							<textarea
								id="policy-excludes"
								value={excludes}
								onChange={(e) => setExcludes(e.target.value)}
								placeholder="One pattern per line, e.g.,&#10;*.tmp&#10;.git"
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
							/>
						</div>

						<div>
							<label
								htmlFor="policy-cron"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Default Schedule (Cron)
							</label>
							<input
								type="text"
								id="policy-cron"
								value={cronExpression}
								onChange={(e) => setCronExpression(e.target.value)}
								placeholder="e.g., 0 2 * * * (daily at 2am)"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
							/>
							<p className="text-xs text-gray-500 mt-1">
								Format: minute hour day-of-month month day-of-week
							</p>
						</div>

						<div className="border-t border-gray-200 pt-4">
							<div className="flex items-center justify-between mb-3">
								<span className="text-sm font-medium text-gray-700">
									Retention Policy
								</span>
								<button
									type="button"
									onClick={() => setShowRetention(!showRetention)}
									className="text-sm text-indigo-600 hover:text-indigo-800"
								>
									{showRetention ? 'Use defaults' : 'Customize'}
								</button>
							</div>
							{showRetention ? (
								<div className="grid grid-cols-2 gap-3">
									<div>
										<label
											htmlFor="keep-last"
											className="block text-xs text-gray-600 mb-1"
										>
											Keep Last
										</label>
										<input
											type="number"
											id="keep-last"
											value={keepLast}
											onChange={(e) =>
												setKeepLast(Number.parseInt(e.target.value, 10) || 0)
											}
											min="0"
											className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-daily"
											className="block text-xs text-gray-600 mb-1"
										>
											Keep Daily
										</label>
										<input
											type="number"
											id="keep-daily"
											value={keepDaily}
											onChange={(e) =>
												setKeepDaily(Number.parseInt(e.target.value, 10) || 0)
											}
											min="0"
											className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-weekly"
											className="block text-xs text-gray-600 mb-1"
										>
											Keep Weekly
										</label>
										<input
											type="number"
											id="keep-weekly"
											value={keepWeekly}
											onChange={(e) =>
												setKeepWeekly(Number.parseInt(e.target.value, 10) || 0)
											}
											min="0"
											className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-monthly"
											className="block text-xs text-gray-600 mb-1"
										>
											Keep Monthly
										</label>
										<input
											type="number"
											id="keep-monthly"
											value={keepMonthly}
											onChange={(e) =>
												setKeepMonthly(Number.parseInt(e.target.value, 10) || 0)
											}
											min="0"
											className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
										/>
									</div>
								</div>
							) : (
								<p className="text-sm text-gray-500">
									Keep 5 latest, 7 daily, 4 weekly, 6 monthly
								</p>
							)}
						</div>

						<div className="border-t border-gray-200 pt-4">
							<div className="flex items-center justify-between mb-3">
								<span className="text-sm font-medium text-gray-700">
									Advanced Settings
								</span>
								<button
									type="button"
									onClick={() => setShowAdvanced(!showAdvanced)}
									className="text-sm text-indigo-600 hover:text-indigo-800"
								>
									{showAdvanced ? 'Hide' : 'Show'}
								</button>
							</div>
							{showAdvanced && (
								<div className="space-y-3">
									<div>
										<label
											htmlFor="bandwidth-limit"
											className="block text-xs text-gray-600 mb-1"
										>
											Bandwidth Limit (KB/s)
										</label>
										<input
											type="number"
											id="bandwidth-limit"
											value={bandwidthLimit}
											onChange={(e) => setBandwidthLimit(e.target.value)}
											placeholder="Unlimited"
											min="0"
											className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
										/>
									</div>
									<div className="grid grid-cols-2 gap-3">
										<div>
											<label
												htmlFor="window-start"
												className="block text-xs text-gray-600 mb-1"
											>
												Backup Window Start
											</label>
											<input
												type="time"
												id="window-start"
												value={windowStart}
												onChange={(e) => setWindowStart(e.target.value)}
												className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
											/>
										</div>
										<div>
											<label
												htmlFor="window-end"
												className="block text-xs text-gray-600 mb-1"
											>
												Backup Window End
											</label>
											<input
												type="time"
												id="window-end"
												value={windowEnd}
												onChange={(e) => setWindowEnd(e.target.value)}
												className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500"
											/>
										</div>
									</div>
								</div>
							)}
						</div>
					</div>

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
							disabled={createPolicy.isPending || !name}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createPolicy.isPending ? 'Creating...' : 'Create Policy'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ApplyPolicyModalProps {
	policy: Policy;
	isOpen: boolean;
	onClose: () => void;
}

function ApplyPolicyModal({ policy, isOpen, onClose }: ApplyPolicyModalProps) {
	const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
	const [repositoryId, setRepositoryId] = useState('');
	const [scheduleName, setScheduleName] = useState('');

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const applyPolicy = useApplyPolicy();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (selectedAgents.length === 0 || !repositoryId) return;

		try {
			await applyPolicy.mutateAsync({
				id: policy.id,
				data: {
					agent_ids: selectedAgents,
					repository_id: repositoryId,
					schedule_name: scheduleName || undefined,
				},
			});
			onClose();
			setSelectedAgents([]);
			setRepositoryId('');
			setScheduleName('');
		} catch {
			// Error handled by mutation
		}
	};

	const toggleAgent = (agentId: string) => {
		setSelectedAgents((prev) =>
			prev.includes(agentId)
				? prev.filter((id) => id !== agentId)
				: [...prev, agentId],
		);
	};

	const selectAllAgents = () => {
		if (agents) {
			setSelectedAgents(agents.map((a) => a.id));
		}
	};

	const deselectAllAgents = () => {
		setSelectedAgents([]);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-2">
					Apply Policy: {policy.name}
				</h3>
				<p className="text-sm text-gray-600 mb-4">
					Create schedules for selected agents using this policy as a template.
				</p>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<div className="flex items-center justify-between mb-2">
								<span className="block text-sm font-medium text-gray-700">
									Select Agents
								</span>
								<div className="flex gap-2">
									<button
										type="button"
										onClick={selectAllAgents}
										className="text-xs text-indigo-600 hover:text-indigo-800"
									>
										Select all
									</button>
									<button
										type="button"
										onClick={deselectAllAgents}
										className="text-xs text-gray-600 hover:text-gray-800"
									>
										Deselect all
									</button>
								</div>
							</div>
							<div className="border border-gray-300 rounded-lg max-h-48 overflow-y-auto">
								{agents?.length ? (
									agents.map((agent) => (
										<label
											key={agent.id}
											className="flex items-center gap-3 px-4 py-2 hover:bg-gray-50 cursor-pointer border-b border-gray-100 last:border-b-0"
										>
											<input
												type="checkbox"
												checked={selectedAgents.includes(agent.id)}
												onChange={() => toggleAgent(agent.id)}
												className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
											/>
											<span className="text-sm text-gray-900">
												{agent.hostname}
											</span>
											<span
												className={`ml-auto text-xs px-2 py-0.5 rounded-full ${
													agent.status === 'active'
														? 'bg-green-100 text-green-800'
														: 'bg-gray-100 text-gray-600'
												}`}
											>
												{agent.status}
											</span>
										</label>
									))
								) : (
									<p className="px-4 py-3 text-sm text-gray-500">
										No agents available
									</p>
								)}
							</div>
							<p className="text-xs text-gray-500 mt-1">
								{selectedAgents.length} agent(s) selected
							</p>
						</div>

						<div>
							<label
								htmlFor="apply-repository"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Repository
							</label>
							<select
								id="apply-repository"
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
								htmlFor="apply-schedule-name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Schedule Name Prefix (optional)
							</label>
							<input
								type="text"
								id="apply-schedule-name"
								value={scheduleName}
								onChange={(e) => setScheduleName(e.target.value)}
								placeholder={`Default: ${policy.name}`}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
							<p className="text-xs text-gray-500 mt-1">
								Agent hostname will be appended to create unique schedule names
							</p>
						</div>
					</div>

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
							disabled={
								applyPolicy.isPending ||
								selectedAgents.length === 0 ||
								!repositoryId
							}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{applyPolicy.isPending
								? 'Applying...'
								: `Apply to ${selectedAgents.length} Agent(s)`}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface PolicySchedulesModalProps {
	policy: Policy;
	isOpen: boolean;
	onClose: () => void;
}

function PolicySchedulesModal({
	policy,
	isOpen,
	onClose,
}: PolicySchedulesModalProps) {
	const { data: schedules, isLoading } = usePolicySchedules(
		isOpen ? policy.id : '',
	);

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-2">
					Schedules using: {policy.name}
				</h3>
				<div className="mt-4">
					{isLoading ? (
						<div className="text-center py-4">
							<div className="w-6 h-6 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto" />
						</div>
					) : schedules?.length ? (
						<div className="space-y-2">
							{schedules.map((schedule) => (
								<div
									key={schedule.id}
									className="flex items-center justify-between px-4 py-3 bg-gray-50 rounded-lg"
								>
									<div>
										<p className="text-sm font-medium text-gray-900">
											{schedule.name}
										</p>
										<p className="text-xs text-gray-500">
											{schedule.cron_expression}
										</p>
									</div>
									<span
										className={`text-xs px-2 py-0.5 rounded-full ${
											schedule.enabled
												? 'bg-green-100 text-green-800'
												: 'bg-gray-100 text-gray-600'
										}`}
									>
										{schedule.enabled ? 'Enabled' : 'Disabled'}
									</span>
								</div>
							))}
						</div>
					) : (
						<p className="text-sm text-gray-500 text-center py-4">
							No schedules are using this policy yet.
						</p>
					)}
				</div>
				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

interface PolicyRowProps {
	policy: Policy;
	onApply: (policy: Policy) => void;
	onViewSchedules: (policy: Policy) => void;
	onDelete: (id: string) => void;
}

function PolicyRow({
	policy,
	onApply,
	onViewSchedules,
	onDelete,
}: PolicyRowProps) {
	const [showActions, setShowActions] = useState(false);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="text-sm font-medium text-gray-900">{policy.name}</div>
				{policy.description && (
					<div className="text-xs text-gray-500 truncate max-w-xs">
						{policy.description}
					</div>
				)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="text-sm text-gray-900 font-mono">
					{policy.cron_expression || '-'}
				</div>
			</td>
			<td className="px-6 py-4 whitespace-nowrap">
				<div className="text-sm text-gray-500">
					{policy.paths?.length || 0} path(s)
				</div>
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
				{formatDate(policy.created_at)}
			</td>
			<td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
				<div className="relative">
					<button
						type="button"
						onClick={() => setShowActions(!showActions)}
						className="text-gray-400 hover:text-gray-600"
					>
						<svg
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
							aria-hidden="true"
						>
							<path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
						</svg>
					</button>
					{showActions && (
						<>
							<div
								className="fixed inset-0 z-10"
								onClick={() => setShowActions(false)}
								onKeyDown={(e) => {
									if (e.key === 'Escape') setShowActions(false);
								}}
								tabIndex={0}
								aria-label="Close menu"
							/>
							<div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-20">
								<button
									type="button"
									onClick={() => {
										onApply(policy);
										setShowActions(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
								>
									Apply to Agents
								</button>
								<button
									type="button"
									onClick={() => {
										onViewSchedules(policy);
										setShowActions(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
								>
									View Schedules
								</button>
								<button
									type="button"
									onClick={() => {
										if (
											window.confirm(
												`Are you sure you want to delete the policy "${policy.name}"?`,
											)
										) {
											onDelete(policy.id);
										}
										setShowActions(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50"
								>
									Delete
								</button>
							</div>
						</>
					)}
				</div>
			</td>
		</tr>
	);
}

export function Policies() {
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [applyPolicy, setApplyPolicy] = useState<Policy | null>(null);
	const [viewSchedulesPolicy, setViewSchedulesPolicy] = useState<Policy | null>(
		null,
	);

	const { data: policies, isLoading, isError } = usePolicies();
	const deletePolicy = useDeletePolicy();

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Policies</h1>
					<p className="text-sm text-gray-500 mt-1">
						Create reusable backup configuration templates
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowCreateModal(true)}
					className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
				>
					<svg
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Create Policy
				</button>
			</div>

			<div className="bg-white rounded-lg shadow overflow-hidden">
				<table className="min-w-full divide-y divide-gray-200">
					<thead className="bg-gray-50">
						<tr>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
								Name
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
								Schedule
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
								Paths
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
								Created
							</th>
							<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
								Actions
							</th>
						</tr>
					</thead>
					<tbody className="bg-white divide-y divide-gray-200">
						{isLoading ? (
							<>
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</>
						) : isError ? (
							<tr>
								<td colSpan={5} className="px-6 py-12 text-center">
									<div className="text-red-600">Failed to load policies</div>
								</td>
							</tr>
						) : policies?.length === 0 ? (
							<tr>
								<td colSpan={5} className="px-6 py-12 text-center">
									<svg
										className="w-12 h-12 text-gray-400 mx-auto mb-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
										aria-hidden="true"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={1.5}
											d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
										/>
									</svg>
									<p className="text-gray-500">No policies created yet</p>
									<p className="text-sm text-gray-400 mt-1">
										Create a policy to define reusable backup configurations
									</p>
								</td>
							</tr>
						) : (
							policies?.map((policy) => (
								<PolicyRow
									key={policy.id}
									policy={policy}
									onApply={setApplyPolicy}
									onViewSchedules={setViewSchedulesPolicy}
									onDelete={(id) => deletePolicy.mutate(id)}
								/>
							))
						)}
					</tbody>
				</table>
			</div>

			<CreatePolicyModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			{applyPolicy && (
				<ApplyPolicyModal
					policy={applyPolicy}
					isOpen={!!applyPolicy}
					onClose={() => setApplyPolicy(null)}
				/>
			)}

			{viewSchedulesPolicy && (
				<PolicySchedulesModal
					policy={viewSchedulesPolicy}
					isOpen={!!viewSchedulesPolicy}
					onClose={() => setViewSchedulesPolicy(null)}
				/>
			)}
		</div>
	);
}

export default Policies;
