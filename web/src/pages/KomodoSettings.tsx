import { useState } from 'react';
import {
	useKomodoIntegrations,
	useCreateKomodoIntegration,
	useDeleteKomodoIntegration,
	useTestKomodoConnection,
	useSyncKomodoIntegration,
	useKomodoContainers,
	useKomodoStacks,
	useKomodoWebhookEvents,
} from '../hooks/useKomodoIntegration';
import type {
	KomodoIntegration,
	KomodoIntegrationConfig,
} from '../lib/types';
import { formatDate } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

function StatusBadge({ status }: { status: string }) {
	const colors = {
		active: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		disconnected: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400',
		error: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
		running: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		stopped: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400',
		restarting: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
		unknown: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400',
		received: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
		processing: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
		processed: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
	};

	return (
		<span
			className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
				colors[status as keyof typeof colors] || colors.unknown
			}`}
		>
			{status}
		</span>
	);
}

interface AddIntegrationModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function AddIntegrationModal({ isOpen, onClose }: AddIntegrationModalProps) {
	const [name, setName] = useState('');
	const [url, setUrl] = useState('');
	const [apiKey, setApiKey] = useState('');
	const [username, setUsername] = useState('');
	const [password, setPassword] = useState('');

	const createIntegration = useCreateKomodoIntegration();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const config: KomodoIntegrationConfig = {
				api_key: apiKey,
				username: username || undefined,
				password: password || undefined,
			};
			await createIntegration.mutateAsync({
				name,
				url,
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
		setUrl('');
		setApiKey('');
		setUsername('');
		setPassword('');
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Add Komodo Integration
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Integration Name
							</label>
							<input
								type="text"
								id="name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Production Komodo"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Komodo URL
							</label>
							<input
								type="url"
								id="url"
								value={url}
								onChange={(e) => setUrl(e.target.value)}
								placeholder="https://komodo.example.com"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="apiKey"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								API Key
							</label>
							<input
								type="password"
								id="apiKey"
								value={apiKey}
								onChange={(e) => setApiKey(e.target.value)}
								placeholder="Your Komodo API key"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div className="border-t border-gray-200 dark:border-gray-700 pt-4">
							<p className="text-sm text-gray-500 dark:text-gray-400 mb-3">
								Alternative: Use username/password authentication
							</p>
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="username"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Username
									</label>
									<input
										type="text"
										id="username"
										value={username}
										onChange={(e) => setUsername(e.target.value)}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
								<div>
									<label
										htmlFor="password"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Password
									</label>
									<input
										type="password"
										id="password"
										value={password}
										onChange={(e) => setPassword(e.target.value)}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
							</div>
						</div>
					</div>

					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={() => {
								resetForm();
								onClose();
							}}
							className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createIntegration.isPending}
							className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg disabled:opacity-50"
						>
							{createIntegration.isPending ? 'Adding...' : 'Add Integration'}
						</button>
					</div>

					{createIntegration.isError && (
						<p className="mt-3 text-sm text-red-600 dark:text-red-400">
							{createIntegration.error?.message || 'Failed to create integration'}
						</p>
					)}
				</form>
			</div>
		</div>
	);
}

function IntegrationRow({ integration }: { integration: KomodoIntegration }) {
	const [showActions, setShowActions] = useState(false);
	const deleteIntegration = useDeleteKomodoIntegration();
	const testConnection = useTestKomodoConnection();
	const syncIntegration = useSyncKomodoIntegration();

	const handleTest = async () => {
		try {
			await testConnection.mutateAsync(integration.id);
		} catch {
			// Error handled by mutation
		}
	};

	const handleSync = async () => {
		try {
			await syncIntegration.mutateAsync(integration.id);
		} catch {
			// Error handled by mutation
		}
	};

	const handleDelete = async () => {
		if (confirm('Are you sure you want to delete this integration? This will also remove all discovered containers and stacks.')) {
			try {
				await deleteIntegration.mutateAsync(integration.id);
			} catch {
				// Error handled by mutation
			}
		}
	};

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">
					{integration.name}
				</div>
				<div className="text-sm text-gray-500 dark:text-gray-400">
					{integration.url}
				</div>
			</td>
			<td className="px-6 py-4">
				<StatusBadge status={integration.status} />
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{integration.last_sync_at
					? formatDate(integration.last_sync_at)
					: 'Never synced'}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{integration.enabled ? (
					<span className="text-green-600 dark:text-green-400">Enabled</span>
				) : (
					<span className="text-gray-400">Disabled</span>
				)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="relative">
					<button
						onClick={() => setShowActions(!showActions)}
						className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded"
					>
						<svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
							<path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
						</svg>
					</button>
					{showActions && (
						<div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-10">
							<button
								onClick={() => {
									setShowActions(false);
									handleTest();
								}}
								disabled={testConnection.isPending}
								className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
							>
								{testConnection.isPending ? 'Testing...' : 'Test Connection'}
							</button>
							<button
								onClick={() => {
									setShowActions(false);
									handleSync();
								}}
								disabled={syncIntegration.isPending}
								className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
							>
								{syncIntegration.isPending ? 'Syncing...' : 'Sync Now'}
							</button>
							<hr className="my-1 border-gray-200 dark:border-gray-700" />
							<button
								onClick={() => {
									setShowActions(false);
									handleDelete();
								}}
								disabled={deleteIntegration.isPending}
								className="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
							>
								{deleteIntegration.isPending ? 'Deleting...' : 'Delete'}
							</button>
						</div>
					)}
				</div>
			</td>
		</tr>
	);
}

function ContainersTable() {
	const { data: containers, isLoading, error } = useKomodoContainers();

	if (error) {
		return (
			<div className="text-center py-8 text-red-600 dark:text-red-400">
				Failed to load containers: {error.message}
			</div>
		);
	}

	return (
		<div className="overflow-x-auto">
			<table className="min-w-full">
				<thead>
					<tr className="border-b border-gray-200 dark:border-gray-700">
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Container
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Stack
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Status
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Backup
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Last Discovered
						</th>
					</tr>
				</thead>
				<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
					{isLoading ? (
						<>
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</>
					) : containers?.length === 0 ? (
						<tr>
							<td colSpan={5} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
								No containers discovered yet. Add an integration and sync to discover containers.
							</td>
						</tr>
					) : (
						containers?.map((container) => (
							<tr key={container.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
								<td className="px-6 py-4">
									<div className="font-medium text-gray-900 dark:text-white">
										{container.name}
									</div>
									{container.image && (
										<div className="text-sm text-gray-500 dark:text-gray-400">
											{container.image}
										</div>
									)}
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{container.stack_name || '-'}
								</td>
								<td className="px-6 py-4">
									<StatusBadge status={container.status} />
								</td>
								<td className="px-6 py-4">
									{container.backup_enabled ? (
										<span className="text-green-600 dark:text-green-400">Enabled</span>
									) : (
										<span className="text-gray-400">Disabled</span>
									)}
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{formatDate(container.last_discovered_at)}
								</td>
							</tr>
						))
					)}
				</tbody>
			</table>
		</div>
	);
}

function StacksTable() {
	const { data: stacks, isLoading, error } = useKomodoStacks();

	if (error) {
		return (
			<div className="text-center py-8 text-red-600 dark:text-red-400">
				Failed to load stacks: {error.message}
			</div>
		);
	}

	return (
		<div className="overflow-x-auto">
			<table className="min-w-full">
				<thead>
					<tr className="border-b border-gray-200 dark:border-gray-700">
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Stack
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Server
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Containers
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Last Discovered
						</th>
					</tr>
				</thead>
				<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
					{isLoading ? (
						<>
							<LoadingRow />
							<LoadingRow />
						</>
					) : stacks?.length === 0 ? (
						<tr>
							<td colSpan={4} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
								No stacks discovered yet.
							</td>
						</tr>
					) : (
						stacks?.map((stack) => (
							<tr key={stack.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
								<td className="px-6 py-4">
									<div className="font-medium text-gray-900 dark:text-white">
										{stack.name}
									</div>
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{stack.server_name || '-'}
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{stack.running_count}/{stack.container_count} running
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{formatDate(stack.last_discovered_at)}
								</td>
							</tr>
						))
					)}
				</tbody>
			</table>
		</div>
	);
}

function WebhookEventsTable() {
	const { data: events, isLoading, error } = useKomodoWebhookEvents();

	if (error) {
		return (
			<div className="text-center py-8 text-red-600 dark:text-red-400">
				Failed to load events: {error.message}
			</div>
		);
	}

	return (
		<div className="overflow-x-auto">
			<table className="min-w-full">
				<thead>
					<tr className="border-b border-gray-200 dark:border-gray-700">
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Event Type
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Status
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Received
						</th>
						<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							Processed
						</th>
					</tr>
				</thead>
				<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
					{isLoading ? (
						<>
							<LoadingRow />
							<LoadingRow />
						</>
					) : events?.length === 0 ? (
						<tr>
							<td colSpan={4} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
								No webhook events received yet.
							</td>
						</tr>
					) : (
						events?.map((event) => (
							<tr key={event.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
								<td className="px-6 py-4">
									<span className="font-mono text-sm text-gray-900 dark:text-white">
										{event.event_type}
									</span>
								</td>
								<td className="px-6 py-4">
									<StatusBadge status={event.status} />
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{formatDate(event.created_at)}
								</td>
								<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
									{event.processed_at ? formatDate(event.processed_at) : '-'}
								</td>
							</tr>
						))
					)}
				</tbody>
			</table>
		</div>
	);
}

export default function KomodoSettings() {
	const [showAddModal, setShowAddModal] = useState(false);
	const [activeTab, setActiveTab] = useState<'integrations' | 'containers' | 'stacks' | 'events'>('integrations');
	const { data: integrations, isLoading, error } = useKomodoIntegrations();

	const tabs = [
		{ id: 'integrations', label: 'Integrations' },
		{ id: 'containers', label: 'Containers' },
		{ id: 'stacks', label: 'Stacks' },
		{ id: 'events', label: 'Webhook Events' },
	] as const;

	return (
		<div className="p-6 max-w-7xl mx-auto">
			<div className="flex justify-between items-center mb-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Komodo Integration
					</h1>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						Connect to Komodo to import stacks and containers, trigger backups, and sync status.
					</p>
				</div>
				<button
					onClick={() => setShowAddModal(true)}
					className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700"
				>
					Add Integration
				</button>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700 mb-6">
				<nav className="flex gap-8">
					{tabs.map((tab) => (
						<button
							key={tab.id}
							onClick={() => setActiveTab(tab.id)}
							className={`pb-4 text-sm font-medium border-b-2 ${
								activeTab === tab.id
									? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
									: 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
							}`}
						>
							{tab.label}
						</button>
					))}
				</nav>
			</div>

			{/* Content */}
			<div className="bg-white dark:bg-gray-800 rounded-lg shadow">
				{activeTab === 'integrations' && (
					<>
						{error ? (
							<div className="p-6 text-center text-red-600 dark:text-red-400">
								Failed to load integrations: {error.message}
							</div>
						) : (
							<table className="min-w-full">
								<thead>
									<tr className="border-b border-gray-200 dark:border-gray-700">
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Integration
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Status
										</th>
										<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
											Last Sync
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
									{isLoading ? (
										<>
											<LoadingRow />
											<LoadingRow />
										</>
									) : integrations?.length === 0 ? (
										<tr>
											<td colSpan={5} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
												No integrations configured. Add one to get started.
											</td>
										</tr>
									) : (
										integrations?.map((integration) => (
											<IntegrationRow key={integration.id} integration={integration} />
										))
									)}
								</tbody>
							</table>
						)}
					</>
				)}

				{activeTab === 'containers' && <ContainersTable />}
				{activeTab === 'stacks' && <StacksTable />}
				{activeTab === 'events' && <WebhookEventsTable />}
			</div>

			{/* Webhook URL Info */}
			{activeTab === 'integrations' && integrations && integrations.length > 0 && (
				<div className="mt-6 bg-gray-50 dark:bg-gray-900 rounded-lg p-4">
					<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
						Webhook Configuration
					</h3>
					<p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
						Configure Komodo to send webhooks to the following URL to trigger backups:
					</p>
					<div className="flex items-center gap-2">
						<code className="flex-1 px-3 py-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded text-sm font-mono text-gray-900 dark:text-white">
							{window.location.origin}/webhooks/komodo/{'{integration_id}'}
						</code>
						<button
							onClick={() => {
								navigator.clipboard.writeText(`${window.location.origin}/webhooks/komodo/`);
							}}
							className="px-3 py-2 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 rounded"
						>
							Copy
						</button>
					</div>
				</div>
			)}

			<AddIntegrationModal isOpen={showAddModal} onClose={() => setShowAddModal(false)} />
		</div>
	);
}
