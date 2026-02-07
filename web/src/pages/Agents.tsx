import { useState } from 'react';
import { AgentDownloads } from '../components/features/AgentDownloads';
import { ExportImportModal } from '../components/features/ExportImportModal';
import { ImportAgentsWizard } from '../components/features/ImportAgentsWizard';
import { type BulkAction, BulkActions } from '../components/ui/BulkActions';
import {
	BulkOperationProgress,
	useBulkOperation,
} from '../components/ui/BulkOperationProgress';
import {
	BulkSelectCheckbox,
	BulkSelectHeader,
	BulkSelectToolbar,
} from '../components/ui/BulkSelect';
import { ConfirmationModal } from '../components/ui/ConfirmationModal';
import { HelpTooltip } from '../components/ui/HelpTooltip';
import { AgentRowSkeleton } from '../components/ui/PageSkeletons';
import { StarButton } from '../components/ui/StarButton';
import { useAddAgentToGroup, useAgentGroups } from '../hooks/useAgentGroups';
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
import { useBulkSelect } from '../hooks/useBulkSelect';
import { useFavoriteIds } from '../hooks/useFavorites';
import { useLocale } from '../hooks/useLocale';
import { useRunSchedule, useSchedules } from '../hooks/useSchedules';
import { statusHelp } from '../lib/help-content';
import type { Agent, AgentStatus, PendingRegistration } from '../lib/types';
import { getAgentStatusColor } from '../lib/utils';

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
	isSelected: boolean;
	onToggleSelect: () => void;
	isFavorite: boolean;
}

function AgentRow({
	agent,
	onDelete,
	onRotateKey,
	onRevokeKey,
	isDeleting,
	isRotating,
	isRevoking,
	isSelected,
	onToggleSelect,
	isFavorite,
}: AgentRowProps) {
	const [showMenu, setShowMenu] = useState(false);
	const statusColor = getAgentStatusColor(agent.status);
	const { t, formatRelativeTime } = useLocale();

	return (
		<tr className={`hover:bg-gray-50 ${isSelected ? 'bg-indigo-50' : ''}`}>
			<td className="px-6 py-4 w-12">
				<BulkSelectCheckbox checked={isSelected} onChange={onToggleSelect} />
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<StarButton
						entityType="agent"
						entityId={agent.id}
						isFavorite={isFavorite}
						size="sm"
					/>
					<div className="font-medium text-gray-900">{agent.hostname}</div>
				</div>
				{agent.os_info && (
					<div className="text-sm text-gray-500">
						{agent.os_info.os} {agent.os_info.arch}
					</div>
				)}
			</td>
			<td className="px-6 py-4">
				<span className="inline-flex items-center gap-1.5">
					<span
						className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
					>
						<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
						{agent.status}
					</span>
					<HelpTooltip
						content={
							agent.status === 'active'
								? statusHelp.agentActive.content
								: agent.status === 'offline'
									? statusHelp.agentOffline.content
									: agent.status === 'pending'
										? statusHelp.agentPending.content
										: statusHelp.agentDisabled.content
						}
						title={
							agent.status === 'active'
								? statusHelp.agentActive.title
								: agent.status === 'offline'
									? statusHelp.agentOffline.title
									: agent.status === 'pending'
										? statusHelp.agentPending.title
										: statusHelp.agentDisabled.title
						}
						position="right"
					/>
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
	const [showFavoritesOnly, setShowFavoritesOnly] = useState(false);
	const [showGenerateModal, setShowGenerateModal] = useState(false);
	const [newCode, setNewCode] = useState<{
		code: string;
		expiresAt: string;
	} | null>(null);
	const [newApiKey, setNewApiKey] = useState<string | null>(null);
	const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
	const [showAddToGroupModal, setShowAddToGroupModal] = useState(false);
	const [selectedGroupId, setSelectedGroupId] = useState('');
	const [showExportModal, setShowExportModal] = useState(false);
	const [selectedAgentForExport, setSelectedAgentForExport] =
		useState<Agent | null>(null);
	const [showBulkImportWizard, setShowBulkImportWizard] = useState(false);
	const [importSuccessMessage, setImportSuccessMessage] = useState<
		string | null
	>(null);

	const { data: agents, isLoading, isError } = useAgents();
	const { data: pendingRegistrations, isLoading: isPendingLoading } =
		usePendingRegistrations();
	const { data: agentGroups } = useAgentGroups();
	const { data: schedules } = useSchedules();
	const favoriteIds = useFavoriteIds('agent');
	const deleteAgent = useDeleteAgent();
	const rotateApiKey = useRotateAgentApiKey();
	const revokeApiKey = useRevokeAgentApiKey();
	const deleteCode = useDeleteRegistrationCode();
	const addAgentToGroup = useAddAgentToGroup();
	const runSchedule = useRunSchedule();
	const { t } = useLocale();

	const bulkOperation = useBulkOperation();

	const filteredAgents = agents?.filter((agent) => {
		const matchesSearch = agent.hostname
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesStatus =
			statusFilter === 'all' || agent.status === statusFilter;
		const matchesFavorites = !showFavoritesOnly || favoriteIds.has(agent.id);
		return matchesSearch && matchesStatus && matchesFavorites;
	});

	const agentIds = filteredAgents?.map((a) => a.id) ?? [];
	const bulkSelect = useBulkSelect(agentIds);

	const bulkActions: BulkAction[] = [
		{
			id: 'add-to-group',
			label: 'Add to Group',
			icon: (
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
						d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
					/>
				</svg>
			),
		},
		{
			id: 'run-backup',
			label: 'Run Backup',
			icon: (
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
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
					/>
				</svg>
			),
		},
		{
			id: 'delete',
			label: 'Delete',
			variant: 'danger',
			icon: (
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
						d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
					/>
				</svg>
			),
			requiresConfirmation: true,
		},
	];

	const handleBulkAction = (actionId: string) => {
		switch (actionId) {
			case 'delete':
				setShowDeleteConfirm(true);
				break;
			case 'add-to-group':
				setShowAddToGroupModal(true);
				break;
			case 'run-backup':
				handleBulkRunBackup();
				break;
		}
	};

	const handleBulkDelete = async () => {
		setShowDeleteConfirm(false);
		await bulkOperation.start(
			[...bulkSelect.selectedIds],
			async (id: string) => {
				await deleteAgent.mutateAsync(id);
			},
		);
		bulkSelect.clear();
	};

	const handleBulkAddToGroup = async () => {
		if (!selectedGroupId) return;
		setShowAddToGroupModal(false);
		await bulkOperation.start(
			[...bulkSelect.selectedIds],
			async (id: string) => {
				await addAgentToGroup.mutateAsync({
					groupId: selectedGroupId,
					data: { agent_id: id },
				});
			},
		);
		bulkSelect.clear();
		setSelectedGroupId('');
	};

	const handleBulkRunBackup = async () => {
		// Find schedules for selected agents and run them
		const agentScheduleMap = new Map<string, string>();
		if (schedules) {
			for (const schedule of schedules) {
				if (bulkSelect.selectedIds.has(schedule.agent_id) && schedule.enabled) {
					agentScheduleMap.set(schedule.agent_id, schedule.id);
				}
			}
		}

		const scheduleIds = [...agentScheduleMap.values()];
		if (scheduleIds.length === 0) {
			alert('No enabled schedules found for selected agents');
			return;
		}

		await bulkOperation.start(scheduleIds, async (id: string) => {
			await runSchedule.mutateAsync(id);
		});
		bulkSelect.clear();
	};

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
				<div className="flex items-center gap-3">
					<button
						type="button"
						onClick={() => setShowBulkImportWizard(true)}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
						title="Import agents from CSV file"
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
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
						Bulk Import
					</button>
					<button
						type="button"
						onClick={() => {
							setSelectedAgentForExport(null);
							setShowExportModal(true);
						}}
						className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
							/>
						</svg>
						Import Config
					</button>
					<button
						type="button"
						onClick={() => setShowGenerateModal(true)}
						data-action="register-agent"
						title={`${t('agents.registerAgent')} (N)`}
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

			{/* Bulk Selection Toolbar */}
			{bulkSelect.selectedCount > 0 && (
				<BulkSelectToolbar
					selectedCount={bulkSelect.selectedCount}
					totalCount={agentIds.length}
					onSelectAll={() => bulkSelect.selectAll(agentIds)}
					onDeselectAll={bulkSelect.deselectAll}
					itemLabel="agent"
				>
					<BulkActions
						actions={bulkActions}
						onAction={handleBulkAction}
						label={t('common.actions')}
					/>
				</BulkSelectToolbar>
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
						<button
							type="button"
							onClick={() => setShowFavoritesOnly(!showFavoritesOnly)}
							className={`flex items-center gap-2 px-4 py-2 border rounded-lg transition-colors ${
								showFavoritesOnly
									? 'border-yellow-400 bg-yellow-50 text-yellow-700'
									: 'border-gray-300 text-gray-700 hover:bg-gray-50'
							}`}
						>
							<svg
								aria-hidden="true"
								className={`w-4 h-4 ${showFavoritesOnly ? 'text-yellow-400 fill-current' : 'text-gray-400'}`}
								fill={showFavoritesOnly ? 'currentColor' : 'none'}
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
								/>
							</svg>
							Favorites
						</button>
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
								<th className="px-6 py-3 w-12" />
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
							<AgentRowSkeleton />
							<AgentRowSkeleton />
							<AgentRowSkeleton />
						</tbody>
					</table>
				) : filteredAgents && filteredAgents.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 w-12">
									<BulkSelectHeader
										isAllSelected={bulkSelect.isAllSelected}
										isPartiallySelected={bulkSelect.isPartiallySelected}
										onToggleAll={() => bulkSelect.toggleAll(agentIds)}
										selectedCount={bulkSelect.selectedCount}
										totalCount={agentIds.length}
									/>
								</th>
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
									isSelected={bulkSelect.isSelected(agent.id)}
									onToggleSelect={() => bulkSelect.toggle(agent.id)}
									isFavorite={favoriteIds.has(agent.id)}
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

			{/* Bulk Delete Confirmation Modal */}
			<ConfirmationModal
				isOpen={showDeleteConfirm}
				onClose={() => setShowDeleteConfirm(false)}
				onConfirm={handleBulkDelete}
				title="Delete Agents"
				message="Are you sure you want to delete the selected agents? This action cannot be undone."
				confirmLabel="Delete"
				variant="danger"
				itemCount={bulkSelect.selectedCount}
			/>

			{/* Add to Group Modal */}
			{showAddToGroupModal && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Add to Group
						</h3>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							Select a group to add {bulkSelect.selectedCount} agent
							{bulkSelect.selectedCount !== 1 ? 's' : ''} to.
						</p>
						<div className="mb-4">
							<label
								htmlFor="bulk-group-select"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Group
							</label>
							<select
								id="bulk-group-select"
								value={selectedGroupId}
								onChange={(e) => setSelectedGroupId(e.target.value)}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="">Select a group</option>
								{agentGroups?.map((group) => (
									<option key={group.id} value={group.id}>
										{group.name}
									</option>
								))}
							</select>
						</div>
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => {
									setShowAddToGroupModal(false);
									setSelectedGroupId('');
								}}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								{t('common.cancel')}
							</button>
							<button
								type="button"
								onClick={handleBulkAddToGroup}
								disabled={!selectedGroupId}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								Add to Group
							</button>
						</div>
					</div>
				</div>
			)}

			{/* Bulk Operation Progress */}
			<BulkOperationProgress
				isOpen={bulkOperation.isRunning || bulkOperation.isComplete}
				onClose={bulkOperation.reset}
				title="Bulk Operation"
				total={bulkOperation.total}
				completed={bulkOperation.completed}
				results={bulkOperation.results}
				isComplete={bulkOperation.isComplete}
			/>

			{/* Download section - always visible */}
			<AgentDownloads showInstallCommands={true} />

			<ExportImportModal
				isOpen={showExportModal}
				onClose={() => {
					setShowExportModal(false);
					setSelectedAgentForExport(null);
				}}
				type="agent"
				item={selectedAgentForExport ?? undefined}
			/>

			<ImportAgentsWizard
				isOpen={showBulkImportWizard}
				onClose={() => setShowBulkImportWizard(false)}
				onSuccess={(importedCount, failedCount) => {
					setImportSuccessMessage(
						`Successfully imported ${importedCount} agent${importedCount !== 1 ? 's' : ''}${failedCount > 0 ? ` (${failedCount} failed)` : ''}`,
					);
					setTimeout(() => setImportSuccessMessage(null), 5000);
				}}
			/>

			{importSuccessMessage && (
				<div className="fixed bottom-4 right-4 bg-green-50 border border-green-200 rounded-lg p-4 shadow-lg z-50">
					<div className="flex items-center gap-3">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-green-500"
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
						<span className="text-sm text-green-800">
							{importSuccessMessage}
						</span>
						<button
							type="button"
							onClick={() => setImportSuccessMessage(null)}
							className="text-green-600 hover:text-green-800"
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
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					</div>
				</div>
			)}
		</div>
	);
}
