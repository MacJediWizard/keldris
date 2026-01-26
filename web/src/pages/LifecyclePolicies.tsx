import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useCreateLifecyclePolicy,
	useDeleteLifecyclePolicy,
	useLifecycleDryRun,
	useLifecyclePolicies,
	useOrgLifecycleDeletions,
	useUpdateLifecyclePolicy,
} from '../hooks/useLifecyclePolicies';
import type {
	ClassificationLevel,
	ClassificationRetention,
	CreateLifecyclePolicyRequest,
	LifecycleDryRunResult,
	LifecyclePolicy,
	LifecyclePolicyStatus,
	RetentionDuration,
} from '../lib/types';
import { formatBytes, formatDateTime } from '../lib/utils';

const CLASSIFICATION_LEVELS: { value: ClassificationLevel; label: string }[] = [
	{ value: 'public', label: 'Public' },
	{ value: 'internal', label: 'Internal' },
	{ value: 'confidential', label: 'Confidential' },
	{ value: 'restricted', label: 'Restricted' },
];

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
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
			<td className="px-6 py-4">
				<div className="h-8 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
		</tr>
	);
}

interface RetentionRuleEditorProps {
	rule: ClassificationRetention;
	onChange: (rule: ClassificationRetention) => void;
	onRemove: () => void;
}

function RetentionRuleEditor({
	rule,
	onChange,
	onRemove,
}: RetentionRuleEditorProps) {
	const updateRetention = (field: keyof RetentionDuration, value: number) => {
		onChange({
			...rule,
			retention: { ...rule.retention, [field]: value },
		});
	};

	return (
		<div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-4">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-3">
					<select
						value={rule.level}
						onChange={(e) =>
							onChange({
								...rule,
								level: e.target.value as ClassificationLevel,
							})
						}
						className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm"
					>
						{CLASSIFICATION_LEVELS.map((level) => (
							<option key={level.value} value={level.value}>
								{level.label}
							</option>
						))}
					</select>
					<span className="text-sm text-gray-500 dark:text-gray-400">
						Classification Level
					</span>
				</div>
				<button
					type="button"
					onClick={onRemove}
					className="text-red-500 hover:text-red-700 text-sm"
				>
					Remove
				</button>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor={`min-retention-${rule.level}`}
						className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1"
					>
						Minimum Retention (days)
					</label>
					<input
						id={`min-retention-${rule.level}`}
						type="number"
						value={rule.retention.min_days}
						onChange={(e) =>
							updateRetention(
								'min_days',
								Number.parseInt(e.target.value, 10) || 0,
							)
						}
						min="0"
						className="w-full px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm"
					/>
					<p className="text-xs text-gray-500 mt-1">
						Compliance: snapshots cannot be deleted before this period
					</p>
				</div>
				<div>
					<label
						htmlFor={`max-retention-${rule.level}`}
						className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1"
					>
						Maximum Retention (days)
					</label>
					<input
						id={`max-retention-${rule.level}`}
						type="number"
						value={rule.retention.max_days}
						onChange={(e) =>
							updateRetention(
								'max_days',
								Number.parseInt(e.target.value, 10) || 0,
							)
						}
						min="0"
						className="w-full px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm"
					/>
					<p className="text-xs text-gray-500 mt-1">
						Auto-delete: 0 means no automatic deletion (keep forever)
					</p>
				</div>
			</div>
		</div>
	);
}

interface CreatePolicyModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreatePolicyModal({ isOpen, onClose }: CreatePolicyModalProps) {
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [status, setStatus] = useState<LifecyclePolicyStatus>('draft');
	const [rules, setRules] = useState<ClassificationRetention[]>([
		{ level: 'public', retention: { min_days: 30, max_days: 90 } },
	]);

	const createPolicy = useCreateLifecyclePolicy();

	const addRule = () => {
		setRules([
			...rules,
			{ level: 'internal', retention: { min_days: 90, max_days: 365 } },
		]);
	};

	const updateRule = (index: number, rule: ClassificationRetention) => {
		const newRules = [...rules];
		newRules[index] = rule;
		setRules(newRules);
	};

	const removeRule = (index: number) => {
		setRules(rules.filter((_, i) => i !== index));
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const data: CreateLifecyclePolicyRequest = {
				name,
				description: description || undefined,
				status,
				rules,
			};
			await createPolicy.mutateAsync(data);
			onClose();
			// Reset form
			setName('');
			setDescription('');
			setStatus('draft');
			setRules([
				{ level: 'public', retention: { min_days: 30, max_days: 90 } },
			]);
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Create Lifecycle Policy
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="policy-name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="policy-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Standard Retention Policy"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 focus:ring-2 focus:ring-indigo-500"
								required
							/>
						</div>

						<div>
							<label
								htmlFor="policy-description"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Description
							</label>
							<textarea
								id="policy-description"
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								placeholder="Optional description of this policy"
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 focus:ring-2 focus:ring-indigo-500"
							/>
						</div>

						<div>
							<label
								htmlFor="policy-status"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Status
							</label>
							<select
								id="policy-status"
								value={status}
								onChange={(e) =>
									setStatus(e.target.value as LifecyclePolicyStatus)
								}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 focus:ring-2 focus:ring-indigo-500"
							>
								<option value="draft">Draft (not enforced)</option>
								<option value="active">Active (enforced)</option>
								<option value="disabled">Disabled</option>
							</select>
						</div>

						<div>
							<div className="flex items-center justify-between mb-2">
								<span className="block text-sm font-medium text-gray-700 dark:text-gray-300">
									Retention Rules by Classification
								</span>
								<button
									type="button"
									onClick={addRule}
									className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400"
								>
									+ Add Rule
								</button>
							</div>
							<div className="space-y-3">
								{rules.map((rule, index) => (
									<RetentionRuleEditor
										key={`${rule.level}-${index}`}
										rule={rule}
										onChange={(r) => updateRule(index, r)}
										onRemove={() => removeRule(index)}
									/>
								))}
							</div>
						</div>
					</div>

					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createPolicy.isPending || !name || rules.length === 0}
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

interface DryRunModalProps {
	policy: LifecyclePolicy | null;
	isOpen: boolean;
	onClose: () => void;
}

function DryRunModal({ policy, isOpen, onClose }: DryRunModalProps) {
	const dryRun = useLifecycleDryRun();
	const [result, setResult] = useState<LifecycleDryRunResult | null>(null);

	const handleDryRun = async () => {
		if (!policy) return;
		try {
			const res = await dryRun.mutateAsync(policy.id);
			setResult(res);
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen || !policy) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
					Dry Run: {policy.name}
				</h3>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					Preview what snapshots would be affected by this policy without making
					any changes.
				</p>

				{!result ? (
					<div className="text-center py-8">
						<button
							type="button"
							onClick={handleDryRun}
							disabled={dryRun.isPending}
							className="px-6 py-3 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{dryRun.isPending ? 'Evaluating...' : 'Run Preview'}
						</button>
					</div>
				) : (
					<div className="space-y-4">
						<div className="grid grid-cols-4 gap-4">
							<div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-gray-900 dark:text-white">
									{result.total_snapshots}
								</div>
								<div className="text-sm text-gray-500 dark:text-gray-400">
									Total Evaluated
								</div>
							</div>
							<div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-green-600 dark:text-green-400">
									{result.keep_count}
								</div>
								<div className="text-sm text-green-600 dark:text-green-400">
									Keep (Within Min)
								</div>
							</div>
							<div className="bg-yellow-50 dark:bg-yellow-900/20 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
									{result.can_delete_count}
								</div>
								<div className="text-sm text-yellow-600 dark:text-yellow-400">
									Can Delete
								</div>
							</div>
							<div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 text-center">
								<div className="text-2xl font-bold text-red-600 dark:text-red-400">
									{result.must_delete_count}
								</div>
								<div className="text-sm text-red-600 dark:text-red-400">
									Must Delete (Past Max)
								</div>
							</div>
						</div>

						{result.hold_count > 0 && (
							<div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4">
								<div className="flex items-center gap-2">
									<svg
										aria-hidden="true"
										className="w-5 h-5 text-amber-500"
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
									<span className="font-medium text-amber-800 dark:text-amber-200">
										{result.hold_count} snapshot(s) under legal hold - cannot be
										deleted
									</span>
								</div>
							</div>
						)}

						<div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4">
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Total space that would be reclaimed:{' '}
								<span className="font-medium text-gray-900 dark:text-white">
									{formatBytes(result.total_size_to_delete)}
								</span>
							</div>
						</div>

						{result.evaluations.length > 0 && (
							<div>
								<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
									Snapshot Details
								</h4>
								<div className="max-h-64 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded-lg">
									<table className="w-full text-sm">
										<thead className="bg-gray-50 dark:bg-gray-900 sticky top-0">
											<tr>
												<th className="px-4 py-2 text-left">Snapshot</th>
												<th className="px-4 py-2 text-left">Age</th>
												<th className="px-4 py-2 text-left">Classification</th>
												<th className="px-4 py-2 text-left">Action</th>
												<th className="px-4 py-2 text-left">Reason</th>
											</tr>
										</thead>
										<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
											{result.evaluations.slice(0, 50).map((eval_item) => (
												<tr key={eval_item.snapshot_id}>
													<td className="px-4 py-2 font-mono text-xs">
														{eval_item.snapshot_id.substring(0, 12)}...
													</td>
													<td className="px-4 py-2">
														{eval_item.snapshot_age_days}d
													</td>
													<td className="px-4 py-2 capitalize">
														{eval_item.classification_level}
													</td>
													<td className="px-4 py-2">
														<span
															className={`px-2 py-0.5 rounded-full text-xs ${
																eval_item.action === 'keep'
																	? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
																	: eval_item.action === 'can_delete'
																		? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
																		: eval_item.action === 'must_delete'
																			? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
																			: 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
															}`}
														>
															{eval_item.action.replace('_', ' ')}
														</span>
													</td>
													<td className="px-4 py-2 text-gray-500 dark:text-gray-400 text-xs max-w-xs truncate">
														{eval_item.reason}
													</td>
												</tr>
											))}
										</tbody>
									</table>
								</div>
								{result.evaluations.length > 50 && (
									<p className="text-xs text-gray-500 mt-2">
										Showing first 50 of {result.evaluations.length} snapshots
									</p>
								)}
							</div>
						)}
					</div>
				)}

				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={() => {
							setResult(null);
							onClose();
						}}
						className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

function getStatusColor(status: LifecyclePolicyStatus): string {
	switch (status) {
		case 'active':
			return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
		case 'draft':
			return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
		case 'disabled':
			return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
		default:
			return 'bg-gray-100 text-gray-800';
	}
}

export function LifecyclePolicies() {
	const { data: user } = useMe();
	const {
		data: policies,
		isLoading,
		isError,
		refetch,
	} = useLifecyclePolicies();
	const { data: deletions } = useOrgLifecycleDeletions(10);
	const deletePolicy = useDeleteLifecyclePolicy();
	const updatePolicy = useUpdateLifecyclePolicy();

	const [showCreateModal, setShowCreateModal] = useState(false);
	const [dryRunPolicy, setDryRunPolicy] = useState<LifecyclePolicy | null>(
		null,
	);
	const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	if (!isAdmin) {
		return (
			<div className="space-y-6">
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
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
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
						Access Denied
					</h3>
					<p className="text-gray-500 dark:text-gray-400">
						You must be an administrator to manage lifecycle policies.
					</p>
				</div>
			</div>
		);
	}

	const handleDelete = async (id: string) => {
		try {
			await deletePolicy.mutateAsync(id);
			setDeleteConfirm(null);
		} catch {
			// Error handling is managed by the mutation
		}
	};

	const toggleStatus = async (policy: LifecyclePolicy) => {
		const newStatus: LifecyclePolicyStatus =
			policy.status === 'active' ? 'disabled' : 'active';
		try {
			await updatePolicy.mutateAsync({
				id: policy.id,
				data: { status: newStatus },
			});
		} catch {
			// Error handling is managed by the mutation
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Lifecycle Policies
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Automated snapshot retention with compliance controls
					</p>
				</div>
				<div className="flex items-center gap-3">
					<button
						type="button"
						onClick={() => refetch()}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
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
						Refresh
					</button>
					<button
						type="button"
						onClick={() => setShowCreateModal(true)}
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
						Create Policy
					</button>
				</div>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load lifecycle policies</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
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
									Rules
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Stats
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
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
				) : policies && policies.length > 0 ? (
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
										Rules
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Stats
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{policies.map((policy) => (
									<tr
										key={policy.id}
										className="hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<td className="px-6 py-4">
											<div className="text-sm font-medium text-gray-900 dark:text-white">
												{policy.name}
											</div>
											{policy.description && (
												<div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
													{policy.description}
												</div>
											)}
										</td>
										<td className="px-6 py-4">
											<span
												className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(policy.status)}`}
											>
												{policy.status}
											</span>
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											{policy.rules.length} rule(s)
											<div className="text-xs mt-1">
												{policy.rules.map((r) => r.level).join(', ')}
											</div>
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											<div>{policy.deletion_count} deleted</div>
											<div className="text-xs">
												{formatBytes(policy.bytes_reclaimed)} reclaimed
											</div>
										</td>
										<td className="px-6 py-4">
											<div className="flex items-center gap-2">
												<button
													type="button"
													onClick={() => setDryRunPolicy(policy)}
													className="px-3 py-1 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/20 rounded"
												>
													Dry Run
												</button>
												<button
													type="button"
													onClick={() => toggleStatus(policy)}
													className={`px-3 py-1 text-sm rounded ${
														policy.status === 'active'
															? 'text-yellow-600 hover:bg-yellow-50 dark:text-yellow-400 dark:hover:bg-yellow-900/20'
															: 'text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/20'
													}`}
												>
													{policy.status === 'active' ? 'Disable' : 'Activate'}
												</button>
												{deleteConfirm === policy.id ? (
													<div className="flex items-center gap-1">
														<button
															type="button"
															onClick={() => handleDelete(policy.id)}
															disabled={deletePolicy.isPending}
															className="px-2 py-1 bg-red-600 text-white text-xs rounded hover:bg-red-700 disabled:opacity-50"
														>
															{deletePolicy.isPending ? '...' : 'Confirm'}
														</button>
														<button
															type="button"
															onClick={() => setDeleteConfirm(null)}
															className="px-2 py-1 border border-gray-300 text-gray-700 dark:border-gray-600 dark:text-gray-300 text-xs rounded hover:bg-gray-50 dark:hover:bg-gray-700"
														>
															Cancel
														</button>
													</div>
												) : (
													<button
														type="button"
														onClick={() => setDeleteConfirm(policy.id)}
														className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 rounded"
													>
														Delete
													</button>
												)}
											</div>
										</td>
									</tr>
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
								d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No Lifecycle Policies
						</h3>
						<p className="mb-4">
							Create a lifecycle policy to automate snapshot retention.
						</p>
						<button
							type="button"
							onClick={() => setShowCreateModal(true)}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Create Policy
						</button>
					</div>
				)}
			</div>

			{/* Recent Deletions */}
			{deletions && deletions.length > 0 && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Recent Deletions
						</h2>
					</div>
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Snapshot
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Reason
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Size
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Deleted At
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{deletions.map((event) => (
									<tr key={event.id}>
										<td className="px-6 py-4 font-mono text-sm text-gray-900 dark:text-white">
											{event.snapshot_id.substring(0, 12)}...
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-md truncate">
											{event.reason}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
											{formatBytes(event.size_bytes)}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
											{formatDateTime(event.deleted_at)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</div>
			)}

			{/* Info card */}
			<div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 p-4">
				<div className="flex items-start gap-3">
					<svg
						aria-hidden="true"
						className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<div className="text-sm text-blue-800 dark:text-blue-200">
						<p className="font-medium mb-1">About Lifecycle Policies</p>
						<ul className="list-disc list-inside space-y-1">
							<li>
								<strong>Minimum retention</strong> ensures compliance by
								preventing deletion before the required period.
							</li>
							<li>
								<strong>Maximum retention</strong> enables automatic cleanup of
								old snapshots to manage storage.
							</li>
							<li>
								Snapshots under <strong>legal hold</strong> are never deleted
								regardless of policy settings.
							</li>
							<li>
								Use <strong>dry run</strong> to preview what would be affected
								before activating a policy.
							</li>
						</ul>
					</div>
				</div>
			</div>

			<CreatePolicyModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			<DryRunModal
				policy={dryRunPolicy}
				isOpen={!!dryRunPolicy}
				onClose={() => setDryRunPolicy(null)}
			/>
		</div>
	);
}
