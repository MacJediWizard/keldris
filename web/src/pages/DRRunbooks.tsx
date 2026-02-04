import { useState } from 'react';
import {
	useActivateDRRunbook,
	useArchiveDRRunbook,
	useCreateDRRunbook,
	useDeleteDRRunbook,
	useDRRunbooks,
	useDRStatus,
	useGenerateDRRunbook,
	useRenderDRRunbook,
} from '../hooks/useDRRunbooks';
import { useDRTests, useRunDRTest } from '../hooks/useDRTests';
import { useSchedules } from '../hooks/useSchedules';
import type { DRRunbook, DRRunbookStatus } from '../lib/types';
import { formatDate } from '../lib/utils';

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

interface CreateRunbookModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateRunbookModal({ isOpen, onClose }: CreateRunbookModalProps) {
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [scheduleId, setScheduleId] = useState('');
	const [useSchedule, setUseSchedule] = useState(false);

	const { data: schedules } = useSchedules();
	const createRunbook = useCreateDRRunbook();
	const generateRunbook = useGenerateDRRunbook();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			if (useSchedule && scheduleId) {
				await generateRunbook.mutateAsync(scheduleId);
			} else {
				await createRunbook.mutateAsync({
					name,
					description,
					schedule_id: scheduleId || undefined,
				});
			}
			onClose();
			setName('');
			setDescription('');
			setScheduleId('');
			setUseSchedule(false);
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	const isPending = createRunbook.isPending || generateRunbook.isPending;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Create DR Runbook
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="use-schedule"
								checked={useSchedule}
								onChange={(e) => setUseSchedule(e.target.checked)}
								className="rounded border-gray-300"
							/>
							<label htmlFor="use-schedule" className="text-sm text-gray-700">
								Generate from existing backup schedule
							</label>
						</div>

						{useSchedule ? (
							<div>
								<label
									htmlFor="schedule"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Backup Schedule
								</label>
								<select
									id="schedule"
									value={scheduleId}
									onChange={(e) => setScheduleId(e.target.value)}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									required
								>
									<option value="">Select a schedule</option>
									{schedules?.map((schedule) => (
										<option key={schedule.id} value={schedule.id}>
											{schedule.name}
										</option>
									))}
								</select>
								<p className="text-xs text-gray-500 mt-1">
									A runbook will be automatically generated with restore steps
								</p>
							</div>
						) : (
							<>
								<div>
									<label
										htmlFor="name"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Name
									</label>
									<input
										type="text"
										id="name"
										value={name}
										onChange={(e) => setName(e.target.value)}
										placeholder="e.g., Production Database Recovery"
										className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										required
									/>
								</div>
								<div>
									<label
										htmlFor="description"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Description
									</label>
									<textarea
										id="description"
										value={description}
										onChange={(e) => setDescription(e.target.value)}
										placeholder="Describe the purpose and scope of this runbook"
										rows={3}
										className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
								<div>
									<label
										htmlFor="schedule-optional"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Associated Schedule (optional)
									</label>
									<select
										id="schedule-optional"
										value={scheduleId}
										onChange={(e) => setScheduleId(e.target.value)}
										className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									>
										<option value="">None</option>
										{schedules?.map((schedule) => (
											<option key={schedule.id} value={schedule.id}>
												{schedule.name}
											</option>
										))}
									</select>
								</div>
							</>
						)}
					</div>

					{(createRunbook.isError || generateRunbook.isError) && (
						<p className="text-sm text-red-600 mt-4">
							Failed to create runbook. Please try again.
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
							disabled={isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{isPending ? 'Creating...' : 'Create Runbook'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ViewRunbookModalProps {
	isOpen: boolean;
	onClose: () => void;
	runbookId: string;
}

function ViewRunbookModal({
	isOpen,
	onClose,
	runbookId,
}: ViewRunbookModalProps) {
	const { data: renderData, isLoading } = useRenderDRRunbook(
		isOpen ? runbookId : '',
	);

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						DR Runbook Document
					</h3>
					<button
						type="button"
						onClick={onClose}
						aria-label="Close"
						className="text-gray-500 hover:text-gray-700"
					>
						<svg
							aria-hidden="true"
							className="w-6 h-6"
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
					</button>
				</div>

				{isLoading ? (
					<div className="animate-pulse space-y-4">
						<div className="h-8 bg-gray-200 rounded w-1/3" />
						<div className="h-4 bg-gray-200 rounded w-full" />
						<div className="h-4 bg-gray-200 rounded w-2/3" />
						<div className="h-4 bg-gray-200 rounded w-3/4" />
					</div>
				) : renderData ? (
					<div className="prose max-w-none">
						<pre className="whitespace-pre-wrap bg-gray-50 p-4 rounded-lg text-sm font-mono overflow-x-auto">
							{renderData.content}
						</pre>
					</div>
				) : (
					<p className="text-gray-500">Failed to load runbook content.</p>
				)}

				<div className="flex justify-end gap-3 mt-6">
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

function getStatusColor(status: DRRunbookStatus) {
	switch (status) {
		case 'active':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'draft':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		case 'archived':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

interface RunbookRowProps {
	runbook: DRRunbook;
	onView: (id: string) => void;
	onActivate: (id: string) => void;
	onArchive: (id: string) => void;
	onDelete: (id: string) => void;
	onRunTest: (id: string) => void;
	isUpdating: boolean;
	isDeleting: boolean;
}

function RunbookRow({
	runbook,
	onView,
	onActivate,
	onArchive,
	onDelete,
	onRunTest,
	isUpdating,
	isDeleting,
}: RunbookRowProps) {
	const statusColor = getStatusColor(runbook.status);

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900">{runbook.name}</div>
				{runbook.description && (
					<div className="text-sm text-gray-500 truncate max-w-xs">
						{runbook.description}
					</div>
				)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{runbook.steps.length} steps
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(runbook.updated_at)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{runbook.status}
				</span>
			</td>
			<td className="px-6 py-4 text-right">
				<div className="flex items-center justify-end gap-2">
					<button
						type="button"
						onClick={() => onView(runbook.id)}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						View
					</button>
					<span className="text-gray-300">|</span>
					<button
						type="button"
						onClick={() => onRunTest(runbook.id)}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						Test
					</button>
					<span className="text-gray-300">|</span>
					{runbook.status === 'draft' && (
						<>
							<button
								type="button"
								onClick={() => onActivate(runbook.id)}
								disabled={isUpdating}
								className="text-green-600 hover:text-green-800 text-sm font-medium disabled:opacity-50"
							>
								Activate
							</button>
							<span className="text-gray-300">|</span>
						</>
					)}
					{runbook.status === 'active' && (
						<>
							<button
								type="button"
								onClick={() => onArchive(runbook.id)}
								disabled={isUpdating}
								className="text-gray-600 hover:text-gray-800 text-sm font-medium disabled:opacity-50"
							>
								Archive
							</button>
							<span className="text-gray-300">|</span>
						</>
					)}
					<button
						type="button"
						onClick={() => onDelete(runbook.id)}
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

export function DRRunbooks() {
	const [searchQuery, setSearchQuery] = useState('');
	const [statusFilter, setStatusFilter] = useState<'all' | DRRunbookStatus>(
		'all',
	);
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [viewRunbookId, setViewRunbookId] = useState<string | null>(null);

	const { data: runbooks, isLoading, isError } = useDRRunbooks();
	const { data: drStatus } = useDRStatus();
	const { data: tests } = useDRTests();
	const activateRunbook = useActivateDRRunbook();
	const archiveRunbook = useArchiveDRRunbook();
	const deleteRunbook = useDeleteDRRunbook();
	const runTest = useRunDRTest();

	const filteredRunbooks = runbooks?.filter((runbook) => {
		const matchesSearch = runbook.name
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesStatus =
			statusFilter === 'all' || runbook.status === statusFilter;
		return matchesSearch && matchesStatus;
	});

	const handleActivate = (id: string) => {
		activateRunbook.mutate(id);
	};

	const handleArchive = (id: string) => {
		archiveRunbook.mutate(id);
	};

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this runbook?')) {
			deleteRunbook.mutate(id);
		}
	};

	const handleRunTest = (id: string) => {
		if (confirm('Run a DR test for this runbook?')) {
			runTest.mutate({ runbook_id: id });
		}
	};

	const recentTests = tests?.slice(0, 3) ?? [];

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">DR Runbooks</h1>
					<p className="text-gray-600 mt-1">
						Disaster recovery procedures and testing
					</p>
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
					Create Runbook
				</button>
			</div>

			{drStatus && (
				<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
					<div className="bg-white rounded-lg border border-gray-200 p-4">
						<p className="text-sm text-gray-600">Total Runbooks</p>
						<p className="text-2xl font-bold text-gray-900">
							{drStatus.total_runbooks}
						</p>
					</div>
					<div className="bg-white rounded-lg border border-gray-200 p-4">
						<p className="text-sm text-gray-600">Active Runbooks</p>
						<p className="text-2xl font-bold text-green-600">
							{drStatus.active_runbooks}
						</p>
					</div>
					<div className="bg-white rounded-lg border border-gray-200 p-4">
						<p className="text-sm text-gray-600">Tests (30 days)</p>
						<p className="text-2xl font-bold text-gray-900">
							{drStatus.tests_last_30_days}
						</p>
					</div>
					<div className="bg-white rounded-lg border border-gray-200 p-4">
						<p className="text-sm text-gray-600">Pass Rate</p>
						<p className="text-2xl font-bold text-gray-900">
							{drStatus.pass_rate.toFixed(1)}%
						</p>
					</div>
				</div>
			)}

			{recentTests.length > 0 && (
				<div className="bg-white rounded-lg border border-gray-200 p-4">
					<h3 className="font-medium text-gray-900 mb-3">Recent Tests</h3>
					<div className="space-y-2">
						{recentTests.map((test) => (
							<div
								key={test.id}
								className="flex items-center justify-between text-sm"
							>
								<span className="text-gray-600">
									{formatDate(test.created_at)}
								</span>
								<span
									className={`px-2 py-0.5 rounded-full text-xs font-medium ${
										test.status === 'completed'
											? 'bg-green-100 text-green-800'
											: test.status === 'failed'
												? 'bg-red-100 text-red-800'
												: test.status === 'running'
													? 'bg-blue-100 text-blue-800'
													: 'bg-gray-100 text-gray-600'
									}`}
								>
									{test.status}
								</span>
							</div>
						))}
					</div>
				</div>
			)}

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search runbooks..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as 'all' | DRRunbookStatus)
							}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="draft">Draft</option>
							<option value="active">Active</option>
							<option value="archived">Archived</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">Failed to load runbooks</p>
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
									Steps
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Last Updated
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
				) : filteredRunbooks && filteredRunbooks.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Runbook
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Steps
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Last Updated
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
							{filteredRunbooks.map((runbook) => (
								<RunbookRow
									key={runbook.id}
									runbook={runbook}
									onView={setViewRunbookId}
									onActivate={handleActivate}
									onArchive={handleArchive}
									onDelete={handleDelete}
									onRunTest={handleRunTest}
									isUpdating={
										activateRunbook.isPending || archiveRunbook.isPending
									}
									isDeleting={deleteRunbook.isPending}
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
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 mb-2">
							No DR runbooks configured
						</h3>
						<p className="mb-4">
							Create a runbook to document your recovery procedures
						</p>
					</div>
				)}
			</div>

			<CreateRunbookModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			{viewRunbookId && (
				<ViewRunbookModal
					isOpen={!!viewRunbookId}
					onClose={() => setViewRunbookId(null)}
					runbookId={viewRunbookId}
				/>
			)}
		</div>
	);
}
