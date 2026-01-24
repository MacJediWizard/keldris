import { useRef, useState } from 'react';
import { useAgentGroups } from '../../hooks/useAgentGroups';
import {
	useAgentImport,
	useAgentImportPreview,
	useAgentImportTemplateDownload,
	useAgentImportTokensExport,
	useAgentRegistrationScript,
} from '../../hooks/useAgentImport';
import type {
	AgentImportJobResult,
	AgentImportPreviewEntry,
	AgentImportPreviewResponse,
	AgentImportResponse,
} from '../../lib/types';

type WizardStep = 'upload' | 'mapping' | 'preview' | 'importing' | 'results';

interface ImportAgentsWizardProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (importedCount: number, failedCount: number) => void;
}

export function ImportAgentsWizard({
	isOpen,
	onClose,
	onSuccess,
}: ImportAgentsWizardProps) {
	const [step, setStep] = useState<WizardStep>('upload');
	const [error, setError] = useState<string | null>(null);
	const [file, setFile] = useState<File | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);

	// Column mapping state
	const [hasHeader, setHasHeader] = useState(true);
	const [hostnameCol, setHostnameCol] = useState(0);
	const [groupCol, setGroupCol] = useState(1);
	const [tagsCol, setTagsCol] = useState(2);
	const [configCol, setConfigCol] = useState(3);

	// Import options
	const [createMissingGroups, setCreateMissingGroups] = useState(true);
	const [tokenExpiryHours, setTokenExpiryHours] = useState(24);

	// Preview and results state
	const [preview, setPreview] = useState<AgentImportPreviewResponse | null>(
		null
	);
	const [importResult, setImportResult] = useState<AgentImportResponse | null>(
		null
	);
	const [selectedScript, setSelectedScript] = useState<{
		hostname: string;
		code: string;
	} | null>(null);
	const [scriptContent, setScriptContent] = useState<string | null>(null);

	// Hooks
	const previewMutation = useAgentImportPreview();
	const importMutation = useAgentImport();
	const downloadTemplate = useAgentImportTemplateDownload();
	const exportTokens = useAgentImportTokensExport();
	const generateScript = useAgentRegistrationScript();
	const { data: existingGroups } = useAgentGroups();

	const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
		const selectedFile = e.target.files?.[0];
		if (selectedFile) {
			if (!selectedFile.name.endsWith('.csv')) {
				setError('Please select a CSV file');
				return;
			}
			setFile(selectedFile);
			setError(null);
		}
	};

	const handleDragOver = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
	};

	const handleDrop = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		const droppedFile = e.dataTransfer.files[0];
		if (droppedFile) {
			if (!droppedFile.name.endsWith('.csv')) {
				setError('Please select a CSV file');
				return;
			}
			setFile(droppedFile);
			setError(null);
		}
	};

	const handlePreview = async () => {
		if (!file) return;
		setError(null);
		try {
			const result = await previewMutation.mutateAsync({
				file,
				options: {
					hasHeader,
					hostnameCol,
					groupCol,
					tagsCol,
					configCol,
				},
			});
			setPreview(result);
			setStep('preview');
		} catch (err) {
			setError(err instanceof Error ? err.message : 'Failed to preview CSV');
		}
	};

	const handleImport = async () => {
		if (!file) return;
		setError(null);
		setStep('importing');
		try {
			const result = await importMutation.mutateAsync({
				file,
				options: {
					hasHeader,
					hostnameCol,
					groupCol,
					tagsCol,
					configCol,
					createMissingGroups,
					tokenExpiryHours,
				},
			});
			setImportResult(result);
			setStep('results');
		} catch (err) {
			setError(err instanceof Error ? err.message : 'Failed to import agents');
			setStep('preview');
		}
	};

	const handleDownloadTemplate = async () => {
		try {
			await downloadTemplate.mutateAsync();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to download template'
			);
		}
	};

	const handleExportTokens = async () => {
		if (!importResult) return;
		try {
			await exportTokens.mutateAsync(importResult.results);
		} catch (err) {
			setError(err instanceof Error ? err.message : 'Failed to export tokens');
		}
	};

	const handleGenerateScript = async (hostname: string, code: string) => {
		setSelectedScript({ hostname, code });
		try {
			const result = await generateScript.mutateAsync({
				hostname,
				registrationCode: code,
			});
			setScriptContent(result.script);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to generate script'
			);
		}
	};

	const handleCopyScript = () => {
		if (scriptContent) {
			navigator.clipboard.writeText(scriptContent);
		}
	};

	const resetForm = () => {
		setStep('upload');
		setError(null);
		setFile(null);
		setHasHeader(true);
		setHostnameCol(0);
		setGroupCol(1);
		setTagsCol(2);
		setConfigCol(3);
		setCreateMissingGroups(true);
		setTokenExpiryHours(24);
		setPreview(null);
		setImportResult(null);
		setSelectedScript(null);
		setScriptContent(null);
	};

	const handleClose = () => {
		if (importResult) {
			onSuccess(importResult.imported_count, importResult.failed_count);
		}
		resetForm();
		onClose();
	};

	if (!isOpen) return null;

	const getNewGroupNames = (): string[] => {
		if (!preview) return [];
		const existingNames = new Set(
			existingGroups?.map((g) => g.name.toLowerCase()) || []
		);
		return preview.detected_groups.filter(
			(g) => !existingNames.has(g.toLowerCase())
		);
	};

	const renderUploadStep = () => (
		<div className="space-y-4">
			<div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
				<div className="flex gap-3">
					<svg
						aria-hidden="true"
						className="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<div>
						<p className="text-sm text-blue-800 font-medium">
							Bulk Import Agents
						</p>
						<p className="text-sm text-blue-700 mt-1">
							Upload a CSV file to register multiple agents at once. Each agent
							will receive a unique registration code.
						</p>
					</div>
				</div>
			</div>

			<div
				onDragOver={handleDragOver}
				onDrop={handleDrop}
				className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
					file ? 'border-green-300 bg-green-50' : 'border-gray-300 hover:border-indigo-300'
				}`}
			>
				{file ? (
					<div>
						<svg
							aria-hidden="true"
							className="w-12 h-12 text-green-500 mx-auto mb-2"
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
						<p className="text-sm font-medium text-green-700">{file.name}</p>
						<p className="text-xs text-green-600 mt-1">
							{(file.size / 1024).toFixed(1)} KB
						</p>
						<button
							type="button"
							onClick={() => setFile(null)}
							className="mt-2 text-sm text-gray-500 hover:text-gray-700"
						>
							Remove
						</button>
					</div>
				) : (
					<div>
						<svg
							aria-hidden="true"
							className="w-12 h-12 text-gray-400 mx-auto mb-2"
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
							Drag and drop a CSV file, or{' '}
							<button
								type="button"
								onClick={() => fileInputRef.current?.click()}
								className="text-indigo-600 hover:text-indigo-700 font-medium"
							>
								browse
							</button>
						</p>
						<input
							ref={fileInputRef}
							type="file"
							accept=".csv"
							onChange={handleFileSelect}
							className="hidden"
						/>
					</div>
				)}
			</div>

			<div className="flex items-center justify-between pt-2">
				<button
					type="button"
					onClick={handleDownloadTemplate}
					disabled={downloadTemplate.isPending}
					className="text-sm text-indigo-600 hover:text-indigo-700 flex items-center gap-1"
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
							d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
						/>
					</svg>
					{downloadTemplate.isPending
						? 'Downloading...'
						: 'Download CSV Template'}
				</button>
			</div>
		</div>
	);

	const renderMappingStep = () => (
		<div className="space-y-4">
			<div className="bg-gray-50 rounded-lg p-4">
				<h4 className="text-sm font-medium text-gray-700 mb-3">
					Column Mapping
				</h4>
				<p className="text-xs text-gray-500 mb-4">
					Configure which CSV columns map to each field
				</p>

				<div className="space-y-3">
					<div className="flex items-center gap-2">
						<input
							type="checkbox"
							id="has-header"
							checked={hasHeader}
							onChange={(e) => setHasHeader(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						<label htmlFor="has-header" className="text-sm text-gray-700">
							First row contains headers
						</label>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div>
							<label
								htmlFor="hostname-col"
								className="block text-xs font-medium text-gray-600 mb-1"
							>
								Hostname Column *
							</label>
							<input
								type="number"
								id="hostname-col"
								min="0"
								value={hostnameCol}
								onChange={(e) => setHostnameCol(Number(e.target.value))}
								className="w-full px-3 py-1.5 border border-gray-300 rounded text-sm"
							/>
						</div>
						<div>
							<label
								htmlFor="group-col"
								className="block text-xs font-medium text-gray-600 mb-1"
							>
								Group Column
							</label>
							<input
								type="number"
								id="group-col"
								min="0"
								value={groupCol}
								onChange={(e) => setGroupCol(Number(e.target.value))}
								className="w-full px-3 py-1.5 border border-gray-300 rounded text-sm"
							/>
						</div>
						<div>
							<label
								htmlFor="tags-col"
								className="block text-xs font-medium text-gray-600 mb-1"
							>
								Tags Column
							</label>
							<input
								type="number"
								id="tags-col"
								min="0"
								value={tagsCol}
								onChange={(e) => setTagsCol(Number(e.target.value))}
								className="w-full px-3 py-1.5 border border-gray-300 rounded text-sm"
							/>
						</div>
						<div>
							<label
								htmlFor="config-col"
								className="block text-xs font-medium text-gray-600 mb-1"
							>
								Config Column
							</label>
							<input
								type="number"
								id="config-col"
								min="0"
								value={configCol}
								onChange={(e) => setConfigCol(Number(e.target.value))}
								className="w-full px-3 py-1.5 border border-gray-300 rounded text-sm"
							/>
						</div>
					</div>
				</div>
			</div>

			<div className="bg-gray-50 rounded-lg p-4">
				<h4 className="text-sm font-medium text-gray-700 mb-3">
					Import Options
				</h4>

				<div className="space-y-3">
					<div className="flex items-center gap-2">
						<input
							type="checkbox"
							id="create-groups"
							checked={createMissingGroups}
							onChange={(e) => setCreateMissingGroups(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						<label htmlFor="create-groups" className="text-sm text-gray-700">
							Create missing groups automatically
						</label>
					</div>

					<div>
						<label
							htmlFor="token-expiry"
							className="block text-xs font-medium text-gray-600 mb-1"
						>
							Registration Code Expiry (hours)
						</label>
						<input
							type="number"
							id="token-expiry"
							min="1"
							max="168"
							value={tokenExpiryHours}
							onChange={(e) => setTokenExpiryHours(Number(e.target.value))}
							className="w-24 px-3 py-1.5 border border-gray-300 rounded text-sm"
						/>
						<span className="text-xs text-gray-500 ml-2">
							(max 168 hours / 7 days)
						</span>
					</div>
				</div>
			</div>
		</div>
	);

	const renderPreviewStep = () => {
		if (!preview) return null;

		const newGroups = getNewGroupNames();

		return (
			<div className="space-y-4">
				<div className="grid grid-cols-3 gap-4">
					<div className="bg-gray-50 rounded-lg p-3 text-center">
						<p className="text-2xl font-bold text-gray-900">
							{preview.total_rows}
						</p>
						<p className="text-xs text-gray-500">Total Rows</p>
					</div>
					<div className="bg-green-50 rounded-lg p-3 text-center">
						<p className="text-2xl font-bold text-green-600">
							{preview.valid_rows}
						</p>
						<p className="text-xs text-gray-500">Valid</p>
					</div>
					<div className="bg-red-50 rounded-lg p-3 text-center">
						<p className="text-2xl font-bold text-red-600">
							{preview.invalid_rows}
						</p>
						<p className="text-xs text-gray-500">Invalid</p>
					</div>
				</div>

				{preview.detected_groups.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Detected Groups
						</h4>
						<div className="flex flex-wrap gap-2">
							{preview.detected_groups.map((group) => (
								<span
									key={group}
									className={`px-2 py-1 rounded text-sm ${
										newGroups.includes(group)
											? 'bg-amber-100 text-amber-700'
											: 'bg-indigo-100 text-indigo-700'
									}`}
								>
									{group}
									{newGroups.includes(group) && (
										<span className="text-xs ml-1">(new)</span>
									)}
								</span>
							))}
						</div>
					</div>
				)}

				{preview.detected_tags.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Detected Tags
						</h4>
						<div className="flex flex-wrap gap-2">
							{preview.detected_tags.map((tag) => (
								<span
									key={tag}
									className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm"
								>
									{tag}
								</span>
							))}
						</div>
					</div>
				)}

				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Preview (first 10 entries)
					</h4>
					<div className="border border-gray-200 rounded-lg divide-y divide-gray-200 max-h-48 overflow-y-auto">
						{preview.entries.slice(0, 10).map((entry: AgentImportPreviewEntry) => (
							<div
								key={entry.row_number}
								className={`p-3 text-sm ${
									entry.is_valid ? '' : 'bg-red-50'
								}`}
							>
								<div className="flex items-center justify-between">
									<span className="font-medium text-gray-900">
										{entry.hostname || '(empty)'}
									</span>
									<span className="text-xs text-gray-500">
										Row {entry.row_number}
									</span>
								</div>
								{entry.group_name && (
									<span className="text-xs text-gray-500">
										Group: {entry.group_name}
									</span>
								)}
								{entry.tags && entry.tags.length > 0 && (
									<div className="flex gap-1 mt-1">
										{entry.tags.map((tag) => (
											<span
												key={tag}
												className="px-1 py-0.5 bg-gray-100 text-gray-600 rounded text-xs"
											>
												{tag}
											</span>
										))}
									</div>
								)}
								{!entry.is_valid && entry.errors && (
									<div className="mt-1 text-xs text-red-600">
										{entry.errors.join('; ')}
									</div>
								)}
							</div>
						))}
					</div>
				</div>

				{preview.invalid_rows > 0 && (
					<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
						<div className="flex gap-3">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-amber-500 flex-shrink-0"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
							<div>
								<p className="text-sm text-amber-800 font-medium">
									{preview.invalid_rows} invalid{' '}
									{preview.invalid_rows === 1 ? 'entry' : 'entries'} will be
									skipped
								</p>
								<p className="text-sm text-amber-700 mt-1">
									Only valid entries will be imported.
								</p>
							</div>
						</div>
					</div>
				)}
			</div>
		);
	};

	const renderImportingStep = () => (
		<div className="py-8 text-center">
			<div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4" />
			<p className="text-gray-600">Importing agents...</p>
			<p className="text-sm text-gray-500 mt-2">This may take a moment</p>
		</div>
	);

	const renderResultsStep = () => {
		if (!importResult) return null;

		const successfulResults = importResult.results.filter(
			(r: AgentImportJobResult) => r.success
		);
		const failedResults = importResult.results.filter(
			(r: AgentImportJobResult) => !r.success
		);

		return (
			<div className="space-y-4">
				<div
					className={`rounded-lg p-4 ${
						importResult.failed_count === 0
							? 'bg-green-50 border border-green-200'
							: 'bg-amber-50 border border-amber-200'
					}`}
				>
					<div className="flex gap-3">
						<svg
							aria-hidden="true"
							className={`w-5 h-5 flex-shrink-0 ${
								importResult.failed_count === 0
									? 'text-green-500'
									: 'text-amber-500'
							}`}
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d={
									importResult.failed_count === 0
										? 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
										: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z'
								}
							/>
						</svg>
						<div>
							<p
								className={`text-sm font-medium ${
									importResult.failed_count === 0
										? 'text-green-800'
										: 'text-amber-800'
								}`}
							>
								Import Complete
							</p>
							<p
								className={`text-sm mt-1 ${
									importResult.failed_count === 0
										? 'text-green-700'
										: 'text-amber-700'
								}`}
							>
								{importResult.imported_count} agent
								{importResult.imported_count !== 1 ? 's' : ''} imported
								{importResult.failed_count > 0 &&
									`, ${importResult.failed_count} failed`}
							</p>
						</div>
					</div>
				</div>

				{importResult.groups_created && importResult.groups_created.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Groups Created
						</h4>
						<div className="flex flex-wrap gap-2">
							{importResult.groups_created.map((group) => (
								<span
									key={group}
									className="px-2 py-1 bg-green-100 text-green-700 rounded text-sm"
								>
									{group}
								</span>
							))}
						</div>
					</div>
				)}

				{successfulResults.length > 0 && (
					<div>
						<div className="flex items-center justify-between mb-2">
							<h4 className="text-sm font-medium text-gray-700">
								Successful Imports ({successfulResults.length})
							</h4>
							<button
								type="button"
								onClick={handleExportTokens}
								disabled={exportTokens.isPending}
								className="text-sm text-indigo-600 hover:text-indigo-700 flex items-center gap-1"
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
										d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
									/>
								</svg>
								{exportTokens.isPending ? 'Exporting...' : 'Export Tokens CSV'}
							</button>
						</div>
						<div className="border border-gray-200 rounded-lg divide-y divide-gray-200 max-h-48 overflow-y-auto">
							{successfulResults.map((result: AgentImportJobResult) => (
								<div key={result.row_number} className="p-3 text-sm">
									<div className="flex items-center justify-between">
										<span className="font-medium text-gray-900">
											{result.hostname}
										</span>
										<button
											type="button"
											onClick={() =>
												handleGenerateScript(
													result.hostname,
													result.registration_code || ''
												)
											}
											className="text-xs text-indigo-600 hover:text-indigo-700"
										>
											Get Script
										</button>
									</div>
									<div className="flex items-center gap-2 mt-1">
										<span className="font-mono text-xs bg-gray-100 px-2 py-0.5 rounded">
											{result.registration_code}
										</span>
										{result.group_name && (
											<span className="text-xs text-gray-500">
												{result.group_name}
											</span>
										)}
									</div>
								</div>
							))}
						</div>
					</div>
				)}

				{failedResults.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Failed Imports ({failedResults.length})
						</h4>
						<div className="border border-red-200 rounded-lg divide-y divide-red-100 max-h-32 overflow-y-auto bg-red-50">
							{failedResults.map((result: AgentImportJobResult) => (
								<div key={result.row_number} className="p-3 text-sm">
									<div className="flex items-center justify-between">
										<span className="font-medium text-gray-900">
											{result.hostname || `Row ${result.row_number}`}
										</span>
									</div>
									<p className="text-xs text-red-600 mt-1">
										{result.error_message}
									</p>
								</div>
							))}
						</div>
					</div>
				)}

				{selectedScript && scriptContent && (
					<div className="border border-gray-200 rounded-lg p-4">
						<div className="flex items-center justify-between mb-2">
							<h4 className="text-sm font-medium text-gray-700">
								Registration Script for {selectedScript.hostname}
							</h4>
							<button
								type="button"
								onClick={handleCopyScript}
								className="text-sm text-indigo-600 hover:text-indigo-700"
							>
								Copy
							</button>
						</div>
						<pre className="bg-gray-900 text-gray-100 rounded p-3 text-xs overflow-x-auto max-h-48">
							{scriptContent}
						</pre>
					</div>
				)}
			</div>
		);
	};

	const renderStepContent = () => {
		switch (step) {
			case 'upload':
				return renderUploadStep();
			case 'mapping':
				return renderMappingStep();
			case 'preview':
				return renderPreviewStep();
			case 'importing':
				return renderImportingStep();
			case 'results':
				return renderResultsStep();
			default:
				return null;
		}
	};

	const getStepTitle = () => {
		switch (step) {
			case 'upload':
				return 'Upload CSV File';
			case 'mapping':
				return 'Column Mapping';
			case 'preview':
				return 'Preview Import';
			case 'importing':
				return 'Importing...';
			case 'results':
				return 'Import Results';
			default:
				return 'Import Agents';
		}
	};

	const isNextDisabled = () => {
		if (step === 'upload') {
			return !file;
		}
		if (step === 'mapping') {
			return previewMutation.isPending;
		}
		if (step === 'preview') {
			return !preview || preview.valid_rows === 0;
		}
		return false;
	};

	const stepOrder: WizardStep[] = ['upload', 'mapping', 'preview', 'importing', 'results'];

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						{getStepTitle()}
					</h3>
					{step !== 'importing' && (
						<button
							type="button"
							onClick={handleClose}
							className="text-gray-400 hover:text-gray-600"
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
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					)}
				</div>

				{/* Step indicator */}
				{!['importing', 'results'].includes(step) && (
					<div className="flex items-center gap-2 mb-6">
						{['upload', 'mapping', 'preview'].map((s, i) => (
							<div key={s} className="flex items-center">
								<div
									className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
										step === s
											? 'bg-indigo-600 text-white'
											: stepOrder.indexOf(step as WizardStep) > i
												? 'bg-indigo-100 text-indigo-600'
												: 'bg-gray-100 text-gray-400'
									}`}
								>
									{i + 1}
								</div>
								{i < 2 && (
									<div
										className={`w-12 h-0.5 mx-1 ${
											stepOrder.indexOf(step as WizardStep) > i
												? 'bg-indigo-200'
												: 'bg-gray-200'
										}`}
									/>
								)}
							</div>
						))}
					</div>
				)}

				{error && (
					<div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
						<p className="text-sm text-red-700">{error}</p>
					</div>
				)}

				{renderStepContent()}

				{step !== 'importing' && step !== 'results' && (
					<div className="flex justify-between mt-6">
						<button
							type="button"
							onClick={() => {
								if (step === 'mapping') setStep('upload');
								else if (step === 'preview') setStep('mapping');
								else handleClose();
							}}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							{step === 'upload' ? 'Cancel' : 'Back'}
						</button>
						<button
							type="button"
							onClick={() => {
								if (step === 'upload') setStep('mapping');
								else if (step === 'mapping') handlePreview();
								else if (step === 'preview') handleImport();
							}}
							disabled={isNextDisabled()}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
						>
							{step === 'mapping' && previewMutation.isPending
								? 'Previewing...'
								: step === 'upload'
									? 'Continue'
									: step === 'mapping'
										? 'Preview'
										: `Import ${preview?.valid_rows || 0} Agents`}
						</button>
					</div>
				)}

				{step === 'results' && (
					<div className="flex justify-end mt-6">
						<button
							type="button"
							onClick={handleClose}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							Done
						</button>
					</div>
				)}
			</div>
		</div>
	);
}
