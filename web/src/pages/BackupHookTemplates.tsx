import { useState } from 'react';
import {
	useApplyBackupHookTemplate,
	useBackupHookTemplates,
	useCreateBackupHookTemplate,
	useDeleteBackupHookTemplate,
	useUpdateBackupHookTemplate,
} from '../hooks/useBackupHookTemplates';
import { useSchedules } from '../hooks/useSchedules';
import type {
	BackupHookTemplate,
	BackupHookTemplateScripts,
	BackupHookTemplateVariable,
	BackupHookTemplateVisibility,
} from '../lib/types';

const SERVICE_ICONS: Record<string, string> = {
	database: 'M4 7v10c0 2 2 3.5 8 3.5s8-1.5 8-3.5V7',
	shield: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
	mail: 'M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z',
	workflow: 'M22 11.08V12a10 10 0 1 1-5.93-9.14',
};

function getIconPath(icon?: string): string {
	return SERVICE_ICONS[icon || 'database'] || SERVICE_ICONS.database;
}

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 animate-pulse">
			<div className="flex items-center gap-4 mb-4">
				<div className="w-12 h-12 bg-gray-200 dark:bg-gray-700 rounded-lg" />
				<div>
					<div className="h-5 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-3 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
				</div>
			</div>
			<div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded mb-2" />
			<div className="h-4 w-3/4 bg-gray-200 dark:bg-gray-700 rounded" />
		</div>
	);
}

interface ApplyTemplateModalProps {
	isOpen: boolean;
	onClose: () => void;
	template: BackupHookTemplate;
}

function ApplyTemplateModal({
	isOpen,
	onClose,
	template,
}: ApplyTemplateModalProps) {
	const [selectedScheduleId, setSelectedScheduleId] = useState('');
	const [variableValues, setVariableValues] = useState<Record<string, string>>(
		{},
	);

	const { data: schedules, isLoading: schedulesLoading } = useSchedules();
	const applyTemplate = useApplyBackupHookTemplate();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!selectedScheduleId) return;

		try {
			await applyTemplate.mutateAsync({
				templateId: template.id,
				data: {
					schedule_id: selectedScheduleId,
					variable_values: variableValues,
				},
			});
			onClose();
			setSelectedScheduleId('');
			setVariableValues({});
		} catch {
			// Error handled by mutation
		}
	};

	const handleVariableChange = (name: string, value: string) => {
		setVariableValues((prev) => ({ ...prev, [name]: value }));
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Apply Template: {template.name}
				</h3>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div>
						<label
							htmlFor="select-schedule"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Select Schedule
						</label>
						<select
							id="select-schedule"
							value={selectedScheduleId}
							onChange={(e) => setSelectedScheduleId(e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							required
						>
							<option value="">Select a schedule...</option>
							{schedulesLoading ? (
								<option disabled>Loading schedules...</option>
							) : (
								schedules?.map((schedule) => (
									<option key={schedule.id} value={schedule.id}>
										{schedule.name}
									</option>
								))
							)}
						</select>
					</div>

					{template.variables && template.variables.length > 0 && (
						<div className="space-y-3">
							<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
								Configure Variables
							</h4>
							{template.variables.map((variable) => (
								<div key={variable.name}>
									<label
										htmlFor={`var-${variable.name}`}
										className="block text-sm text-gray-600 dark:text-gray-400 mb-1"
									>
										{variable.name}
										{variable.required && (
											<span className="text-red-500 ml-1">*</span>
										)}
									</label>
									<input
										id={`var-${variable.name}`}
										type={variable.sensitive ? 'password' : 'text'}
										placeholder={variable.default || variable.description}
										value={variableValues[variable.name] || ''}
										onChange={(e) =>
											handleVariableChange(variable.name, e.target.value)
										}
										className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
										required={variable.required}
									/>
									<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
										{variable.description}
									</p>
								</div>
							))}
						</div>
					)}

					<div className="flex justify-end gap-3 pt-4">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={applyTemplate.isPending || !selectedScheduleId}
							className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded-md"
						>
							{applyTemplate.isPending ? 'Applying...' : 'Apply Template'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface CreateTemplateModalProps {
	isOpen: boolean;
	onClose: () => void;
	editTemplate?: BackupHookTemplate;
}

function CreateTemplateModal({
	isOpen,
	onClose,
	editTemplate,
}: CreateTemplateModalProps) {
	const [name, setName] = useState(editTemplate?.name || '');
	const [description, setDescription] = useState(
		editTemplate?.description || '',
	);
	const [serviceType, setServiceType] = useState(
		editTemplate?.service_type || '',
	);
	const [icon, setIcon] = useState(editTemplate?.icon || 'database');
	const [visibility, setVisibility] = useState<BackupHookTemplateVisibility>(
		editTemplate?.visibility === 'built_in'
			? 'private'
			: editTemplate?.visibility || 'private',
	);
	const [tags, setTags] = useState<string[]>(editTemplate?.tags || []);
	const [tagInput, setTagInput] = useState('');
	const [variables, setVariables] = useState<BackupHookTemplateVariable[]>(
		editTemplate?.variables || [],
	);
	const [scripts, setScripts] = useState<BackupHookTemplateScripts>(
		editTemplate?.scripts || {},
	);

	const createTemplate = useCreateBackupHookTemplate();
	const updateTemplate = useUpdateBackupHookTemplate();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();

		try {
			if (editTemplate && editTemplate.visibility !== 'built_in') {
				await updateTemplate.mutateAsync({
					id: editTemplate.id,
					data: {
						name,
						description,
						service_type: serviceType,
						icon,
						visibility,
						tags,
						variables,
						scripts,
					},
				});
			} else {
				await createTemplate.mutateAsync({
					name,
					description,
					service_type: serviceType,
					icon,
					visibility,
					tags,
					variables,
					scripts,
				});
			}
			resetForm();
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const resetForm = () => {
		setName('');
		setDescription('');
		setServiceType('');
		setIcon('database');
		setVisibility('private');
		setTags([]);
		setTagInput('');
		setVariables([]);
		setScripts({});
	};

	const addTag = () => {
		if (tagInput.trim() && !tags.includes(tagInput.trim())) {
			setTags([...tags, tagInput.trim()]);
			setTagInput('');
		}
	};

	const removeTag = (tag: string) => {
		setTags(tags.filter((t) => t !== tag));
	};

	const addVariable = () => {
		setVariables([
			...variables,
			{ name: '', description: '', default: '', required: false },
		]);
	};

	const updateVariable = (
		index: number,
		field: keyof BackupHookTemplateVariable,
		value: string | boolean,
	) => {
		const updated = [...variables];
		updated[index] = { ...updated[index], [field]: value };
		setVariables(updated);
	};

	const removeVariable = (index: number) => {
		setVariables(variables.filter((_, i) => i !== index));
	};

	const updateScript = (
		type: 'pre_backup' | 'post_success' | 'post_failure' | 'post_always',
		field: 'script' | 'timeout_seconds' | 'fail_on_error',
		value: string | number | boolean,
	) => {
		setScripts((prev) => ({
			...prev,
			[type]: {
				...prev[type],
				script: prev[type]?.script || '',
				timeout_seconds: prev[type]?.timeout_seconds || 300,
				fail_on_error: prev[type]?.fail_on_error || false,
				[field]: value,
			},
		}));
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					{editTemplate ? 'Edit Template' : 'Create Custom Template'}
				</h3>

				<form onSubmit={handleSubmit} className="space-y-6">
					{/* Basic Info */}
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="template-name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Name
							</label>
							<input
								id="template-name"
								type="text"
								value={name}
								onChange={(e) => setName(e.target.value)}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								required
							/>
						</div>
						<div>
							<label
								htmlFor="template-service-type"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Service Type
							</label>
							<input
								id="template-service-type"
								type="text"
								value={serviceType}
								onChange={(e) => setServiceType(e.target.value)}
								placeholder="e.g., postgresql, custom-app"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								required
							/>
						</div>
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
							rows={2}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						/>
					</div>

					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="template-icon"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Icon
							</label>
							<select
								id="template-icon"
								value={icon}
								onChange={(e) => setIcon(e.target.value)}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							>
								<option value="database">Database</option>
								<option value="shield">Shield</option>
								<option value="mail">Mail</option>
								<option value="workflow">Workflow</option>
							</select>
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
									setVisibility(e.target.value as BackupHookTemplateVisibility)
								}
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							>
								<option value="private">Private (Only me)</option>
								<option value="organization">Organization (All members)</option>
							</select>
						</div>
					</div>

					{/* Tags */}
					<div>
						<label
							htmlFor="template-tags"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Tags
						</label>
						<div className="flex gap-2 mb-2">
							<input
								id="template-tags"
								type="text"
								value={tagInput}
								onChange={(e) => setTagInput(e.target.value)}
								onKeyDown={(e) => {
									if (e.key === 'Enter') {
										e.preventDefault();
										addTag();
									}
								}}
								placeholder="Add a tag..."
								className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
							<button
								type="button"
								onClick={addTag}
								className="px-3 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md"
							>
								Add
							</button>
						</div>
						<div className="flex flex-wrap gap-2">
							{tags.map((tag) => (
								<span
									key={tag}
									className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
								>
									{tag}
									<button
										type="button"
										onClick={() => removeTag(tag)}
										className="ml-1 text-blue-600 hover:text-blue-800"
									>
										&times;
									</button>
								</span>
							))}
						</div>
					</div>

					{/* Variables */}
					<div>
						<div className="flex justify-between items-center mb-2">
							<span className="block text-sm font-medium text-gray-700 dark:text-gray-300">
								Variables
							</span>
							<button
								type="button"
								onClick={addVariable}
								className="text-sm text-blue-600 hover:text-blue-800"
							>
								+ Add Variable
							</button>
						</div>
						<div className="space-y-3">
							{variables.map((variable, index) => (
								<div
									key={variable.name || `var-${index}`}
									className="flex gap-2 items-start bg-gray-50 dark:bg-gray-700 p-3 rounded-md"
								>
									<input
										type="text"
										value={variable.name}
										onChange={(e) =>
											updateVariable(index, 'name', e.target.value)
										}
										placeholder="Name"
										className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
									/>
									<input
										type="text"
										value={variable.description}
										onChange={(e) =>
											updateVariable(index, 'description', e.target.value)
										}
										placeholder="Description"
										className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
									/>
									<input
										type="text"
										value={variable.default}
										onChange={(e) =>
											updateVariable(index, 'default', e.target.value)
										}
										placeholder="Default"
										className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
									/>
									<label className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
										<input
											type="checkbox"
											checked={variable.required}
											onChange={(e) =>
												updateVariable(index, 'required', e.target.checked)
											}
											className="rounded"
										/>
										Required
									</label>
									<button
										type="button"
										onClick={() => removeVariable(index)}
										className="text-red-600 hover:text-red-800"
									>
										&times;
									</button>
								</div>
							))}
						</div>
					</div>

					{/* Scripts */}
					<div>
						<span className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Scripts
						</span>
						<div className="space-y-4">
							{(
								[
									'pre_backup',
									'post_success',
									'post_failure',
									'post_always',
								] as const
							).map((type) => (
								<div
									key={type}
									className="border border-gray-200 dark:border-gray-700 rounded-md p-4"
								>
									<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 capitalize">
										{type.replace('_', ' ')}
									</h4>
									<textarea
										value={scripts[type]?.script || ''}
										onChange={(e) =>
											updateScript(type, 'script', e.target.value)
										}
										rows={4}
										placeholder="#!/bin/bash&#10;# Your script here..."
										className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
									/>
									<div className="flex gap-4 mt-2">
										<label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
											Timeout (seconds):
											<input
												type="number"
												value={scripts[type]?.timeout_seconds || 300}
												onChange={(e) =>
													updateScript(
														type,
														'timeout_seconds',
														Number.parseInt(e.target.value, 10),
														Number.parseInt(e.target.value),
													)
												}
												min={1}
												max={3600}
												className="w-20 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
											/>
										</label>
										<label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
											<input
												type="checkbox"
												checked={scripts[type]?.fail_on_error || false}
												onChange={(e) =>
													updateScript(type, 'fail_on_error', e.target.checked)
												}
												className="rounded"
											/>
											Fail on error
										</label>
									</div>
								</div>
							))}
						</div>
					</div>

					<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
						<button
							type="button"
							onClick={() => {
								resetForm();
								onClose();
							}}
							className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createTemplate.isPending || updateTemplate.isPending}
							className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded-md"
						>
							{createTemplate.isPending || updateTemplate.isPending
								? 'Saving...'
								: editTemplate
									? 'Update Template'
									: 'Create Template'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface TemplateCardProps {
	template: BackupHookTemplate;
	onApply: () => void;
	onEdit?: () => void;
	onDelete?: () => void;
}

function TemplateCard({
	template,
	onApply,
	onEdit,
	onDelete,
}: TemplateCardProps) {
	const isBuiltIn = template.visibility === 'built_in';

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-md transition-shadow p-6">
			<div className="flex items-start justify-between mb-4">
				<div className="flex items-center gap-4">
					<div className="w-12 h-12 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center">
						<svg
							className="w-6 h-6 text-blue-600 dark:text-blue-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d={getIconPath(template.icon)}
							/>
						</svg>
					</div>
					<div>
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							{template.name}
						</h3>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{template.service_type}
						</p>
					</div>
				</div>
				<div className="flex items-center gap-2">
					{isBuiltIn && (
						<span className="px-2 py-1 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 rounded-full">
							Built-in
						</span>
					)}
					{template.visibility === 'organization' && (
						<span className="px-2 py-1 text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200 rounded-full">
							Shared
						</span>
					)}
				</div>
			</div>

			<p className="text-sm text-gray-600 dark:text-gray-300 mb-4 line-clamp-2">
				{template.description}
			</p>

			{template.tags && template.tags.length > 0 && (
				<div className="flex flex-wrap gap-1 mb-4">
					{template.tags.slice(0, 4).map((tag) => (
						<span
							key={tag}
							className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded"
						>
							{tag}
						</span>
					))}
					{template.tags.length > 4 && (
						<span className="px-2 py-0.5 text-xs text-gray-500">
							+{template.tags.length - 4} more
						</span>
					)}
				</div>
			)}

			<div className="flex items-center justify-between pt-4 border-t border-gray-100 dark:border-gray-700">
				<div className="text-xs text-gray-500 dark:text-gray-400">
					{template.usage_count > 0 && `Used ${template.usage_count} times`}
				</div>
				<div className="flex gap-2">
					{!isBuiltIn && onEdit && (
						<button
							type="button"
							onClick={onEdit}
							className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md"
						>
							Edit
						</button>
					)}
					{!isBuiltIn && onDelete && (
						<button
							type="button"
							onClick={onDelete}
							className="px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md"
						>
							Delete
						</button>
					)}
					<button
						type="button"
						onClick={onApply}
						className="px-4 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md"
					>
						Apply
					</button>
				</div>
			</div>
		</div>
	);
}

export function BackupHookTemplates() {
	const [filter, setFilter] = useState<'all' | 'built_in' | 'custom'>('all');
	const [serviceTypeFilter, setServiceTypeFilter] = useState('');
	const [tagFilter, setTagFilter] = useState('');
	const [searchQuery, setSearchQuery] = useState('');
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [editTemplate, setEditTemplate] = useState<
		BackupHookTemplate | undefined
	>();
	const [applyTemplate, setApplyTemplate] = useState<BackupHookTemplate | null>(
		null,
	);

	const { data: templates, isLoading } = useBackupHookTemplates();
	const deleteTemplate = useDeleteBackupHookTemplate();

	// Get unique service types and tags for filters
	const serviceTypes = Array.from(
		new Set(templates?.map((t) => t.service_type) || []),
	);
	const allTags = Array.from(
		new Set(templates?.flatMap((t) => t.tags || []) || []),
	);

	// Filter templates
	const filteredTemplates = templates?.filter((t) => {
		if (filter === 'built_in' && t.visibility !== 'built_in') return false;
		if (filter === 'custom' && t.visibility === 'built_in') return false;
		if (serviceTypeFilter && t.service_type !== serviceTypeFilter) return false;
		if (tagFilter && !t.tags?.includes(tagFilter)) return false;
		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			return (
				t.name.toLowerCase().includes(query) ||
				t.description?.toLowerCase().includes(query) ||
				t.service_type.toLowerCase().includes(query)
			);
		}
		return true;
	});

	const handleDelete = async (id: string) => {
		if (!confirm('Are you sure you want to delete this template?')) return;
		try {
			await deleteTemplate.mutateAsync(id);
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="p-6">
			<div className="flex justify-between items-center mb-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Backup Hook Templates
					</h1>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Pre-built and custom templates for backup hooks
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowCreateModal(true)}
					className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md"
				>
					Create Template
				</button>
			</div>

			{/* Filters */}
			<div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-6">
				<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
					<div>
						<label
							htmlFor="filter-search"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Search
						</label>
						<input
							id="filter-search"
							type="text"
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							placeholder="Search templates..."
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						/>
					</div>
					<div>
						<label
							htmlFor="filter-type"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Type
						</label>
						<select
							id="filter-type"
							value={filter}
							onChange={(e) =>
								setFilter(e.target.value as 'all' | 'built_in' | 'custom')
							}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						>
							<option value="all">All Templates</option>
							<option value="built_in">Built-in Only</option>
							<option value="custom">Custom Only</option>
						</select>
					</div>
					<div>
						<label
							htmlFor="filter-service-type"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Service Type
						</label>
						<select
							id="filter-service-type"
							value={serviceTypeFilter}
							onChange={(e) => setServiceTypeFilter(e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						>
							<option value="">All Services</option>
							{serviceTypes.map((type) => (
								<option key={type} value={type}>
									{type}
								</option>
							))}
						</select>
					</div>
					<div>
						<label
							htmlFor="filter-tag"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Tag
						</label>
						<select
							id="filter-tag"
							value={tagFilter}
							onChange={(e) => setTagFilter(e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						>
							<option value="">All Tags</option>
							{allTags.map((tag) => (
								<option key={tag} value={tag}>
									{tag}
								</option>
							))}
						</select>
					</div>
				</div>
			</div>

			{/* Templates Grid */}
			{isLoading ? (
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{[...Array(6)].map((_, i) => (
						// biome-ignore lint/suspicious/noArrayIndexKey: Static loading skeleton has no state
						<LoadingCard key={`loading-card-${i}`} />
					))}
				</div>
			) : filteredTemplates && filteredTemplates.length > 0 ? (
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{filteredTemplates.map((template) => (
						<TemplateCard
							key={template.id}
							template={template}
							onApply={() => setApplyTemplate(template)}
							onEdit={
								template.visibility !== 'built_in'
									? () => setEditTemplate(template)
									: undefined
							}
							onDelete={
								template.visibility !== 'built_in'
									? () => handleDelete(template.id)
									: undefined
							}
						/>
					))}
				</div>
			) : (
				<div className="bg-white dark:bg-gray-800 rounded-lg shadow p-12 text-center">
					<svg
						className="mx-auto h-12 w-12 text-gray-400"
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
					<h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
						No templates found
					</h3>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						{searchQuery || filter !== 'all' || serviceTypeFilter || tagFilter
							? 'Try adjusting your filters'
							: 'Get started by creating a custom template'}
					</p>
					<div className="mt-6">
						<button
							type="button"
							onClick={() => setShowCreateModal(true)}
							className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md"
						>
							Create Template
						</button>
					</div>
				</div>
			)}

			{/* Modals */}
			<CreateTemplateModal
				isOpen={showCreateModal || !!editTemplate}
				onClose={() => {
					setShowCreateModal(false);
					setEditTemplate(undefined);
				}}
				editTemplate={editTemplate}
			/>

			{applyTemplate && (
				<ApplyTemplateModal
					isOpen={!!applyTemplate}
					onClose={() => setApplyTemplate(null)}
					template={applyTemplate}
				/>
			)}
		</div>
	);
}
