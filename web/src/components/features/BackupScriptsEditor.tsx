import { useState } from 'react';
import {
	useBackupScripts,
	useCreateBackupScript,
	useDeleteBackupScript,
	useUpdateBackupScript,
} from '../../hooks/useBackupScripts';
import type { BackupScript, BackupScriptType } from '../../lib/types';

interface BackupScriptsEditorProps {
	scheduleId: string;
	onClose: () => void;
}

const SCRIPT_TYPES: {
	value: BackupScriptType;
	label: string;
	description: string;
}[] = [
	{
		value: 'pre_backup',
		label: 'Pre-Backup',
		description: 'Runs before the backup starts',
	},
	{
		value: 'post_success',
		label: 'Post-Backup (Success)',
		description: 'Runs after a successful backup',
	},
	{
		value: 'post_failure',
		label: 'Post-Backup (Failure)',
		description: 'Runs after a failed backup',
	},
	{
		value: 'post_always',
		label: 'Post-Backup (Always)',
		description: 'Runs after backup regardless of outcome',
	},
];

interface ScriptFormProps {
	script?: BackupScript;
	scriptType: BackupScriptType;
	scheduleId: string;
	onSave: () => void;
	onCancel: () => void;
}

function ScriptForm({
	script,
	scriptType,
	scheduleId,
	onSave,
	onCancel,
}: ScriptFormProps) {
	const [scriptContent, setScriptContent] = useState(script?.script ?? '');
	const [timeoutSeconds, setTimeoutSeconds] = useState(
		script?.timeout_seconds ?? 300,
	);
	const [failOnError, setFailOnError] = useState(
		script?.fail_on_error ?? false,
	);
	const [enabled, setEnabled] = useState(script?.enabled ?? true);

	const createScript = useCreateBackupScript();
	const updateScript = useUpdateBackupScript();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();

		try {
			if (script) {
				await updateScript.mutateAsync({
					scheduleId,
					id: script.id,
					data: {
						script: scriptContent,
						timeout_seconds: timeoutSeconds,
						fail_on_error: failOnError,
						enabled,
					},
				});
			} else {
				await createScript.mutateAsync({
					scheduleId,
					data: {
						type: scriptType,
						script: scriptContent,
						timeout_seconds: timeoutSeconds,
						fail_on_error: failOnError,
						enabled,
					},
				});
			}
			onSave();
		} catch {
			// Error handled by mutation
		}
	};

	const isPending = createScript.isPending || updateScript.isPending;
	const typeInfo = SCRIPT_TYPES.find((t) => t.value === scriptType);

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<h4 className="text-sm font-medium text-gray-900">{typeInfo?.label}</h4>
				<p className="text-xs text-gray-500">{typeInfo?.description}</p>
			</div>

			<div>
				<label
					htmlFor="script-content"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Script
				</label>
				<textarea
					id="script-content"
					value={scriptContent}
					onChange={(e) => setScriptContent(e.target.value)}
					placeholder="#!/bin/bash&#10;# Your script here"
					rows={8}
					className="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					required
				/>
				<p className="text-xs text-gray-500 mt-1">
					Shell script to execute. Runs with /bin/sh -c.
				</p>
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

			{scriptType === 'pre_backup' && (
				<div>
					<label className="flex items-center gap-2 cursor-pointer">
						<input
							type="checkbox"
							checked={failOnError}
							onChange={(e) => setFailOnError(e.target.checked)}
							className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
						/>
						<span className="text-sm text-gray-700">
							Fail backup if script fails
						</span>
					</label>
					<p className="text-xs text-gray-500 mt-1 ml-6">
						If enabled and this script fails, the backup will be aborted.
					</p>
				</div>
			)}

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
					disabled={isPending || !scriptContent.trim()}
					className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isPending ? 'Saving...' : script ? 'Update Script' : 'Create Script'}
				</button>
			</div>
		</form>
	);
}

interface ScriptCardProps {
	script: BackupScript;
	scheduleId: string;
	onEdit: (script: BackupScript) => void;
}

function ScriptCard({ script, scheduleId, onEdit }: ScriptCardProps) {
	const deleteScript = useDeleteBackupScript();
	const typeInfo = SCRIPT_TYPES.find((t) => t.value === script.type);

	const handleDelete = async () => {
		if (confirm('Are you sure you want to delete this script?')) {
			await deleteScript.mutateAsync({ scheduleId, id: script.id });
		}
	};

	return (
		<div
			className={`border rounded-lg p-4 ${script.enabled ? 'bg-white' : 'bg-gray-50 border-gray-200'}`}
		>
			<div className="flex items-start justify-between mb-2">
				<div>
					<div className="flex items-center gap-2">
						<h4 className="text-sm font-medium text-gray-900">
							{typeInfo?.label}
						</h4>
						{!script.enabled && (
							<span className="text-xs px-2 py-0.5 bg-gray-200 text-gray-600 rounded">
								Disabled
							</span>
						)}
					</div>
					<p className="text-xs text-gray-500">{typeInfo?.description}</p>
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => onEdit(script)}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						Edit
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={deleteScript.isPending}
						className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
					>
						Delete
					</button>
				</div>
			</div>
			<pre className="text-xs bg-gray-100 p-2 rounded overflow-x-auto max-h-32 whitespace-pre-wrap">
				{script.script.substring(0, 500)}
				{script.script.length > 500 && '...'}
			</pre>
			<div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
				<span>Timeout: {script.timeout_seconds}s</span>
				{script.type === 'pre_backup' && script.fail_on_error && (
					<span className="text-amber-600">Fails backup on error</span>
				)}
			</div>
		</div>
	);
}

export function BackupScriptsEditor({
	scheduleId,
	onClose,
}: BackupScriptsEditorProps) {
	const { data: scripts, isLoading, isError } = useBackupScripts(scheduleId);
	const [editingScript, setEditingScript] = useState<BackupScript | null>(null);
	const [creatingType, setCreatingType] = useState<BackupScriptType | null>(
		null,
	);

	const existingTypes = new Set(scripts?.map((s) => s.type) ?? []);
	const availableTypes = SCRIPT_TYPES.filter(
		(t) => !existingTypes.has(t.value),
	);

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
					<p className="text-red-600">Failed to load scripts</p>
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
						Backup Scripts
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

				{editingScript || creatingType ? (
					<ScriptForm
						script={editingScript ?? undefined}
						scriptType={editingScript?.type ?? creatingType ?? 'pre_backup'}
						scheduleId={scheduleId}
						onSave={() => {
							setEditingScript(null);
							setCreatingType(null);
						}}
						onCancel={() => {
							setEditingScript(null);
							setCreatingType(null);
						}}
					/>
				) : (
					<>
						<p className="text-sm text-gray-600 mb-4">
							Configure scripts to run before and after backups. Scripts are
							executed on the backup server, not the agent.
						</p>

						{scripts && scripts.length > 0 && (
							<div className="space-y-3 mb-4">
								{scripts.map((script) => (
									<ScriptCard
										key={script.id}
										script={script}
										scheduleId={scheduleId}
										onEdit={setEditingScript}
									/>
								))}
							</div>
						)}

						{availableTypes.length > 0 && (
							<div className="border-t border-gray-200 pt-4">
								<p className="text-sm font-medium text-gray-700 mb-2">
									Add a script:
								</p>
								<div className="grid grid-cols-2 gap-2">
									{availableTypes.map((type) => (
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
						)}

						{scripts?.length === 0 && availableTypes.length === 0 && (
							<p className="text-gray-500 text-center py-4">
								All script types have been configured.
							</p>
						)}

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
