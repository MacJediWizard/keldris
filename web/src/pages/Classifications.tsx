import { useState } from 'react';
import {
	ClassificationBadge,
	ClassificationLevelSelect,
	DataTypeMultiSelect,
} from '../components/ClassificationBadge';
import {
	useAutoClassifySchedule,
	useClassificationRules,
	useClassificationSummary,
	useCreateClassificationRule,
	useDeleteClassificationRule,
	useScheduleClassifications,
	useUpdateClassificationRule,
} from '../hooks/useClassifications';
import type {
	ClassificationLevel,
	DataType,
	PathClassificationRule,
} from '../lib/types';

interface CreateRuleModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateRuleModal({ isOpen, onClose }: CreateRuleModalProps) {
	const [pattern, setPattern] = useState('');
	const [level, setLevel] = useState<ClassificationLevel>('public');
	const [dataTypes, setDataTypes] = useState<DataType[]>(['general']);
	const [description, setDescription] = useState('');

	const createRule = useCreateClassificationRule();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		await createRule.mutateAsync({
			pattern,
			level,
			data_types: dataTypes,
			description: description || undefined,
		});
		setPattern('');
		setLevel('public');
		setDataTypes(['general']);
		setDescription('');
		onClose();
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Create Classification Rule
				</h3>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div>
						<label
							htmlFor="pattern"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Path Pattern
						</label>
						<input
							id="pattern"
							type="text"
							value={pattern}
							onChange={(e) => setPattern(e.target.value)}
							placeholder="**/medical/**"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500"
							required
						/>
						<p className="mt-1 text-xs text-gray-500">
							Use glob patterns like **/pattern/** or /path/to/files/*
						</p>
					</div>

					<div>
						<label
							htmlFor="classification-level"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Classification Level
						</label>
						<ClassificationLevelSelect
							id="classification-level"
							value={level}
							onChange={(l) => setLevel(l as ClassificationLevel)}
						/>
					</div>

					<div>
						<span className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Data Types
						</span>
						<DataTypeMultiSelect
							value={dataTypes}
							onChange={(types) => setDataTypes(types as DataType[])}
						/>
					</div>

					<div>
						<label
							htmlFor="description"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Description (optional)
						</label>
						<input
							id="description"
							type="text"
							value={description}
							onChange={(e) => setDescription(e.target.value)}
							placeholder="Medical records and health information"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500"
						/>
					</div>

					<div className="flex justify-end gap-3 pt-4">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createRule.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50"
						>
							{createRule.isPending ? 'Creating...' : 'Create Rule'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface RuleRowProps {
	rule: PathClassificationRule;
	onDelete: (id: string) => void;
	onToggle: (id: string, enabled: boolean) => void;
}

function RuleRow({ rule, onDelete, onToggle }: RuleRowProps) {
	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<code className="text-sm font-mono bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
					{rule.pattern}
				</code>
				{rule.description && (
					<p className="text-sm text-gray-500 mt-1">{rule.description}</p>
				)}
			</td>
			<td className="px-6 py-4">
				<ClassificationBadge
					level={rule.level}
					dataTypes={rule.data_types}
					showDataTypes
					size="sm"
				/>
			</td>
			<td className="px-6 py-4">
				<button
					type="button"
					onClick={() => onToggle(rule.id, !rule.enabled)}
					disabled={rule.is_builtin}
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
						rule.enabled
							? 'bg-green-100 text-green-800'
							: 'bg-gray-100 text-gray-600'
					} ${rule.is_builtin ? 'opacity-50 cursor-not-allowed' : ''}`}
				>
					<span
						className={`w-1.5 h-1.5 rounded-full ${
							rule.enabled ? 'bg-green-500' : 'bg-gray-400'
						}`}
					/>
					{rule.enabled ? 'Active' : 'Disabled'}
				</button>
			</td>
			<td className="px-6 py-4 text-right">
				{!rule.is_builtin && (
					<button
						type="button"
						onClick={() => onDelete(rule.id)}
						className="text-red-600 hover:text-red-800 text-sm font-medium"
					>
						Delete
					</button>
				)}
				{rule.is_builtin && (
					<span className="text-xs text-gray-400">Built-in</span>
				)}
			</td>
		</tr>
	);
}

function ComplianceSummaryCard({
	title,
	count,
	color,
	icon,
}: {
	title: string;
	count: number;
	color: string;
	icon: React.ReactNode;
}) {
	return (
		<div className={`${color} rounded-lg p-4`}>
			<div className="flex items-center gap-3">
				<div className="flex-shrink-0">{icon}</div>
				<div>
					<p className="text-2xl font-bold">{count}</p>
					<p className="text-sm opacity-80">{title}</p>
				</div>
			</div>
		</div>
	);
}

export function Classifications() {
	const [activeTab, setActiveTab] = useState<'summary' | 'rules' | 'schedules'>(
		'summary',
	);
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [levelFilter, setLevelFilter] = useState<string>('');

	const { data: summary, isLoading: summaryLoading } =
		useClassificationSummary();
	const { data: rules, isLoading: rulesLoading } = useClassificationRules();
	const { data: schedules, isLoading: schedulesLoading } =
		useScheduleClassifications(levelFilter || undefined);

	const deleteRule = useDeleteClassificationRule();
	const updateRule = useUpdateClassificationRule();
	const autoClassify = useAutoClassifySchedule();

	const handleDeleteRule = (id: string) => {
		if (confirm('Are you sure you want to delete this rule?')) {
			deleteRule.mutate(id);
		}
	};

	const handleToggleRule = (id: string, enabled: boolean) => {
		updateRule.mutate({ id, data: { enabled } });
	};

	const handleAutoClassify = (scheduleId: string) => {
		autoClassify.mutate(scheduleId);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Data Classification
					</h1>
					<p className="text-gray-500 dark:text-gray-400 mt-1">
						Manage sensitive data tagging and compliance reporting
					</p>
				</div>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('summary')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'summary'
								? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Compliance Summary
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('rules')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'rules'
								? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Classification Rules
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('schedules')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'schedules'
								? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Schedule Classifications
					</button>
				</nav>
			</div>

			{/* Summary Tab */}
			{activeTab === 'summary' && (
				<div className="space-y-6">
					{summaryLoading ? (
						<div className="animate-pulse space-y-4">
							<div className="grid grid-cols-4 gap-4">
								{[1, 2, 3, 4].map((i) => (
									<div
										key={i}
										className="h-24 bg-gray-200 dark:bg-gray-700 rounded-lg"
									/>
								))}
							</div>
						</div>
					) : summary ? (
						<>
							<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
								<ComplianceSummaryCard
									title="Restricted"
									count={summary.restricted_count}
									color="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300"
									icon={
										<svg
											className="w-8 h-8"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
											/>
										</svg>
									}
								/>
								<ComplianceSummaryCard
									title="Confidential"
									count={summary.confidential_count}
									color="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300"
									icon={
										<svg
											className="w-8 h-8"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
											/>
										</svg>
									}
								/>
								<ComplianceSummaryCard
									title="Internal"
									count={summary.internal_count}
									color="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300"
									icon={
										<svg
											className="w-8 h-8"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
											/>
										</svg>
									}
								/>
								<ComplianceSummaryCard
									title="Public"
									count={summary.public_count}
									color="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300"
									icon={
										<svg
											className="w-8 h-8"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
											/>
										</svg>
									}
								/>
							</div>

							<div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
								<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
									Classification Overview
								</h3>
								<div className="grid grid-cols-2 gap-6">
									<div>
										<p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
											Total Schedules
										</p>
										<p className="text-3xl font-bold text-gray-900 dark:text-white">
											{summary.total_schedules}
										</p>
									</div>
									<div>
										<p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
											Total Backups
										</p>
										<p className="text-3xl font-bold text-gray-900 dark:text-white">
											{summary.total_backups}
										</p>
									</div>
								</div>
							</div>
						</>
					) : (
						<div className="text-center py-12 text-gray-500">
							No classification data available
						</div>
					)}
				</div>
			)}

			{/* Rules Tab */}
			{activeTab === 'rules' && (
				<div className="space-y-4">
					<div className="flex justify-end">
						<button
							type="button"
							onClick={() => setShowCreateModal(true)}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 flex items-center gap-2"
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
							Add Rule
						</button>
					</div>

					<div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
						{rulesLoading ? (
							<div className="p-8 text-center">
								<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto" />
							</div>
						) : rules && rules.length > 0 ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Pattern
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Classification
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
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
											onDelete={handleDeleteRule}
											onToggle={handleToggleRule}
										/>
									))}
								</tbody>
							</table>
						) : (
							<div className="p-8 text-center text-gray-500">
								No custom classification rules defined. Add rules to
								automatically classify backup paths.
							</div>
						)}
					</div>
				</div>
			)}

			{/* Schedules Tab */}
			{activeTab === 'schedules' && (
				<div className="space-y-4">
					<div className="flex gap-4">
						<select
							value={levelFilter}
							onChange={(e) => setLevelFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg"
						>
							<option value="">All Levels</option>
							<option value="public">Public</option>
							<option value="internal">Internal</option>
							<option value="confidential">Confidential</option>
							<option value="restricted">Restricted</option>
						</select>
					</div>

					<div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
						{schedulesLoading ? (
							<div className="p-8 text-center">
								<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto" />
							</div>
						) : schedules && schedules.length > 0 ? (
							<table className="w-full">
								<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
									<tr>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Schedule
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Paths
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Classification
										</th>
										<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Actions
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
									{schedules.map((schedule) => (
										<tr
											key={schedule.id}
											className="hover:bg-gray-50 dark:hover:bg-gray-700"
										>
											<td className="px-6 py-4">
												<div className="font-medium text-gray-900 dark:text-white">
													{schedule.name}
												</div>
											</td>
											<td className="px-6 py-4">
												<div className="text-sm text-gray-500 dark:text-gray-400">
													{schedule.paths.slice(0, 2).join(', ')}
													{schedule.paths.length > 2 && (
														<span className="text-gray-400">
															{' '}
															+{schedule.paths.length - 2} more
														</span>
													)}
												</div>
											</td>
											<td className="px-6 py-4">
												<ClassificationBadge
													level={schedule.classification_level || 'public'}
													dataTypes={schedule.classification_data_types}
													showDataTypes
													size="sm"
												/>
											</td>
											<td className="px-6 py-4 text-right">
												<button
													type="button"
													onClick={() => handleAutoClassify(schedule.id)}
													disabled={autoClassify.isPending}
													className="text-indigo-600 hover:text-indigo-800 text-sm font-medium disabled:opacity-50"
												>
													Auto-Classify
												</button>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						) : (
							<div className="p-8 text-center text-gray-500">
								No schedules found matching the filter criteria.
							</div>
						)}
					</div>
				</div>
			)}

			<CreateRuleModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>
		</div>
	);
}
