import { useState } from 'react';
import {
	downloadExport,
	useExportAgent,
	useExportSchedule,
	useImportConfig,
	useValidateImport,
} from '../../hooks/useConfigExport';
import { useLocale } from '../../hooks/useLocale';
import type {
	Agent,
	ConflictResolution,
	ExportFormat,
	Schedule,
	ValidationResult,
} from '../../lib/types';

type ExportType = 'agent' | 'schedule';

interface ExportImportModalProps {
	isOpen: boolean;
	onClose: () => void;
	type: ExportType;
	// For export mode
	item?: Agent | Schedule;
	// For import mode
	agents?: Agent[];
}

export function ExportImportModal({
	isOpen,
	onClose,
	type,
	item,
	agents,
}: ExportImportModalProps) {
	const [mode, setMode] = useState<'export' | 'import'>('export');
	const [format, setFormat] = useState<ExportFormat>('json');
	const [configText, setConfigText] = useState('');
	const [targetAgentId, setTargetAgentId] = useState('');
	const [conflictResolution, setConflictResolution] =
		useState<ConflictResolution>('skip');
	const [validationResult, setValidationResult] =
		useState<ValidationResult | null>(null);
	const [importStep, setImportStep] = useState<'input' | 'validate' | 'import'>(
		'input',
	);

	const exportAgent = useExportAgent();
	const exportSchedule = useExportSchedule();
	const validateImport = useValidateImport();
	const importConfig = useImportConfig();
	const { t } = useLocale();

	const handleExport = async () => {
		if (!item) return;

		try {
			let content: string;
			if (type === 'agent') {
				content = await exportAgent.mutateAsync({ id: item.id, format });
			} else {
				content = await exportSchedule.mutateAsync({ id: item.id, format });
			}

			const name =
				type === 'agent' ? (item as Agent).hostname : (item as Schedule).name;
			downloadExport(content, `${type}-${name}`, format);
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const handleValidate = async () => {
		if (!configText.trim()) return;

		try {
			const result = await validateImport.mutateAsync({
				config: configText,
				format,
			});
			setValidationResult(result);
			setImportStep('validate');
		} catch {
			// Error handled by mutation
		}
	};

	const handleImport = async () => {
		if (!configText.trim()) return;

		try {
			const result = await importConfig.mutateAsync({
				config: configText,
				format,
				target_agent_id: targetAgentId || undefined,
				conflict_resolution: conflictResolution,
			});

			if (result.success) {
				onClose();
				// Reset state
				setConfigText('');
				setValidationResult(null);
				setImportStep('input');
			}
		} catch {
			// Error handled by mutation
		}
	};

	const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file) return;

		const reader = new FileReader();
		reader.onload = (event) => {
			const content = event.target?.result as string;
			setConfigText(content);
			// Auto-detect format from file extension
			if (file.name.endsWith('.yaml') || file.name.endsWith('.yml')) {
				setFormat('yaml');
			} else {
				setFormat('json');
			}
		};
		reader.readAsText(file);
	};

	const handleClose = () => {
		onClose();
		// Reset state
		setMode('export');
		setConfigText('');
		setValidationResult(null);
		setImportStep('input');
		setTargetAgentId('');
	};

	if (!isOpen) return null;

	const isExporting = exportAgent.isPending || exportSchedule.isPending;
	const isValidating = validateImport.isPending;
	const isImporting = importConfig.isPending;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						{type === 'agent' ? 'Agent' : 'Schedule'} Configuration
					</h3>
					<button
						type="button"
						onClick={handleClose}
						className="text-gray-400 hover:text-gray-600"
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

				{/* Mode toggle */}
				<div className="flex gap-2 mb-6">
					<button
						type="button"
						onClick={() => setMode('export')}
						className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
							mode === 'export'
								? 'bg-indigo-600 text-white'
								: 'bg-gray-100 text-gray-700 hover:bg-gray-200'
						}`}
					>
						<span className="flex items-center justify-center gap-2">
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
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
								/>
							</svg>
							Export
						</span>
					</button>
					<button
						type="button"
						onClick={() => setMode('import')}
						className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
							mode === 'import'
								? 'bg-indigo-600 text-white'
								: 'bg-gray-100 text-gray-700 hover:bg-gray-200'
						}`}
					>
						<span className="flex items-center justify-center gap-2">
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
							Import
						</span>
					</button>
				</div>

				{mode === 'export' ? (
					/* Export Mode */
					<div className="space-y-4">
						{item ? (
							<>
								<div className="bg-gray-50 rounded-lg p-4">
									<p className="text-sm text-gray-600">
										Export configuration for:{' '}
										<span className="font-medium text-gray-900">
											{type === 'agent'
												? (item as Agent).hostname
												: (item as Schedule).name}
										</span>
									</p>
								</div>

								<div>
									<span className="block text-sm font-medium text-gray-700 mb-2">
										Export Format
									</span>
									<div className="flex gap-4">
										<label className="flex items-center gap-2">
											<input
												type="radio"
												name="format"
												value="json"
												checked={format === 'json'}
												onChange={() => setFormat('json')}
												className="text-indigo-600 focus:ring-indigo-500"
											/>
											<span className="text-sm">JSON</span>
										</label>
										<label className="flex items-center gap-2">
											<input
												type="radio"
												name="format"
												value="yaml"
												checked={format === 'yaml'}
												onChange={() => setFormat('yaml')}
												className="text-indigo-600 focus:ring-indigo-500"
											/>
											<span className="text-sm">YAML</span>
										</label>
									</div>
								</div>

								{(exportAgent.isError || exportSchedule.isError) && (
									<p className="text-sm text-red-600">
										Failed to export configuration. Please try again.
									</p>
								)}

								<div className="flex justify-end gap-3 pt-4">
									<button
										type="button"
										onClick={handleClose}
										className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
									>
										{t('common.cancel')}
									</button>
									<button
										type="button"
										onClick={handleExport}
										disabled={isExporting}
										className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
									>
										{isExporting ? 'Exporting...' : 'Export'}
									</button>
								</div>
							</>
						) : (
							<p className="text-gray-600 text-center py-8">
								Select an item to export from the list.
							</p>
						)}
					</div>
				) : (
					/* Import Mode */
					<div className="space-y-4">
						{importStep === 'input' && (
							<>
								<div>
									<span className="block text-sm font-medium text-gray-700 mb-2">
										Configuration File
									</span>
									<div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center hover:border-indigo-400 transition-colors">
										<input
											type="file"
											accept=".json,.yaml,.yml"
											onChange={handleFileUpload}
											className="hidden"
											id="config-file"
										/>
										<label htmlFor="config-file" className="cursor-pointer">
											<svg
												aria-hidden="true"
												className="w-12 h-12 mx-auto text-gray-400 mb-2"
												fill="none"
												stroke="currentColor"
												viewBox="0 0 24 24"
											>
												<path
													strokeLinecap="round"
													strokeLinejoin="round"
													strokeWidth={2}
													d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
												/>
											</svg>
											<p className="text-sm text-gray-600">
												Drop a file here or{' '}
												<span className="text-indigo-600">click to upload</span>
											</p>
											<p className="text-xs text-gray-400 mt-1">
												JSON or YAML format
											</p>
										</label>
									</div>
								</div>

								<div>
									<label
										htmlFor="config-paste"
										className="block text-sm font-medium text-gray-700 mb-2"
									>
										Or paste configuration:
									</label>
									<textarea
										id="config-paste"
										value={configText}
										onChange={(e) => setConfigText(e.target.value)}
										placeholder="Paste exported configuration here..."
										className="w-full h-40 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
									/>
								</div>

								{type === 'schedule' && agents && agents.length > 0 && (
									<div>
										<label
											htmlFor="target-agent"
											className="block text-sm font-medium text-gray-700 mb-2"
										>
											Target Agent (for schedule imports)
										</label>
										<select
											id="target-agent"
											value={targetAgentId}
											onChange={(e) => setTargetAgentId(e.target.value)}
											className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										>
											<option value="">Select an agent...</option>
											{agents.map((agent) => (
												<option key={agent.id} value={agent.id}>
													{agent.hostname}
												</option>
											))}
										</select>
									</div>
								)}

								<div className="flex justify-end gap-3 pt-4">
									<button
										type="button"
										onClick={handleClose}
										className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
									>
										{t('common.cancel')}
									</button>
									<button
										type="button"
										onClick={handleValidate}
										disabled={!configText.trim() || isValidating}
										className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
									>
										{isValidating ? 'Validating...' : 'Validate & Continue'}
									</button>
								</div>
							</>
						)}

						{importStep === 'validate' && validationResult && (
							<>
								<div
									className={`p-4 rounded-lg ${
										validationResult.valid
											? 'bg-green-50 border border-green-200'
											: 'bg-red-50 border border-red-200'
									}`}
								>
									<div className="flex items-center gap-2">
										{validationResult.valid ? (
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-green-600"
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
												className="w-5 h-5 text-red-600"
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
										)}
										<span
											className={`font-medium ${
												validationResult.valid
													? 'text-green-800'
													: 'text-red-800'
											}`}
										>
											{validationResult.valid
												? 'Configuration is valid'
												: 'Configuration has errors'}
										</span>
									</div>
								</div>

								{validationResult.errors &&
									validationResult.errors.length > 0 && (
										<div className="bg-red-50 rounded-lg p-4">
											<h4 className="font-medium text-red-800 mb-2">Errors:</h4>
											<ul className="text-sm text-red-700 space-y-1">
												{validationResult.errors.map((error) => (
													<li key={`${error.field}-${error.message}`}>
														<span className="font-medium">{error.field}:</span>{' '}
														{error.message}
													</li>
												))}
											</ul>
										</div>
									)}

								{validationResult.warnings &&
									validationResult.warnings.length > 0 && (
										<div className="bg-yellow-50 rounded-lg p-4">
											<h4 className="font-medium text-yellow-800 mb-2">
												Warnings:
											</h4>
											<ul className="text-sm text-yellow-700 space-y-1">
												{validationResult.warnings.map((warning) => (
													<li key={warning}>{warning}</li>
												))}
											</ul>
										</div>
									)}

								{validationResult.conflicts &&
									validationResult.conflicts.length > 0 && (
										<div className="bg-orange-50 rounded-lg p-4">
											<h4 className="font-medium text-orange-800 mb-2">
												Conflicts Found:
											</h4>
											<ul className="text-sm text-orange-700 space-y-1">
												{validationResult.conflicts.map((conflict) => (
													<li key={conflict.name}>
														<span className="font-medium">{conflict.name}</span>
														: {conflict.message}
													</li>
												))}
											</ul>

											<div className="mt-3">
												<label
													htmlFor="conflict-resolution"
													className="block text-sm font-medium text-orange-800 mb-2"
												>
													Conflict Resolution:
												</label>
												<select
													id="conflict-resolution"
													value={conflictResolution}
													onChange={(e) =>
														setConflictResolution(
															e.target.value as ConflictResolution,
														)
													}
													className="w-full px-3 py-2 border border-orange-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
												>
													<option value="skip">Skip conflicting items</option>
													<option value="replace">
														Replace existing items
													</option>
													<option value="rename">Rename imported items</option>
													<option value="fail">Fail if any conflicts</option>
												</select>
											</div>
										</div>
									)}

								{validationResult.suggestions &&
									validationResult.suggestions.length > 0 && (
										<div className="bg-blue-50 rounded-lg p-4">
											<h4 className="font-medium text-blue-800 mb-2">
												Suggestions:
											</h4>
											<ul className="text-sm text-blue-700 space-y-1">
												{validationResult.suggestions.map((suggestion) => (
													<li key={suggestion}>{suggestion}</li>
												))}
											</ul>
										</div>
									)}

								{importConfig.isError && (
									<p className="text-sm text-red-600">
										Failed to import configuration. Please try again.
									</p>
								)}

								<div className="flex justify-end gap-3 pt-4">
									<button
										type="button"
										onClick={() => {
											setImportStep('input');
											setValidationResult(null);
										}}
										className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
									>
										Back
									</button>
									<button
										type="button"
										onClick={handleImport}
										disabled={!validationResult.valid || isImporting}
										className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
									>
										{isImporting ? 'Importing...' : 'Import Configuration'}
									</button>
								</div>
							</>
						)}
					</div>
				)}
			</div>
		</div>
	);
}
