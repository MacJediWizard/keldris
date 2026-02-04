import { useState } from 'react';
import {
	useCreateNotificationRule,
	useDeleteNotificationRule,
	useNotificationRules,
	useTestNotificationRule,
	useUpdateNotificationRule,
} from '../hooks/useNotificationRules';
import { useNotificationChannels } from '../hooks/useNotifications';
import type {
	NotificationRule,
	RuleAction,
	RuleActionType,
	RuleConditions,
	RuleTriggerType,
} from '../lib/types';

interface ActionWithId extends RuleAction {
	_id: string;
}

function generateId(): string {
	return Math.random().toString(36).substring(2, 11);
}

const TRIGGER_TYPES: { value: RuleTriggerType; label: string }[] = [
	{ value: 'backup_failed', label: 'Backup Failed' },
	{ value: 'backup_success', label: 'Backup Success' },
	{ value: 'agent_offline', label: 'Agent Offline' },
	{ value: 'agent_health_warning', label: 'Agent Health Warning' },
	{ value: 'agent_health_critical', label: 'Agent Health Critical' },
	{ value: 'storage_usage_high', label: 'Storage Usage High' },
	{ value: 'replication_lag', label: 'Replication Lag' },
	{ value: 'ransomware_suspected', label: 'Ransomware Suspected' },
	{ value: 'maintenance_scheduled', label: 'Maintenance Scheduled' },
];

const ACTION_TYPES: { value: RuleActionType; label: string }[] = [
	{ value: 'notify_channel', label: 'Send Notification' },
	{ value: 'escalate', label: 'Escalate to PagerDuty' },
	{ value: 'suppress', label: 'Suppress Further Notifications' },
	{ value: 'webhook', label: 'Call Webhook' },
];

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-24 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

interface AddRuleModalProps {
	isOpen: boolean;
	onClose: () => void;
	editRule?: NotificationRule;
}

function AddRuleModal({ isOpen, onClose, editRule }: AddRuleModalProps) {
	const [name, setName] = useState(editRule?.name || '');
	const [description, setDescription] = useState(editRule?.description || '');
	const [triggerType, setTriggerType] = useState<RuleTriggerType>(
		editRule?.trigger_type || 'backup_failed',
	);
	const [enabled, setEnabled] = useState(editRule?.enabled ?? true);
	const [priority, setPriority] = useState(editRule?.priority ?? 0);
	const [count, setCount] = useState(editRule?.conditions.count ?? 3);
	const [timeWindow, setTimeWindow] = useState(
		editRule?.conditions.time_window_minutes ?? 60,
	);
	const [actions, setActions] = useState<ActionWithId[]>(
		editRule?.actions.map((a) => ({ ...a, _id: generateId() })) || [
			{ type: 'notify_channel', _id: generateId() },
		],
	);

	const { data: channels } = useNotificationChannels();
	const createRule = useCreateNotificationRule();
	const updateRule = useUpdateNotificationRule();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		const conditions: RuleConditions = {
			count,
			time_window_minutes: timeWindow,
		};
		// Strip _id from actions before sending to API
		const cleanActions: RuleAction[] = actions.map(
			({ _id, ...rest }) => rest as RuleAction,
		);

		try {
			if (editRule) {
				await updateRule.mutateAsync({
					id: editRule.id,
					data: {
						name,
						description,
						enabled,
						priority,
						conditions,
						actions: cleanActions,
					},
				});
			} else {
				await createRule.mutateAsync({
					name,
					description,
					trigger_type: triggerType,
					enabled,
					priority,
					conditions,
					actions: cleanActions,
				});
			}
			resetForm();
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const resetForm = () => {
		setName('');
		setDescription('');
		setTriggerType('backup_failed');
		setEnabled(true);
		setPriority(0);
		setCount(3);
		setTimeWindow(60);
		setActions([{ type: 'notify_channel', _id: generateId() }]);
	};

	const addAction = () => {
		setActions([...actions, { type: 'notify_channel', _id: generateId() }]);
	};

	const removeAction = (index: number) => {
		setActions(actions.filter((_, i) => i !== index));
	};

	const updateAction = (index: number, updates: Partial<RuleAction>) => {
		const newActions = [...actions];
		newActions[index] = { ...newActions[index], ...updates };
		setActions(newActions);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					{editRule ? 'Edit Notification Rule' : 'Create Notification Rule'}
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Rule Name
							</label>
							<input
								type="text"
								id="name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Escalate on 3 Backup Failures"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>

						<div>
							<label
								htmlFor="description"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Description
							</label>
							<textarea
								id="description"
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								placeholder="Describe what this rule does..."
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>

						{!editRule && (
							<div>
								<label
									htmlFor="triggerType"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Trigger Event
								</label>
								<select
									id="triggerType"
									value={triggerType}
									onChange={(e) =>
										setTriggerType(e.target.value as RuleTriggerType)
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									{TRIGGER_TYPES.map((t) => (
										<option key={t.value} value={t.value}>
											{t.label}
										</option>
									))}
								</select>
							</div>
						)}

						<div className="border-t border-gray-200 dark:border-gray-700 pt-4">
							<h4 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
								Conditions
							</h4>
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="count"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Event Count
									</label>
									<input
										type="number"
										id="count"
										min={1}
										value={count}
										onChange={(e) =>
											setCount(Number.parseInt(e.target.value, 10))
										}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
									<p className="text-xs text-gray-500 mt-1">
										Number of events required to trigger
									</p>
								</div>
								<div>
									<label
										htmlFor="timeWindow"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Time Window (minutes)
									</label>
									<input
										type="number"
										id="timeWindow"
										min={1}
										value={timeWindow}
										onChange={(e) =>
											setTimeWindow(Number.parseInt(e.target.value, 10))
										}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
									<p className="text-xs text-gray-500 mt-1">
										Window to count events within
									</p>
								</div>
							</div>
						</div>

						<div className="border-t border-gray-200 dark:border-gray-700 pt-4">
							<div className="flex items-center justify-between mb-3">
								<h4 className="text-sm font-medium text-gray-900 dark:text-white">
									Actions
								</h4>
								<button
									type="button"
									onClick={addAction}
									className="text-sm text-indigo-600 hover:text-indigo-800"
								>
									+ Add Action
								</button>
							</div>
							<div className="space-y-3">
								{actions.map((action, index) => (
									<div
										key={action._id}
										className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
									>
										<div className="flex items-center gap-3 mb-2">
											<select
												value={action.type}
												onChange={(e) =>
													updateAction(index, {
														type: e.target.value as RuleActionType,
													})
												}
												className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg text-sm"
											>
												{ACTION_TYPES.map((a) => (
													<option key={a.value} value={a.value}>
														{a.label}
													</option>
												))}
											</select>
											{actions.length > 1 && (
												<button
													type="button"
													onClick={() => removeAction(index)}
													className="text-red-500 hover:text-red-700"
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
															d="M6 18L18 6M6 6l12 12"
														/>
													</svg>
												</button>
											)}
										</div>
										{(action.type === 'notify_channel' ||
											action.type === 'escalate') && (
											<select
												value={
													action.type === 'escalate'
														? action.escalate_to_channel_id || ''
														: action.channel_id || ''
												}
												onChange={(e) =>
													updateAction(
														index,
														action.type === 'escalate'
															? { escalate_to_channel_id: e.target.value }
															: { channel_id: e.target.value },
													)
												}
												className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg text-sm"
											>
												<option value="">Select a channel...</option>
												{channels?.map((ch) => (
													<option key={ch.id} value={ch.id}>
														{ch.name} ({ch.type})
													</option>
												))}
											</select>
										)}
										{action.type === 'webhook' && (
											<input
												type="url"
												value={action.webhook_url || ''}
												onChange={(e) =>
													updateAction(index, { webhook_url: e.target.value })
												}
												placeholder="https://example.com/webhook"
												className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg text-sm"
											/>
										)}
										{action.type === 'suppress' && (
											<input
												type="number"
												value={action.suppress_duration_minutes || 60}
												onChange={(e) =>
													updateAction(index, {
														suppress_duration_minutes: Number.parseInt(
															e.target.value,
															10,
														),
													})
												}
												placeholder="Duration in minutes"
												className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg text-sm"
											/>
										)}
										<input
											type="text"
											value={action.message || ''}
											onChange={(e) =>
												updateAction(index, { message: e.target.value })
											}
											placeholder="Custom message (optional)"
											className="w-full mt-2 px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg text-sm"
										/>
									</div>
								))}
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div className="flex items-center">
								<input
									type="checkbox"
									id="enabled"
									checked={enabled}
									onChange={(e) => setEnabled(e.target.checked)}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<label
									htmlFor="enabled"
									className="ml-2 text-sm text-gray-700 dark:text-gray-300"
								>
									Rule Enabled
								</label>
							</div>
							<div>
								<label
									htmlFor="priority"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Priority
								</label>
								<input
									type="number"
									id="priority"
									min={0}
									value={priority}
									onChange={(e) =>
										setPriority(Number.parseInt(e.target.value, 10))
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
								<p className="text-xs text-gray-500 mt-1">
									Lower = higher priority
								</p>
							</div>
						</div>
					</div>

					{(createRule.isError || updateRule.isError) && (
						<p className="text-sm text-red-600 mt-4">
							Failed to save rule. Please try again.
						</p>
					)}

					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={() => {
								resetForm();
								onClose();
							}}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createRule.isPending || updateRule.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createRule.isPending || updateRule.isPending
								? 'Saving...'
								: editRule
									? 'Update Rule'
									: 'Create Rule'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface RuleRowProps {
	rule: NotificationRule;
	onEdit: (rule: NotificationRule) => void;
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

function RuleRow({ rule, onEdit, onDelete, isDeleting }: RuleRowProps) {
	const updateRule = useUpdateNotificationRule();
	const testRule = useTestNotificationRule();

	const handleToggle = async () => {
		await updateRule.mutateAsync({
			id: rule.id,
			data: { enabled: !rule.enabled },
		});
	};

	const handleTest = async () => {
		const result = await testRule.mutateAsync({ id: rule.id });
		if (result.success) {
			alert('Rule test successful! Actions would be triggered.');
		} else {
			alert(`Rule test failed: ${result.message}`);
		}
	};

	const triggerLabel =
		TRIGGER_TYPES.find((t) => t.value === rule.trigger_type)?.label ||
		rule.trigger_type;

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div>
					<p className="font-medium text-gray-900 dark:text-white">
						{rule.name}
					</p>
					{rule.description && (
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{rule.description}
						</p>
					)}
				</div>
			</td>
			<td className="px-6 py-4">
				<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
					{triggerLabel}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{rule.conditions.count && rule.conditions.count > 1 ? (
					<span>
						{rule.conditions.count}x in {rule.conditions.time_window_minutes}min
					</span>
				) : (
					<span>Immediate</span>
				)}
			</td>
			<td className="px-6 py-4">
				<button
					type="button"
					onClick={handleToggle}
					disabled={updateRule.isPending}
					className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 ${
						rule.enabled ? 'bg-indigo-600' : 'bg-gray-200'
					}`}
				>
					<span
						className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
							rule.enabled ? 'translate-x-5' : 'translate-x-0'
						}`}
					/>
				</button>
			</td>
			<td className="px-6 py-4 text-right space-x-2">
				<button
					type="button"
					onClick={handleTest}
					disabled={testRule.isPending}
					className="text-indigo-600 hover:text-indigo-800 text-sm font-medium disabled:opacity-50"
				>
					Test
				</button>
				<button
					type="button"
					onClick={() => onEdit(rule)}
					className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200 text-sm font-medium"
				>
					Edit
				</button>
				<button
					type="button"
					onClick={() => onDelete(rule.id)}
					disabled={isDeleting}
					className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
				>
					Delete
				</button>
			</td>
		</tr>
	);
}

export function NotificationRules() {
	const [showAddModal, setShowAddModal] = useState(false);
	const [editingRule, setEditingRule] = useState<
		NotificationRule | undefined
	>();

	const { data: rules, isLoading, isError } = useNotificationRules();
	const deleteRule = useDeleteNotificationRule();

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this notification rule?')) {
			deleteRule.mutate(id);
		}
	};

	const handleEdit = (rule: NotificationRule) => {
		setEditingRule(rule);
		setShowAddModal(true);
	};

	const handleCloseModal = () => {
		setShowAddModal(false);
		setEditingRule(undefined);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Notification Rules
					</h1>
					<p className="text-gray-500 mt-1">
						Create rules to escalate notifications based on conditions
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowAddModal(true)}
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
					Create Rule
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load notification rules</p>
						<p className="text-sm mt-1">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Rule
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Trigger
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Conditions
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Enabled
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{[1, 2, 3].map((i) => (
								<LoadingRow key={i} />
							))}
						</tbody>
					</table>
				) : rules && rules.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Rule
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Trigger
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Conditions
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Enabled
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{rules.map((rule) => (
								<RuleRow
									key={rule.id}
									rule={rule}
									onEdit={handleEdit}
									onDelete={handleDelete}
									isDeleting={deleteRule.isPending}
								/>
							))}
						</tbody>
					</table>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<div className="inline-flex items-center justify-center w-12 h-12 bg-gray-100 rounded-full mb-4">
							<svg
								className="w-6 h-6 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"
								/>
							</svg>
						</div>
						<p className="font-medium">No notification rules configured</p>
						<p className="text-sm mt-1">
							Create a rule to escalate alerts when conditions are met
						</p>
						<button
							type="button"
							onClick={() => setShowAddModal(true)}
							className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Create Rule
						</button>
					</div>
				)}
			</div>

			<div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
				<h3 className="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">
					Example: Escalate to PagerDuty on 3 Backup Failures
				</h3>
				<p className="text-sm text-blue-700 dark:text-blue-300">
					Create a rule with trigger &quot;Backup Failed&quot;, condition
					&quot;3 events in 60 minutes&quot;, and action &quot;Escalate to
					PagerDuty&quot; to automatically escalate when backups fail
					repeatedly.
				</p>
			</div>

			<AddRuleModal
				isOpen={showAddModal}
				onClose={handleCloseModal}
				editRule={editingRule}
			/>
		</div>
	);
}
