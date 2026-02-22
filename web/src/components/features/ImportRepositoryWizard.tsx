import { useState } from 'react';
import { useAgents } from '../../hooks/useAgents';
import {
	useImportPreview,
	useImportRepository,
	useVerifyImportAccess,
} from '../../hooks/useRepositoryImport';
import type {
	Agent,
	B2BackendConfig,
	BackendConfig,
	DropboxBackendConfig,
	ImportPreviewResponse,
	LocalBackendConfig,
	RepositoryType,
	RestBackendConfig,
	S3BackendConfig,
	SFTPBackendConfig,
	SnapshotPreview,
} from '../../lib/types';
import { formatBytes } from '../../lib/utils';

interface FormFieldProps {
	label: string;
	id: string;
	value: string;
	onChange: (value: string) => void;
	placeholder?: string;
	required?: boolean;
	type?: 'text' | 'password' | 'number';
	helpText?: string;
}

function FormField({
	label,
	id,
	value,
	onChange,
	placeholder,
	required = false,
	type = 'text',
	helpText,
}: FormFieldProps) {
	return (
		<div>
			<label
				htmlFor={id}
				className="block text-sm font-medium text-gray-700 mb-1"
			>
				{label}
				{required && <span className="text-red-500 ml-1">*</span>}
			</label>
			<input
				type={type}
				id={id}
				value={value}
				onChange={(e) => onChange(e.target.value)}
				placeholder={placeholder}
				className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				required={required}
			/>
			{helpText && <p className="mt-1 text-xs text-gray-500">{helpText}</p>}
		</div>
	);
}

type WizardStep = 'connection' | 'preview' | 'configure' | 'importing';

interface ImportRepositoryWizardProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (repositoryName: string, snapshotsImported: number) => void;
}

export function ImportRepositoryWizard({
	isOpen,
	onClose,
	onSuccess,
}: ImportRepositoryWizardProps) {
	const [step, setStep] = useState<WizardStep>('connection');
	const [error, setError] = useState<string | null>(null);

	// Connection form state
	const [name, setName] = useState('');
	const [type, setType] = useState<RepositoryType>('local');
	const [password, setPassword] = useState('');
	const [escrowEnabled, setEscrowEnabled] = useState(false);

	// Local backend fields
	const [localPath, setLocalPath] = useState('');

	// S3 backend fields
	const [s3Endpoint, setS3Endpoint] = useState('');
	const [s3Bucket, setS3Bucket] = useState('');
	const [s3Prefix, setS3Prefix] = useState('');
	const [s3Region, setS3Region] = useState('');
	const [s3AccessKey, setS3AccessKey] = useState('');
	const [s3SecretKey, setS3SecretKey] = useState('');
	const [s3UseSsl, setS3UseSsl] = useState(true);

	// B2 backend fields
	const [b2Bucket, setB2Bucket] = useState('');
	const [b2Prefix, setB2Prefix] = useState('');
	const [b2AccountId, setB2AccountId] = useState('');
	const [b2AppKey, setB2AppKey] = useState('');

	// SFTP backend fields
	const [sftpHost, setSftpHost] = useState('');
	const [sftpPort, setSftpPort] = useState('22');
	const [sftpUser, setSftpUser] = useState('');
	const [sftpPath, setSftpPath] = useState('');
	const [sftpPassword, setSftpPassword] = useState('');
	const [sftpPrivateKey, setSftpPrivateKey] = useState('');

	// REST backend fields
	const [restUrl, setRestUrl] = useState('');
	const [restUsername, setRestUsername] = useState('');
	const [restPassword, setRestPassword] = useState('');

	// Dropbox backend fields
	const [dropboxRemoteName, setDropboxRemoteName] = useState('');
	const [dropboxPath, setDropboxPath] = useState('');
	const [dropboxToken, setDropboxToken] = useState('');
	const [dropboxAppKey, setDropboxAppKey] = useState('');
	const [dropboxAppSecret, setDropboxAppSecret] = useState('');

	// Preview state
	const [preview, setPreview] = useState<ImportPreviewResponse | null>(null);

	// Configure state
	const [selectedHostnames, setSelectedHostnames] = useState<string[]>([]);
	const [selectedSnapshots, setSelectedSnapshots] = useState<string[]>([]);
	const [selectedAgentId, setSelectedAgentId] = useState<string>('');

	// Hooks
	const verifyAccess = useVerifyImportAccess();
	const importPreview = useImportPreview();
	const importRepository = useImportRepository();
	const { data: agents } = useAgents();

	const buildConfig = (): BackendConfig => {
		switch (type) {
			case 'local':
				return { path: localPath } as LocalBackendConfig;
			case 's3':
				return {
					endpoint: s3Endpoint || undefined,
					bucket: s3Bucket,
					prefix: s3Prefix || undefined,
					region: s3Region || undefined,
					access_key_id: s3AccessKey,
					secret_access_key: s3SecretKey,
					use_ssl: s3UseSsl,
				} as S3BackendConfig;
			case 'b2':
				return {
					bucket: b2Bucket,
					prefix: b2Prefix || undefined,
					account_id: b2AccountId,
					application_key: b2AppKey,
				} as B2BackendConfig;
			case 'sftp':
				return {
					host: sftpHost,
					port: sftpPort ? Number.parseInt(sftpPort, 10) : undefined,
					user: sftpUser,
					path: sftpPath,
					password: sftpPassword || undefined,
					private_key: sftpPrivateKey || undefined,
				} as SFTPBackendConfig;
			case 'rest':
				return {
					url: restUrl,
					username: restUsername || undefined,
					password: restPassword || undefined,
				} as RestBackendConfig;
			case 'dropbox':
				return {
					remote_name: dropboxRemoteName,
					path: dropboxPath || undefined,
					token: dropboxToken || undefined,
					app_key: dropboxAppKey || undefined,
					app_secret: dropboxAppSecret || undefined,
				} as DropboxBackendConfig;
			default:
				return { path: localPath } as LocalBackendConfig;
		}
	};

	const handleVerifyAndPreview = async () => {
		setError(null);
		try {
			const config = buildConfig();

			// First verify access
			const verifyResult = await verifyAccess.mutateAsync({
				type,
				config,
				password,
			});

			if (!verifyResult.success) {
				setError(verifyResult.message);
				return;
			}

			// Then get preview
			const previewResult = await importPreview.mutateAsync({
				type,
				config,
				password,
			});

			setPreview(previewResult);
			setSelectedHostnames(previewResult.hostnames);
			setSelectedSnapshots([]);
			setStep('preview');
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to access repository',
			);
		}
	};

	const handleImport = async () => {
		setError(null);
		setStep('importing');
		try {
			const result = await importRepository.mutateAsync({
				name,
				type,
				config: buildConfig(),
				password,
				escrow_enabled: escrowEnabled,
				hostnames: selectedHostnames.length > 0 ? selectedHostnames : undefined,
				snapshot_ids:
					selectedSnapshots.length > 0 ? selectedSnapshots : undefined,
				agent_id: selectedAgentId || undefined,
			});

			onSuccess(result.repository.name, result.snapshots_imported);
			resetForm();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to import repository',
			);
			setStep('configure');
		}
	};

	const resetForm = () => {
		setStep('connection');
		setError(null);
		setName('');
		setType('local');
		setPassword('');
		setEscrowEnabled(false);
		setLocalPath('');
		setS3Endpoint('');
		setS3Bucket('');
		setS3Prefix('');
		setS3Region('');
		setS3AccessKey('');
		setS3SecretKey('');
		setS3UseSsl(true);
		setB2Bucket('');
		setB2Prefix('');
		setB2AccountId('');
		setB2AppKey('');
		setSftpHost('');
		setSftpPort('22');
		setSftpUser('');
		setSftpPath('');
		setSftpPassword('');
		setSftpPrivateKey('');
		setRestUrl('');
		setRestUsername('');
		setRestPassword('');
		setDropboxRemoteName('');
		setDropboxPath('');
		setDropboxToken('');
		setDropboxAppKey('');
		setDropboxAppSecret('');
		setPreview(null);
		setSelectedHostnames([]);
		setSelectedSnapshots([]);
		setSelectedAgentId('');
	};

	const handleClose = () => {
		resetForm();
		onClose();
	};

	if (!isOpen) return null;

	const renderBackendFields = () => {
		switch (type) {
			case 'local':
				return (
					<FormField
						label="Path"
						id="import-local-path"
						value={localPath}
						onChange={setLocalPath}
						placeholder="/var/backups/restic"
						required
						helpText="Absolute path to the existing Restic repository"
					/>
				);

			case 's3':
				return (
					<>
						<FormField
							label="Bucket"
							id="import-s3-bucket"
							value={s3Bucket}
							onChange={setS3Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<FormField
							label="Access Key ID"
							id="import-s3-access-key"
							value={s3AccessKey}
							onChange={setS3AccessKey}
							placeholder="AKIAIOSFODNN7EXAMPLE"
							required
						/>
						<FormField
							label="Secret Access Key"
							id="import-s3-secret-key"
							value={s3SecretKey}
							onChange={setS3SecretKey}
							type="password"
							required
						/>
						<FormField
							label="Region"
							id="import-s3-region"
							value={s3Region}
							onChange={setS3Region}
							placeholder="us-east-1"
						/>
						<FormField
							label="Endpoint"
							id="import-s3-endpoint"
							value={s3Endpoint}
							onChange={setS3Endpoint}
							placeholder="minio.example.com:9000"
							helpText="For MinIO, Wasabi, or other S3-compatible services"
						/>
						<FormField
							label="Prefix"
							id="import-s3-prefix"
							value={s3Prefix}
							onChange={setS3Prefix}
							placeholder="backups/server1"
						/>
						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="import-s3-use-ssl"
								checked={s3UseSsl}
								onChange={(e) => setS3UseSsl(e.target.checked)}
								className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
							/>
							<label
								htmlFor="import-s3-use-ssl"
								className="text-sm text-gray-700"
							>
								Use SSL/TLS
							</label>
						</div>
					</>
				);

			case 'b2':
				return (
					<>
						<FormField
							label="Bucket"
							id="import-b2-bucket"
							value={b2Bucket}
							onChange={setB2Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<FormField
							label="Account ID"
							id="import-b2-account-id"
							value={b2AccountId}
							onChange={setB2AccountId}
							placeholder="0012345678abcdef"
							required
						/>
						<FormField
							label="Application Key"
							id="import-b2-app-key"
							value={b2AppKey}
							onChange={setB2AppKey}
							type="password"
							required
						/>
						<FormField
							label="Prefix"
							id="import-b2-prefix"
							value={b2Prefix}
							onChange={setB2Prefix}
							placeholder="backups/server1"
						/>
					</>
				);

			case 'sftp':
				return (
					<>
						<div className="grid grid-cols-3 gap-3">
							<div className="col-span-2">
								<FormField
									label="Host"
									id="import-sftp-host"
									value={sftpHost}
									onChange={setSftpHost}
									placeholder="backup.example.com"
									required
								/>
							</div>
							<FormField
								label="Port"
								id="import-sftp-port"
								value={sftpPort}
								onChange={setSftpPort}
								placeholder="22"
								type="number"
							/>
						</div>
						<FormField
							label="Username"
							id="import-sftp-user"
							value={sftpUser}
							onChange={setSftpUser}
							placeholder="backup"
							required
						/>
						<FormField
							label="Remote Path"
							id="import-sftp-path"
							value={sftpPath}
							onChange={setSftpPath}
							placeholder="/var/backups/restic"
							required
						/>
						<FormField
							label="Password"
							id="import-sftp-password"
							value={sftpPassword}
							onChange={setSftpPassword}
							type="password"
						/>
						<div>
							<label
								htmlFor="import-sftp-private-key"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Private Key
							</label>
							<textarea
								id="import-sftp-private-key"
								value={sftpPrivateKey}
								onChange={(e) => setSftpPrivateKey(e.target.value)}
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-xs"
								rows={4}
							/>
						</div>
					</>
				);

			case 'rest':
				return (
					<>
						<FormField
							label="URL"
							id="import-rest-url"
							value={restUrl}
							onChange={setRestUrl}
							placeholder="https://backup.example.com:8000"
							required
						/>
						<FormField
							label="Username"
							id="import-rest-username"
							value={restUsername}
							onChange={setRestUsername}
							placeholder="backup"
						/>
						<FormField
							label="Password"
							id="import-rest-password"
							value={restPassword}
							onChange={setRestPassword}
							type="password"
						/>
					</>
				);

			case 'dropbox':
				return (
					<>
						<FormField
							label="Remote Name"
							id="import-dropbox-remote-name"
							value={dropboxRemoteName}
							onChange={setDropboxRemoteName}
							placeholder="dropbox"
							required
						/>
						<FormField
							label="Path"
							id="import-dropbox-path"
							value={dropboxPath}
							onChange={setDropboxPath}
							placeholder="/Backups/server1"
						/>
						<FormField
							label="Token"
							id="import-dropbox-token"
							value={dropboxToken}
							onChange={setDropboxToken}
							type="password"
						/>
						<FormField
							label="App Key"
							id="import-dropbox-app-key"
							value={dropboxAppKey}
							onChange={setDropboxAppKey}
						/>
						<FormField
							label="App Secret"
							id="import-dropbox-app-secret"
							value={dropboxAppSecret}
							onChange={setDropboxAppSecret}
							type="password"
						/>
					</>
				);

			default:
				return null;
		}
	};

	const renderConnectionStep = () => (
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
							Import Existing Repository
						</p>
						<p className="text-sm text-blue-700 mt-1">
							Connect to an existing Restic repository to import its snapshots
							into Keldris. You'll need the repository password.
						</p>
					</div>
				</div>
			</div>

			<FormField
				label="Repository Name"
				id="import-repo-name"
				value={name}
				onChange={setName}
				placeholder="e.g., imported-backup"
				required
				helpText="A name to identify this repository in Keldris"
			/>

			<div>
				<label
					htmlFor="import-repo-type"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Type
				</label>
				<select
					id="import-repo-type"
					value={type}
					onChange={(e) => setType(e.target.value as RepositoryType)}
					className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				>
					<option value="local">Local Filesystem</option>
					<option value="s3">Amazon S3 / MinIO / Wasabi</option>
					<option value="b2">Backblaze B2</option>
					<option value="sftp">SFTP</option>
					<option value="rest">Restic REST Server</option>
					<option value="dropbox">Dropbox (via rclone)</option>
				</select>
			</div>

			<hr className="my-4" />

			{renderBackendFields()}

			<hr className="my-4" />

			<FormField
				label="Repository Password"
				id="import-repo-password"
				value={password}
				onChange={setPassword}
				type="password"
				required
				helpText="The password used to encrypt this Restic repository"
			/>

			<div className="flex items-start gap-3 pt-2">
				<input
					type="checkbox"
					id="import-escrow-enabled"
					checked={escrowEnabled}
					onChange={(e) => setEscrowEnabled(e.target.checked)}
					className="mt-1 h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
				/>
				<div>
					<label
						htmlFor="import-escrow-enabled"
						className="block text-sm font-medium text-gray-700"
					>
						Enable key escrow
					</label>
					<p className="text-xs text-gray-500">
						Store an encrypted copy of the password for recovery by
						administrators
					</p>
				</div>
			</div>
		</div>
	);

	const renderPreviewStep = () => {
		if (!preview) return null;

		return (
			<div className="space-y-4">
				<div className="bg-green-50 border border-green-200 rounded-lg p-4">
					<div className="flex gap-3">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5"
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
						<div>
							<p className="text-sm text-green-800 font-medium">
								Repository Found
							</p>
							<p className="text-sm text-green-700 mt-1">
								Successfully connected to the repository. Review the contents
								below.
							</p>
						</div>
					</div>
				</div>

				<div className="grid grid-cols-2 gap-4">
					<div className="bg-gray-50 rounded-lg p-4">
						<p className="text-sm text-gray-500">Snapshots</p>
						<p className="text-2xl font-bold text-gray-900">
							{preview.snapshot_count}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-4">
						<p className="text-sm text-gray-500">Total Size</p>
						<p className="text-2xl font-bold text-gray-900">
							{formatBytes(preview.total_size)}
						</p>
					</div>
				</div>

				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Hostnames Found
					</h4>
					<div className="flex flex-wrap gap-2">
						{preview.hostnames.map((hostname) => (
							<span
								key={hostname}
								className="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-sm"
							>
								{hostname}
							</span>
						))}
					</div>
				</div>

				{preview.snapshots.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Recent Snapshots (showing {Math.min(5, preview.snapshots.length)}{' '}
							of {preview.snapshots.length})
						</h4>
						<div className="border border-gray-200 rounded-lg divide-y divide-gray-200 max-h-48 overflow-y-auto">
							{preview.snapshots.slice(0, 5).map((snap: SnapshotPreview) => (
								<div key={snap.id} className="p-3 text-sm">
									<div className="flex items-center justify-between">
										<span className="font-mono text-xs text-gray-500">
											{snap.short_id}
										</span>
										<span className="text-gray-500">
											{new Date(snap.time).toLocaleString()}
										</span>
									</div>
									<div className="text-gray-700 mt-1">
										<span className="font-medium">{snap.hostname}</span>
										{snap.paths.length > 0 && (
											<span className="text-gray-500 ml-2">
												{snap.paths[0]}
												{snap.paths.length > 1 &&
													` +${snap.paths.length - 1} more`}
											</span>
										)}
									</div>
								</div>
							))}
						</div>
					</div>
				)}
			</div>
		);
	};

	const renderConfigureStep = () => {
		if (!preview) return null;

		return (
			<div className="space-y-4">
				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Filter by Hostname
					</h4>
					<p className="text-xs text-gray-500 mb-2">
						Select which hostnames to import (all selected by default)
					</p>
					<div className="space-y-2 max-h-32 overflow-y-auto border border-gray-200 rounded-lg p-3">
						{preview.hostnames.map((hostname) => (
							<label key={hostname} className="flex items-center gap-2">
								<input
									type="checkbox"
									checked={selectedHostnames.includes(hostname)}
									onChange={(e) => {
										if (e.target.checked) {
											setSelectedHostnames([...selectedHostnames, hostname]);
										} else {
											setSelectedHostnames(
												selectedHostnames.filter((h) => h !== hostname),
											);
										}
									}}
									className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
								/>
								<span className="text-sm text-gray-700">{hostname}</span>
							</label>
						))}
					</div>
				</div>

				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Assign to Agent
					</h4>
					<p className="text-xs text-gray-500 mb-2">
						Optionally assign imported snapshots to an existing agent
					</p>
					<select
						value={selectedAgentId}
						onChange={(e) => setSelectedAgentId(e.target.value)}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="">No agent (unassigned)</option>
						{agents?.map((agent: Agent) => (
							<option key={agent.id} value={agent.id}>
								{agent.hostname}
							</option>
						))}
					</select>
				</div>

				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<div className="flex gap-3">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5"
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
								Import Summary
							</p>
							<p className="text-sm text-amber-700 mt-1">
								{selectedHostnames.length === preview.hostnames.length
									? `All ${preview.snapshot_count} snapshots`
									: `Snapshots from ${selectedHostnames.length} hostname(s)`}{' '}
								will be imported.
							</p>
						</div>
					</div>
				</div>
			</div>
		);
	};

	const renderImportingStep = () => (
		<div className="py-8 text-center">
			<div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4" />
			<p className="text-gray-600">Importing repository...</p>
			<p className="text-sm text-gray-500 mt-2">This may take a moment</p>
		</div>
	);

	const renderStepContent = () => {
		switch (step) {
			case 'connection':
				return renderConnectionStep();
			case 'preview':
				return renderPreviewStep();
			case 'configure':
				return renderConfigureStep();
			case 'importing':
				return renderImportingStep();
			default:
				return null;
		}
	};

	const getStepTitle = () => {
		switch (step) {
			case 'connection':
				return 'Connect to Repository';
			case 'preview':
				return 'Preview Repository';
			case 'configure':
				return 'Configure Import';
			case 'importing':
				return 'Importing...';
			default:
				return 'Import Repository';
		}
	};

	const isNextDisabled = () => {
		if (step === 'connection') {
			return (
				!name || !password || verifyAccess.isPending || importPreview.isPending
			);
		}
		if (step === 'configure') {
			return selectedHostnames.length === 0;
		}
		return false;
	};

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
				{step !== 'importing' && (
					<div className="flex items-center gap-2 mb-6">
						{['connection', 'preview', 'configure'].map((s, i) => (
							<div key={s} className="flex items-center">
								<div
									className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
										step === s
											? 'bg-indigo-600 text-white'
											: ['preview', 'configure'].indexOf(step) > i
												? 'bg-indigo-100 text-indigo-600'
												: 'bg-gray-100 text-gray-400'
									}`}
								>
									{i + 1}
								</div>
								{i < 2 && (
									<div
										className={`w-12 h-0.5 mx-1 ${
											['preview', 'configure'].indexOf(step) > i
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

				{step !== 'importing' && (
					<div className="flex justify-between mt-6">
						<button
							type="button"
							onClick={() => {
								if (step === 'preview') setStep('connection');
								else if (step === 'configure') setStep('preview');
								else handleClose();
							}}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							{step === 'connection' ? 'Cancel' : 'Back'}
						</button>
						<button
							type="button"
							onClick={() => {
								if (step === 'connection') handleVerifyAndPreview();
								else if (step === 'preview') setStep('configure');
								else if (step === 'configure') handleImport();
							}}
							disabled={isNextDisabled()}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
						>
							{step === 'connection' &&
							(verifyAccess.isPending || importPreview.isPending)
								? 'Connecting...'
								: step === 'connection'
									? 'Connect'
									: step === 'preview'
										? 'Continue'
										: 'Import Repository'}
						</button>
					</div>
				)}
			</div>
		</div>
	);
}
