import { useEffect, useState } from 'react';
import { ImportRepositoryWizard } from '../components/features/ImportRepositoryWizard';
import { HelpTooltip } from '../components/ui/HelpTooltip';
import { StarButton } from '../components/ui/StarButton';
import { useMe } from '../hooks/useAuth';
import { useFavoriteIds } from '../hooks/useFavorites';
import { useOrganizations } from '../hooks/useOrganizations';
import {
	useCloneRepository,
	useCreateRepository,
	useDeleteRepository,
	useRecoverRepositoryKey,
	useRepositories,
	useTestConnection,
	useTestRepository,
} from '../hooks/useRepositories';
import {
	useTriggerVerification,
	useVerificationStatus,
} from '../hooks/useVerifications';
import type {
	B2BackendConfig,
	CloneRepositoryResponse,
	CreateRepositoryResponse,
	DropboxBackendConfig,
	LocalBackendConfig,
	Repository,
	RepositoryType,
	RestBackendConfig,
	S3BackendConfig,
	SFTPBackendConfig,
	TestRepositoryResponse,
	VerificationStatus,
	VerificationType,
} from '../lib/types';
import { formatDate, getRepositoryTypeBadge } from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse">
			<div className="flex items-start justify-between mb-4">
				<div className="h-5 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
				<div className="h-6 w-12 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</div>
			<div className="h-4 w-24 bg-gray-100 rounded" />
		</div>
	);
}

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
				className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
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
				className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				required={required}
			/>
			{helpText && (
				<p className="mt-1 text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
					{helpText}
				</p>
			)}
		</div>
	);
}

interface PasswordModalProps {
	isOpen: boolean;
	onClose: () => void;
	repositoryName: string;
	password: string;
}

function PasswordModal({
	isOpen,
	onClose,
	repositoryName,
	password,
}: PasswordModalProps) {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		await navigator.clipboard.writeText(password);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="flex-shrink-0 w-10 h-10 bg-yellow-100 rounded-full flex items-center justify-center">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-yellow-600"
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
					</div>
					<div>
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							Repository Password
						</h3>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							Save this password - it will only be shown once
						</p>
					</div>
				</div>

				<div className="mb-4">
					<p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
						Repository <span className="font-medium">{repositoryName}</span> has
						been created. Use this password to access your Restic repository:
					</p>
				</div>

				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-4">
						<code className="text-sm font-mono break-all flex-1">
							{password}
						</code>
						<button
							type="button"
							onClick={handleCopy}
							className="flex-shrink-0 px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors"
						>
							{copied ? 'Copied!' : 'Copy'}
						</button>
					</div>
				</div>

				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-6">
					<p className="text-sm text-amber-800">
						<strong>Important:</strong> This password is required to decrypt
						your backups. Store it securely - without it, your backup data
						cannot be recovered.
					</p>
				</div>

				<div className="flex justify-end">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						I've saved the password
					</button>
				</div>
			</div>
		</div>
	);
}

interface AddRepositoryModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (response: CreateRepositoryResponse) => void;
	initialType?: RepositoryType;
}

function AddRepositoryModal({
	isOpen,
	onClose,
	onSuccess,
	initialType,
}: AddRepositoryModalProps) {
	const [name, setName] = useState('');
	const [type, setType] = useState<RepositoryType>(initialType ?? 'local');
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

	// Test connection state
	const [testResult, setTestResult] = useState<TestRepositoryResponse | null>(
		null,
	);

	const createRepository = useCreateRepository();
	const testConnection = useTestConnection();

	useEffect(() => {
		if (initialType) {
			setType(initialType);
		}
	}, [initialType]);

	// Reset test result when type changes
	// biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset when type changes
	useEffect(() => {
		setTestResult(null);
	}, [type]);

	const resetForm = () => {
		setName('');
		setType('local');
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
		setTestResult(null);
		setEscrowEnabled(false);
	};

	const handleTestConnection = async () => {
		setTestResult(null);
		try {
			const result = await testConnection.mutateAsync({
				type,
				config: buildConfig(),
			});
			setTestResult(result);
		} catch {
			setTestResult({
				success: false,
				message: 'Failed to test connection',
			});
		}
	};

	const buildConfig = ():
		| LocalBackendConfig
		| S3BackendConfig
		| B2BackendConfig
		| SFTPBackendConfig
		| RestBackendConfig
		| DropboxBackendConfig => {
		switch (type) {
			case 'local':
				return { path: localPath };
			case 's3':
				return {
					endpoint: s3Endpoint || undefined,
					bucket: s3Bucket,
					prefix: s3Prefix || undefined,
					region: s3Region || undefined,
					access_key_id: s3AccessKey,
					secret_access_key: s3SecretKey,
					use_ssl: s3UseSsl,
				};
			case 'b2':
				return {
					bucket: b2Bucket,
					prefix: b2Prefix || undefined,
					account_id: b2AccountId,
					application_key: b2AppKey,
				};
			case 'sftp':
				return {
					host: sftpHost,
					port: sftpPort ? Number.parseInt(sftpPort, 10) : undefined,
					user: sftpUser,
					path: sftpPath,
					password: sftpPassword || undefined,
					private_key: sftpPrivateKey || undefined,
				};
			case 'rest':
				return {
					url: restUrl,
					username: restUsername || undefined,
					password: restPassword || undefined,
				};
			case 'dropbox':
				return {
					remote_name: dropboxRemoteName,
					path: dropboxPath || undefined,
					token: dropboxToken || undefined,
					app_key: dropboxAppKey || undefined,
					app_secret: dropboxAppSecret || undefined,
				};
			default:
				return { path: localPath };
		}
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			const response = await createRepository.mutateAsync({
				name,
				type,
				config: buildConfig(),
				escrow_enabled: escrowEnabled,
			});
			onSuccess(response);
			resetForm();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	const renderBackendFields = () => {
		switch (type) {
			case 'local':
				return (
					<FormField
						label="Path"
						id="local-path"
						value={localPath}
						onChange={setLocalPath}
						placeholder="/var/backups/restic"
						required
						helpText="Absolute path to the backup directory"
					/>
				);

			case 's3':
				return (
					<>
						<FormField
							label="Bucket"
							id="s3-bucket"
							value={s3Bucket}
							onChange={setS3Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<FormField
							label="Access Key ID"
							id="s3-access-key"
							value={s3AccessKey}
							onChange={setS3AccessKey}
							placeholder="AKIAIOSFODNN7EXAMPLE"
							required
						/>
						<FormField
							label="Secret Access Key"
							id="s3-secret-key"
							value={s3SecretKey}
							onChange={setS3SecretKey}
							type="password"
							required
						/>
						<FormField
							label="Region"
							id="s3-region"
							value={s3Region}
							onChange={setS3Region}
							placeholder="us-east-1"
							helpText="Required for AWS S3"
						/>
						<FormField
							label="Endpoint"
							id="s3-endpoint"
							value={s3Endpoint}
							onChange={setS3Endpoint}
							placeholder="minio.example.com:9000"
							helpText="For MinIO, Wasabi, or other S3-compatible services"
						/>
						<FormField
							label="Prefix"
							id="s3-prefix"
							value={s3Prefix}
							onChange={setS3Prefix}
							placeholder="backups/server1"
							helpText="Optional path prefix within the bucket"
						/>
						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="s3-use-ssl"
								checked={s3UseSsl}
								onChange={(e) => setS3UseSsl(e.target.checked)}
								className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
							/>
							<label
								htmlFor="s3-use-ssl"
								className="text-sm text-gray-700 dark:text-gray-300 dark:text-gray-600"
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
							id="b2-bucket"
							value={b2Bucket}
							onChange={setB2Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<FormField
							label="Account ID"
							id="b2-account-id"
							value={b2AccountId}
							onChange={setB2AccountId}
							placeholder="0012345678abcdef"
							required
						/>
						<FormField
							label="Application Key"
							id="b2-app-key"
							value={b2AppKey}
							onChange={setB2AppKey}
							type="password"
							required
						/>
						<FormField
							label="Prefix"
							id="b2-prefix"
							value={b2Prefix}
							onChange={setB2Prefix}
							placeholder="backups/server1"
							helpText="Optional path prefix within the bucket"
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
									id="sftp-host"
									value={sftpHost}
									onChange={setSftpHost}
									placeholder="backup.example.com"
									required
								/>
							</div>
							<FormField
								label="Port"
								id="sftp-port"
								value={sftpPort}
								onChange={setSftpPort}
								placeholder="22"
								type="number"
							/>
						</div>
						<FormField
							label="Username"
							id="sftp-user"
							value={sftpUser}
							onChange={setSftpUser}
							placeholder="backup"
							required
						/>
						<FormField
							label="Remote Path"
							id="sftp-path"
							value={sftpPath}
							onChange={setSftpPath}
							placeholder="/var/backups/restic"
							required
							helpText="Absolute path on the remote server"
						/>
						<FormField
							label="Password"
							id="sftp-password"
							value={sftpPassword}
							onChange={setSftpPassword}
							type="password"
							helpText="Password or private key required"
						/>
						<div>
							<label
								htmlFor="sftp-private-key"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Private Key
							</label>
							<textarea
								id="sftp-private-key"
								value={sftpPrivateKey}
								onChange={(e) => setSftpPrivateKey(e.target.value)}
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-xs"
								rows={4}
							/>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
								Paste your SSH private key (PEM format)
							</p>
						</div>
					</>
				);

			case 'rest':
				return (
					<>
						<FormField
							label="URL"
							id="rest-url"
							value={restUrl}
							onChange={setRestUrl}
							placeholder="https://backup.example.com:8000"
							required
							helpText="URL of the Restic REST server"
						/>
						<FormField
							label="Username"
							id="rest-username"
							value={restUsername}
							onChange={setRestUsername}
							placeholder="backup"
							helpText="Optional authentication"
						/>
						<FormField
							label="Password"
							id="rest-password"
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
							id="dropbox-remote-name"
							value={dropboxRemoteName}
							onChange={setDropboxRemoteName}
							placeholder="dropbox"
							required
							helpText="Name for the rclone remote configuration"
						/>
						<FormField
							label="Path"
							id="dropbox-path"
							value={dropboxPath}
							onChange={setDropboxPath}
							placeholder="/Backups/server1"
							helpText="Path within your Dropbox"
						/>
						<FormField
							label="Token"
							id="dropbox-token"
							value={dropboxToken}
							onChange={setDropboxToken}
							type="password"
							helpText="OAuth token from rclone config (optional if rclone is pre-configured)"
						/>
						<FormField
							label="App Key"
							id="dropbox-app-key"
							value={dropboxAppKey}
							onChange={setDropboxAppKey}
							helpText="Your Dropbox App Key (optional)"
						/>
						<FormField
							label="App Secret"
							id="dropbox-app-secret"
							value={dropboxAppSecret}
							onChange={setDropboxAppSecret}
							type="password"
							helpText="Your Dropbox App Secret (optional)"
						/>
						<p className="text-xs text-gray-500 dark:text-gray-400 bg-gray-50 p-3 rounded-lg">
							Dropbox backend requires rclone to be installed on the agent. You
							can either pre-configure rclone with `rclone config` or provide
							the OAuth token here.
						</p>
					</>
				);

			default:
				return null;
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Add Repository
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<FormField
							label="Name"
							id="repo-name"
							value={name}
							onChange={setName}
							placeholder="e.g., primary-backup"
							required
						/>
						<div>
							<label
								htmlFor="repo-type"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Type
							</label>
							<select
								id="repo-type"
								value={type}
								onChange={(e) => setType(e.target.value as RepositoryType)}
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
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
						<div className="flex items-start gap-3 pt-4">
							<input
								type="checkbox"
								id="escrow-enabled"
								checked={escrowEnabled}
								onChange={(e) => setEscrowEnabled(e.target.checked)}
								className="mt-1 h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
							/>
							<div>
								<label
									htmlFor="escrow-enabled"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300"
								>
									Enable key escrow
								</label>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Store an encrypted copy of the password server-side for
									recovery by administrators
								</p>
							</div>
						</div>
					</div>
					{testResult && (
						<div
							className={`mt-4 p-3 rounded-lg text-sm ${
								testResult.success
									? 'bg-green-50 text-green-800 border border-green-200'
									: 'bg-red-50 text-red-800 border border-red-200'
							}`}
						>
							<div className="flex items-center gap-2">
								{testResult.success ? (
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="currentColor"
										viewBox="0 0 20 20"
									>
										<path
											fillRule="evenodd"
											d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
											clipRule="evenodd"
										/>
									</svg>
								) : (
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="currentColor"
										viewBox="0 0 20 20"
									>
										<path
											fillRule="evenodd"
											d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
											clipRule="evenodd"
										/>
									</svg>
								)}
								<span>{testResult.message}</span>
							</div>
						</div>
					)}
					{createRepository.isError && (
						<p className="text-sm text-red-600 mt-4">
							Failed to create repository. Please try again.
						</p>
					)}
					<div className="flex justify-between items-center mt-6">
						<button
							type="button"
							onClick={handleTestConnection}
							disabled={testConnection.isPending}
							className="px-4 py-2 text-indigo-600 border border-indigo-300 rounded-lg hover:bg-indigo-50 transition-colors disabled:opacity-50"
						>
							{testConnection.isPending ? 'Testing...' : 'Test Connection'}
						</button>
						<div className="flex gap-3">
							<button
								type="button"
								onClick={() => {
									onClose();
									resetForm();
								}}
								className="px-4 py-2 text-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="submit"
								disabled={createRepository.isPending}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
							>
								{createRepository.isPending ? 'Creating...' : 'Add Repository'}
							</button>
						</div>
					</div>
				</form>
			</div>
		</div>
	);
}

function getVerificationStatusBadge(status?: VerificationStatus) {
	switch (status) {
		case 'passed':
			return { label: 'Verified', className: 'bg-green-100 text-green-800' };
		case 'failed':
			return { label: 'Failed', className: 'bg-red-100 text-red-800' };
		case 'running':
			return { label: 'Running', className: 'bg-blue-100 text-blue-800' };
		case 'pending':
			return { label: 'Pending', className: 'bg-yellow-100 text-yellow-800' };
		default:
			return { label: 'Not verified', className: 'bg-gray-100 text-gray-600' };
	}
}

function formatRelativeTime(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMins / 60);
	const diffDays = Math.floor(diffHours / 24);

	if (diffMins < 1) return 'just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	if (diffDays < 7) return `${diffDays}d ago`;
	return formatDate(dateStr);
}

interface RepositoryCardProps {
	repository: Repository;
	onDelete: (id: string) => void;
	onTest: (id: string) => void;
	onVerify: (id: string, type: VerificationType) => void;
	onRecoverKey: (id: string) => void;
	onClone: (repository: Repository) => void;
	isDeleting: boolean;
	isTesting: boolean;
	isVerifying: boolean;
	isRecovering: boolean;
	testResult?: { success: boolean; message: string };
	isFavorite: boolean;
}

function RepositoryCard({
	repository,
	onDelete,
	onTest,
	onVerify,
	onRecoverKey,
	onClone,
	isDeleting,
	isTesting,
	isVerifying,
	isRecovering,
	testResult,
	isFavorite,
}: RepositoryCardProps) {
	const typeBadge = getRepositoryTypeBadge(repository.type);
	const { data: verificationStatus } = useVerificationStatus(repository.id);

	const lastVerification = verificationStatus?.last_verification;
	const verificationBadge = getVerificationStatusBadge(
		lastVerification?.status,
	);
	const consecutiveFails = verificationStatus?.consecutive_fails ?? 0;

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
			<div className="flex items-start justify-between mb-4">
				<div className="flex items-start gap-2">
					<StarButton
						entityType="repository"
						entityId={repository.id}
						isFavorite={isFavorite}
						size="sm"
					/>
					<div>
						<h3 className="font-semibold text-gray-900">{repository.name}</h3>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							Created {formatDate(repository.created_at)}
						</p>
					</div>
				</div>
				<div className="flex items-center gap-2">
					{repository.escrow_enabled && (
						<span className="px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
							Escrow
						</span>
					)}
					<span
						className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${typeBadge.className}`}
					>
						{typeBadge.label}
					</span>
				</div>
			</div>

			{/* Verification Status */}
			<div className="mb-4 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
				<div className="flex items-center justify-between mb-2">
					<span className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300 dark:text-gray-600">
						Integrity
					</span>
					<span
						className={`px-2 py-0.5 rounded-full text-xs font-medium ${verificationBadge.className}`}
					>
						{verificationBadge.label}
					</span>
				</div>
				{lastVerification && (
					<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
						Last checked: {formatRelativeTime(lastVerification.started_at)}
						{lastVerification.status === 'failed' &&
							lastVerification.error_message && (
								<span
									className="block text-red-600 mt-1 truncate"
									title={lastVerification.error_message}
								>
									{lastVerification.error_message}
								</span>
							)}
					</p>
				)}
				{consecutiveFails > 0 && (
					<p className="text-xs text-red-600 mt-1">
						{consecutiveFails} consecutive failure
						{consecutiveFails > 1 ? 's' : ''}
					</p>
				)}
				{verificationStatus?.next_scheduled_at && (
					<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
						Next: {formatRelativeTime(verificationStatus.next_scheduled_at)}
					</p>
				)}
			</div>

			{testResult && (
				<div
					className={`mb-4 p-3 rounded-lg text-sm ${
						testResult.success
							? 'bg-green-50 text-green-800'
							: 'bg-red-50 text-red-800'
					}`}
				>
					{testResult.message}
				</div>
			)}
			<div className="flex items-center gap-2 flex-wrap">
				<button
					type="button"
					onClick={() => onVerify(repository.id, 'check')}
					disabled={isVerifying}
					className="text-sm text-green-600 hover:text-green-800 font-medium disabled:opacity-50"
				>
					{isVerifying ? 'Verifying...' : 'Verify'}
				</button>
				<span className="text-gray-300 dark:text-gray-600">|</span>
				<button
					type="button"
					onClick={() => onTest(repository.id)}
					disabled={isTesting}
					className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 font-medium disabled:opacity-50"
				>
					{isTesting ? 'Testing...' : 'Test'}
				</button>
				{repository.escrow_enabled && (
					<>
						<span className="text-gray-300 dark:text-gray-600">|</span>
						<button
							type="button"
							onClick={() => onRecoverKey(repository.id)}
							disabled={isRecovering}
							className="text-sm text-amber-600 hover:text-amber-800 font-medium disabled:opacity-50"
						>
							{isRecovering ? 'Recovering...' : 'Recover Key'}
						</button>
					</>
				)}
				<span className="text-gray-300 dark:text-gray-600">|</span>
				<button
					type="button"
					onClick={() => onClone(repository)}
					className="text-sm text-purple-600 hover:text-purple-800 font-medium"
				>
					Clone
				</button>
				<span className="text-gray-300 dark:text-gray-600">|</span>
				<button
					type="button"
					onClick={() => onDelete(repository.id)}
					disabled={isDeleting}
					className="text-sm text-red-600 hover:text-red-800 font-medium disabled:opacity-50"
				>
					Delete
				</button>
			</div>
		</div>
	);
}

interface RecoveredKeyModalProps {
	isOpen: boolean;
	onClose: () => void;
	repositoryName: string;
	password: string;
}

function RecoveredKeyModal({
	isOpen,
	onClose,
	repositoryName,
	password,
}: RecoveredKeyModalProps) {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		await navigator.clipboard.writeText(password);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
				<div className="flex items-center gap-3 mb-4">
					<div className="flex-shrink-0 w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
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
								d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
							/>
						</svg>
					</div>
					<div>
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							Recovered Password
						</h3>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							Password for repository: {repositoryName}
						</p>
					</div>
				</div>

				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4 mb-4">
					<div className="flex items-center justify-between gap-4">
						<code className="text-sm font-mono break-all flex-1">
							{password}
						</code>
						<button
							type="button"
							onClick={handleCopy}
							className="flex-shrink-0 px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors"
						>
							{copied ? 'Copied!' : 'Copy'}
						</button>
					</div>
				</div>

				<div className="flex justify-end">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

interface CloneRepositoryModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (response: CloneRepositoryResponse) => void;
	repository: Repository | null;
	isAdmin: boolean;
}

function CloneRepositoryModal({
	isOpen,
	onClose,
	onSuccess,
	repository,
	isAdmin,
}: CloneRepositoryModalProps) {
	const [name, setName] = useState('');
	const [targetOrgId, setTargetOrgId] = useState<string | undefined>(undefined);

	// Credential fields based on repository type
	// S3
	const [s3AccessKey, setS3AccessKey] = useState('');
	const [s3SecretKey, setS3SecretKey] = useState('');

	// B2
	const [b2AccountId, setB2AccountId] = useState('');
	const [b2AppKey, setB2AppKey] = useState('');

	// SFTP
	const [sftpPassword, setSftpPassword] = useState('');
	const [sftpPrivateKey, setSftpPrivateKey] = useState('');

	// REST
	const [restUsername, setRestUsername] = useState('');
	const [restPassword, setRestPassword] = useState('');

	// Dropbox
	const [dropboxToken, setDropboxToken] = useState('');
	const [dropboxAppKey, setDropboxAppKey] = useState('');
	const [dropboxAppSecret, setDropboxAppSecret] = useState('');

	const { data: organizations } = useOrganizations();
	const cloneRepository = useCloneRepository();

	// Reset form when modal opens with new repository
	useEffect(() => {
		if (isOpen && repository) {
			setName(`${repository.name} (Copy)`);
			setTargetOrgId(undefined);
			// Reset credential fields
			setS3AccessKey('');
			setS3SecretKey('');
			setB2AccountId('');
			setB2AppKey('');
			setSftpPassword('');
			setSftpPrivateKey('');
			setRestUsername('');
			setRestPassword('');
			setDropboxToken('');
			setDropboxAppKey('');
			setDropboxAppSecret('');
		}
	}, [isOpen, repository]);

	const buildCredentials = (): Record<string, unknown> => {
		if (!repository) return {};

		switch (repository.type) {
			case 's3':
				return {
					access_key_id: s3AccessKey,
					secret_access_key: s3SecretKey,
				};
			case 'b2':
				return {
					account_id: b2AccountId,
					application_key: b2AppKey,
				};
			case 'sftp':
				return {
					...(sftpPassword ? { password: sftpPassword } : {}),
					...(sftpPrivateKey ? { private_key: sftpPrivateKey } : {}),
				};
			case 'rest':
				return {
					...(restUsername ? { username: restUsername } : {}),
					...(restPassword ? { password: restPassword } : {}),
				};
			case 'dropbox':
				return {
					...(dropboxToken ? { token: dropboxToken } : {}),
					...(dropboxAppKey ? { app_key: dropboxAppKey } : {}),
					...(dropboxAppSecret ? { app_secret: dropboxAppSecret } : {}),
				};
			default:
				return {};
		}
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!repository) return;

		try {
			const response = await cloneRepository.mutateAsync({
				id: repository.id,
				data: {
					name,
					credentials: buildCredentials(),
					...(targetOrgId ? { target_org_id: targetOrgId } : {}),
				},
			});
			onSuccess(response);
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen || !repository) return null;

	const renderCredentialFields = () => {
		switch (repository.type) {
			case 's3':
				return (
					<>
						<FormField
							label="Access Key ID"
							id="clone-s3-access-key"
							value={s3AccessKey}
							onChange={setS3AccessKey}
							placeholder="AKIAIOSFODNN7EXAMPLE"
							required
						/>
						<FormField
							label="Secret Access Key"
							id="clone-s3-secret-key"
							value={s3SecretKey}
							onChange={setS3SecretKey}
							type="password"
							required
						/>
					</>
				);

			case 'b2':
				return (
					<>
						<FormField
							label="Account ID"
							id="clone-b2-account-id"
							value={b2AccountId}
							onChange={setB2AccountId}
							placeholder="0012345678abcdef"
							required
						/>
						<FormField
							label="Application Key"
							id="clone-b2-app-key"
							value={b2AppKey}
							onChange={setB2AppKey}
							type="password"
							required
						/>
					</>
				);

			case 'sftp':
				return (
					<>
						<FormField
							label="Password"
							id="clone-sftp-password"
							value={sftpPassword}
							onChange={setSftpPassword}
							type="password"
							helpText="Password or private key required"
						/>
						<div>
							<label
								htmlFor="clone-sftp-private-key"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Private Key
							</label>
							<textarea
								id="clone-sftp-private-key"
								value={sftpPrivateKey}
								onChange={(e) => setSftpPrivateKey(e.target.value)}
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-xs"
								rows={4}
							/>
						</div>
					</>
				);

			case 'rest':
				return (
					<>
						<FormField
							label="Username"
							id="clone-rest-username"
							value={restUsername}
							onChange={setRestUsername}
							placeholder="backup"
							helpText="Optional authentication"
						/>
						<FormField
							label="Password"
							id="clone-rest-password"
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
							label="Token"
							id="clone-dropbox-token"
							value={dropboxToken}
							onChange={setDropboxToken}
							type="password"
							helpText="OAuth token from rclone config"
						/>
						<FormField
							label="App Key"
							id="clone-dropbox-app-key"
							value={dropboxAppKey}
							onChange={setDropboxAppKey}
							helpText="Your Dropbox App Key (optional)"
						/>
						<FormField
							label="App Secret"
							id="clone-dropbox-app-secret"
							value={dropboxAppSecret}
							onChange={setDropboxAppSecret}
							type="password"
							helpText="Your Dropbox App Secret (optional)"
						/>
					</>
				);

			default:
				return (
					<p className="text-sm text-gray-500 dark:text-gray-400">
						Local repositories do not require credentials.
					</p>
				);
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					Clone Repository
				</h3>
				<p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
					Create a copy of{' '}
					<span className="font-medium">{repository.name}</span> with new
					credentials.
				</p>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<FormField
							label="Name"
							id="clone-repo-name"
							value={name}
							onChange={setName}
							placeholder="e.g., primary-backup-copy"
							required
						/>

						{isAdmin && organizations && organizations.length > 1 && (
							<div>
								<label
									htmlFor="clone-target-org"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Target Organization
								</label>
								<select
									id="clone-target-org"
									value={targetOrgId ?? ''}
									onChange={(e) => setTargetOrgId(e.target.value || undefined)}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								>
									<option value="">Current organization</option>
									{organizations.map((org) => (
										<option key={org.id} value={org.id}>
											{org.name}
										</option>
									))}
								</select>
								<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
									Admin only: Clone to a different organization
								</p>
							</div>
						)}

						<hr className="my-4" />
						<p className="text-sm font-medium text-gray-700 dark:text-gray-300">
							New Credentials
						</p>
						{renderCredentialFields()}
					</div>

					{cloneRepository.isError && (
						<p className="text-sm text-red-600 mt-4">
							Failed to clone repository. Please check your credentials and try
							again.
						</p>
					)}

					<div className="flex justify-end items-center mt-6 gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={cloneRepository.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{cloneRepository.isPending ? 'Cloning...' : 'Clone Repository'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

export function Repositories() {
	const [searchQuery, setSearchQuery] = useState('');
	const [typeFilter, setTypeFilter] = useState<RepositoryType | 'all'>('all');
	const [showFavoritesOnly, setShowFavoritesOnly] = useState(false);
	const [showAddModal, setShowAddModal] = useState(false);
	const [selectedType, setSelectedType] = useState<RepositoryType | undefined>(
		undefined,
	);
	const [testResults, setTestResults] = useState<
		Record<string, { success: boolean; message: string }>
	>({});
	const [passwordModal, setPasswordModal] = useState<{
		isOpen: boolean;
		repositoryName: string;
		password: string;
	}>({ isOpen: false, repositoryName: '', password: '' });
	const [recoveredKeyModal, setRecoveredKeyModal] = useState<{
		isOpen: boolean;
		repositoryName: string;
		password: string;
	}>({ isOpen: false, repositoryName: '', password: '' });
	const [showImportWizard, setShowImportWizard] = useState(false);
	const [importSuccessModal, setImportSuccessModal] = useState<{
		isOpen: boolean;
		repositoryName: string;
		snapshotsImported: number;
	}>({ isOpen: false, repositoryName: '', snapshotsImported: 0 });
	const [cloneModal, setCloneModal] = useState<{
		isOpen: boolean;
		repository: Repository | null;
	}>({ isOpen: false, repository: null });

	const { data: user } = useMe();
	const { data: repositories, isLoading, isError } = useRepositories();
	const favoriteIds = useFavoriteIds('repository');
	const deleteRepository = useDeleteRepository();
	const testRepository = useTestRepository();
	const triggerVerification = useTriggerVerification();
	const recoverKey = useRecoverRepositoryKey();

	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	const filteredRepositories = repositories?.filter((repo) => {
		const matchesSearch = repo.name
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesType = typeFilter === 'all' || repo.type === typeFilter;
		const matchesFavorites = !showFavoritesOnly || favoriteIds.has(repo.id);
		return matchesSearch && matchesType && matchesFavorites;
	});

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this repository?')) {
			deleteRepository.mutate(id);
		}
	};

	const handleTest = async (id: string) => {
		try {
			const result = await testRepository.mutateAsync(id);
			setTestResults((prev) => ({ ...prev, [id]: result }));
		} catch {
			setTestResults((prev) => ({
				...prev,
				[id]: { success: false, message: 'Connection test failed' },
			}));
		}
	};

	const handleVerify = (id: string, type: VerificationType) => {
		triggerVerification.mutate({ repoId: id, type });
	};

	const handleRecoverKey = async (id: string) => {
		try {
			const result = await recoverKey.mutateAsync(id);
			setRecoveredKeyModal({
				isOpen: true,
				repositoryName: result.repository_name,
				password: result.password,
			});
		} catch {
			alert('Failed to recover key. You may not have permission.');
		}
	};

	const handleClone = (repository: Repository) => {
		setCloneModal({ isOpen: true, repository });
	};

	const handleCloneSuccess = (response: CloneRepositoryResponse) => {
		setCloneModal({ isOpen: false, repository: null });
		setPasswordModal({
			isOpen: true,
			repositoryName: response.repository.name,
			password: response.password,
		});
	};

	const handleTypeClick = (type: RepositoryType) => {
		setSelectedType(type);
		setShowAddModal(true);
	};

	const handleCreateSuccess = (response: CreateRepositoryResponse) => {
		setShowAddModal(false);
		setSelectedType(undefined);
		setPasswordModal({
			isOpen: true,
			repositoryName: response.repository.name,
			password: response.password,
		});
	};

	const handleImportSuccess = (
		repositoryName: string,
		snapshotsImported: number,
	) => {
		setShowImportWizard(false);
		setImportSuccessModal({
			isOpen: true,
			repositoryName,
			snapshotsImported,
		});
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<div className="flex items-center gap-2">
						<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
							Repositories
						</h1>
						<HelpTooltip
							content="Configure storage backends for your backups. Supports S3, B2, SFTP, local storage, and more."
							title="Repository Configuration"
							docsUrl="/docs/configuration#storage-backend-configuration"
							size="md"
						/>
					</div>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Configure backup storage destinations
					</p>
				</div>
				<div className="flex items-center gap-3">
					<button
						type="button"
						onClick={() => setShowImportWizard(true)}
						className="inline-flex items-center gap-2 px-4 py-2 border border-indigo-300 text-indigo-600 rounded-lg hover:bg-indigo-50 transition-colors"
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
							/>
						</svg>
						Import Existing
					</button>
					<button
						type="button"
						onClick={() => {
							setSelectedType(undefined);
							setShowAddModal(true);
						}}
						data-action="create-repository"
						title="Add Repository (N)"
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
						Add Repository
					</button>
				</div>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-6 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search repositories..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={typeFilter}
							onChange={(e) =>
								setTypeFilter(e.target.value as RepositoryType | 'all')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Types</option>
							<option value="local">Local</option>
							<option value="s3">Amazon S3</option>
							<option value="b2">Backblaze B2</option>
							<option value="sftp">SFTP</option>
							<option value="rest">REST Server</option>
							<option value="dropbox">Dropbox</option>
						</select>
						<button
							type="button"
							onClick={() => setShowFavoritesOnly(!showFavoritesOnly)}
							className={`flex items-center gap-2 px-4 py-2 border rounded-lg transition-colors ${
								showFavoritesOnly
									? 'border-yellow-400 bg-yellow-50 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400'
									: 'border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
							}`}
						>
							<svg
								aria-hidden="true"
								className={`w-4 h-4 ${showFavoritesOnly ? 'text-yellow-400 fill-current' : 'text-gray-400'}`}
								fill={showFavoritesOnly ? 'currentColor' : 'none'}
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
								/>
							</svg>
							Favorites
						</button>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500 dark:text-red-400 dark:text-red-400">
						<p className="font-medium">Failed to load repositories</p>
						<p className="text-sm">Please try refreshing the page</p>
					</div>
				) : isLoading ? (
					<div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
						<LoadingCard />
						<LoadingCard />
						<LoadingCard />
					</div>
				) : filteredRepositories && filteredRepositories.length > 0 ? (
					<div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
						{filteredRepositories.map((repo) => (
							<RepositoryCard
								key={repo.id}
								repository={repo}
								onDelete={handleDelete}
								onTest={handleTest}
								onVerify={handleVerify}
								onRecoverKey={handleRecoverKey}
								onClone={handleClone}
								isDeleting={deleteRepository.isPending}
								isTesting={testRepository.isPending}
								isVerifying={triggerVerification.isPending}
								isRecovering={recoverKey.isPending}
								testResult={testResults[repo.id]}
								isFavorite={favoriteIds.has(repo.id)}
							/>
						))}
					</div>
				) : (
					<div className="p-12 text-center text-gray-500 dark:text-gray-400">
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
								d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
							/>
						</svg>
						<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
							No repositories configured
						</h3>
						<p className="mb-4">Add a repository to store your backup data</p>
						<div className="grid grid-cols-2 md:grid-cols-3 gap-4 max-w-2xl mx-auto">
							<button
								type="button"
								onClick={() => handleTypeClick('local')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">
									Local
								</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Filesystem path
								</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('s3')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">S3</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									AWS / MinIO / Wasabi
								</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('b2')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">B2</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Backblaze
								</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('sftp')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">
									SFTP
								</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Remote server
								</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('rest')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">
									REST
								</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Restic REST server
								</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('dropbox')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900 dark:text-white">
									Dropbox
								</p>
								<p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
									Via rclone
								</p>
							</button>
						</div>
					</div>
				)}
			</div>

			<AddRepositoryModal
				isOpen={showAddModal}
				onClose={() => {
					setShowAddModal(false);
					setSelectedType(undefined);
				}}
				onSuccess={handleCreateSuccess}
				initialType={selectedType}
			/>

			<PasswordModal
				isOpen={passwordModal.isOpen}
				onClose={() =>
					setPasswordModal({ isOpen: false, repositoryName: '', password: '' })
				}
				repositoryName={passwordModal.repositoryName}
				password={passwordModal.password}
			/>

			<RecoveredKeyModal
				isOpen={recoveredKeyModal.isOpen}
				onClose={() =>
					setRecoveredKeyModal({
						isOpen: false,
						repositoryName: '',
						password: '',
					})
				}
				repositoryName={recoveredKeyModal.repositoryName}
				password={recoveredKeyModal.password}
			/>

			<ImportRepositoryWizard
				isOpen={showImportWizard}
				onClose={() => setShowImportWizard(false)}
				onSuccess={handleImportSuccess}
			/>

			<CloneRepositoryModal
				isOpen={cloneModal.isOpen}
				onClose={() => setCloneModal({ isOpen: false, repository: null })}
				onSuccess={handleCloneSuccess}
				repository={cloneModal.repository}
				isAdmin={isAdmin}
			/>

			{importSuccessModal.isOpen && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
						<div className="flex items-center gap-3 mb-4">
							<div className="flex-shrink-0 w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
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
							</div>
							<div>
								<h3 className="text-lg font-semibold text-gray-900">
									Repository Imported
								</h3>
								<p className="text-sm text-gray-500">
									Successfully imported existing repository
								</p>
							</div>
						</div>

						<div className="bg-gray-50 rounded-lg p-4 mb-4">
							<p className="text-sm text-gray-700">
								<span className="font-medium">
									{importSuccessModal.repositoryName}
								</span>{' '}
								has been imported with{' '}
								<span className="font-medium">
									{importSuccessModal.snapshotsImported}
								</span>{' '}
								snapshot{importSuccessModal.snapshotsImported !== 1 ? 's' : ''}.
							</p>
						</div>

						<div className="flex justify-end">
							<button
								type="button"
								onClick={() =>
									setImportSuccessModal({
										isOpen: false,
										repositoryName: '',
										snapshotsImported: 0,
									})
								}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Done
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
