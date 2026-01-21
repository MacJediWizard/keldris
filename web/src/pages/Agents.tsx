import { useState } from 'react';
import { Link } from 'react-router-dom';
import { AgentDownloads } from '../components/features/AgentDownloads';
import {
	useAgentGroups,
	useAgentsWithGroups,
} from '../hooks/useAgentGroups';
import {
	useCreateAgent,
	useDeleteAgent,
	useRevokeAgentApiKey,
	useRotateAgentApiKey,
} from '../hooks/useAgents';
import type { AgentGroup, AgentStatus, AgentWithGroups } from '../lib/types';
import {
	formatDate,
	getAgentStatusColor,
	getHealthStatusColor,
	getHealthStatusLabel,
} from '../lib/utils';

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
				<div className="h-6 w-16 bg-gray-200 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="flex gap-1">
					<div className="h-5 w-16 bg-gray-200 rounded-full" />
				</div>
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

interface GroupBadgeProps {
	group: AgentGroup;
}

function GroupBadge({ group }: GroupBadgeProps) {
	const bgColor = group.color
		? `${group.color}20`
		: 'rgba(99, 102, 241, 0.1)';
	const textColor = group.color || '#6366f1';

	return (
		<span
			className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
			style={{ backgroundColor: bgColor, color: textColor }}
		>
			{group.color && (
				<span
					className="w-1.5 h-1.5 rounded-full"
					style={{ backgroundColor: textColor }}
				/>
			)}
			{group.name}
		</span>
	);
}

interface RegisterModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (apiKey: string) => void;
}

function RegisterModal({ isOpen, onClose, onSuccess }: RegisterModalProps) {
	const [hostname, setHostname] = useState('');
	const createAgent = useCreateAgent();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const result = await createAgent.mutateAsync({ hostname });
			onSuccess(result.api_key);
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
					Register New Agent
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="hostname"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Hostname
						</label>
						<input
							type="text"
							id="hostname"
							value={hostname}
							onChange={(e) => setHostname(e.target.value)}
							placeholder="e.g., server-01"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>
					{createAgent.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to create agent. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createAgent.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createAgent.isPending ? 'Creating...' : 'Register'}
						</button>
					</div>
				</form>
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
						Agent Registered Successfully
					</h3>
				</div>
				<p className="text-sm text-gray-600 mb-4">
					Save this API key now. You won't be able to see it again!
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
						Use this key to configure your agent:
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
						Done
					</button>
				</div>
			</div>
		</div>
	);
}

interface AgentRowProps {
	agent: AgentWithGroups;
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
	const healthColor = getHealthStatusColor(agent.health_status || 'unknown');

	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<Link
					to={`/agents/${agent.id}`}
					className="font-medium text-indigo-600 hover:text-indigo-700"
				>
					{agent.hostname}
				</Link>
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
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${healthColor.bg} ${healthColor.text}`}
					title={
						agent.health_metrics
							? `CPU: ${agent.health_metrics.cpu_usage?.toFixed(1)}% | Memory: ${agent.health_metrics.memory_usage?.toFixed(1)}% | Disk: ${agent.health_metrics.disk_usage?.toFixed(1)}%`
							: 'No health data'
					}
				>
					<span className={`w-1.5 h-1.5 ${healthColor.dot} rounded-full`} />
					{getHealthStatusLabel(agent.health_status || 'unknown')}
				</span>
			</td>
			<td className="px-6 py-4">
				{agent.groups && agent.groups.length > 0 ? (
					<div className="flex flex-wrap gap-1">
						{agent.groups.map((group) => (
							<GroupBadge key={group.id} group={group} />
						))}
					</div>
				) : (
					<span className="text-sm text-gray-400">-</span>
				)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(agent.last_seen)}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(agent.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="relative inline-block text-left">
					<button
						type="button"
						onClick={() => setShowMenu(!showMenu)}
						className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
					>
						Actions
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
									{isRotating ? 'Rotating...' : 'Rotate API Key'}
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
									{isRevoking ? 'Revoking...' : 'Revoke API Key'}
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
									{isDeleting ? 'Deleting...' : 'Delete Agent'}
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
	const [groupFilter, setGroupFilter] = useState<string>('all');
	const [showRegisterModal, setShowRegisterModal] = useState(false);
	const [newApiKey, setNewApiKey] = useState<string | null>(null);

	const { data: agents, isLoading, isError } = useAgentsWithGroups();
	const { data: groups } = useAgentGroups();
	const deleteAgent = useDeleteAgent();
	const rotateApiKey = useRotateAgentApiKey();
	const revokeApiKey = useRevokeAgentApiKey();

	const filteredAgents = agents?.filter((agent) => {
		const matchesSearch = agent.hostname
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesStatus =
			statusFilter === 'all' || agent.status === statusFilter;
		const matchesGroup =
			groupFilter === 'all' ||
			(groupFilter === 'none'
				? !agent.groups || agent.groups.length === 0
				: agent.groups?.some((g) => g.id === groupFilter));
		return matchesSearch && matchesStatus && matchesGroup;
	});

	const handleRegisterSuccess = (apiKey: string) => {
		setShowRegisterModal(false);
		setNewApiKey(apiKey);
	};

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this agent?')) {
			deleteAgent.mutate(id);
		}
	};

	const handleRotateKey = async (id: string) => {
		if (
			confirm(
				'Are you sure you want to rotate this API key? The old key will be invalidated immediately.',
			)
		) {
			try {
				const result = await rotateApiKey.mutateAsync(id);
				setNewApiKey(result.api_key);
			} catch {
				// Error handled by mutation
			}
		}
	};

	const handleRevokeKey = (id: string) => {
		if (
			confirm(
				'Are you sure you want to revoke this API key? The agent will no longer be able to authenticate.',
			)
		) {
			revokeApiKey.mutate(id);
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Agents</h1>
					<p className="text-gray-600 mt-1">
						Manage backup agents across your infrastructure
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowRegisterModal(true)}
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
					Register Agent
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search agents..."
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
							<option value="all">All Status</option>
							<option value="active">Active</option>
							<option value="offline">Offline</option>
							<option value="pending">Pending</option>
							<option value="disabled">Disabled</option>
						</select>
						<select
							value={groupFilter}
							onChange={(e) => setGroupFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Groups</option>
							<option value="none">No Group</option>
							{groups?.map((group) => (
								<option key={group.id} value={group.id}>
									{group.name}
								</option>
							))}
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">Failed to load agents</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Hostname
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Health
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Groups
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Last Seen
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Registered
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
				) : filteredAgents && filteredAgents.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 border-b border-gray-200">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Hostname
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Status
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Health
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Groups
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Last Seen
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
									Registered
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
									Actions
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
							No agents registered
						</h3>
						<p className="mb-6">
							Install and register an agent to start backing up your systems
						</p>
					</div>
				)}
			</div>

			<RegisterModal
				isOpen={showRegisterModal}
				onClose={() => setShowRegisterModal(false)}
				onSuccess={handleRegisterSuccess}
			/>

			{newApiKey && (
				<ApiKeyModal apiKey={newApiKey} onClose={() => setNewApiKey(null)} />
			)}

			{/* Download section - always visible */}
			<AgentDownloads showInstallCommands={true} />
		</div>
	);
}
