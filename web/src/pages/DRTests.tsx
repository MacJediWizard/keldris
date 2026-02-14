import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useCancelDRTest, useDRTests } from '../hooks/useDRTests';
import type { DRTest, DRTestStatus } from '../lib/types';
import { formatDate, formatDateTime, formatDuration } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-20 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

function getStatusColor(status: DRTestStatus) {
	switch (status) {
		case 'passed':
		case 'completed':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'failed':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'running':
			return {
				bg: 'bg-blue-100',
				text: 'text-blue-800',
				dot: 'bg-blue-500',
			};
		case 'pending':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		case 'skipped':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

function getDurationMinutes(test: DRTest): string | null {
	if (!test.started_at || !test.completed_at) return null;
	const start = new Date(test.started_at).getTime();
	const end = new Date(test.completed_at).getTime();
	const diffMs = end - start;
	if (diffMs < 0) return null;
	return formatDuration(diffMs);
}

interface TestRowProps {
	test: DRTest;
	onCancel: (id: string) => void;
	isCancelling: boolean;
}

function TestRow({ test, onCancel, isCancelling }: TestRowProps) {
	const statusColor = getStatusColor(test.status);
	const duration = getDurationMinutes(test);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				{test.runbook_name ? (
					<Link
						to="/dr-runbooks"
						className="font-medium text-indigo-600 hover:text-indigo-800"
					>
						{test.runbook_name}
					</Link>
				) : (
					<span className="text-sm text-gray-500">
						{test.runbook_id.slice(0, 8)}...
					</span>
				)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{test.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{test.started_at ? (
					<span title={formatDateTime(test.started_at)}>
						{formatDate(test.started_at)}
					</span>
				) : (
					'Not started'
				)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">{duration ?? '-'}</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{test.actual_rto_minutes != null && (
					<span className="mr-3">RTO: {test.actual_rto_minutes}m</span>
				)}
				{test.actual_rpo_minutes != null && (
					<span>RPO: {test.actual_rpo_minutes}m</span>
				)}
				{test.actual_rto_minutes == null &&
					test.actual_rpo_minutes == null &&
					'-'}
			</td>
			<td className="px-6 py-4 text-right">
				{(test.status === 'pending' || test.status === 'running') && (
					<button
						type="button"
						onClick={() => onCancel(test.id)}
						disabled={isCancelling}
						className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
					>
						Cancel
					</button>
				)}
				{test.notes && (
					<span title={test.notes} className="ml-2 text-gray-400 cursor-help">
						<svg
							aria-hidden="true"
							className="w-4 h-4 inline"
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
					</span>
				)}
			</td>
		</tr>
	);
}

export function DRTests() {
	const [statusFilter, setStatusFilter] = useState<DRTestStatus | 'all'>('all');
	const [dateFilter, setDateFilter] = useState<string>('all');

	const {
		data: tests,
		isLoading,
		isError,
	} = useDRTests(statusFilter !== 'all' ? { status: statusFilter } : undefined);
	const cancelTest = useCancelDRTest();

	const filteredTests = tests?.filter((test) => {
		if (dateFilter === 'all') return true;
		const testDate = new Date(test.created_at);
		const now = new Date();
		switch (dateFilter) {
			case '7d':
				return now.getTime() - testDate.getTime() <= 7 * 86400000;
			case '30d':
				return now.getTime() - testDate.getTime() <= 30 * 86400000;
			case '90d':
				return now.getTime() - testDate.getTime() <= 90 * 86400000;
			default:
				return true;
		}
	});

	const passedCount =
		tests?.filter((t) => t.status === 'passed' || t.status === 'completed')
			.length ?? 0;
	const failedCount = tests?.filter((t) => t.status === 'failed').length ?? 0;
	const runningCount = tests?.filter((t) => t.status === 'running').length ?? 0;

	const handleCancel = (id: string) => {
		cancelTest.mutate({ id });
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						DR Tests
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Disaster recovery test results and history
					</p>
				</div>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
				<button
					type="button"
					onClick={() =>
						setStatusFilter(statusFilter === 'passed' ? 'all' : 'passed')
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'passed'
							? 'bg-green-50 border-green-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-green-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-green-600"
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
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{passedCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Passed
							</div>
						</div>
					</div>
				</button>

				<button
					type="button"
					onClick={() =>
						setStatusFilter(statusFilter === 'failed' ? 'all' : 'failed')
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'failed'
							? 'bg-red-50 border-red-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-red-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-red-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{failedCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Failed
							</div>
						</div>
					</div>
				</button>

				<button
					type="button"
					onClick={() =>
						setStatusFilter(statusFilter === 'running' ? 'all' : 'running')
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'running'
							? 'bg-blue-50 border-blue-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-blue-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-blue-600"
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
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{runningCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Running
							</div>
						</div>
					</div>
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-4 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as DRTestStatus | 'all')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="pending">Pending</option>
							<option value="running">Running</option>
							<option value="completed">Completed</option>
							<option value="passed">Passed</option>
							<option value="failed">Failed</option>
							<option value="skipped">Skipped</option>
						</select>
						<select
							value={dateFilter}
							onChange={(e) => setDateFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Time</option>
							<option value="7d">Last 7 days</option>
							<option value="30d">Last 30 days</option>
							<option value="90d">Last 90 days</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load DR tests</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Runbook
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Started
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Duration
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
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : filteredTests && filteredTests.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Runbook
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Started
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Duration
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
							{filteredTests.map((test) => (
								<TestRow
									key={test.id}
									test={test}
									onCancel={handleCancel}
									isCancelling={cancelTest.isPending}
								/>
							))}
						</tbody>
					</table>
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
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							{statusFilter === 'all' && dateFilter === 'all'
								? 'No DR tests'
								: 'No matching tests'}
						</h3>
						<p>
							{statusFilter === 'all' && dateFilter === 'all'
								? 'Run a test from the DR Runbooks page to get started'
								: 'Try adjusting your filters'}
						</p>
					</div>
				)}
			</div>
		</div>
	);
}

export default DRTests;
