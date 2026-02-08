import { useState } from 'react';
import {
	useCreateWebhookEndpoint,
	useDeleteWebhookEndpoint,
	useTestWebhookEndpoint,
	useUpdateWebhookEndpoint,
	useWebhookEndpointDeliveries,
	useWebhookEndpoints,
	useWebhookEventTypes,
	useRetryWebhookDelivery,
} from '../hooks/useWebhooks';
import type {
	WebhookDelivery,
	WebhookDeliveryStatus,
	WebhookEndpoint,
	WebhookEventType,
} from '../lib/types';
import { formatDate } from '../lib/utils';

const EVENT_TYPE_LABELS: Record<WebhookEventType, string> = {
	'backup.started': 'Backup Started',
	'backup.completed': 'Backup Completed',
	'backup.failed': 'Backup Failed',
	'agent.online': 'Agent Online',
	'agent.offline': 'Agent Offline',
	'restore.started': 'Restore Started',
	'restore.completed': 'Restore Completed',
	'restore.failed': 'Restore Failed',
	'alert.triggered': 'Alert Triggered',
	'alert.resolved': 'Alert Resolved',
};

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

function StatusBadge({ status }: { status: WebhookDeliveryStatus }) {
	const styles: Record<WebhookDeliveryStatus, string> = {
		pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
		delivered: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
		failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
		retrying: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
	};

	return (
		<span className={`px-2 py-1 text-xs font-medium rounded-full ${styles[status]}`}>
			{status}
		</span>
	);
}

interface AddEndpointModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function AddEndpointModal({ isOpen, onClose }: AddEndpointModalProps) {
	const [name, setName] = useState('');
	const [url, setUrl] = useState('');
	const [secret, setSecret] = useState('');
	const [selectedEvents, setSelectedEvents] = useState<WebhookEventType[]>([]);
	const [retryCount, setRetryCount] = useState('3');
	const [timeoutSeconds, setTimeoutSeconds] = useState('30');

	const { data: eventTypesData } = useWebhookEventTypes();
	const createEndpoint = useCreateWebhookEndpoint();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createEndpoint.mutateAsync({
				name,
				url,
				secret,
				event_types: selectedEvents,
				retry_count: Number.parseInt(retryCount, 10),
				timeout_seconds: Number.parseInt(timeoutSeconds, 10),
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
		setSecret('');
		setSelectedEvents([]);
		setRetryCount('3');
		setTimeoutSeconds('30');
	};

	const toggleEventType = (eventType: WebhookEventType) => {
		setSelectedEvents((prev) =>
			prev.includes(eventType)
				? prev.filter((e) => e !== eventType)
				: [...prev, eventType]
		);
	};

	const generateSecret = () => {
		const array = new Uint8Array(32);
		crypto.getRandomValues(array);
		const secret = Array.from(array, (b) => b.toString(16).padStart(2, '0')).join('');
		setSecret(secret);
	};

	if (!isOpen) return null;

	const eventTypes = eventTypesData?.event_types ?? Object.keys(EVENT_TYPE_LABELS) as WebhookEventType[];

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Add Webhook Endpoint
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Slack Integration"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Endpoint URL
							</label>
							<input
								type="url"
								id="url"
								value={url}
								onChange={(e) => setUrl(e.target.value)}
								placeholder="https://example.com/webhooks"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="secret"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Signing Secret
							</label>
							<div className="flex gap-2">
								<input
									type="text"
									id="secret"
									value={secret}
									onChange={(e) => setSecret(e.target.value)}
									placeholder="Minimum 16 characters"
									className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
									required
									minLength={16}
								/>
								<button
									type="button"
									onClick={generateSecret}
									className="px-3 py-2 bg-gray-100 dark:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-500"
								>
									Generate
								</button>
							</div>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Used to sign webhook payloads with HMAC-SHA256
							</p>
						</div>
						<div>
							<label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
								Event Types
							</label>
							<div className="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto border border-gray-200 dark:border-gray-600 rounded-lg p-3">
								{eventTypes.map((eventType) => (
									<label
										key={eventType}
										className="flex items-center space-x-2 cursor-pointer"
									>
										<input
											type="checkbox"
											checked={selectedEvents.includes(eventType)}
											onChange={() => toggleEventType(eventType)}
											className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
										/>
										<span className="text-sm text-gray-700 dark:text-gray-300">
											{EVENT_TYPE_LABELS[eventType] ?? eventType}
										</span>
									</label>
								))}
							</div>
						</div>
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="retryCount"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Retry Count
								</label>
								<input
									type="number"
									id="retryCount"
									value={retryCount}
									onChange={(e) => setRetryCount(e.target.value)}
									min="0"
									max="10"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div>
								<label
									htmlFor="timeoutSeconds"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Timeout (seconds)
								</label>
								<input
									type="number"
									id="timeoutSeconds"
									value={timeoutSeconds}
									onChange={(e) => setTimeoutSeconds(e.target.value)}
									min="5"
									max="120"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
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
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createEndpoint.isPending || selectedEvents.length === 0}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50"
						>
							{createEndpoint.isPending ? 'Creating...' : 'Create Endpoint'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface DeliveryLogModalProps {
	isOpen: boolean;
	onClose: () => void;
	endpoint: WebhookEndpoint | null;
}

function DeliveryLogModal({ isOpen, onClose, endpoint }: DeliveryLogModalProps) {
	const { data, isLoading } = useWebhookEndpointDeliveries(endpoint?.id ?? '', 50, 0);
	const retryDelivery = useRetryWebhookDelivery();

	if (!isOpen || !endpoint) return null;

	const deliveries = data?.deliveries ?? [];

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-4xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
				<div className="flex justify-between items-center mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Delivery Log: {endpoint.name}
					</h3>
					<button
						onClick={onClose}
						className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
					>
						<svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
				<div className="flex-1 overflow-y-auto">
					{isLoading ? (
						<div className="text-center py-8 text-gray-500">Loading...</div>
					) : deliveries.length === 0 ? (
						<div className="text-center py-8 text-gray-500">No deliveries yet</div>
					) : (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-700 sticky top-0">
								<tr>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Event
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Status
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Response
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Attempts
									</th>
									<th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Time
									</th>
									<th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{deliveries.map((delivery: WebhookDelivery) => (
									<tr key={delivery.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
										<td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
											{EVENT_TYPE_LABELS[delivery.event_type] ?? delivery.event_type}
										</td>
										<td className="px-4 py-3">
											<StatusBadge status={delivery.status} />
										</td>
										<td className="px-4 py-3 text-sm">
											{delivery.response_status ? (
												<span className={delivery.response_status >= 200 && delivery.response_status < 300 ? 'text-green-600' : 'text-red-600'}>
													{delivery.response_status}
												</span>
											) : delivery.error_message ? (
												<span className="text-red-600 truncate max-w-xs block" title={delivery.error_message}>
													{delivery.error_message}
												</span>
											) : (
												<span className="text-gray-400">-</span>
											)}
										</td>
										<td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
											{delivery.attempt_number}/{delivery.max_attempts}
										</td>
										<td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
											{formatDate(delivery.created_at)}
										</td>
										<td className="px-4 py-3 text-right">
											{delivery.status === 'failed' && (
												<button
													onClick={() => retryDelivery.mutate(delivery.id)}
													disabled={retryDelivery.isPending}
													className="text-indigo-600 hover:text-indigo-900 dark:hover:text-indigo-400 text-sm"
												>
													Retry
												</button>
											)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					)}
				</div>
			</div>
		</div>
	);
}

export default function Webhooks() {
	const [showAddModal, setShowAddModal] = useState(false);
	const [selectedEndpoint, setSelectedEndpoint] = useState<WebhookEndpoint | null>(null);
	const [showDeliveryLog, setShowDeliveryLog] = useState(false);
	const [testingEndpointId, setTestingEndpointId] = useState<string | null>(null);

	const { data: endpoints, isLoading } = useWebhookEndpoints();
	const updateEndpoint = useUpdateWebhookEndpoint();
	const deleteEndpoint = useDeleteWebhookEndpoint();
	const testEndpoint = useTestWebhookEndpoint();

	const handleToggleEnabled = async (endpoint: WebhookEndpoint) => {
		await updateEndpoint.mutateAsync({
			id: endpoint.id,
			data: { enabled: !endpoint.enabled },
		});
	};

	const handleDelete = async (endpoint: WebhookEndpoint) => {
		if (confirm(`Delete webhook endpoint "${endpoint.name}"?`)) {
			await deleteEndpoint.mutateAsync(endpoint.id);
		}
	};

	const handleTest = async (endpoint: WebhookEndpoint) => {
		setTestingEndpointId(endpoint.id);
		try {
			const result = await testEndpoint.mutateAsync({ id: endpoint.id });
			if (result.success) {
				alert(`Test successful! Response status: ${result.response_status} (${result.duration_ms}ms)`);
			} else {
				alert(`Test failed: ${result.error_message ?? `Status ${result.response_status}`}`);
			}
		} catch (err) {
			alert(`Test failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			setTestingEndpointId(null);
		}
	};

	const handleViewLog = (endpoint: WebhookEndpoint) => {
		setSelectedEndpoint(endpoint);
		setShowDeliveryLog(true);
	};

	return (
		<div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
			<div className="sm:flex sm:items-center sm:justify-between mb-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Webhooks
					</h1>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						Configure outbound webhooks to receive real-time event notifications
					</p>
				</div>
				<button
					onClick={() => setShowAddModal(true)}
					className="mt-4 sm:mt-0 inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium"
				>
					<svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
					</svg>
					Add Endpoint
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
				<table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
					<thead className="bg-gray-50 dark:bg-gray-700">
						<tr>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
								Name
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
								URL
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
								Events
							</th>
							<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
								Status
							</th>
							<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
								Actions
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
						) : !endpoints || endpoints.length === 0 ? (
							<tr>
								<td colSpan={5} className="px-6 py-12 text-center">
									<div className="text-gray-500 dark:text-gray-400">
										<svg className="mx-auto h-12 w-12 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
										</svg>
										<p className="text-lg font-medium">No webhook endpoints configured</p>
										<p className="mt-1">Add an endpoint to receive real-time event notifications</p>
									</div>
								</td>
							</tr>
						) : (
							endpoints.map((endpoint: WebhookEndpoint) => (
								<tr key={endpoint.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
									<td className="px-6 py-4">
										<div className="font-medium text-gray-900 dark:text-white">
											{endpoint.name}
										</div>
									</td>
									<td className="px-6 py-4">
										<div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-xs" title={endpoint.url}>
											{endpoint.url}
										</div>
									</td>
									<td className="px-6 py-4">
										<div className="flex flex-wrap gap-1">
											{endpoint.event_types.slice(0, 2).map((eventType) => (
												<span
													key={eventType}
													className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded"
												>
													{EVENT_TYPE_LABELS[eventType] ?? eventType}
												</span>
											))}
											{endpoint.event_types.length > 2 && (
												<span className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded">
													+{endpoint.event_types.length - 2} more
												</span>
											)}
										</div>
									</td>
									<td className="px-6 py-4">
										<button
											onClick={() => handleToggleEnabled(endpoint)}
											className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
												endpoint.enabled ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
											}`}
										>
											<span
												className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
													endpoint.enabled ? 'translate-x-6' : 'translate-x-1'
												}`}
											/>
										</button>
									</td>
									<td className="px-6 py-4 text-right">
										<div className="flex justify-end gap-2">
											<button
												onClick={() => handleTest(endpoint)}
												disabled={testingEndpointId === endpoint.id}
												className="text-indigo-600 hover:text-indigo-900 dark:hover:text-indigo-400 text-sm font-medium"
											>
												{testingEndpointId === endpoint.id ? 'Testing...' : 'Test'}
											</button>
											<button
												onClick={() => handleViewLog(endpoint)}
												className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white text-sm font-medium"
											>
												Log
											</button>
											<button
												onClick={() => handleDelete(endpoint)}
												className="text-red-600 hover:text-red-900 dark:hover:text-red-400 text-sm font-medium"
											>
												Delete
											</button>
										</div>
									</td>
								</tr>
							))
						)}
					</tbody>
				</table>
			</div>

			<div className="mt-8 bg-gray-50 dark:bg-gray-800/50 rounded-lg p-6">
				<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
					Webhook Signature Verification
				</h2>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					All webhook payloads are signed using HMAC-SHA256. Verify the signature using the{' '}
					<code className="px-1.5 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-xs">X-Keldris-Signature-256</code> header.
				</p>
				<pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto">
{`// Example verification (Node.js)
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}`}
				</pre>
			</div>

			<AddEndpointModal
				isOpen={showAddModal}
				onClose={() => setShowAddModal(false)}
			/>

			<DeliveryLogModal
				isOpen={showDeliveryLog}
				onClose={() => {
					setShowDeliveryLog(false);
					setSelectedEndpoint(null);
				}}
				endpoint={selectedEndpoint}
			/>
		</div>
	);
}
