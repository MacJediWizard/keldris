import { useState } from 'react';
import {
	useCreateReportSchedule,
	useDeleteReportSchedule,
	usePreviewReport,
	useReportHistory,
	useReportSchedules,
	useSendReport,
	useUpdateReportSchedule,
} from '../hooks/useReports';
import { useNotificationChannels } from '../hooks/useNotifications';
import type {
	ReportData,
	ReportFrequency,
	ReportHistory,
	ReportSchedule,
} from '../lib/types';
import { formatDate } from '../lib/utils';

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
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

interface AddScheduleModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function AddScheduleModal({ isOpen, onClose }: AddScheduleModalProps) {
	const [name, setName] = useState('');
	const [frequency, setFrequency] = useState<ReportFrequency>('weekly');
	const [recipients, setRecipients] = useState('');
	const [timezone, setTimezone] = useState('UTC');
	const [channelId, setChannelId] = useState('');

	const { data: channels } = useNotificationChannels();
	const createSchedule = useCreateReportSchedule();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const recipientList = recipients
				.split(',')
				.map((r) => r.trim())
				.filter((r) => r);
			await createSchedule.mutateAsync({
				name,
				frequency,
				recipients: recipientList,
				timezone,
				channel_id: channelId || undefined,
				enabled: true,
			});
			resetForm();
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const resetForm = () => {
		setName('');
		setFrequency('weekly');
		setRecipients('');
		setTimezone('UTC');
		setChannelId('');
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Create Report Schedule
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Schedule Name
							</label>
							<input
								type="text"
								id="name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Weekly Summary"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="frequency"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Frequency
							</label>
							<select
								id="frequency"
								value={frequency}
								onChange={(e) =>
									setFrequency(e.target.value as ReportFrequency)
								}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="daily">Daily (8:00 AM)</option>
								<option value="weekly">Weekly (Monday 8:00 AM)</option>
								<option value="monthly">Monthly (1st 8:00 AM)</option>
							</select>
						</div>
						<div>
							<label
								htmlFor="recipients"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Recipients (comma-separated)
							</label>
							<input
								type="text"
								id="recipients"
								value={recipients}
								onChange={(e) => setRecipients(e.target.value)}
								placeholder="admin@example.com, team@example.com"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="timezone"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Timezone
							</label>
							<select
								id="timezone"
								value={timezone}
								onChange={(e) => setTimezone(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="UTC">UTC</option>
								<option value="America/New_York">Eastern Time</option>
								<option value="America/Chicago">Central Time</option>
								<option value="America/Denver">Mountain Time</option>
								<option value="America/Los_Angeles">Pacific Time</option>
								<option value="Europe/London">London</option>
								<option value="Europe/Paris">Paris</option>
								<option value="Asia/Tokyo">Tokyo</option>
							</select>
						</div>
						<div>
							<label
								htmlFor="channelId"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Email Channel
							</label>
							<select
								id="channelId"
								value={channelId}
								onChange={(e) => setChannelId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							>
								<option value="">Select a channel...</option>
								{channels
									?.filter((c) => c.type === 'email' && c.enabled)
									.map((channel) => (
										<option key={channel.id} value={channel.id}>
											{channel.name}
										</option>
									))}
							</select>
							<p className="mt-1 text-xs text-gray-500">
								Configure email channels in Notifications settings
							</p>
						</div>
					</div>
					<div className="flex justify-end space-x-3 mt-6">
						<button
							type="button"
							onClick={() => {
								resetForm();
								onClose();
							}}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createSchedule.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50"
						>
							{createSchedule.isPending ? 'Creating...' : 'Create Schedule'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface PreviewModalProps {
	isOpen: boolean;
	onClose: () => void;
	data: ReportData | null;
	periodStart?: string;
	periodEnd?: string;
}

function PreviewModal({
	isOpen,
	onClose,
	data,
	periodStart,
	periodEnd,
}: PreviewModalProps) {
	if (!isOpen || !data) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex justify-between items-center mb-4">
					<h3 className="text-lg font-semibold text-gray-900">Report Preview</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-500"
					>
						<span className="sr-only">Close</span>
						<svg
							className="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
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
				{periodStart && periodEnd && (
					<p className="text-sm text-gray-500 mb-4">
						Period: {formatDate(periodStart)} - {formatDate(periodEnd)}
					</p>
				)}

				<div className="space-y-6">
					{/* Backup Summary */}
					<div className="bg-gray-50 rounded-lg p-4">
						<h4 className="font-medium text-gray-900 mb-3">Backup Summary</h4>
						<div className="grid grid-cols-3 gap-4 text-center">
							<div>
								<div className="text-2xl font-bold text-gray-900">
									{data.backup_summary.total_backups}
								</div>
								<div className="text-xs text-gray-500">Total</div>
							</div>
							<div>
								<div className="text-2xl font-bold text-green-600">
									{data.backup_summary.successful_backups}
								</div>
								<div className="text-xs text-gray-500">Successful</div>
							</div>
							<div>
								<div className="text-2xl font-bold text-red-600">
									{data.backup_summary.failed_backups}
								</div>
								<div className="text-xs text-gray-500">Failed</div>
							</div>
						</div>
						<div className="mt-3 text-center">
							<span
								className={`text-lg font-semibold ${
									data.backup_summary.success_rate >= 95
										? 'text-green-600'
										: data.backup_summary.success_rate >= 80
											? 'text-yellow-600'
											: 'text-red-600'
								}`}
							>
								{data.backup_summary.success_rate.toFixed(1)}% Success Rate
							</span>
						</div>
					</div>

					{/* Storage Summary */}
					<div className="bg-gray-50 rounded-lg p-4">
						<h4 className="font-medium text-gray-900 mb-3">Storage Overview</h4>
						<div className="space-y-2 text-sm">
							<div className="flex justify-between">
								<span className="text-gray-500">Repositories</span>
								<span className="font-medium">
									{data.storage_summary.repository_count}
								</span>
							</div>
							<div className="flex justify-between">
								<span className="text-gray-500">Total Snapshots</span>
								<span className="font-medium">
									{data.storage_summary.total_snapshots}
								</span>
							</div>
							<div className="flex justify-between">
								<span className="text-gray-500">Original Data</span>
								<span className="font-medium">
									{formatBytes(data.storage_summary.total_restore_size)}
								</span>
							</div>
							<div className="flex justify-between">
								<span className="text-gray-500">Storage Used</span>
								<span className="font-medium">
									{formatBytes(data.storage_summary.total_raw_size)}
								</span>
							</div>
							<div className="flex justify-between">
								<span className="text-gray-500">Space Saved</span>
								<span className="font-medium text-green-600">
									{formatBytes(data.storage_summary.space_saved)} (
									{data.storage_summary.space_saved_pct.toFixed(1)}%)
								</span>
							</div>
						</div>
					</div>

					{/* Agent Summary */}
					<div className="bg-gray-50 rounded-lg p-4">
						<h4 className="font-medium text-gray-900 mb-3">Agent Status</h4>
						<div className="grid grid-cols-4 gap-4 text-center">
							<div>
								<div className="text-xl font-bold text-gray-900">
									{data.agent_summary.total_agents}
								</div>
								<div className="text-xs text-gray-500">Total</div>
							</div>
							<div>
								<div className="text-xl font-bold text-green-600">
									{data.agent_summary.active_agents}
								</div>
								<div className="text-xs text-gray-500">Active</div>
							</div>
							<div>
								<div className="text-xl font-bold text-red-600">
									{data.agent_summary.offline_agents}
								</div>
								<div className="text-xs text-gray-500">Offline</div>
							</div>
							<div>
								<div className="text-xl font-bold text-yellow-600">
									{data.agent_summary.pending_agents}
								</div>
								<div className="text-xs text-gray-500">Pending</div>
							</div>
						</div>
					</div>

					{/* Alerts Summary */}
					{data.alert_summary.total_alerts > 0 && (
						<div className="bg-gray-50 rounded-lg p-4">
							<h4 className="font-medium text-gray-900 mb-3">Alerts</h4>
							<div className="grid grid-cols-3 gap-4 text-center">
								<div>
									<div className="text-xl font-bold text-red-600">
										{data.alert_summary.critical_alerts}
									</div>
									<div className="text-xs text-gray-500">Critical</div>
								</div>
								<div>
									<div className="text-xl font-bold text-yellow-600">
										{data.alert_summary.warning_alerts}
									</div>
									<div className="text-xs text-gray-500">Warning</div>
								</div>
								<div>
									<div className="text-xl font-bold text-green-600">
										{data.alert_summary.resolved_alerts}
									</div>
									<div className="text-xs text-gray-500">Resolved</div>
								</div>
							</div>
						</div>
					)}

					{/* Top Issues */}
					{data.top_issues && data.top_issues.length > 0 && (
						<div className="bg-red-50 rounded-lg p-4">
							<h4 className="font-medium text-red-900 mb-3">
								Issues Requiring Attention
							</h4>
							<div className="space-y-2">
								{data.top_issues.map((issue, idx) => (
									<div
										key={`${issue.type}-${idx}`}
										className="text-sm border-b border-red-200 pb-2 last:border-0"
									>
										<div className="font-medium text-red-800">{issue.title}</div>
										<div className="text-red-700">{issue.description}</div>
									</div>
								))}
							</div>
						</div>
					)}
				</div>

				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

function ScheduleRow({ schedule }: { schedule: ReportSchedule }) {
	const [showPreview, setShowPreview] = useState(false);
	const [previewData, setPreviewData] = useState<{
		data: ReportData;
		start: string;
		end: string;
	} | null>(null);

	const updateSchedule = useUpdateReportSchedule();
	const deleteSchedule = useDeleteReportSchedule();
	const sendReport = useSendReport();
	const previewReport = usePreviewReport();

	const handleToggle = async () => {
		await updateSchedule.mutateAsync({
			id: schedule.id,
			data: { enabled: !schedule.enabled },
		});
	};

	const handleDelete = async () => {
		if (confirm('Are you sure you want to delete this schedule?')) {
			await deleteSchedule.mutateAsync(schedule.id);
		}
	};

	const handlePreview = async () => {
		try {
			const result = await previewReport.mutateAsync({
				frequency: schedule.frequency,
				timezone: schedule.timezone,
			});
			setPreviewData({
				data: result.data,
				start: result.period.start,
				end: result.period.end,
			});
			setShowPreview(true);
		} catch {
			// Error handled by mutation
		}
	};

	const handleSend = async () => {
		if (confirm('Send this report now to all recipients?')) {
			await sendReport.mutateAsync({ id: schedule.id, preview: false });
		}
	};

	return (
		<>
			<tr className="hover:bg-gray-50">
				<td className="px-6 py-4">
					<div className="font-medium text-gray-900">{schedule.name}</div>
					<div className="text-sm text-gray-500">
						{schedule.recipients.join(', ')}
					</div>
				</td>
				<td className="px-6 py-4">
					<span className="px-2 py-1 text-xs font-medium rounded-full bg-indigo-100 text-indigo-800 capitalize">
						{schedule.frequency}
					</span>
				</td>
				<td className="px-6 py-4 text-sm text-gray-500">
					{schedule.last_sent_at ? formatDate(schedule.last_sent_at) : 'Never'}
				</td>
				<td className="px-6 py-4">
					<button
						type="button"
						onClick={handleToggle}
						disabled={updateSchedule.isPending}
						className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 ${
							schedule.enabled ? 'bg-indigo-600' : 'bg-gray-200'
						}`}
					>
						<span
							className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
								schedule.enabled ? 'translate-x-5' : 'translate-x-0'
							}`}
						/>
					</button>
				</td>
				<td className="px-6 py-4 text-right space-x-2">
					<button
						type="button"
						onClick={handlePreview}
						disabled={previewReport.isPending}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						Preview
					</button>
					<button
						type="button"
						onClick={handleSend}
						disabled={sendReport.isPending || !schedule.enabled}
						className="text-green-600 hover:text-green-800 text-sm font-medium disabled:opacity-50"
					>
						Send Now
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={deleteSchedule.isPending}
						className="text-red-600 hover:text-red-800 text-sm font-medium"
					>
						Delete
					</button>
				</td>
			</tr>
			<PreviewModal
				isOpen={showPreview}
				onClose={() => setShowPreview(false)}
				data={previewData?.data ?? null}
				periodStart={previewData?.start}
				periodEnd={previewData?.end}
			/>
		</>
	);
}

function HistoryRow({ history }: { history: ReportHistory }) {
	const [showDetails, setShowDetails] = useState(false);

	return (
		<>
			<tr className="hover:bg-gray-50">
				<td className="px-6 py-4">
					<span className="px-2 py-1 text-xs font-medium rounded-full bg-indigo-100 text-indigo-800 capitalize">
						{history.report_type}
					</span>
				</td>
				<td className="px-6 py-4 text-sm text-gray-500">
					{formatDate(history.period_start)} - {formatDate(history.period_end)}
				</td>
				<td className="px-6 py-4 text-sm text-gray-500">
					{history.recipients.join(', ')}
				</td>
				<td className="px-6 py-4">
					<span
						className={`px-2 py-1 text-xs font-medium rounded-full ${
							history.status === 'sent'
								? 'bg-green-100 text-green-800'
								: history.status === 'failed'
									? 'bg-red-100 text-red-800'
									: 'bg-gray-100 text-gray-800'
						}`}
					>
						{history.status}
					</span>
				</td>
				<td className="px-6 py-4 text-sm text-gray-500">
					{history.sent_at ? formatDate(history.sent_at) : '-'}
				</td>
				<td className="px-6 py-4 text-right">
					{history.report_data && (
						<button
							type="button"
							onClick={() => setShowDetails(true)}
							className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
						>
							View
						</button>
					)}
				</td>
			</tr>
			<PreviewModal
				isOpen={showDetails}
				onClose={() => setShowDetails(false)}
				data={history.report_data ?? null}
				periodStart={history.period_start}
				periodEnd={history.period_end}
			/>
		</>
	);
}

export default function Reports() {
	const [activeTab, setActiveTab] = useState<'schedules' | 'history'>(
		'schedules',
	);
	const [showAddModal, setShowAddModal] = useState(false);

	const { data: schedules, isLoading: schedulesLoading } = useReportSchedules();
	const { data: history, isLoading: historyLoading } = useReportHistory();

	return (
		<div className="p-6">
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Email Reports</h1>
					<p className="text-sm text-gray-500 mt-1">
						Schedule automated backup summary reports
					</p>
				</div>
				{activeTab === 'schedules' && (
					<button
						type="button"
						onClick={() => setShowAddModal(true)}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
					>
						Create Schedule
					</button>
				)}
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 mb-6">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('schedules')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'schedules'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Schedules
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('history')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'history'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						History
					</button>
				</nav>
			</div>

			{/* Schedules Tab */}
			{activeTab === 'schedules' && (
				<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
					<table className="min-w-full divide-y divide-gray-200">
						<thead className="bg-gray-50">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Schedule
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Frequency
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Last Sent
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Enabled
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="bg-white divide-y divide-gray-200">
							{schedulesLoading ? (
								<>
									<LoadingRow />
									<LoadingRow />
								</>
							) : schedules?.length === 0 ? (
								<tr>
									<td
										colSpan={5}
										className="px-6 py-12 text-center text-gray-500"
									>
										No report schedules configured. Create one to start
										receiving automated reports.
									</td>
								</tr>
							) : (
								schedules?.map((schedule) => (
									<ScheduleRow key={schedule.id} schedule={schedule} />
								))
							)}
						</tbody>
					</table>
				</div>
			)}

			{/* History Tab */}
			{activeTab === 'history' && (
				<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
					<table className="min-w-full divide-y divide-gray-200">
						<thead className="bg-gray-50">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Type
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Period
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Recipients
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Sent At
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="bg-white divide-y divide-gray-200">
							{historyLoading ? (
								<>
									<LoadingRow />
									<LoadingRow />
								</>
							) : history?.length === 0 ? (
								<tr>
									<td
										colSpan={6}
										className="px-6 py-12 text-center text-gray-500"
									>
										No reports have been sent yet.
									</td>
								</tr>
							) : (
								history?.map((item) => (
									<HistoryRow key={item.id} history={item} />
								))
							)}
						</tbody>
					</table>
				</div>
			)}

			<AddScheduleModal
				isOpen={showAddModal}
				onClose={() => setShowAddModal(false)}
			/>
		</div>
	);
}
