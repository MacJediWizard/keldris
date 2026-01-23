import { useState } from 'react';
import {
	useCreateTemplate,
	useDeleteTemplate,
	useTemplates,
	useUseTemplate,
} from '../hooks/useConfigExport';
import type {
	ConfigTemplate,
	ConfigType,
	ConflictResolution,
	CreateTemplateRequest,
	TemplateVisibility,
} from '../lib/types';
import { formatDate } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-32" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-20" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-24" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-16" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-24" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

interface CreateTemplateModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateTemplateModal({ isOpen, onClose }: CreateTemplateModalProps) {
	const [name, setName] = useState('');
	const [description, setDescription] = useState('');
	const [visibility, setVisibility] = useState<TemplateVisibility>('organization');
	const [tags, setTags] = useState('');
	const [config, setConfig] = useState('');

	const createTemplate = useCreateTemplate();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const tagList = tags
				.split(',')
				.map((t) => t.trim())
				.filter(Boolean);

			// Validate JSON
			try {
				JSON.parse(config);
			} catch {
				alert('Invalid JSON configuration');
				return;
			}

			const data: CreateTemplateRequest = {
				name,
				description: description || undefined,
				visibility,
				tags: tagList.length > 0 ? tagList : undefined,
				config, // Send as string
			};

			await createTemplate.mutateAsync(data);
			onClose();
			// Reset form
			setName('');
			setDescription('');
			setVisibility('organization');
			setTags('');
			setConfig('');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Create Template
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="template-name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="template-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., Web Server Backup"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>

						<div>
							<label
								htmlFor="template-description"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Description
							</label>
							<textarea
								id="template-description"
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								placeholder="Describe what this template is for..."
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>

						<div>
							<label
								htmlFor="template-visibility"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Visibility
							</label>
							<select
								id="template-visibility"
								value={visibility}
								onChange={(e) =>
									setVisibility(e.target.value as TemplateVisibility)
								}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							>
								<option value="private">Private</option>
								<option value="organization">Organization</option>
								<option value="public">Public</option>
							</select>
							<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
								The type is auto-detected from the configuration
							</p>
						</div>

						<div>
							<label
								htmlFor="template-tags"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Tags (comma-separated)
							</label>
							<input
								type="text"
								id="template-tags"
								value={tags}
								onChange={(e) => setTags(e.target.value)}
								placeholder="e.g., web, production, nginx"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							/>
						</div>

						<div>
							<label
								htmlFor="template-config"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Configuration (JSON)
							</label>
							<textarea
								id="template-config"
								value={config}
								onChange={(e) => setConfig(e.target.value)}
								placeholder='{"paths": ["/var/www"], "cron_expression": "0 2 * * *"}'
								rows={6}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
								required
							/>
							<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
								Paste an exported configuration or create one manually
							</p>
						</div>
					</div>

					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createTemplate.isPending || !name || !config}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createTemplate.isPending ? 'Creating...' : 'Create Template'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface UseTemplateModalProps {
	template: ConfigTemplate;
	isOpen: boolean;
	onClose: () => void;
}

function UseTemplateModal({ template, isOpen, onClose }: UseTemplateModalProps) {
	const [conflictResolution, setConflictResolution] = useState<ConflictResolution>('skip');
	const useTemplate = useUseTemplate();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await useTemplate.mutateAsync({
				id: template.id,
				data: {
					conflict_resolution: conflictResolution,
				},
			});
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
					Use Template: {template.name}
				</h3>
				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					This will create new resources based on this template.
				</p>

				<div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 mb-4">
					<div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
						Template Configuration:
					</div>
					<pre className="text-xs text-gray-700 dark:text-gray-300 overflow-auto max-h-40 font-mono">
						{JSON.stringify(template.config, null, 2)}
					</pre>
				</div>

				<form onSubmit={handleSubmit}>
					<div>
						<label
							htmlFor="conflict-resolution"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Conflict Resolution
						</label>
						<select
							id="conflict-resolution"
							value={conflictResolution}
							onChange={(e) => setConflictResolution(e.target.value as ConflictResolution)}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="skip">Skip conflicting items</option>
							<option value="replace">Replace existing items</option>
							<option value="rename">Rename imported items</option>
							<option value="fail">Fail if any conflicts</option>
						</select>
						<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
							Choose how to handle items that already exist
						</p>
					</div>

					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={useTemplate.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{useTemplate.isPending ? 'Creating...' : 'Use Template'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

function getTypeColor(type: ConfigType): string {
	switch (type) {
		case 'schedule':
			return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
		case 'agent':
			return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
		case 'repository':
			return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200';
		case 'bundle':
			return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200';
		default:
			return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
	}
}

function getVisibilityIcon(visibility: TemplateVisibility): JSX.Element {
	switch (visibility) {
		case 'public':
			return (
				<svg
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			);
		case 'organization':
			return (
				<svg
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
					/>
				</svg>
			);
		case 'private':
		default:
			return (
				<svg
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
					/>
				</svg>
			);
	}
}

interface TemplateRowProps {
	template: ConfigTemplate;
	onUse: (template: ConfigTemplate) => void;
	onDelete: (id: string) => void;
}

function TemplateRow({ template, onUse, onDelete }: TemplateRowProps) {
	const [showActions, setShowActions] = useState(false);

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="font-medium text-gray-900 dark:text-white">
					{template.name}
				</div>
				{template.description && (
					<div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-xs">
						{template.description}
					</div>
				)}
				{template.tags && template.tags.length > 0 && (
					<div className="flex flex-wrap gap-1 mt-1">
						{template.tags.map((tag) => (
							<span
								key={tag}
								className="px-1.5 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded"
							>
								{tag}
							</span>
						))}
					</div>
				)}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${getTypeColor(template.type)}`}
				>
					{template.type}
				</span>
			</td>
			<td className="px-6 py-4">
				<span
					className="inline-flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400"
					title={template.visibility}
				>
					{getVisibilityIcon(template.visibility)}
					<span className="capitalize">{template.visibility}</span>
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{template.usage_count}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(template.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="relative">
					<button
						type="button"
						onClick={() => setShowActions(!showActions)}
						className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
					>
						<svg
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
							aria-hidden="true"
						>
							<path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
						</svg>
					</button>
					{showActions && (
						<>
							<div
								className="fixed inset-0 z-10"
								onClick={() => setShowActions(false)}
								onKeyDown={(e) => {
									if (e.key === 'Escape') setShowActions(false);
								}}
								role="button"
								tabIndex={0}
								aria-label="Close menu"
							/>
							<div className="absolute right-0 mt-2 w-40 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-20">
								<button
									type="button"
									onClick={() => {
										onUse(template);
										setShowActions(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
								>
									Use Template
								</button>
								<button
									type="button"
									onClick={() => {
										if (
											window.confirm(
												`Are you sure you want to delete "${template.name}"?`,
											)
										) {
											onDelete(template.id);
										}
										setShowActions(false);
									}}
									className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
								>
									Delete
								</button>
							</div>
						</>
					)}
				</div>
			</td>
		</tr>
	);
}

export function Templates() {
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [useTemplateItem, setUseTemplateItem] = useState<ConfigTemplate | null>(
		null,
	);
	const [typeFilter, setTypeFilter] = useState<ConfigType | 'all'>('all');
	const [searchQuery, setSearchQuery] = useState('');

	const { data: templates, isLoading, isError } = useTemplates();
	const deleteTemplate = useDeleteTemplate();

	const filteredTemplates = templates?.filter((template) => {
		const matchesType = typeFilter === 'all' || template.type === typeFilter;
		const matchesSearch =
			template.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			template.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
			template.tags?.some((tag) =>
				tag.toLowerCase().includes(searchQuery.toLowerCase()),
			);
		return matchesType && matchesSearch;
	});

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Templates
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Reusable configuration templates for quick setup
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
					Create Template
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search templates..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={typeFilter}
							onChange={(e) => setTypeFilter(e.target.value as ConfigType | 'all')}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Types</option>
							<option value="schedule">Schedule</option>
							<option value="agent">Agent</option>
							<option value="repository">Repository</option>
							<option value="bundle">Bundle</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400">
						<p className="font-medium">Failed to load templates</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Template
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Type
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Visibility
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Uses
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
				) : filteredTemplates && filteredTemplates.length > 0 ? (
					<table className="w-full">
						<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Template
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Type
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Visibility
								</th>
								<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
									Uses
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
							{filteredTemplates.map((template) => (
								<TemplateRow
									key={template.id}
									template={template}
									onUse={setUseTemplateItem}
									onDelete={(id) => deleteTemplate.mutate(id)}
								/>
							))}
						</tbody>
					</table>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
						<svg
							aria-hidden="true"
							className="w-16 h-16 mx-auto mb-4 text-gray-300 dark:text-gray-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No templates yet
						</h3>
						<p className="mb-4">
							Create templates from exported configurations for quick reuse
						</p>
						<button
							type="button"
							onClick={() => setShowCreateModal(true)}
							className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 font-medium"
						>
							Create your first template
						</button>
					</div>
				)}
			</div>

			<CreateTemplateModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			{useTemplateItem && (
				<UseTemplateModal
					template={useTemplateItem}
					isOpen={!!useTemplateItem}
					onClose={() => setUseTemplateItem(null)}
				/>
			)}
		</div>
	);
}
