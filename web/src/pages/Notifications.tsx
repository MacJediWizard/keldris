import { useState } from 'react';
import {
	useCreateNotificationChannel,
	useCreateNotificationPreference,
	useDeleteNotificationChannel,
	useNotificationChannels,
	useNotificationLogs,
	useNotificationPreferences,
	useUpdateNotificationChannel,
	useUpdateNotificationPreference,
} from '../hooks/useNotifications';
import type {
	EmailChannelConfig,
	NotificationChannel,
	NotificationChannelType,
	NotificationEventType,
	NotificationLog,
	NotificationPreference,
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

interface AddChannelModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function AddChannelModal({ isOpen, onClose }: AddChannelModalProps) {
	const [name, setName] = useState('');
	const [host, setHost] = useState('');
	const [port, setPort] = useState('587');
	const [username, setUsername] = useState('');
	const [password, setPassword] = useState('');
	const [from, setFrom] = useState('');
	const [useTLS, setUseTLS] = useState(true);

	const createChannel = useCreateNotificationChannel();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const config: EmailChannelConfig = {
				host,
				port: Number.parseInt(port, 10),
				username,
				password,
				from,
				tls: useTLS,
			};
			await createChannel.mutateAsync({
				name,
				type: 'email' as NotificationChannelType,
				config,
			});
			resetForm();
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const resetForm = () => {
		setName('');
		setHost('');
		setPort('587');
		setUsername('');
		setPassword('');
		setFrom('');
		setUseTLS(true);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Add Email Notification Channel
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Channel Name
							</label>
							<input
								type="text"
								id="name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Primary Email"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="host"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									SMTP Host
								</label>
								<input
									type="text"
									id="host"
									value={host}
									onChange={(e) => setHost(e.target.value)}
									placeholder="smtp.example.com"
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									required
								/>
							</div>
							<div>
								<label
									htmlFor="port"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Port
								</label>
								<input
									type="number"
									id="port"
									value={port}
									onChange={(e) => setPort(e.target.value)}
									className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									required
								/>
							</div>
						</div>
						<div>
							<label
								htmlFor="username"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Username
							</label>
							<input
								type="text"
								id="username"
								value={username}
								onChange={(e) => setUsername(e.target.value)}
								placeholder="user@example.com"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="password"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Password
							</label>
							<input
								type="password"
								id="password"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>
						<div>
							<label
								htmlFor="from"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								From Address
							</label>
							<input
								type="email"
								id="from"
								value={from}
								onChange={(e) => setFrom(e.target.value)}
								placeholder="notifications@example.com"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div className="flex items-center">
							<input
								type="checkbox"
								id="tls"
								checked={useTLS}
								onChange={(e) => setUseTLS(e.target.checked)}
								className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
							/>
							<label htmlFor="tls" className="ml-2 text-sm text-gray-700">
								Use TLS
							</label>
						</div>
					</div>
					{createChannel.isError && (
						<p className="text-sm text-red-600 mt-4">
							Failed to create channel. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={() => {
								resetForm();
								onClose();
							}}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createChannel.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createChannel.isPending ? 'Creating...' : 'Create Channel'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ChannelRowProps {
	channel: NotificationChannel;
	preferences: NotificationPreference[];
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

const EVENT_TYPES: { type: NotificationEventType; label: string }[] = [
	{ type: 'backup_success', label: 'Backup Success' },
	{ type: 'backup_failed', label: 'Backup Failed' },
	{ type: 'agent_offline', label: 'Agent Offline' },
];

function ChannelRow({
	channel,
	preferences,
	onDelete,
	isDeleting,
}: ChannelRowProps) {
	const updateChannel = useUpdateNotificationChannel();
	const createPreference = useCreateNotificationPreference();
	const updatePreference = useUpdateNotificationPreference();

	const handleToggleChannel = async () => {
		await updateChannel.mutateAsync({
			id: channel.id,
			data: { enabled: !channel.enabled },
		});
	};

	const handleToggleEvent = async (eventType: NotificationEventType) => {
		const pref = preferences.find((p) => p.event_type === eventType);
		if (pref) {
			await updatePreference.mutateAsync({
				id: pref.id,
				data: { enabled: !pref.enabled },
			});
		} else {
			await createPreference.mutateAsync({
				channel_id: channel.id,
				event_type: eventType,
				enabled: true,
			});
		}
	};

	const isEventEnabled = (eventType: NotificationEventType) => {
		const pref = preferences.find((p) => p.event_type === eventType);
		return pref?.enabled ?? false;
	};

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="flex items-center gap-3">
					<div className="p-2 bg-indigo-100 rounded-lg">
						<svg
							className="w-5 h-5 text-indigo-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
							/>
						</svg>
					</div>
					<div>
						<p className="font-medium text-gray-900">{channel.name}</p>
						<p className="text-sm text-gray-500">{channel.type}</p>
					</div>
				</div>
			</td>
			<td className="px-6 py-4">
				<button
					type="button"
					onClick={handleToggleChannel}
					disabled={updateChannel.isPending}
					className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 ${
						channel.enabled ? 'bg-indigo-600' : 'bg-gray-200'
					}`}
				>
					<span
						className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
							channel.enabled ? 'translate-x-5' : 'translate-x-0'
						}`}
					/>
				</button>
			</td>
			<td className="px-6 py-4">
				<div className="flex flex-wrap gap-2">
					{EVENT_TYPES.map(({ type, label }) => {
						const enabled = isEventEnabled(type);
						return (
							<button
								key={type}
								type="button"
								onClick={() => handleToggleEvent(type)}
								disabled={
									createPreference.isPending || updatePreference.isPending
								}
								className={`px-2 py-1 text-xs font-medium rounded-full transition-colors ${
									enabled
										? 'bg-green-100 text-green-800 hover:bg-green-200'
										: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
								}`}
							>
								{label}
							</button>
						);
					})}
				</div>
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onDelete(channel.id)}
					disabled={isDeleting}
					className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
				>
					Delete
				</button>
			</td>
		</tr>
	);
}

function getStatusColor(status: string): { bg: string; text: string } {
	switch (status) {
		case 'sent':
			return { bg: 'bg-green-100', text: 'text-green-800' };
		case 'failed':
			return { bg: 'bg-red-100', text: 'text-red-800' };
		case 'queued':
			return { bg: 'bg-yellow-100', text: 'text-yellow-800' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-800' };
	}
}

function LogRow({ log }: { log: NotificationLog }) {
	const statusColor = getStatusColor(log.status);
	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4 text-sm text-gray-900">
				{log.subject || log.event_type}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">{log.recipient}</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					{log.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(log.sent_at || log.created_at)}
			</td>
			<td className="px-6 py-4 text-sm text-red-500">
				{log.error_message || '-'}
			</td>
		</tr>
	);
}

export function Notifications() {
	const [showAddModal, setShowAddModal] = useState(false);
	const [activeTab, setActiveTab] = useState<'channels' | 'logs'>('channels');

	const {
		data: channels,
		isLoading: channelsLoading,
		isError: channelsError,
	} = useNotificationChannels();
	const { data: preferences } = useNotificationPreferences();
	const {
		data: logs,
		isLoading: logsLoading,
		isError: logsError,
	} = useNotificationLogs();
	const deleteChannel = useDeleteNotificationChannel();

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this notification channel?')) {
			deleteChannel.mutate(id);
		}
	};

	const getPreferencesForChannel = (channelId: string) =>
		preferences?.filter((p) => p.channel_id === channelId) ?? [];

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Notifications</h1>
					<p className="text-gray-500 mt-1">
						Configure email notifications for backup events
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
					Add Channel
				</button>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('channels')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'channels'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Channels
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('logs')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'logs'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						History
					</button>
				</nav>
			</div>

			{activeTab === 'channels' && (
				<div className="bg-white rounded-lg border border-gray-200">
					{channelsError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">
								Failed to load notification channels
							</p>
							<p className="text-sm mt-1">Please try refreshing the page</p>
						</div>
					) : channelsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Channel
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Enabled
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Events
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{[1, 2, 3].map((i) => (
									<LoadingRow key={i} />
								))}
							</tbody>
						</table>
					) : channels && channels.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Channel
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Enabled
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Events
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{channels.map((channel) => (
									<ChannelRow
										key={channel.id}
										channel={channel}
										preferences={getPreferencesForChannel(channel.id)}
										onDelete={handleDelete}
										isDeleting={deleteChannel.isPending}
									/>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center text-gray-500">
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
										d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
									/>
								</svg>
							</div>
							<p className="font-medium">No notification channels configured</p>
							<p className="text-sm mt-1">
								Add an email channel to receive backup notifications
							</p>
							<button
								type="button"
								onClick={() => setShowAddModal(true)}
								className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Add Channel
							</button>
						</div>
					)}
				</div>
			)}

			{activeTab === 'logs' && (
				<div className="bg-white rounded-lg border border-gray-200">
					{logsError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">Failed to load notification history</p>
							<p className="text-sm mt-1">Please try refreshing the page</p>
						</div>
					) : logsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Subject
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Recipient
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Sent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Error
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{[1, 2, 3].map((i) => (
									<LoadingRow key={i} />
								))}
							</tbody>
						</table>
					) : logs && logs.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Subject
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Recipient
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Sent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Error
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{logs.map((log) => (
									<LogRow key={log.id} log={log} />
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center text-gray-500">
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
										d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
									/>
								</svg>
							</div>
							<p className="font-medium">No notification history</p>
							<p className="text-sm mt-1">
								Notifications will appear here once they are sent
							</p>
						</div>
					)}
				</div>
			)}

			<AddChannelModal
				isOpen={showAddModal}
				onClose={() => setShowAddModal(false)}
			/>
		</div>
	);
}
