import { useState } from 'react';
import { MultiRepoSelector } from '../components/features/MultiRepoSelector';
import { useAgents } from '../hooks/useAgents';
import { usePolicies } from '../hooks/usePolicies';
import { useRepositories } from '../hooks/useRepositories';
import {
	useCreateSchedule,
	useDeleteSchedule,
	useRunSchedule,
	useSchedules,
	useUpdateSchedule,
} from '../hooks/useSchedules';
import type {
	CompressionLevel,
	Schedule,
	ScheduleRepositoryRequest,
} from '../lib/types';
import { PatternLibraryModal } from '../components/features/PatternLibraryModal';

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
	const [selectedRepos, setSelectedRepos] = useState<
		ScheduleRepositoryRequest[]
	>([]);
	const [cronExpression, setCronExpression] = useState('0 2 * * *');
	const [paths, setPaths] = useState('/home');
	// Policy template state
	const [selectedPolicyId, setSelectedPolicyId] = useState('');
	// Retention policy state
	const [showRetention, setShowRetention] = useState(false);
	const [keepLast, setKeepLast] = useState(5);
	const [keepDaily, setKeepDaily] = useState(7);
	const [keepWeekly, setKeepWeekly] = useState(4);
	const [keepMonthly, setKeepMonthly] = useState(6);
	const [keepYearly, setKeepYearly] = useState(0);
	// Bandwidth control state
	const [bandwidthLimitKb, setBandwidthLimitKb] = useState('');
	const [windowStart, setWindowStart] = useState('');
	const [windowEnd, setWindowEnd] = useState('');
	const [excludedHours, setExcludedHours] = useState<number[]>([]);
	const [compressionLevel, setCompressionLevel] = useState<
		CompressionLevel | ''
	>('');
	const [showAdvanced, setShowAdvanced] = useState(false);
	// Exclude patterns state
	const [excludes, setExcludes] = useState<string[]>([]);
	const [showPatternLibrary, setShowPatternLibrary] = useState(false);

	const { data: agents } = useAgents();
	const { data: repositories } = useRepositories();
	const { data: policies } = usePolicies();
	const createSchedule = useCreateSchedule();

	const handlePolicySelect = (policyId: string) => {
		setSelectedPolicyId(policyId);
		if (!policyId) return;

		const policy = policies?.find((p) => p.id === policyId);
		if (!policy) return;

		// Apply policy values to form
		if (policy.paths && policy.paths.length > 0) {
			setPaths(policy.paths.join('\n'));
		}
		if (policy.cron_expression) {
			setCronExpression(policy.cron_expression);
		}
		if (policy.retention_policy) {
			setShowRetention(true);
			setKeepLast(policy.retention_policy.keep_last || 5);
			setKeepDaily(policy.retention_policy.keep_daily || 7);
			setKeepWeekly(policy.retention_policy.keep_weekly || 4);
			setKeepMonthly(policy.retention_policy.keep_monthly || 6);
			setKeepYearly(policy.retention_policy.keep_yearly || 0);
		}
		if (policy.bandwidth_limit_kb) {
			setBandwidthLimitKb(policy.bandwidth_limit_kb.toString());
			setShowAdvanced(true);
		}
		if (policy.backup_window) {
			setWindowStart(policy.backup_window.start || '');
			setWindowEnd(policy.backup_window.end || '');
			setShowAdvanced(true);
		}
		if (policy.excluded_hours && policy.excluded_hours.length > 0) {
			setExcludedHours(policy.excluded_hours);
			setShowAdvanced(true);
		}
	};

	const toggleExcludedHour = (hour: number) => {
		setExcludedHours((prev) =>
			prev.includes(hour) ? prev.filter((h) => h !== hour) : [...prev, hour],
		);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (selectedRepos.length === 0) {
			return; // Don't submit without repositories
		}
		try {
			const retentionPolicy = showRetention
				? {
						keep_last: keepLast > 0 ? keepLast : undefined,
						keep_daily: keepDaily > 0 ? keepDaily : undefined,
						keep_weekly: keepWeekly > 0 ? keepWeekly : undefined,
						keep_monthly: keepMonthly > 0 ? keepMonthly : undefined,
						keep_yearly: keepYearly > 0 ? keepYearly : undefined,
					}
				: undefined;

			const data: Parameters<typeof createSchedule.mutateAsync>[0] = {
				name,
				agent_id: agentId,
				repositories: selectedRepos,
				cron_expression: cronExpression,
				paths: paths.split('\n').filter((p) => p.trim()),
				excludes: excludes.length > 0 ? excludes : undefined,
				retention_policy: retentionPolicy,
				enabled: true,
			};

			if (bandwidthLimitKb && Number.parseInt(bandwidthLimitKb, 10) > 0) {
				data.bandwidth_limit_kb = Number.parseInt(bandwidthLimitKb, 10);
			}

			if (windowStart || windowEnd) {
				data.backup_window = {
					start: windowStart || undefined,
					end: windowEnd || undefined,
				};
			}

			if (excludedHours.length > 0) {
				data.excluded_hours = excludedHours;
			}

			if (compressionLevel) {
				data.compression_level = compressionLevel;
			}

			await createSchedule.mutateAsync(data);
			onClose();
			setName('');
			setAgentId('');
			setSelectedRepos([]);
			setSelectedPolicyId('');
			setCronExpression('0 2 * * *');
			setPaths('/home');
			// Reset retention policy state
			setShowRetention(false);
			setKeepLast(5);
			setKeepDaily(7);
			setKeepWeekly(4);
			setKeepMonthly(6);
			setKeepYearly(0);
			// Reset bandwidth control state
			setBandwidthLimitKb('');
			setWindowStart('');
			setWindowEnd('');
			setExcludedHours([]);
			setCompressionLevel('');
			setShowAdvanced(false);
			// Reset exclude patterns state
			setExcludes([]);
			setShowPatternLibrary(false);
		} catch {
			// Error handled by mutation
		}
	};

	const handleAddPatterns = (patterns: string[]) => {
		setExcludes((prev) => [...prev, ...patterns.filter((p) => !prev.includes(p))]);
	};

	const handleRemovePattern = (pattern: string) => {
		setExcludes((prev) => prev.filter((p) => p !== pattern));
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
							<MultiRepoSelector
								repositories={repositories ?? []}
								selectedRepos={selectedRepos}
								onChange={setSelectedRepos}
							/>
						</div>
						{policies && policies.length > 0 && (
							<div>
								<label
									htmlFor="schedule-policy"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Policy Template (optional)
								</label>
								<select
									id="schedule-policy"
									value={selectedPolicyId}
									onChange={(e) => handlePolicySelect(e.target.value)}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									<option value="">No template - configure manually</option>
									{policies.map((policy) => (
										<option key={policy.id} value={policy.id}>
											{policy.name}
										</option>
									))}
								</select>
								<p className="text-xs text-gray-500 mt-1">
									Select a policy to pre-fill the form with template values
								</p>
							</div>
						)}
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

						{/* Exclude Patterns Section */}
						<div className="border-t border-gray-200 pt-4">
							<div className="flex items-center justify-between mb-2">
								<label className="block text-sm font-medium text-gray-700">
									Exclude Patterns
								</label>
								<button
									type="button"
									onClick={() => setShowPatternLibrary(true)}
									className="text-sm text-indigo-600 hover:text-indigo-800 flex items-center gap-1"
								>
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
											d="M12 6v6m0 0v6m0-6h6m-6 0H6"
										/>
									</svg>
									Browse Library
								</button>
							</div>
							{excludes.length > 0 ? (
								<div className="space-y-2">
									<div className="flex flex-wrap gap-1.5 p-3 bg-gray-50 rounded-lg max-h-32 overflow-y-auto">
										{excludes.map((pattern) => (
											<span
												key={pattern}
												className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-white border border-gray-200 rounded group"
											>
												<code className="text-gray-700">{pattern}</code>
												<button
													type="button"
													onClick={() => handleRemovePattern(pattern)}
													className="text-gray-400 hover:text-red-500 transition-colors"
												>
													<svg
														className="w-3 h-3"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
														aria-hidden="true"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M6 18L18 6M6 6l12 12"
														/>
													</svg>
												</button>
											</span>
										))}
									</div>
									<p className="text-xs text-gray-500">
										{excludes.length} pattern{excludes.length !== 1 ? 's' : ''} will be excluded from backup
									</p>
								</div>
							) : (
								<p className="text-sm text-gray-500">
									No patterns selected. Click "Browse Library" to add common patterns.
								</p>
							)}
						</div>

						{/* Retention Policy Section */}
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
							{!showRetention ? (
								<p className="text-sm text-gray-500">
									Using default policy: Keep last 5, 7 daily, 4 weekly, 6
									monthly
								</p>
							) : (
								<div className="grid grid-cols-2 gap-3">
									<div>
										<label
											htmlFor="keep-last"
											className="block text-xs font-medium text-gray-600 mb-1"
										>
											Keep Last
										</label>
										<input
											type="number"
											id="keep-last"
											min="0"
											value={keepLast}
											onChange={(e) =>
												setKeepLast(Number.parseInt(e.target.value, 10) || 0)
											}
											className="w-full px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-daily"
											className="block text-xs font-medium text-gray-600 mb-1"
										>
											Keep Daily
										</label>
										<input
											type="number"
											id="keep-daily"
											min="0"
											value={keepDaily}
											onChange={(e) =>
												setKeepDaily(Number.parseInt(e.target.value, 10) || 0)
											}
											className="w-full px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-weekly"
											className="block text-xs font-medium text-gray-600 mb-1"
										>
											Keep Weekly
										</label>
										<input
											type="number"
											id="keep-weekly"
											min="0"
											value={keepWeekly}
											onChange={(e) =>
												setKeepWeekly(Number.parseInt(e.target.value, 10) || 0)
											}
											className="w-full px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-monthly"
											className="block text-xs font-medium text-gray-600 mb-1"
										>
											Keep Monthly
										</label>
										<input
											type="number"
											id="keep-monthly"
											min="0"
											value={keepMonthly}
											onChange={(e) =>
												setKeepMonthly(Number.parseInt(e.target.value, 10) || 0)
											}
											className="w-full px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
									</div>
									<div>
										<label
											htmlFor="keep-yearly"
											className="block text-xs font-medium text-gray-600 mb-1"
										>
											Keep Yearly
										</label>
										<input
											type="number"
											id="keep-yearly"
											min="0"
											value={keepYearly}
											onChange={(e) =>
												setKeepYearly(Number.parseInt(e.target.value, 10) || 0)
											}
											className="w-full px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
									</div>
								</div>
							)}
							<p className="text-xs text-gray-500 mt-2">
								Retention policy automatically removes old backups to save
								storage space.
							</p>
						</div>

						{/* Advanced Settings Section (Bandwidth Controls) */}
						<div className="border-t border-gray-200 pt-4">
							<button
								type="button"
								onClick={() => setShowAdvanced(!showAdvanced)}
								className="flex items-center gap-2 text-sm font-medium text-gray-700 hover:text-gray-900"
							>
								<svg
									className={`w-4 h-4 transition-transform ${showAdvanced ? 'rotate-90' : ''}`}
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M9 5l7 7-7 7"
									/>
								</svg>
								Advanced Settings
							</button>

							{showAdvanced && (
								<div className="mt-4 space-y-4">
									<div>
										<label
											htmlFor="bandwidth-limit"
											className="block text-sm font-medium text-gray-700 mb-1"
										>
											Bandwidth Limit (KB/s)
										</label>
										<input
											type="number"
											id="bandwidth-limit"
											value={bandwidthLimitKb}
											onChange={(e) => setBandwidthLimitKb(e.target.value)}
											placeholder="e.g., 1024 (leave empty for unlimited)"
											min="0"
											className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										/>
										<p className="text-xs text-gray-500 mt-1">
											Limit upload speed during backups. Leave empty for
											unlimited.
										</p>
									</div>

									<fieldset>
										<legend className="block text-sm font-medium text-gray-700 mb-1">
											Backup Window
										</legend>
										<div className="flex items-center gap-2">
											<input
												type="time"
												value={windowStart}
												onChange={(e) => setWindowStart(e.target.value)}
												className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
												aria-label="Window start time"
											/>
											<span className="text-gray-500">to</span>
											<input
												type="time"
												value={windowEnd}
												onChange={(e) => setWindowEnd(e.target.value)}
												className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
												aria-label="Window end time"
											/>
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Only run backups within this time window. Leave empty to
											allow any time.
										</p>
									</fieldset>

									<fieldset>
										<legend className="block text-sm font-medium text-gray-700 mb-2">
											Excluded Hours
										</legend>
										<div className="grid grid-cols-6 gap-1">
											{Array.from({ length: 24 }, (_, i) => (
												<button
													// biome-ignore lint/suspicious/noArrayIndexKey: Static list of 24 hours, order never changes
													key={i}
													type="button"
													onClick={() => toggleExcludedHour(i)}
													className={`px-2 py-1 text-xs rounded transition-colors ${
														excludedHours.includes(i)
															? 'bg-red-100 text-red-800 border border-red-300'
															: 'bg-gray-100 text-gray-600 border border-gray-200 hover:bg-gray-200'
													}`}
												>
													{i.toString().padStart(2, '0')}
												</button>
											))}
										</div>
										<p className="text-xs text-gray-500 mt-1">
											Click to exclude hours when backups should not run.
										</p>
									</fieldset>

									<div>
										<label
											htmlFor="compression-level"
											className="block text-sm font-medium text-gray-700 mb-1"
										>
											Compression Level
										</label>
										<select
											id="compression-level"
											value={compressionLevel}
											onChange={(e) =>
												setCompressionLevel(
													e.target.value as CompressionLevel | '',
												)
											}
											className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										>
											<option value="">Auto (default)</option>
											<option value="off">
												Off - No compression (fastest, largest files)
											</option>
											<option value="auto">Auto - Balanced compression</option>
											<option value="max">
												Max - Maximum compression (slowest, smallest files)
											</option>
										</select>
										<p className="text-xs text-gray-500 mt-1">
											<strong>Off:</strong> Best for already-compressed data
											(videos, images, archives).
											<br />
											<strong>Auto:</strong> Good balance for most data types.
											<br />
											<strong>Max:</strong> Best for text files, logs, and
											databases.
										</p>
									</div>
								</div>
							)}
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
			<PatternLibraryModal
				isOpen={showPatternLibrary}
				onClose={() => setShowPatternLibrary(false)}
				onAddPatterns={handleAddPatterns}
				existingPatterns={excludes}
			/>
		</div>
	);
}

function formatBandwidth(kbps?: number): string {
	if (!kbps) return 'Unlimited';
	if (kbps >= 1024) {
		return `${(kbps / 1024).toFixed(1)} MB/s`;
	}
	return `${kbps} KB/s`;
}

function formatBackupWindow(window?: { start?: string; end?: string }):
	| string
	| null {
	if (!window || (!window.start && !window.end)) return null;
	const start = window.start || '00:00';
	const end = window.end || '23:59';
	return `${start} - ${end}`;
}

interface ScheduleRowProps {
	schedule: Schedule;
	agentName?: string;
	repoNames: string[];
	policyName?: string;
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
	repoNames,
	policyName,
	onToggle,
	onDelete,
	onRun,
	isUpdating,
	isDeleting,
	isRunning,
}: ScheduleRowProps) {
	const hasResourceControls =
		schedule.bandwidth_limit_kb ||
		schedule.backup_window ||
		(schedule.excluded_hours && schedule.excluded_hours.length > 0) ||
		schedule.compression_level;

	const hasBadges = hasResourceControls || policyName;

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900">{schedule.name}</div>
				<div className="text-sm text-gray-500">
					{agentName ?? 'Unknown Agent'} →{' '}
					{repoNames.length > 0 ? repoNames.join(', ') : 'No repos'}
				</div>
				{hasBadges && (
					<div className="mt-1 flex flex-wrap gap-1.5">
						{policyName && (
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-indigo-50 text-indigo-700 rounded">
								<svg
									className="w-3 h-3"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
									/>
								</svg>
								{policyName}
							</span>
						)}
					</div>
				)}
				{hasResourceControls && (
					<div className="mt-1 flex flex-wrap gap-1.5">
						{schedule.bandwidth_limit_kb && (
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-blue-50 text-blue-700 rounded">
								<svg
									className="w-3 h-3"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M13 10V3L4 14h7v7l9-11h-7z"
									/>
								</svg>
								{formatBandwidth(schedule.bandwidth_limit_kb)}
							</span>
						)}
						{formatBackupWindow(schedule.backup_window) && (
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-purple-50 text-purple-700 rounded">
								<svg
									className="w-3 h-3"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
									/>
								</svg>
								{formatBackupWindow(schedule.backup_window)}
							</span>
						)}
						{schedule.excluded_hours && schedule.excluded_hours.length > 0 && (
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-amber-50 text-amber-700 rounded">
								<svg
									className="w-3 h-3"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636"
									/>
								</svg>
								{schedule.excluded_hours.length} excluded hour
								{schedule.excluded_hours.length !== 1 ? 's' : ''}
							</span>
						)}
						{schedule.compression_level && (
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-cyan-50 text-cyan-700 rounded">
								<svg
									className="w-3 h-3"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
									/>
								</svg>
								{schedule.compression_level === 'off'
									? 'No compression'
									: schedule.compression_level === 'max'
										? 'Max compression'
										: 'Auto compression'}
							</span>
						)}
					</div>
				)}
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
	const { data: policies } = usePolicies();
	const updateSchedule = useUpdateSchedule();
	const deleteSchedule = useDeleteSchedule();
	const runSchedule = useRunSchedule();

	const agentMap = new Map(agents?.map((a) => [a.id, a.hostname]));
	const repoMap = new Map(repositories?.map((r) => [r.id, r.name]));
	const policyMap = new Map(policies?.map((p) => [p.id, p.name]));

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
							{filteredSchedules.map((schedule) => {
								const repoNames = (schedule.repositories ?? [])
									.sort((a, b) => a.priority - b.priority)
									.map((r) => repoMap.get(r.repository_id) ?? 'Unknown');
								return (
									<ScheduleRow
										key={schedule.id}
										schedule={schedule}
										agentName={agentMap.get(schedule.agent_id)}
										repoNames={repoNames}
										policyName={
											schedule.policy_id
												? policyMap.get(schedule.policy_id)
												: undefined
										}
										onToggle={handleToggle}
										onDelete={handleDelete}
										onRun={handleRun}
										isUpdating={updateSchedule.isPending}
										isDeleting={deleteSchedule.isPending}
										isRunning={runSchedule.isPending}
									/>
								);
							})}
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
