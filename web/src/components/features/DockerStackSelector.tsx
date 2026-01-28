import { useState } from 'react';
import type { DockerStack } from '../../lib/types';
import { HelpTooltip } from '../ui/HelpTooltip';

interface DockerStackSelectorProps {
	stacks: DockerStack[];
	selectedStackId: string | null;
	onChange: (stackId: string | null, config: DockerStackBackupConfig | null) => void;
	agentId: string;
	isLoading?: boolean;
	onDiscover?: () => void;
}

export interface DockerStackBackupConfig {
	stack_id: string;
	export_images: boolean;
	include_env_files: boolean;
	stop_for_backup: boolean;
}

export function DockerStackSelector({
	stacks,
	selectedStackId,
	onChange,
	agentId,
	isLoading,
	onDiscover,
}: DockerStackSelectorProps) {
	const [showConfig, setShowConfig] = useState(false);
	const [exportImages, setExportImages] = useState(false);
	const [includeEnvFiles, setIncludeEnvFiles] = useState(true);
	const [stopForBackup, setStopForBackup] = useState(false);

	const filteredStacks = stacks.filter((s) => s.agent_id === agentId);
	const selectedStack = filteredStacks.find((s) => s.id === selectedStackId);

	const handleStackSelect = (stackId: string) => {
		if (!stackId) {
			onChange(null, null);
			setShowConfig(false);
			return;
		}

		const stack = filteredStacks.find((s) => s.id === stackId);
		if (stack) {
			// Initialize config from stack defaults
			setExportImages(stack.export_images);
			setIncludeEnvFiles(stack.include_env_files);
			setStopForBackup(stack.stop_for_backup);
			setShowConfig(true);

			onChange(stackId, {
				stack_id: stackId,
				export_images: stack.export_images,
				include_env_files: stack.include_env_files,
				stop_for_backup: stack.stop_for_backup,
			});
		}
	};

	const handleConfigChange = (field: keyof Omit<DockerStackBackupConfig, 'stack_id'>, value: boolean) => {
		if (!selectedStackId) return;

		let newConfig: DockerStackBackupConfig;

		switch (field) {
			case 'export_images':
				setExportImages(value);
				newConfig = {
					stack_id: selectedStackId,
					export_images: value,
					include_env_files: includeEnvFiles,
					stop_for_backup: stopForBackup,
				};
				break;
			case 'include_env_files':
				setIncludeEnvFiles(value);
				newConfig = {
					stack_id: selectedStackId,
					export_images: exportImages,
					include_env_files: value,
					stop_for_backup: stopForBackup,
				};
				break;
			case 'stop_for_backup':
				setStopForBackup(value);
				newConfig = {
					stack_id: selectedStackId,
					export_images: exportImages,
					include_env_files: includeEnvFiles,
					stop_for_backup: value,
				};
				break;
		}

		onChange(selectedStackId, newConfig);
	};

	if (!agentId) {
		return (
			<div className="text-sm text-gray-500 dark:text-gray-400">
				Select an agent first to see available Docker stacks.
			</div>
		);
	}

	return (
		<div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-4">
			<div className="flex items-center justify-between">
				<span className="flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300">
					Docker Stack Backup (optional)
					<HelpTooltip
						content="Back up a Docker Compose stack including volumes, bind mounts, compose files, and optionally Docker images."
						title="Docker Stack Backup"
					/>
				</span>
				{onDiscover && (
					<button
						type="button"
						onClick={onDiscover}
						className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 flex items-center gap-1"
					>
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
								d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
							/>
						</svg>
						Discover Stacks
					</button>
				)}
			</div>

			{isLoading ? (
				<div className="flex items-center gap-2 text-sm text-gray-500">
					<svg
						className="w-4 h-4 animate-spin"
						fill="none"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<circle
							className="opacity-25"
							cx="12"
							cy="12"
							r="10"
							stroke="currentColor"
							strokeWidth="4"
						/>
						<path
							className="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						/>
					</svg>
					Loading stacks...
				</div>
			) : filteredStacks.length === 0 ? (
				<div className="text-sm text-gray-500 dark:text-gray-400">
					No Docker stacks registered for this agent.
					{onDiscover && (
						<button
							type="button"
							onClick={onDiscover}
							className="ml-1 text-indigo-600 hover:text-indigo-800 dark:text-indigo-400"
						>
							Discover stacks
						</button>
					)}
				</div>
			) : (
				<>
					<select
						value={selectedStackId || ''}
						onChange={(e) => handleStackSelect(e.target.value)}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="">No Docker stack (file-based backup only)</option>
						{filteredStacks.map((stack) => (
							<option key={stack.id} value={stack.id}>
								{stack.name} - {stack.compose_path}
								{stack.is_running ? ' (running)' : ' (stopped)'}
							</option>
						))}
					</select>

					{selectedStack && showConfig && (
						<div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
							<div className="flex items-center gap-2 text-sm">
								<svg
									className="w-5 h-5 text-blue-500"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
									/>
								</svg>
								<span className="font-medium text-gray-900 dark:text-white">
									{selectedStack.name}
								</span>
								<span className={`px-2 py-0.5 rounded-full text-xs ${
									selectedStack.is_running
										? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
										: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
								}`}>
									{selectedStack.is_running ? 'Running' : 'Stopped'}
								</span>
							</div>

							<div className="text-xs text-gray-500 dark:text-gray-400 font-mono">
								{selectedStack.compose_path}
							</div>

							<div className="grid grid-cols-1 gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
								<label className="flex items-center gap-2 text-sm">
									<input
										type="checkbox"
										checked={includeEnvFiles}
										onChange={(e) => handleConfigChange('include_env_files', e.target.checked)}
										className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="text-gray-700 dark:text-gray-300">
										Include .env files
									</span>
									<HelpTooltip
										content="Backup .env files alongside the docker-compose.yml"
										title="Environment Files"
									/>
								</label>

								<label className="flex items-center gap-2 text-sm">
									<input
										type="checkbox"
										checked={stopForBackup}
										onChange={(e) => handleConfigChange('stop_for_backup', e.target.checked)}
										className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="text-gray-700 dark:text-gray-300">
										Stop containers during backup
									</span>
									<HelpTooltip
										content="Stop containers to ensure data consistency during backup. Containers will restart after backup completes."
										title="Stop for Backup"
									/>
								</label>

								<label className="flex items-center gap-2 text-sm">
									<input
										type="checkbox"
										checked={exportImages}
										onChange={(e) => handleConfigChange('export_images', e.target.checked)}
										className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
									/>
									<span className="text-gray-700 dark:text-gray-300">
										Export Docker images
									</span>
									<span className="text-xs text-amber-600 dark:text-amber-400">
										(large)
									</span>
									<HelpTooltip
										content="Export Docker images as tar files. This can significantly increase backup size but allows full offline restore."
										title="Export Images"
									/>
								</label>
							</div>

							{exportImages && (
								<div className="flex items-start gap-2 p-2 bg-amber-50 dark:bg-amber-900/20 rounded text-xs text-amber-700 dark:text-amber-300">
									<svg
										className="w-4 h-4 flex-shrink-0 mt-0.5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
										aria-hidden="true"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									<span>
										Exporting Docker images can significantly increase backup size.
										Images will be saved as tar archives alongside volume data.
									</span>
								</div>
							)}
						</div>
					)}
				</>
			)}
		</div>
	);
}
