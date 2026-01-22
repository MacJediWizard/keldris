import { useState } from 'react';
import { AgentDownloads } from '../components/features/AgentDownloads';
import {
	useCreateRegistrationCode,
	useDeleteRegistrationCode,
	usePendingRegistrations,
} from '../hooks/useAgentRegistration';
import {
	useAgents,
	useDeleteAgent,
	useRevokeAgentApiKey,
	useRotateAgentApiKey,
} from '../hooks/useAgents';
import { useLocale } from '../hooks/useLocale';
import type { Agent, AgentStatus, PendingRegistration } from '../lib/types';
import { getAgentStatusColor } from '../lib/utils';

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
				<div className="h-4 w-32 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

interface GenerateCodeModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (code: string, expiresAt: string) => void;
}

function GenerateCodeModal({
	isOpen,
	onClose,
	onSuccess,
}: GenerateCodeModalProps) {
	const [hostname, setHostname] = useState('');
	const createCode = useCreateRegistrationCode();
	const { t } = useLocale();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const result = await createCode.mutateAsync({
				hostname: hostname || undefined,
			});
			onSuccess(result.code, result.expires_at);
			setHostname('');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Generate Registration Code
				</h3>
				<p className="text-sm text-gray-600 mb-4">
					Generate a one-time code that an agent can use to register. The code
					expires in 10 minutes.
				</p>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="hostname"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							{t('agents.hostname')} (optional)
						</label>
						<input
							type="text"
							id="hostname"
							value={hostname}
							onChange={(e) => setHostname(e.target.value)}
							placeholder={t('agents.hostnamePlaceholder')}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<p className="mt-1 text-xs text-gray-500">
							If provided, the agent must register with this exact hostname.
						</p>
					</div>
					{createCode.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to generate code. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							{t('common.cancel')}
						</button>
						<button
							type="submit"
							disabled={createCode.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createCode.isPending ? 'Generating...' : 'Generate Code'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface RegistrationCodeModalProps {
	code: string;
	expiresAt: string;
	onClose: () => void;
}

function RegistrationCodeModal({
	code,
	expiresAt,
	onClose,
}: RegistrationCodeModalProps) {
	const [copied, setCopied] = useState(false);
	const { t } = useLocale();

	const copyToClipboard = async () => {
		await navigator.clipboard.writeText(code);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	const expiresDate = new Date(expiresAt);
	const minutesLeft = Math.max(
		0,
		Math.ceil((expiresDate.getTime() - Date.now()) / 60000),
	);

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-green-100 rounded-full">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-green-600"
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
					<h3 className="text-lg font-semibold text-gray-900">
						Registration Code Generated
					</h3>
				</div>
				<p className="text-sm text-gray-600 mb-4">
					Use this code to register your agent. The code expires in{' '}
					<span className="font-medium text-orange-600">
						{minutesLeft} minutes
					</span>
					.
				</p>
				<div className="bg-gray-50 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-2">
						<code className="text-2xl font-mono font-bold text-gray-800 tracking-wider">
							{code}
						</code>
						<button
							type="button"
							onClick={copyToClipboard}
							className="flex-shrink-0 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded transition-colors"
						>
							{copied ? (
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
										d="M5 13l4 4L19 7"
									/>
								</svg>
							) : (
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
										d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
									/>
								</svg>
							)}
						</button>
					</div>
				</div>
				<div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
					<p className="text-sm text-blue-800 font-medium mb-2">
						To register an agent, run:
					</p>
					<code className="text-xs text-blue-700 block">
						keldris-agent register --server YOUR_SERVER_URL --code {code}
					</code>
				</div>
				<div className="flex justify-end">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						{t('common.done')}
					</button>
				</div>
			</div>
		</div>
	);
}

interface ApiKeyModalProps {
	apiKey: string;
	onClose: () => void;
}

function ApiKeyModal({ apiKey, onClose }: ApiKeyModalProps) {
	const [copied, setCopied] = useState(false);
	const { t } = useLocale();

	const copyToClipboard = async () => {
		await navigator.clipboard.writeText(apiKey);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-green-100 rounded-full">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-green-600"
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
					<h3 className="text-lg font-semibold text-gray-900">
						New API Key Generated
					</h3>
				</div>
				<p className="text-sm text-gray-600 mb-4">
					{t('agents.saveApiKeyWarning')}
				</p>
				<div className="bg-gray-50 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-2">
						<code className="text-sm font-mono text-gray-800 break-all">
							{apiKey}
						</code>
						<button
							type="button"
							onClick={copyToClipboard}
							className="flex-shrink-0 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded transition-colors"
						>
							{copied ? (
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
										d="M5 13l4 4L19 7"
									/>
								</svg>
							) : (
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
										d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
									/>
								</svg>
							)}
						</button>
					</div>
				</div>
				<div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
					<p className="text-sm text-yellow-800">
						{t('agents.useKeyToConfigure')}
					</p>
					<code className="text-xs text-yellow-700 block mt-2">
						keldris-agent config --api-key {apiKey.substring(0, 20)}...
					</code>
				</div>
				<div className="flex justify-end">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						{t('common.done')}
					</button>
				</div>
			</div>
		</div>
	);
}

interface PendingRegistrationRowProps {
	registration: PendingRegistration;
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

function PendingRegistrationRow({
	registration,
	onDelete,
	isDeleting,
}: PendingRegistrationRowProps) {
	const [copied, setCopied] = useState(false);
	const expiresDate = new Date(registration.expires_at);
	const isExpired = expiresDate < new Date();
	const minutesLeft = Math.max(
		0,
		Math.ceil((expiresDate.getTime() - Date.now()) / 60000),
	);

	const copyCode = async () => {
		await navigator.clipboard.writeText(registration.code);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<tr className={`hover:bg-gray-50 ${isExpired ? 'opacity-50' : ''}`}>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<code className="text-lg font-mono font-bold tracking-wider">
						{registration.code}
					</code>
					<button
						type="button"
						onClick={copyCode}
						className="p-1 text-gray-400 hover:text-gray-600 rounded"
						title="Copy code"
					>
						{copied ? (
							<svg
								aria-hidden="true"
								className="w-4 h-4 text-green-500"
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
						) : (
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
									d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
								/>
							</svg>
						)}
					</button>
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{registration.hostname || (
					<span className="text-gray-400 italic">Any hostname</span>
				)}
			</td>
			<td className="px-6 py-4">
				{isExpired ? (
					<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700">
						Expired
					</span>
				) : (
					<span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-700">
						Expires in {minutesLeft}m
					</span>
				)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{registration.created_by}
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onDelete(registration.id)}
					disabled={isDeleting}
					className="text-sm text-red-600 hover:text-red-800 disabled:opacity-50"
				>
					{isDeleting ? 'Canceling...' : 'Cancel'}
				</button>
			</td>
		</tr>
	);
}

interface AgentRowProps {
	agent: Agent;
	onDelete: (id: string) => void;
	onRotateKey: (id: string) => void;
	onRevokeKey: (id: string) => void;
	isDeleting: boolean;
	isRotating: boolean;
	isRevoking: boolean;
}

function AgentRow({
	agent,
	onDelete,
	onRotateKey,
	onRevokeKey,
	isDeleting,
	isRotating,
	isRevoking,
}: AgentRowProps) {
	const [showMenu, setShowMenu] = useState(false);
	const statusColor = getAgentStatusColor(agent.status);
	const { t, formatRelativeTime } = useLocale();

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900">{agent.hostname}</div>
				{agent.os_info && (
					<div className="text-sm text-gray-500">
						{agent.os_info.os} {agent.os_info.arch}
					</div>
				)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
					{agent.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatRelativeTime(agent.last_seen)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatRelativeTime(agent.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="relative inline-block text-left">
					<button
						type="button"
						onClick={() => setShowMenu(!showMenu)}
						className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
					>
						{t('common.actions')}
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
								d="M19 9l-7 7-7-7"
							/>
						</svg>
					</button>
					{showMenu && (
						<>
							<div
								className="fixed inset-0 z-10"
								onClick={() => setShowMenu(false)}
								onKeyDown={(e) => e.key === 'Escape' && setShowMenu(false)}
							/>
							<div className="absolute right-0 z-20 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1">
								<button
									type="button"
									onClick={() => {
										onRotateKey(agent.id);
										setShowMenu(false);
									}}
									disabled={isRotating}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50"
								>
									{isRotating ? t('agents.rotating') : t('agents.rotateApiKey')}
								</button>
								<button
									type="button"
									onClick={() => {
										onRevokeKey(agent.id);
										setShowMenu(false);
									}}
									disabled={isRevoking || agent.status === 'pending'}
									className="w-full text-left px-4 py-2 text-sm text-yellow-700 hover:bg-yellow-50 disabled:opacity-50"
								>
									{isRevoking ? t('agents.revoking') : t('agents.revokeApiKey')}
								</button>
								<div className="border-t border-gray-100 my-1" />
								<button
									type="button"
									onClick={() => {
										onDelete(agent.id);
										setShowMenu(false);
									}}
									disabled={isDeleting}
									className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 disabled:opacity-50"
								>
									{isDeleting ? t('agents.deleting') : t('agents.deleteAgent')}
								</button>
							</div>
						</>
					)}
				</div>
			</td>
		</tr>
	);
}

export function Agents() {
	const [searchQuery, setSearchQuery] = useState('');
	const [statusFilter, setStatusFilter] = useState<AgentStatus | 'all'>('all');
	const [showGenerateModal, setShowGenerateModal] = useState(false);
	const [newCode, setNewCode] = useState<{
		code: string;
		expiresAt: string;
	} | null>(null);
	const [newApiKey, setNewApiKey] = useState<string | null>(null);

	const { data: agents, isLoading, isError } = useAgents();
	const { data: pendingRegistrations, isLoading: isPendingLoading } =
		usePendingRegistrations();
	const deleteAgent = useDeleteAgent();
	const rotateApiKey = useRotateAgentApiKey();
	const revokeApiKey = useRevokeAgentApiKey();
	const deleteCode = useDeleteRegistrationCode();
	const { t } = useLocale();

	const filteredAgents = agents?.filter((agent) => {
		const matchesSearch = agent.hostname
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesStatus =
			statusFilter === 'all' || agent.status === statusFilter;
		return matchesSearch && matchesStatus;
	});

	const handleGenerateSuccess = (code: string, expiresAt: string) => {
		setShowGenerateModal(false);
		setNewCode({ code, expiresAt });
	};

	const handleDelete = (id: string) => {
		if (confirm(t('agents.confirmDelete'))) {
			deleteAgent.mutate(id);
		}
	};

	const handleRotateKey = async (id: string) => {
		if (confirm(t('agents.confirmRotate'))) {
			try {
				const result = await rotateApiKey.mutateAsync(id);
				setNewApiKey(result.api_key);
			} catch {
				// Error handled by mutation
			}
		}
	};

	const handleRevokeKey = (id: string) => {
		if (confirm(t('agents.confirmRevoke'))) {
			revokeApiKey.mutate(id);
		}
	};

	const handleDeleteCode = (id: string) => {
		if (confirm('Are you sure you want to cancel this registration code?')) {
			deleteCode.mutate(id);
		}
	};

	// Filter out expired registrations for display count
	const activePendingCount =
		pendingRegistrations?.filter((r) => new Date(r.expires_at) > new Date())
			.length ?? 0;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">
						{t('agents.title')}
					</h1>
					<p className="text-gray-600 mt-1">{t('agents.subtitle')}</p>
				</div>
				<button
					type="button"
					onClick={() => setShowGenerateModal(true)}
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
					{t('agents.registerAgent')}
				</button>
			</div>

			{/* Pending Registrations Section */}
			{(activePendingCount > 0 || isPendingLoading) && (
				<div className="bg-white rounded-lg border border-gray-200">
					<div className="p-4 border-b border-gray-200">
						<div className="flex items-center gap-2">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-yellow-500"
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
							<h2 className="text-lg font-semibold text-gray-900">
								Pending Registrations
							</h2>
							{activePendingCount > 0 && (
								<span className="inline-flex items-center justify-center px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 rounded-full">
									{activePendingCount}
								</span>
							)}
						</div>
						<p className="text-sm text-gray-500 mt-1">
							Registration codes waiting for agents to connect
						</p>
					</div>
					{isPendingLoading ? (
						<div className="p-8 text-center text-gray-500">
							Loading pending registrations...
						</div>
					) : pendingRegistrations && pendingRegistrations.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Code
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										{t('agents.hostname')}
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										{t('common.status')}
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created By
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										{t('common.actions')}
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{pendingRegistrations.map((reg) => (
									<PendingRegistrationRow
										key={reg.id}
										registration={reg}
										onDelete={handleDeleteCode}
										isDeleting={deleteCode.isPending}
									/>
								))}
							</tbody>
						</table>
					) : null}
				</div>
			)}

			{/* Registered Agents Section */}
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder={t('agents.searchAgents')}
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as AgentStatus | 'all')
							}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">{t('agents.allStatus')}</option>
							<option value="active">{t('agents.active')}</option>
							<option value="offline">{t('agents.offline')}</option>
							<option value="pending">{t('agents.pending')}</option>
							<option value="disabled">{t('agents.disabled')}</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">{t('agents.failedToLoad')}</p>
						<p className="text-sm">{t('agents.tryRefreshing')}</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.hostname')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('common.status')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.lastSeen')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.registered')}
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('common.actions')}
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200">
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : filteredAgents && filteredAgents.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.hostname')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('common.status')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.lastSeen')}
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('agents.registered')}
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									{t('common.actions')}
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200">
							{filteredAgents.map((agent) => (
								<AgentRow
									key={agent.id}
									agent={agent}
									onDelete={handleDelete}
									onRotateKey={handleRotateKey}
									onRevokeKey={handleRevokeKey}
									isDeleting={deleteAgent.isPending}
									isRotating={rotateApiKey.isPending}
									isRevoking={revokeApiKey.isPending}
								/>
							))}
						</tbody>
					</table>
				) : (
					<div className="p-8 text-center text-gray-500">
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
								d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 mb-2">
							{t('agents.noAgentsRegistered')}
						</h3>
						<p className="mb-6">
							Generate a registration code to start backing up your systems
						</p>
						<button
							type="button"
							onClick={() => setShowGenerateModal(true)}
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
							Generate Registration Code
						</button>
					</div>
				)}
			</div>

			<GenerateCodeModal
				isOpen={showGenerateModal}
				onClose={() => setShowGenerateModal(false)}
				onSuccess={handleGenerateSuccess}
			/>

			{newCode && (
				<RegistrationCodeModal
					code={newCode.code}
					expiresAt={newCode.expiresAt}
					onClose={() => setNewCode(null)}
				/>
			)}

			{newApiKey && (
				<ApiKeyModal apiKey={newApiKey} onClose={() => setNewApiKey(null)} />
			)}

			{/* Download section - always visible */}
			<AgentDownloads showInstallCommands={true} />
		</div>
	);
}
