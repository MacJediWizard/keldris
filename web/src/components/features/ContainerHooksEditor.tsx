import { useState } from 'react';
import {
	useContainerHooks,
	useCreateContainerHook,
	useDeleteContainerHook,
	useUpdateContainerHook,
	useContainerHookTemplates,
} from '../../hooks/useContainerHooks';
import type {
	ContainerBackupHook,
	ContainerHookType,
	ContainerHookTemplate,
	ContainerHookTemplateInfo,
} from '../../lib/types';

interface ContainerHooksEditorProps {
	scheduleId: string;
	onClose: () => void;
}

const HOOK_TYPES: {
	value: ContainerHookType;
	label: string;
	description: string;
}[] = [
	{
		value: 'pre_backup',
		label: 'Pre-Backup',
		description: 'Runs before the backup starts (e.g., dump database)',
	},
	{
		value: 'post_backup',
		label: 'Post-Backup',
		description: 'Runs after the backup completes (e.g., cleanup)',
	},
];

interface HookFormProps {
	hook?: ContainerBackupHook;
	hookType: ContainerHookType;
	scheduleId: string;
	templates: ContainerHookTemplateInfo[];
	onSave: () => void;
	onCancel: () => void;
}

function HookForm({
	hook,
	hookType,
	scheduleId,
	templates,
	onSave,
	onCancel,
}: HookFormProps) {
	const [containerName, setContainerName] = useState(hook?.container_name ?? '');
	const [selectedTemplate, setSelectedTemplate] = useState<ContainerHookTemplate>(
		hook?.template ?? 'none',
	);
	const [command, setCommand] = useState(hook?.command ?? '');
	const [workingDir, setWorkingDir] = useState(hook?.working_dir ?? '');
	const [user, setUser] = useState(hook?.user ?? '');
	const [timeoutSeconds, setTimeoutSeconds] = useState(
		hook?.timeout_seconds ?? 300,
	);
	const [failOnError, setFailOnError] = useState(hook?.fail_on_error ?? false);
	const [enabled, setEnabled] = useState(hook?.enabled ?? true);
	const [description, setDescription] = useState(hook?.description ?? '');
	const [templateVars, setTemplateVars] = useState<Record<string, string>>(
		hook?.template_vars ?? {},
	);

	const createHook = useCreateContainerHook();
	const updateHook = useUpdateContainerHook();

	const templateInfo = templates.find((t) => t.template === selectedTemplate);

	const handleTemplateChange = (newTemplate: ContainerHookTemplate) => {
		setSelectedTemplate(newTemplate);
		const info = templates.find((t) => t.template === newTemplate);
		if (info) {
			// Initialize template vars with defaults
			setTemplateVars({ ...info.default_vars });
			// Set command from template
			const cmd = hookType === 'pre_backup' ? info.pre_backup_cmd : info.post_backup_cmd;
			setCommand(cmd);
		} else {
			setCommand('');
			setTemplateVars({});
		}
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();

		try {
			if (hook) {
				await updateHook.mutateAsync({
					scheduleId,
					id: hook.id,
					data: {
						container_name: containerName,
						command: selectedTemplate === 'none' ? command : undefined,
						working_dir: workingDir || undefined,
						user: user || undefined,
						timeout_seconds: timeoutSeconds,
						fail_on_error: failOnError,
						enabled,
						description: description || undefined,
						template_vars: selectedTemplate !== 'none' ? templateVars : undefined,
					},
				});
			} else {
				await createHook.mutateAsync({
					scheduleId,
					data: {
						container_name: containerName,
						type: hookType,
						template: selectedTemplate !== 'none' ? selectedTemplate : undefined,
						command: selectedTemplate === 'none' ? command : undefined,
						working_dir: workingDir || undefined,
						user: user || undefined,
						timeout_seconds: timeoutSeconds,
						fail_on_error: failOnError,
						enabled,
						description: description || undefined,
						template_vars: selectedTemplate !== 'none' ? templateVars : undefined,
					},
				});
			}
			onSave();
		} catch {
			// Error handled by mutation
		}
	};

	const isPending = createHook.isPending || updateHook.isPending;
	const typeInfo = HOOK_TYPES.find((t) => t.value === hookType);

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<h4 className="text-sm font-medium text-gray-900">{typeInfo?.label}</h4>
				<p className="text-xs text-gray-500">{typeInfo?.description}</p>
			</div>

			<div>
				<label
					htmlFor="container-name"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Container Name
				</label>
				<input
					type="text"
					id="container-name"
					value={containerName}
					onChange={(e) => setContainerName(e.target.value)}
					placeholder="my-database"
					className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					required
				/>
				<p className="text-xs text-gray-500 mt-1">
					The Docker container name or ID to execute the hook in.
				</p>
			</div>

			<div>
				<label
					htmlFor="template"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Template
				</label>
				<select
					id="template"
					value={selectedTemplate}
					onChange={(e) => handleTemplateChange(e.target.value as ContainerHookTemplate)}
					className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					disabled={!!hook}
				>
					<option value="none">Custom Command</option>
					{templates.map((t) => (
						<option key={t.template} value={t.template}>
							{t.name} - {t.description}
						</option>
					))}
				</select>
			</div>

			{selectedTemplate === 'none' ? (
				<div>
					<label
						htmlFor="command"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						Command
					</label>
					<textarea
						id="command"
						value={command}
						onChange={(e) => setCommand(e.target.value)}
						placeholder="pg_dump -U postgres -d mydb > /tmp/backup.sql"
						rows={4}
						className="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						required
					/>
					<p className="text-xs text-gray-500 mt-1">
						Shell command to execute inside the container (runs with sh -c).
					</p>
				</div>
			) : (
				<>
					{templateInfo && (
						<div className="bg-gray-50 border border-gray-200 rounded-lg p-3">
							<p className="text-sm text-gray-700 mb-2">{templateInfo.description}</p>
							<div className="bg-gray-100 p-2 rounded text-xs font-mono text-gray-600 whitespace-pre-wrap">
								{hookType === 'pre_backup'
									? templateInfo.pre_backup_cmd
									: templateInfo.post_backup_cmd}
							</div>
						</div>
					)}

					{templateInfo && (templateInfo.required_vars.length > 0 || templateInfo.optional_vars.length > 0) && (
						<div className="space-y-3">
							<label className="block text-sm font-medium text-gray-700">
								Template Variables
							</label>
							{templateInfo.required_vars.map((varName) => (
								<div key={varName}>
									<label
										htmlFor={`var-${varName}`}
										className="block text-xs font-medium text-gray-600 mb-1"
									>
										{varName} <span className="text-red-500">*</span>
									</label>
									<input
										type="text"
										id={`var-${varName}`}
										value={templateVars[varName] ?? ''}
										onChange={(e) =>
											setTemplateVars({ ...templateVars, [varName]: e.target.value })
										}
										className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										required
									/>
								</div>
							))}
							{templateInfo.optional_vars.map((varName) => (
								<div key={varName}>
									<label
										htmlFor={`var-${varName}`}
										className="block text-xs font-medium text-gray-600 mb-1"
									>
										{varName}
										{templateInfo.default_vars[varName] && (
											<span className="text-gray-400 ml-1">
												(default: {templateInfo.default_vars[varName]})
											</span>
										)}
									</label>
									<input
										type="text"
										id={`var-${varName}`}
										value={templateVars[varName] ?? ''}
										onChange={(e) =>
											setTemplateVars({ ...templateVars, [varName]: e.target.value })
										}
										placeholder={templateInfo.default_vars[varName]}
										className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
							))}
						</div>
					)}
				</>
			)}

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="working-dir"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						Working Directory
					</label>
					<input
						type="text"
						id="working-dir"
						value={workingDir}
						onChange={(e) => setWorkingDir(e.target.value)}
						placeholder="/app"
						className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					/>
				</div>

				<div>
					<label
						htmlFor="user"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						User
					</label>
					<input
						type="text"
						id="user"
						value={user}
						onChange={(e) => setUser(e.target.value)}
						placeholder="postgres"
						className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					/>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="timeout-seconds"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						Timeout (seconds)
					</label>
					<input
						type="number"
						id="timeout-seconds"
						value={timeoutSeconds}
						onChange={(e) =>
							setTimeoutSeconds(Number.parseInt(e.target.value, 10) || 300)
						}
						min={1}
						max={3600}
						className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					/>
				</div>

				<div className="flex flex-col justify-end">
					<label className="flex items-center gap-2 cursor-pointer">
						<input
							type="checkbox"
							checked={enabled}
							onChange={(e) => setEnabled(e.target.checked)}
							className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
						/>
						<span className="text-sm text-gray-700">Enabled</span>
					</label>
				</div>
			</div>

			{hookType === 'pre_backup' && (
				<div>
					<label className="flex items-center gap-2 cursor-pointer">
						<input
							type="checkbox"
							checked={failOnError}
							onChange={(e) => setFailOnError(e.target.checked)}
							className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
						/>
						<span className="text-sm text-gray-700">
							Fail backup if hook fails
						</span>
					</label>
					<p className="text-xs text-gray-500 mt-1 ml-6">
						If enabled and this hook fails, the backup will be aborted.
					</p>
				</div>
			)}

			<div>
				<label
					htmlFor="description"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Description
				</label>
				<input
					type="text"
					id="description"
					value={description}
					onChange={(e) => setDescription(e.target.value)}
					placeholder="Optional description"
					className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				/>
			</div>

			<div className="flex justify-end gap-3 pt-2">
				<button
					type="button"
					onClick={onCancel}
					className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={isPending || !containerName.trim() || (selectedTemplate === 'none' && !command.trim())}
					className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isPending ? 'Saving...' : hook ? 'Update Hook' : 'Create Hook'}
				</button>
			</div>
		</form>
	);
}

interface HookCardProps {
	hook: ContainerBackupHook;
	scheduleId: string;
	onEdit: (hook: ContainerBackupHook) => void;
}

function HookCard({ hook, scheduleId, onEdit }: HookCardProps) {
	const deleteHook = useDeleteContainerHook();
	const typeInfo = HOOK_TYPES.find((t) => t.value === hook.type);

	const handleDelete = async () => {
		if (confirm('Are you sure you want to delete this container hook?')) {
			await deleteHook.mutateAsync({ scheduleId, id: hook.id });
		}
	};

	return (
		<div
			className={`border rounded-lg p-4 ${hook.enabled ? 'bg-white' : 'bg-gray-50 border-gray-200'}`}
		>
			<div className="flex items-start justify-between mb-2">
				<div>
					<div className="flex items-center gap-2">
						<h4 className="text-sm font-medium text-gray-900">
							{hook.container_name}
						</h4>
						<span className="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded">
							{typeInfo?.label}
						</span>
						{hook.template !== 'none' && (
							<span className="text-xs px-2 py-0.5 bg-purple-100 text-purple-700 rounded">
								{hook.template}
							</span>
						)}
						{!hook.enabled && (
							<span className="text-xs px-2 py-0.5 bg-gray-200 text-gray-600 rounded">
								Disabled
							</span>
						)}
					</div>
					{hook.description && (
						<p className="text-xs text-gray-500 mt-1">{hook.description}</p>
					)}
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => onEdit(hook)}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						Edit
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={deleteHook.isPending}
						className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
					>
						Delete
					</button>
				</div>
			</div>
			<pre className="text-xs bg-gray-100 p-2 rounded overflow-x-auto max-h-24 whitespace-pre-wrap">
				{hook.command.substring(0, 300)}
				{hook.command.length > 300 && '...'}
			</pre>
			<div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
				<span>Timeout: {hook.timeout_seconds}s</span>
				{hook.working_dir && <span>Dir: {hook.working_dir}</span>}
				{hook.user && <span>User: {hook.user}</span>}
				{hook.type === 'pre_backup' && hook.fail_on_error && (
					<span className="text-amber-600">Fails backup on error</span>
				)}
			</div>
		</div>
	);
}

export function ContainerHooksEditor({
	scheduleId,
	onClose,
}: ContainerHooksEditorProps) {
	const { data: hooks, isLoading, isError } = useContainerHooks(scheduleId);
	const { data: templates = [] } = useContainerHookTemplates();
	const [editingHook, setEditingHook] = useState<ContainerBackupHook | null>(null);
	const [creatingType, setCreatingType] = useState<ContainerHookType | null>(null);

	if (isLoading) {
		return (
			<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
				<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4">
					<div className="animate-pulse">
						<div className="h-6 w-48 bg-gray-200 rounded mb-4" />
						<div className="h-32 bg-gray-200 rounded" />
					</div>
				</div>
			</div>
		);
	}

	if (isError) {
		return (
			<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
				<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4">
					<p className="text-red-600">Failed to load container hooks</p>
					<button
						type="button"
						onClick={onClose}
						className="mt-4 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
					>
						Close
					</button>
				</div>
			</div>
		);
	}

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						Container Backup Hooks
					</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-600"
						aria-label="Close"
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
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>

				{editingHook || creatingType ? (
					<HookForm
						hook={editingHook ?? undefined}
						hookType={editingHook?.type ?? creatingType ?? 'pre_backup'}
						scheduleId={scheduleId}
						templates={templates}
						onSave={() => {
							setEditingHook(null);
							setCreatingType(null);
						}}
						onCancel={() => {
							setEditingHook(null);
							setCreatingType(null);
						}}
					/>
				) : (
					<>
						<p className="text-sm text-gray-600 mb-4">
							Configure hooks to run inside Docker containers before and after backups.
							Use pre-backup hooks to dump databases or stop writes, and post-backup
							hooks to clean up or resume operations.
						</p>

						{hooks && hooks.length > 0 && (
							<div className="space-y-3 mb-4">
								{hooks.map((hook) => (
									<HookCard
										key={hook.id}
										hook={hook}
										scheduleId={scheduleId}
										onEdit={setEditingHook}
									/>
								))}
							</div>
						)}

						<div className="border-t border-gray-200 pt-4">
							<p className="text-sm font-medium text-gray-700 mb-2">
								Add a container hook:
							</p>
							<div className="grid grid-cols-2 gap-2">
								{HOOK_TYPES.map((type) => (
									<button
										key={type.value}
										type="button"
										onClick={() => setCreatingType(type.value)}
										className="text-left px-3 py-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
									>
										<div className="text-sm font-medium text-gray-900">
											{type.label}
										</div>
										<div className="text-xs text-gray-500">
											{type.description}
										</div>
									</button>
								))}
							</div>
						</div>

						<div className="flex justify-end mt-6">
							<button
								type="button"
								onClick={onClose}
								className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
							>
								Close
							</button>
						</div>
					</>
				)}
			</div>
		</div>
	);
}
