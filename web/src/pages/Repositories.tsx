import { useEffect, useState } from 'react';
import {
	useCreateRepository,
	useDeleteRepository,
	useRepositories,
	useTestConnection,
	useTestRepository,
} from '../hooks/useRepositories';
import type {
	B2BackendConfig,
	DropboxBackendConfig,
	LocalBackendConfig,
	Repository,
	RepositoryType,
	RestBackendConfig,
	S3BackendConfig,
	SFTPBackendConfig,
	TestRepositoryResponse,
} from '../lib/types';
import { formatDate, getRepositoryTypeBadge } from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
			<div className="flex items-start justify-between mb-4">
				<div className="h-5 w-32 bg-gray-200 rounded" />
				<div className="h-6 w-12 bg-gray-200 rounded-full" />
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

interface AddRepositoryModalProps {
	isOpen: boolean;
	onClose: () => void;
	initialType?: RepositoryType;
}

function AddRepositoryModal({
	isOpen,
	onClose,
	initialType,
}: AddRepositoryModalProps) {
	const [name, setName] = useState('');
	const [type, setType] = useState<RepositoryType>(initialType ?? 'local');

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
			await createRepository.mutateAsync({
				name,
				type,
				config: buildConfig(),
			});
			onClose();
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
							<label htmlFor="s3-use-ssl" className="text-sm text-gray-700">
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
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Private Key
							</label>
							<textarea
								id="sftp-private-key"
								value={sftpPrivateKey}
								onChange={(e) => setSftpPrivateKey(e.target.value)}
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-xs"
								rows={4}
							/>
							<p className="mt-1 text-xs text-gray-500">
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
						<p className="text-xs text-gray-500 bg-gray-50 p-3 rounded-lg">
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
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
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
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Type
							</label>
							<select
								id="repo-type"
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
								className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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

interface RepositoryCardProps {
	repository: Repository;
	onDelete: (id: string) => void;
	onTest: (id: string) => void;
	isDeleting: boolean;
	isTesting: boolean;
	testResult?: { success: boolean; message: string };
}

function RepositoryCard({
	repository,
	onDelete,
	onTest,
	isDeleting,
	isTesting,
	testResult,
}: RepositoryCardProps) {
	const typeBadge = getRepositoryTypeBadge(repository.type);

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-start justify-between mb-4">
				<div>
					<h3 className="font-semibold text-gray-900">{repository.name}</h3>
					<p className="text-sm text-gray-500">
						Created {formatDate(repository.created_at)}
					</p>
				</div>
				<span
					className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${typeBadge.className}`}
				>
					{typeBadge.label}
				</span>
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
			<div className="flex items-center gap-2">
				<button
					type="button"
					onClick={() => onTest(repository.id)}
					disabled={isTesting}
					className="text-sm text-indigo-600 hover:text-indigo-800 font-medium disabled:opacity-50"
				>
					{isTesting ? 'Testing...' : 'Test Connection'}
				</button>
				<span className="text-gray-300">|</span>
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

export function Repositories() {
	const [searchQuery, setSearchQuery] = useState('');
	const [typeFilter, setTypeFilter] = useState<RepositoryType | 'all'>('all');
	const [showAddModal, setShowAddModal] = useState(false);
	const [selectedType, setSelectedType] = useState<RepositoryType | undefined>(
		undefined,
	);
	const [testResults, setTestResults] = useState<
		Record<string, { success: boolean; message: string }>
	>({});

	const { data: repositories, isLoading, isError } = useRepositories();
	const deleteRepository = useDeleteRepository();
	const testRepository = useTestRepository();

	const filteredRepositories = repositories?.filter((repo) => {
		const matchesSearch = repo.name
			.toLowerCase()
			.includes(searchQuery.toLowerCase());
		const matchesType = typeFilter === 'all' || repo.type === typeFilter;
		return matchesSearch && matchesType;
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

	const handleTypeClick = (type: RepositoryType) => {
		setSelectedType(type);
		setShowAddModal(true);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Repositories</h1>
					<p className="text-gray-600 mt-1">
						Configure backup storage destinations
					</p>
				</div>
				<button
					type="button"
					onClick={() => {
						setSelectedType(undefined);
						setShowAddModal(true);
					}}
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

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="p-6 border-b border-gray-200">
					<div className="flex items-center gap-4">
						<input
							type="text"
							placeholder="Search repositories..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<select
							value={typeFilter}
							onChange={(e) =>
								setTypeFilter(e.target.value as RepositoryType | 'all')
							}
							className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Types</option>
							<option value="local">Local</option>
							<option value="s3">Amazon S3</option>
							<option value="b2">Backblaze B2</option>
							<option value="sftp">SFTP</option>
							<option value="rest">REST Server</option>
							<option value="dropbox">Dropbox</option>
						</select>
					</div>
				</div>

				{isError ? (
					<div className="p-12 text-center text-red-500">
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
								isDeleting={deleteRepository.isPending}
								isTesting={testRepository.isPending}
								testResult={testResults[repo.id]}
							/>
						))}
					</div>
				) : (
					<div className="p-12 text-center text-gray-500">
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
						<h3 className="text-lg font-medium text-gray-900 mb-2">
							No repositories configured
						</h3>
						<p className="mb-4">Add a repository to store your backup data</p>
						<div className="grid grid-cols-2 md:grid-cols-3 gap-4 max-w-2xl mx-auto">
							<button
								type="button"
								onClick={() => handleTypeClick('local')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">Local</p>
								<p className="text-xs text-gray-500">Filesystem path</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('s3')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">S3</p>
								<p className="text-xs text-gray-500">AWS / MinIO / Wasabi</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('b2')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">B2</p>
								<p className="text-xs text-gray-500">Backblaze</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('sftp')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">SFTP</p>
								<p className="text-xs text-gray-500">Remote server</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('rest')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">REST</p>
								<p className="text-xs text-gray-500">Restic REST server</p>
							</button>
							<button
								type="button"
								onClick={() => handleTypeClick('dropbox')}
								className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition-colors text-left"
							>
								<p className="font-medium text-gray-900">Dropbox</p>
								<p className="text-xs text-gray-500">Via rclone</p>
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
				initialType={selectedType}
			/>
		</div>
	);
}
