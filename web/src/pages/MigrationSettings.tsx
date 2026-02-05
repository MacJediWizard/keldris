import { useRef, useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	downloadMigrationExport,
	readFileAsText,
	useGenerateExportKey,
	useMigrationExport,
	useMigrationImport,
	useValidateMigrationImport,
} from '../hooks/useMigration';
import type {
	MigrationImportResult,
	MigrationValidationResult,
} from '../lib/types';

type MigrationTab = 'export' | 'import';

export function MigrationSettings() {
	const { data: user, isLoading: userLoading } = useMe();
	const [activeTab, setActiveTab] = useState<MigrationTab>('export');

	// Export state
	const [includeSecrets, setIncludeSecrets] = useState(false);
	const [includeSystemConfig, setIncludeSystemConfig] = useState(true);
	const [useEncryption, setUseEncryption] = useState(true);
	const [encryptionKey, setEncryptionKey] = useState('');
	const [exportDescription, setExportDescription] = useState('');
	const [exportError, setExportError] = useState<string | null>(null);
	const [exportSuccess, setExportSuccess] = useState(false);

	// Import state
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [importFile, setImportFile] = useState<File | null>(null);
	const [importFileContent, setImportFileContent] = useState<string | null>(
		null,
	);
	const [decryptionKey, setDecryptionKey] = useState('');
	const [conflictResolution, setConflictResolution] = useState<
		'skip' | 'replace' | 'rename' | 'fail'
	>('skip');
	const [dryRun, setDryRun] = useState(true);
	const [validationResult, setValidationResult] =
		useState<MigrationValidationResult | null>(null);
	const [importResult, setImportResult] =
		useState<MigrationImportResult | null>(null);
	const [importError, setImportError] = useState<string | null>(null);

	// Mutations
	const generateKey = useGenerateExportKey();
	const exportMutation = useMigrationExport();
	const validateMutation = useValidateMigrationImport();
	const importMutation = useMigrationImport();

	const handleGenerateKey = async () => {
		try {
			const result = await generateKey.mutateAsync();
			setEncryptionKey(result.key);
		} catch {
			setExportError('Failed to generate encryption key');
		}
	};

	const handleExport = async () => {
		setExportError(null);
		setExportSuccess(false);

		try {
			const blob = await exportMutation.mutateAsync({
				include_secrets: includeSecrets,
				include_system_config: includeSystemConfig,
				encryption_key: useEncryption ? encryptionKey : undefined,
				description: exportDescription || undefined,
			});

			downloadMigrationExport(blob, useEncryption && !!encryptionKey);
			setExportSuccess(true);
		} catch (err) {
			setExportError(
				err instanceof Error ? err.message : 'Export failed. Please try again.',
			);
		}
	};

	const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file) return;

		setImportFile(file);
		setValidationResult(null);
		setImportResult(null);
		setImportError(null);

		try {
			const content = await readFileAsText(file);
			setImportFileContent(content);
		} catch {
			setImportError('Failed to read file');
		}
	};

	const handleValidate = async () => {
		if (!importFileContent) return;

		setValidationResult(null);
		setImportError(null);

		try {
			const result = await validateMutation.mutateAsync({
				data: importFileContent,
				decryption_key: decryptionKey || undefined,
			});
			setValidationResult(result);
		} catch (err) {
			setImportError(
				err instanceof Error
					? err.message
					: 'Validation failed. Please try again.',
			);
		}
	};

	const handleImport = async () => {
		if (!importFileContent || !validationResult?.valid) return;

		setImportResult(null);
		setImportError(null);

		try {
			const result = await importMutation.mutateAsync({
				data: importFileContent,
				decryption_key: decryptionKey || undefined,
				conflict_resolution: conflictResolution,
				dry_run: dryRun,
			});
			setImportResult(result);

			// If this was a dry run and successful, suggest running the actual import
			if (result.success && result.dry_run) {
				// Keep the form state so user can run actual import
			}
		} catch (err) {
			setImportError(
				err instanceof Error ? err.message : 'Import failed. Please try again.',
			);
		}
	};

	const resetImportState = () => {
		setImportFile(null);
		setImportFileContent(null);
		setDecryptionKey('');
		setValidationResult(null);
		setImportResult(null);
		setImportError(null);
		if (fileInputRef.current) {
			fileInputRef.current.value = '';
		}
	};

	if (userLoading) {
		return (
			<div className="space-y-6">
				<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div
								key={`skeleton-${i}`}
								className="h-12 bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
							/>
						))}
					</div>
				</div>
			</div>
		);
	}

	if (!user?.is_superuser) {
		return (
			<div className="text-center py-12">
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
						d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
					/>
				</svg>
				<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
					Access Denied
				</h2>
				<p className="text-gray-500 dark:text-gray-400">
					You need superuser privileges to access migration settings.
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					Migration
				</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
					Export or import your Keldris configuration for backup or migration to
					a new server
				</p>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('export')}
						className={`pb-4 px-1 border-b-2 font-medium text-sm transition-colors ${
							activeTab === 'export'
								? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
						}`}
					>
						Export
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('import')}
						className={`pb-4 px-1 border-b-2 font-medium text-sm transition-colors ${
							activeTab === 'import'
								? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
						}`}
					>
						Import
					</button>
				</nav>
			</div>

			{/* Export Tab */}
			{activeTab === 'export' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Export Configuration
						</h2>
						<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							Export your entire Keldris configuration including organizations,
							users, agents, repositories, and policies
						</p>
					</div>
					<div className="p-6 space-y-6">
						{/* Export Options */}
						<div className="space-y-4">
							<h3 className="text-sm font-medium text-gray-900 dark:text-white">
								Export Options
							</h3>

							<div className="space-y-3">
								<label className="flex items-start gap-3">
									<input
										type="checkbox"
										checked={includeSystemConfig}
										onChange={(e) => setIncludeSystemConfig(e.target.checked)}
										className="mt-1 h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<div>
										<span className="text-sm font-medium text-gray-700 dark:text-gray-300">
											Include system configuration
										</span>
										<p className="text-xs text-gray-500 dark:text-gray-400">
											Export SMTP, OIDC, storage, and security settings
										</p>
									</div>
								</label>

								<label className="flex items-start gap-3">
									<input
										type="checkbox"
										checked={includeSecrets}
										onChange={(e) => setIncludeSecrets(e.target.checked)}
										className="mt-1 h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<div>
										<span className="text-sm font-medium text-gray-700 dark:text-gray-300">
											Include secrets
										</span>
										<p className="text-xs text-gray-500 dark:text-gray-400">
											Include API keys, passwords, and other sensitive data
											(requires encryption)
										</p>
									</div>
								</label>
							</div>
						</div>

						{/* Encryption */}
						<div className="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
							<h3 className="text-sm font-medium text-gray-900 dark:text-white">
								Encryption
							</h3>

							<label className="flex items-start gap-3">
								<input
									type="checkbox"
									checked={useEncryption}
									onChange={(e) => setUseEncryption(e.target.checked)}
									className="mt-1 h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
								/>
								<div>
									<span className="text-sm font-medium text-gray-700 dark:text-gray-300">
										Encrypt export file
									</span>
									<p className="text-xs text-gray-500 dark:text-gray-400">
										Use AES-256-GCM encryption to protect the export file
									</p>
								</div>
							</label>

							{useEncryption && (
								<div className="ml-7 space-y-3">
									<div>
										<label
											htmlFor="encryption-key"
											className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
										>
											Encryption Key
										</label>
										<div className="flex gap-2">
											<input
												type="text"
												id="encryption-key"
												value={encryptionKey}
												onChange={(e) => setEncryptionKey(e.target.value)}
												placeholder="Enter or generate a key"
												className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
											/>
											<button
												type="button"
												onClick={handleGenerateKey}
												disabled={generateKey.isPending}
												className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50 whitespace-nowrap"
											>
												{generateKey.isPending ? 'Generating...' : 'Generate'}
											</button>
										</div>
										<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
											Save this key securely - you&apos;ll need it to import
											the data
										</p>
									</div>
								</div>
							)}

							{includeSecrets && !useEncryption && (
								<div className="ml-7 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
									<p className="text-sm text-yellow-800 dark:text-yellow-200">
										Warning: Exporting secrets without encryption is not
										recommended. Enable encryption to protect sensitive data.
									</p>
								</div>
							)}
						</div>

						{/* Description */}
						<div className="space-y-2 pt-4 border-t border-gray-200 dark:border-gray-700">
							<label
								htmlFor="export-description"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Description (optional)
							</label>
							<textarea
								id="export-description"
								value={exportDescription}
								onChange={(e) => setExportDescription(e.target.value)}
								placeholder="Add a note about this export..."
								rows={2}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
						</div>

						{/* Export Button and Status */}
						<div className="pt-4 border-t border-gray-200 dark:border-gray-700">
							{exportError && (
								<div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
									<p className="text-sm text-red-800 dark:text-red-200">
										{exportError}
									</p>
								</div>
							)}

							{exportSuccess && (
								<div className="mb-4 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
									<p className="text-sm text-green-800 dark:text-green-200">
										Export completed successfully. The file has been downloaded.
									</p>
								</div>
							)}

							<button
								type="button"
								onClick={handleExport}
								disabled={
									exportMutation.isPending ||
									(useEncryption && !encryptionKey)
								}
								className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								{exportMutation.isPending ? 'Exporting...' : 'Export'}
							</button>
						</div>
					</div>
				</div>
			)}

			{/* Import Tab */}
			{activeTab === 'import' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Import Configuration
						</h2>
						<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							Import a previously exported Keldris configuration
						</p>
					</div>
					<div className="p-6 space-y-6">
						{/* File Selection */}
						<div className="space-y-4">
							<h3 className="text-sm font-medium text-gray-900 dark:text-white">
								Select Export File
							</h3>

							<div className="flex items-center gap-4">
								<input
									ref={fileInputRef}
									type="file"
									accept=".json,.encrypted"
									onChange={handleFileSelect}
									className="block w-full text-sm text-gray-500 dark:text-gray-400
										file:mr-4 file:py-2 file:px-4
										file:rounded-lg file:border-0
										file:text-sm file:font-medium
										file:bg-indigo-50 file:text-indigo-700
										dark:file:bg-indigo-900/30 dark:file:text-indigo-400
										hover:file:bg-indigo-100 dark:hover:file:bg-indigo-900/50
										file:cursor-pointer"
								/>
								{importFile && (
									<button
										type="button"
										onClick={resetImportState}
										className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
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

							{importFile && (
								<p className="text-sm text-gray-600 dark:text-gray-400">
									Selected: {importFile.name} (
									{(importFile.size / 1024).toFixed(1)} KB)
								</p>
							)}
						</div>

						{/* Decryption Key */}
						{importFileContent && (
							<div className="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white">
									Decryption Key
								</h3>
								<div>
									<input
										type="text"
										value={decryptionKey}
										onChange={(e) => setDecryptionKey(e.target.value)}
										placeholder="Enter decryption key if the file is encrypted"
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
									/>
								</div>

								{/* Validate Button */}
								<button
									type="button"
									onClick={handleValidate}
									disabled={validateMutation.isPending}
									className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
								>
									{validateMutation.isPending
										? 'Validating...'
										: 'Validate File'}
								</button>
							</div>
						)}

						{/* Validation Result */}
						{validationResult && (
							<div className="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white">
									Validation Result
								</h3>

								{validationResult.valid ? (
									<div className="p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg space-y-3">
										<div className="flex items-center gap-2">
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-green-600 dark:text-green-400"
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
											<span className="font-medium text-green-800 dark:text-green-200">
												File is valid and ready to import
											</span>
										</div>

										{validationResult.metadata && (
											<div className="grid grid-cols-2 gap-2 text-sm">
												<div>
													<span className="text-gray-600 dark:text-gray-400">
														Version:
													</span>{' '}
													<span className="text-gray-900 dark:text-white">
														{validationResult.metadata.version}
													</span>
												</div>
												<div>
													<span className="text-gray-600 dark:text-gray-400">
														Exported:
													</span>{' '}
													<span className="text-gray-900 dark:text-white">
														{new Date(
															validationResult.metadata.exported_at,
														).toLocaleString()}
													</span>
												</div>
												{validationResult.metadata.description && (
													<div className="col-span-2">
														<span className="text-gray-600 dark:text-gray-400">
															Description:
														</span>{' '}
														<span className="text-gray-900 dark:text-white">
															{validationResult.metadata.description}
														</span>
													</div>
												)}
											</div>
										)}

										{validationResult.entity_counts && (
											<div className="grid grid-cols-3 gap-2 text-sm pt-2 border-t border-green-200 dark:border-green-800">
												{validationResult.entity_counts.organizations !==
													undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Organizations:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.organizations}
														</span>
													</div>
												)}
												{validationResult.entity_counts.users !== undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Users:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.users}
														</span>
													</div>
												)}
												{validationResult.entity_counts.agents !==
													undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Agents:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.agents}
														</span>
													</div>
												)}
												{validationResult.entity_counts.repositories !==
													undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Repositories:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.repositories}
														</span>
													</div>
												)}
												{validationResult.entity_counts.schedules !==
													undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Schedules:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.schedules}
														</span>
													</div>
												)}
												{validationResult.entity_counts.policies !==
													undefined && (
													<div>
														<span className="text-gray-600 dark:text-gray-400">
															Policies:
														</span>{' '}
														<span className="font-medium text-gray-900 dark:text-white">
															{validationResult.entity_counts.policies}
														</span>
													</div>
												)}
											</div>
										)}

										{validationResult.warnings &&
											validationResult.warnings.length > 0 && (
												<div className="pt-2 border-t border-yellow-200 dark:border-yellow-800">
													<p className="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-1">
														Warnings:
													</p>
													<ul className="text-sm text-yellow-700 dark:text-yellow-300 list-disc list-inside">
														{validationResult.warnings.map((warning, idx) => (
															<li key={`validation-warning-${idx}`}>{warning}</li>
														))}
													</ul>
												</div>
											)}
									</div>
								) : (
									<div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
										<div className="flex items-center gap-2 mb-2">
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-red-600 dark:text-red-400"
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
											<span className="font-medium text-red-800 dark:text-red-200">
												Validation failed
											</span>
										</div>
										{validationResult.requires_key && (
											<p className="text-sm text-red-700 dark:text-red-300 mb-2">
												This file is encrypted. Please provide the decryption
												key.
											</p>
										)}
										{validationResult.errors &&
											validationResult.errors.length > 0 && (
												<ul className="text-sm text-red-700 dark:text-red-300 list-disc list-inside">
													{validationResult.errors.map((error, idx) => (
														<li key={`validation-error-${idx}`}>{error}</li>
													))}
												</ul>
											)}
									</div>
								)}
							</div>
						)}

						{/* Import Options */}
						{validationResult?.valid && (
							<div className="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white">
									Import Options
								</h3>

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
										onChange={(e) =>
											setConflictResolution(
												e.target.value as typeof conflictResolution,
											)
										}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
									>
										<option value="skip">
											Skip - Keep existing data, skip conflicts
										</option>
										<option value="replace">
											Replace - Overwrite existing data with imported data
										</option>
										<option value="rename">
											Rename - Create new entries with modified names
										</option>
										<option value="fail">
											Fail - Stop import if any conflict is found
										</option>
									</select>
								</div>

								<label className="flex items-start gap-3">
									<input
										type="checkbox"
										checked={dryRun}
										onChange={(e) => setDryRun(e.target.checked)}
										className="mt-1 h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<div>
										<span className="text-sm font-medium text-gray-700 dark:text-gray-300">
											Dry run
										</span>
										<p className="text-xs text-gray-500 dark:text-gray-400">
											Preview the import without making any changes
										</p>
									</div>
								</label>
							</div>
						)}

						{/* Import Result */}
						{importResult && (
							<div className="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white">
									Import Result {importResult.dry_run && '(Dry Run)'}
								</h3>

								{importResult.success ? (
									<div className="p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg space-y-3">
										<div className="flex items-center gap-2">
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-green-600 dark:text-green-400"
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
											<span className="font-medium text-green-800 dark:text-green-200">
												{importResult.dry_run
													? 'Dry run completed successfully'
													: 'Import completed successfully'}
											</span>
										</div>

										{importResult.message && (
											<p className="text-sm text-green-700 dark:text-green-300">
												{importResult.message}
											</p>
										)}

										<div className="grid grid-cols-2 gap-4 text-sm">
											<div>
												<h4 className="font-medium text-gray-700 dark:text-gray-300 mb-1">
													Imported:
												</h4>
												<ul className="space-y-1 text-gray-600 dark:text-gray-400">
													{importResult.imported.organizations !== undefined &&
														importResult.imported.organizations > 0 && (
															<li>
																Organizations:{' '}
																{importResult.imported.organizations}
															</li>
														)}
													{importResult.imported.users !== undefined &&
														importResult.imported.users > 0 && (
															<li>Users: {importResult.imported.users}</li>
														)}
													{importResult.imported.agents !== undefined &&
														importResult.imported.agents > 0 && (
															<li>Agents: {importResult.imported.agents}</li>
														)}
													{importResult.imported.repositories !== undefined &&
														importResult.imported.repositories > 0 && (
															<li>
																Repositories:{' '}
																{importResult.imported.repositories}
															</li>
														)}
													{importResult.imported.schedules !== undefined &&
														importResult.imported.schedules > 0 && (
															<li>
																Schedules: {importResult.imported.schedules}
															</li>
														)}
													{importResult.imported.policies !== undefined &&
														importResult.imported.policies > 0 && (
															<li>
																Policies: {importResult.imported.policies}
															</li>
														)}
												</ul>
											</div>
											{(importResult.skipped.organizations ||
												importResult.skipped.users ||
												importResult.skipped.agents ||
												importResult.skipped.repositories ||
												importResult.skipped.schedules ||
												importResult.skipped.policies) && (
												<div>
													<h4 className="font-medium text-gray-700 dark:text-gray-300 mb-1">
														Skipped:
													</h4>
													<ul className="space-y-1 text-gray-600 dark:text-gray-400">
														{importResult.skipped.organizations !== undefined &&
															importResult.skipped.organizations > 0 && (
																<li>
																	Organizations:{' '}
																	{importResult.skipped.organizations}
																</li>
															)}
														{importResult.skipped.users !== undefined &&
															importResult.skipped.users > 0 && (
																<li>Users: {importResult.skipped.users}</li>
															)}
														{importResult.skipped.agents !== undefined &&
															importResult.skipped.agents > 0 && (
																<li>Agents: {importResult.skipped.agents}</li>
															)}
														{importResult.skipped.repositories !== undefined &&
															importResult.skipped.repositories > 0 && (
																<li>
																	Repositories:{' '}
																	{importResult.skipped.repositories}
																</li>
															)}
														{importResult.skipped.schedules !== undefined &&
															importResult.skipped.schedules > 0 && (
																<li>
																	Schedules: {importResult.skipped.schedules}
																</li>
															)}
														{importResult.skipped.policies !== undefined &&
															importResult.skipped.policies > 0 && (
																<li>
																	Policies: {importResult.skipped.policies}
																</li>
															)}
													</ul>
												</div>
											)}
										</div>

										{importResult.warnings &&
											importResult.warnings.length > 0 && (
												<div className="pt-2 border-t border-yellow-200 dark:border-yellow-800">
													<p className="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-1">
														Warnings:
													</p>
													<ul className="text-sm text-yellow-700 dark:text-yellow-300 list-disc list-inside">
														{importResult.warnings.map((warning, idx) => (
															<li key={`import-warning-${idx}`}>{warning}</li>
														))}
													</ul>
												</div>
											)}

										{importResult.dry_run && (
											<div className="pt-2 border-t border-green-200 dark:border-green-800">
												<p className="text-sm text-green-700 dark:text-green-300">
													This was a dry run. Uncheck &quot;Dry run&quot; and
													click Import again to apply the changes.
												</p>
											</div>
										)}
									</div>
								) : (
									<div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
										<div className="flex items-center gap-2 mb-2">
											<svg
												aria-hidden="true"
												className="w-5 h-5 text-red-600 dark:text-red-400"
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
											<span className="font-medium text-red-800 dark:text-red-200">
												Import failed
											</span>
										</div>
										{importResult.message && (
											<p className="text-sm text-red-700 dark:text-red-300 mb-2">
												{importResult.message}
											</p>
										)}
										{importResult.errors && importResult.errors.length > 0 && (
											<ul className="text-sm text-red-700 dark:text-red-300 list-disc list-inside">
												{importResult.errors.map((error, idx) => (
													<li key={`import-error-${idx}`}>{error}</li>
												))}
											</ul>
										)}
									</div>
								)}
							</div>
						)}

						{/* Import Error */}
						{importError && (
							<div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
								<p className="text-sm text-red-800 dark:text-red-200">
									{importError}
								</p>
							</div>
						)}

						{/* Import Button */}
						{validationResult?.valid && (
							<div className="pt-4 border-t border-gray-200 dark:border-gray-700">
								<button
									type="button"
									onClick={handleImport}
									disabled={importMutation.isPending}
									className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{importMutation.isPending
										? 'Importing...'
										: dryRun
											? 'Preview Import'
											: 'Import'}
								</button>
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
}
