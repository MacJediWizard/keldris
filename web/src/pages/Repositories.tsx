import { useState } from 'react';
import {
	useCreateRepository,
	useDeleteRepository,
	useRepositories,
	useTestRepository,
} from '../hooks/useRepositories';
import {
	useTriggerVerification,
	useVerificationStatus,
} from '../hooks/useVerifications';
import type {
	Repository,
	RepositoryType,
	VerificationStatus,
	VerificationType,
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
	const [path, setPath] = useState('');
	const createRepository = useCreateRepository();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createRepository.mutateAsync({
				name,
				type,
				config: { path },
			});
			onClose();
			setName('');
			setPath('');
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Add Repository
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="repo-name"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								Name
							</label>
							<input
								type="text"
								id="repo-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="e.g., primary-backup"
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
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
								<option value="local">Local</option>
								<option value="s3">Amazon S3</option>
								<option value="b2">Backblaze B2</option>
								<option value="sftp">SFTP</option>
								<option value="rest">REST Server</option>
							</select>
						</div>
						<div>
							<label
								htmlFor="repo-path"
								className="block text-sm font-medium text-gray-700 mb-1"
							>
								{type === 'local' ? 'Path' : 'Connection String'}
							</label>
							<input
								type="text"
								id="repo-path"
								value={path}
								onChange={(e) => setPath(e.target.value)}
								placeholder={
									type === 'local' ? '/var/backups' : 's3://bucket/path'
								}
								className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								required
							/>
						</div>
					</div>
					{createRepository.isError && (
						<p className="text-sm text-red-600 mt-4">
							Failed to create repository. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3 mt-6">
						<button
							type="button"
							onClick={onClose}
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
	isDeleting: boolean;
	isTesting: boolean;
	isVerifying: boolean;
	testResult?: { success: boolean; message: string };
}

function RepositoryCard({
	repository,
	onDelete,
	onTest,
	onVerify,
	isDeleting,
	isTesting,
	isVerifying,
	testResult,
}: RepositoryCardProps) {
	const typeBadge = getRepositoryTypeBadge(repository.type);
	const { data: verificationStatus } = useVerificationStatus(repository.id);

	const lastVerification = verificationStatus?.last_verification;
	const verificationBadge = getVerificationStatusBadge(
		lastVerification?.status,
	);
	const consecutiveFails = verificationStatus?.consecutive_fails ?? 0;

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

			{/* Verification Status */}
			<div className="mb-4 p-3 bg-gray-50 rounded-lg">
				<div className="flex items-center justify-between mb-2">
					<span className="text-sm font-medium text-gray-700">Integrity</span>
					<span
						className={`px-2 py-0.5 rounded-full text-xs font-medium ${verificationBadge.className}`}
					>
						{verificationBadge.label}
					</span>
				</div>
				{lastVerification && (
					<p className="text-xs text-gray-500">
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
					<p className="text-xs text-gray-500 mt-1">
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
				<span className="text-gray-300">|</span>
				<button
					type="button"
					onClick={() => onTest(repository.id)}
					disabled={isTesting}
					className="text-sm text-indigo-600 hover:text-indigo-800 font-medium disabled:opacity-50"
				>
					{isTesting ? 'Testing...' : 'Test'}
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
	const triggerVerification = useTriggerVerification();

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

	const handleVerify = (id: string, type: VerificationType) => {
		triggerVerification.mutate({ repoId: id, type });
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
								onVerify={handleVerify}
								isDeleting={deleteRepository.isPending}
								isTesting={testRepository.isPending}
								isVerifying={triggerVerification.isPending}
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
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-2xl mx-auto">
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
								<p className="text-xs text-gray-500">AWS / MinIO</p>
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
