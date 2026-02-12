import { useState } from 'react';
import {
	useCreateSLAPolicy,
	useDeleteSLAPolicy,
	useSLAPolicies,
	useSLAPolicyHistory,
	useSLAPolicyStatus,
} from '../hooks/useSLAPolicies';
import type { SLAPolicy } from '../lib/types';
import { formatDateTime } from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse">
			<div className="h-5 w-3/4 bg-gray-200 dark:bg-gray-700 rounded mb-4" />
			<div className="grid grid-cols-3 gap-4">
				<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
				<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
				<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
			</div>
		</div>
	);
}

interface PolicyStatusBadgeProps {
	policyId: string;
}

function PolicyStatusBadge({ policyId }: PolicyStatusBadgeProps) {
	const { data: status, isLoading } = useSLAPolicyStatus(policyId);

	if (isLoading) {
		return (
			<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
				Loading...
			</span>
		);
	}

	if (!status) {
		return (
			<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
				No data
			</span>
		);
	}

	return (
		<span
			className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium ${
				status.compliant
					? 'bg-green-100 text-green-800'
					: 'bg-red-100 text-red-800'
			}`}
		>
			<span
				className={`w-1.5 h-1.5 rounded-full ${status.compliant ? 'bg-green-500' : 'bg-red-500'}`}
			/>
			{status.compliant ? 'Compliant' : 'Non-Compliant'}
		</span>
	);
}

interface PolicyStatusDetailsProps {
	policyId: string;
	policy: SLAPolicy;
}

function PolicyStatusDetails({ policyId, policy }: PolicyStatusDetailsProps) {
	const { data: status } = useSLAPolicyStatus(policyId);

	if (!status) return null;

	return (
		<div className="mt-4 grid grid-cols-3 gap-4">
			<div className="text-center p-3 rounded-lg bg-gray-50 dark:bg-gray-700">
				<div className="text-sm text-gray-500 dark:text-gray-400">RPO</div>
				<div
					className={`text-lg font-semibold ${status.current_rpo_hours <= policy.target_rpo_hours ? 'text-green-600' : 'text-red-600'}`}
				>
					{status.current_rpo_hours.toFixed(1)}h
				</div>
				<div className="text-xs text-gray-400">
					Target: {policy.target_rpo_hours}h
				</div>
			</div>
			<div className="text-center p-3 rounded-lg bg-gray-50 dark:bg-gray-700">
				<div className="text-sm text-gray-500 dark:text-gray-400">RTO</div>
				<div
					className={`text-lg font-semibold ${status.current_rto_hours <= policy.target_rto_hours ? 'text-green-600' : 'text-red-600'}`}
				>
					{status.current_rto_hours.toFixed(1)}h
				</div>
				<div className="text-xs text-gray-400">
					Target: {policy.target_rto_hours}h
				</div>
			</div>
			<div className="text-center p-3 rounded-lg bg-gray-50 dark:bg-gray-700">
				<div className="text-sm text-gray-500 dark:text-gray-400">
					Success Rate
				</div>
				<div
					className={`text-lg font-semibold ${status.success_rate >= policy.target_success_rate ? 'text-green-600' : 'text-red-600'}`}
				>
					{status.success_rate.toFixed(1)}%
				</div>
				<div className="text-xs text-gray-400">
					Target: {policy.target_success_rate}%
				</div>
			</div>
		</div>
	);
}

interface PolicyHistoryProps {
	policyId: string;
}

function PolicyHistory({ policyId }: PolicyHistoryProps) {
	const { data: history, isLoading } = useSLAPolicyHistory(policyId);

	if (isLoading) {
		return (
			<div className="mt-4 animate-pulse">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
				<div className="h-20 bg-gray-200 dark:bg-gray-700 rounded" />
			</div>
		);
	}

	if (!history || history.length === 0) {
		return (
			<div className="mt-4 text-sm text-gray-500 dark:text-gray-400">
				No compliance history available yet.
			</div>
		);
	}

	return (
		<div className="mt-4">
			<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
				Compliance History
			</h4>
			<div className="overflow-x-auto">
				<table className="min-w-full text-sm">
					<thead>
						<tr className="border-b border-gray-200 dark:border-gray-700">
							<th className="text-left py-2 pr-4 font-medium text-gray-500 dark:text-gray-400">
								Date
							</th>
							<th className="text-right py-2 px-4 font-medium text-gray-500 dark:text-gray-400">
								RPO
							</th>
							<th className="text-right py-2 px-4 font-medium text-gray-500 dark:text-gray-400">
								RTO
							</th>
							<th className="text-right py-2 px-4 font-medium text-gray-500 dark:text-gray-400">
								Success
							</th>
							<th className="text-right py-2 pl-4 font-medium text-gray-500 dark:text-gray-400">
								Status
							</th>
						</tr>
					</thead>
					<tbody>
						{history.slice(0, 10).map((snap) => (
							<tr
								key={snap.id}
								className="border-b border-gray-100 dark:border-gray-800"
							>
								<td className="py-2 pr-4 text-gray-700 dark:text-gray-300">
									{formatDateTime(snap.calculated_at)}
								</td>
								<td className="py-2 px-4 text-right text-gray-700 dark:text-gray-300">
									{snap.rpo_hours.toFixed(1)}h
								</td>
								<td className="py-2 px-4 text-right text-gray-700 dark:text-gray-300">
									{snap.rto_hours.toFixed(1)}h
								</td>
								<td className="py-2 px-4 text-right text-gray-700 dark:text-gray-300">
									{snap.success_rate.toFixed(1)}%
								</td>
								<td className="py-2 pl-4 text-right">
									<span
										className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
											snap.compliant
												? 'bg-green-100 text-green-800'
												: 'bg-red-100 text-red-800'
										}`}
									>
										{snap.compliant ? 'Pass' : 'Fail'}
									</span>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			</div>
		</div>
	);
}

interface PolicyCardProps {
	policy: SLAPolicy;
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

function PolicyCard({ policy, onDelete, isDeleting }: PolicyCardProps) {
	const [expanded, setExpanded] = useState(false);

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-3">
					<h3 className="font-medium text-gray-900 dark:text-white">
						{policy.name}
					</h3>
					<PolicyStatusBadge policyId={policy.id} />
					{!policy.enabled && (
						<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
							Disabled
						</span>
					)}
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => setExpanded(!expanded)}
						className="px-3 py-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-800 transition-colors"
					>
						{expanded ? 'Collapse' : 'Details'}
					</button>
					<button
						type="button"
						onClick={() => onDelete(policy.id)}
						disabled={isDeleting}
						className="px-3 py-1.5 text-sm font-medium text-red-600 hover:text-red-800 transition-colors disabled:opacity-50"
					>
						Delete
					</button>
				</div>
			</div>
			{policy.description && (
				<p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
					{policy.description}
				</p>
			)}
			<div className="mt-3 grid grid-cols-3 gap-4 text-sm">
				<div>
					<span className="text-gray-500 dark:text-gray-400">Target RPO:</span>{' '}
					<span className="font-medium text-gray-900 dark:text-white">
						{policy.target_rpo_hours}h
					</span>
				</div>
				<div>
					<span className="text-gray-500 dark:text-gray-400">Target RTO:</span>{' '}
					<span className="font-medium text-gray-900 dark:text-white">
						{policy.target_rto_hours}h
					</span>
				</div>
				<div>
					<span className="text-gray-500 dark:text-gray-400">
						Target Success:
					</span>{' '}
					<span className="font-medium text-gray-900 dark:text-white">
						{policy.target_success_rate}%
					</span>
				</div>
			</div>

			{expanded && (
				<>
					<PolicyStatusDetails policyId={policy.id} policy={policy} />
					<PolicyHistory policyId={policy.id} />
				</>
			)}
		</div>
	);
}

interface CreatePolicyFormProps {
	onSuccess: () => void;
	onCancel: () => void;
}

function CreatePolicyForm({ onSuccess, onCancel }: CreatePolicyFormProps) {
	const createPolicy = useCreateSLAPolicy();
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [targetRPO, setTargetRPO] = useState('24');
	const [targetRTO, setTargetRTO] = useState('4');
	const [targetSuccess, setTargetSuccess] = useState('99');

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		createPolicy.mutate(
			{
				name,
				description: description || undefined,
				target_rpo_hours: Number.parseFloat(targetRPO),
				target_rto_hours: Number.parseFloat(targetRTO),
				target_success_rate: Number.parseFloat(targetSuccess),
			},
			{
				onSuccess: () => {
					onSuccess();
				},
			},
		);
	};

	return (
		<form
			onSubmit={handleSubmit}
			className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6"
		>
			<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
				Create SLA Policy
			</h3>
			<div className="space-y-4">
				<div>
					<label
						htmlFor="sla-name"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Policy Name
					</label>
					<input
						id="sla-name"
						type="text"
						value={name}
						onChange={(e) => setName(e.target.value)}
						required
						className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="Gold SLA"
					/>
				</div>
				<div>
					<label
						htmlFor="sla-description"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Description
					</label>
					<input
						id="sla-description"
						type="text"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
						className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="Primary production SLA policy"
					/>
				</div>
				<div className="grid grid-cols-3 gap-4">
					<div>
						<label
							htmlFor="sla-rpo"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Target RPO (hours)
						</label>
						<input
							id="sla-rpo"
							type="number"
							step="0.5"
							min="0.5"
							value={targetRPO}
							onChange={(e) => setTargetRPO(e.target.value)}
							required
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div>
						<label
							htmlFor="sla-rto"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Target RTO (hours)
						</label>
						<input
							id="sla-rto"
							type="number"
							step="0.5"
							min="0.5"
							value={targetRTO}
							onChange={(e) => setTargetRTO(e.target.value)}
							required
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
					<div>
						<label
							htmlFor="sla-success"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Target Success Rate (%)
						</label>
						<input
							id="sla-success"
							type="number"
							step="0.1"
							min="0"
							max="100"
							value={targetSuccess}
							onChange={(e) => setTargetSuccess(e.target.value)}
							required
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</div>
			</div>
			<div className="mt-6 flex items-center gap-3">
				<button
					type="submit"
					disabled={createPolicy.isPending || !name}
					className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 font-medium"
				>
					{createPolicy.isPending ? 'Creating...' : 'Create Policy'}
				</button>
				<button
					type="button"
					onClick={onCancel}
					className="px-4 py-2 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium"
				>
					Cancel
				</button>
			</div>
			{createPolicy.isError && (
				<p className="mt-2 text-sm text-red-600">
					Failed to create policy. Please try again.
				</p>
			)}
		</form>
	);
}

export function SLATracking() {
	const [showCreate, setShowCreate] = useState(false);
	const { data: policies, isLoading, isError } = useSLAPolicies();
	const deletePolicy = useDeleteSLAPolicy();

	const handleDelete = (id: string) => {
		deletePolicy.mutate(id);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						SLA Tracking
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Monitor backup service level agreements
					</p>
				</div>
				{!showCreate && (
					<button
						type="button"
						onClick={() => setShowCreate(true)}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors font-medium"
					>
						Create Policy
					</button>
				)}
			</div>

			{showCreate && (
				<CreatePolicyForm
					onSuccess={() => setShowCreate(false)}
					onCancel={() => setShowCreate(false)}
				/>
			)}

			{isError ? (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
					<p className="font-medium text-red-500 dark:text-red-400">
						Failed to load SLA policies
					</p>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Please try refreshing the page
					</p>
				</div>
			) : isLoading ? (
				<div className="space-y-4">
					<LoadingCard />
					<LoadingCard />
				</div>
			) : policies && policies.length > 0 ? (
				<div className="space-y-4">
					{policies.map((policy) => (
						<PolicyCard
							key={policy.id}
							policy={policy}
							onDelete={handleDelete}
							isDeleting={deletePolicy.isPending}
						/>
					))}
				</div>
			) : (
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
							d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
						No SLA policies
					</h3>
					<p className="text-gray-500 dark:text-gray-400">
						Create your first SLA policy to start tracking backup compliance
					</p>
				</div>
			)}
		</div>
	);
}

export default SLATracking;
