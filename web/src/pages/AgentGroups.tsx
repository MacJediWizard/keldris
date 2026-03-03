import { useState } from 'react';
import { EmptyStateNoGroups } from '../components/ui/EmptyState';
import {
	useAddAgentToGroup,
	useAgentGroupMembers,
	useAgentGroups,
	useCreateAgentGroup,
	useDeleteAgentGroup,
	useRemoveAgentFromGroup,
	useUpdateAgentGroup,
} from '../hooks/useAgentGroups';
import { useAgents } from '../hooks/useAgents';
import type { AgentGroup } from '../lib/types';
import { formatDate } from '../lib/utils';

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
				<div className="h-6 w-8 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

interface CreateModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: () => void;
}

function CreateModal({ isOpen, onClose, onSuccess }: CreateModalProps) {
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [color, setColor] = useState('#6366f1');
	const createGroup = useCreateAgentGroup();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createGroup.mutateAsync({
				name,
				description: description || undefined,
				color,
			});
			onSuccess();
			setName('');
			setDescription('');
			setColor('#6366f1');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Create Agent Group
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
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
							placeholder="e.g., Production Servers"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
							required
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="description"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Description (optional)
						</label>
						<textarea
							id="description"
							value={description}
							onChange={(e) => setDescription(e.target.value)}
							placeholder="Describe this group..."
							rows={2}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="color"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Color
						</label>
						<div className="flex items-center gap-2">
							<input
								type="color"
								id="color"
								value={color}
								onChange={(e) => setColor(e.target.value)}
								className="w-10 h-10 border border-gray-300 dark:border-gray-600 rounded cursor-pointer"
							/>
							<input
								type="text"
								value={color}
								onChange={(e) => setColor(e.target.value)}
								placeholder="#6366f1"
								className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
							/>
						</div>
					</div>
					{createGroup.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to create group. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createGroup.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createGroup.isPending ? 'Creating...' : 'Create'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface EditModalProps {
	group: AgentGroup;
	isOpen: boolean;
	onClose: () => void;
	onSuccess: () => void;
}

function EditModal({ group, isOpen, onClose, onSuccess }: EditModalProps) {
	const [name, setName] = useState(group.name);
	const [description, setDescription] = useState(group.description || '');
	const [color, setColor] = useState(group.color || '#6366f1');
	const updateGroup = useUpdateAgentGroup();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updateGroup.mutateAsync({
				id: group.id,
				data: {
					name,
					description: description || undefined,
					color,
				},
			});
			onSuccess();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Edit Agent Group
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="edit-name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Name
						</label>
						<input
							type="text"
							id="edit-name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
							required
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="edit-description"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Description (optional)
						</label>
						<textarea
							id="edit-description"
							value={description}
							onChange={(e) => setDescription(e.target.value)}
							rows={2}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div className="mb-4">
						<label
							htmlFor="edit-color"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Color
						</label>
						<div className="flex items-center gap-2">
							<input
								type="color"
								id="edit-color"
								value={color}
								onChange={(e) => setColor(e.target.value)}
								className="w-10 h-10 border border-gray-300 dark:border-gray-600 rounded cursor-pointer"
							/>
							<input
								type="text"
								value={color}
								onChange={(e) => setColor(e.target.value)}
								className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
							/>
						</div>
					</div>
					{updateGroup.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to update group. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={updateGroup.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{updateGroup.isPending ? 'Saving...' : 'Save'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface ManageMembersModalProps {
	group: AgentGroup;
	isOpen: boolean;
	onClose: () => void;
}

function ManageMembersModal({
	group,
	isOpen,
	onClose,
}: ManageMembersModalProps) {
	const { data: members, isLoading: membersLoading } = useAgentGroupMembers(
		group.id,
	);
	const { data: allAgents, isLoading: agentsLoading } = useAgents();
	const addAgent = useAddAgentToGroup();
	const removeAgent = useRemoveAgentFromGroup();

	const memberIds = new Set(members?.map((m) => m.id) || []);
	const availableAgents = allAgents?.filter((a) => !memberIds.has(a.id)) || [];

	const handleAddAgent = async (agentId: string) => {
		try {
			await addAgent.mutateAsync({
				groupId: group.id,
				data: { agent_id: agentId },
			});
		} catch {
			// Error handled by mutation
		}
	};

	const handleRemoveAgent = async (agentId: string) => {
		if (confirm('Are you sure you want to remove this agent from the group?')) {
			try {
				await removeAgent.mutateAsync({
					groupId: group.id,
					agentId,
				});
			} catch {
				// Error handled by mutation
			}
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden flex flex-col">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Manage Group Members: {group.name}
					</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
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

				<div className="flex-1 overflow-auto">
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						{/* Current Members */}
						<div>
							<h4 className="font-medium text-gray-900 dark:text-white mb-2">
								Current Members ({members?.length || 0})
							</h4>
							<div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
								{membersLoading ? (
									<div className="p-4 text-center text-gray-500 dark:text-gray-400">
										Loading...
									</div>
								) : members && members.length > 0 ? (
									<ul className="divide-y divide-gray-200 dark:divide-gray-700 max-h-64 overflow-auto">
										{members.map((agent) => (
											<li
												key={agent.id}
												className="p-3 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700"
											>
												<span className="text-sm text-gray-900 dark:text-white">
													{agent.hostname}
												</span>
												<button
													type="button"
													onClick={() => handleRemoveAgent(agent.id)}
													disabled={removeAgent.isPending}
													className="text-red-600 hover:text-red-800 text-sm"
												>
													Remove
												</button>
											</li>
										))}
									</ul>
								) : (
									<div className="p-4 text-center text-gray-500 dark:text-gray-400 text-sm">
										No agents in this group
									</div>
								)}
							</div>
						</div>

						{/* Available Agents */}
						<div>
							<h4 className="font-medium text-gray-900 dark:text-white mb-2">
								Available Agents ({availableAgents.length})
							</h4>
							<div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
								{agentsLoading ? (
									<div className="p-4 text-center text-gray-500 dark:text-gray-400">
										Loading...
									</div>
								) : availableAgents.length > 0 ? (
									<ul className="divide-y divide-gray-200 dark:divide-gray-700 max-h-64 overflow-auto">
										{availableAgents.map((agent) => (
											<li
												key={agent.id}
												className="p-3 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700"
											>
												<span className="text-sm text-gray-900 dark:text-white">
													{agent.hostname}
												</span>
												<button
													type="button"
													onClick={() => handleAddAgent(agent.id)}
													disabled={addAgent.isPending}
													className="text-indigo-600 hover:text-indigo-800 text-sm"
												>
													Add
												</button>
											</li>
										))}
									</ul>
								) : (
									<div className="p-4 text-center text-gray-500 dark:text-gray-400 text-sm">
										All agents are already in this group
									</div>
								)}
							</div>
						</div>
					</div>
				</div>

				<div className="flex justify-end mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
					>
						Done
					</button>
				</div>
			</div>
		</div>
	);
}

interface GroupRowProps {
	group: AgentGroup;
	onEdit: (group: AgentGroup) => void;
	onManageMembers: (group: AgentGroup) => void;
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

function GroupRow({
	group,
	onEdit,
	onManageMembers,
	onDelete,
	isDeleting,
}: GroupRowProps) {
	const [showMenu, setShowMenu] = useState(false);

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					{group.color && (
						<div
							className="w-3 h-3 rounded-full"
							style={{ backgroundColor: group.color }}
						/>
					)}
					<span className="font-medium text-gray-900 dark:text-white">
						{group.name}
					</span>
				</div>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{group.description || '-'}
			</td>
			<td className="px-6 py-4">
				<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
					{group.agent_count} agent{group.agent_count !== 1 ? 's' : ''}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(group.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="relative inline-block text-left">
					<button
						type="button"
						onClick={() => setShowMenu(!showMenu)}
						className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
								aria-label="Close menu"
							/>
							<div className="absolute right-0 z-20 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1">
								<button
									type="button"
									onClick={() => {
										onManageMembers(group);
										setShowMenu(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
								>
									Manage Agents
								</button>
								<button
									type="button"
									onClick={() => {
										onEdit(group);
										setShowMenu(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
								>
									Edit Group
								</button>
								<div className="border-t border-gray-100 dark:border-gray-700 my-1" />
								<button
									type="button"
									onClick={() => {
										onDelete(group.id);
										setShowMenu(false);
									}}
									disabled={isDeleting}
									className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 disabled:opacity-50"
								>
									{isDeleting ? 'Deleting...' : 'Delete Group'}
								</button>
							</div>
						</>
					)}
				</div>
			</td>
		</tr>
	);
}

export function AgentGroups() {
	const [searchQuery, setSearchQuery] = useState('');
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [editingGroup, setEditingGroup] = useState<AgentGroup | null>(null);
	const [managingGroup, setManagingGroup] = useState<AgentGroup | null>(null);

	const { data: groups, isLoading, isError } = useAgentGroups();
	const deleteGroup = useDeleteAgentGroup();

	const filteredGroups = groups?.filter((group) =>
		group.name.toLowerCase().includes(searchQuery.toLowerCase()),
	);

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this group?')) {
			deleteGroup.mutate(id);
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Agent Groups
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Organize agents by environment or purpose
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
					Create Group
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<input
						type="text"
						placeholder="Search groups..."
						value={searchQuery}
						onChange={(e) => setSearchQuery(e.target.value)}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
					/>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
						<p className="font-medium">Failed to load groups</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Description
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Agents
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Created
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							<LoadingRow />
							<LoadingRow />
							<LoadingRow />
						</tbody>
					</table>
				) : filteredGroups && filteredGroups.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Name
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Description
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Agents
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Created
								</th>
								<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
							{filteredGroups.map((group) => (
								<GroupRow
									key={group.id}
									group={group}
									onEdit={setEditingGroup}
									onManageMembers={setManagingGroup}
									onDelete={handleDelete}
									isDeleting={deleteGroup.isPending}
								/>
							))}
						</tbody>
					</table>
				) : (
					<EmptyStateNoGroups onCreateGroup={() => setShowCreateModal(true)} />
				)}
			</div>

			<CreateModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
				onSuccess={() => setShowCreateModal(false)}
			/>

			{editingGroup && (
				<EditModal
					group={editingGroup}
					isOpen={true}
					onClose={() => setEditingGroup(null)}
					onSuccess={() => setEditingGroup(null)}
				/>
			)}

			{managingGroup && (
				<ManageMembersModal
					group={managingGroup}
					isOpen={true}
					onClose={() => setManagingGroup(null)}
				/>
			)}
		</div>
	);
}

export default AgentGroups;
