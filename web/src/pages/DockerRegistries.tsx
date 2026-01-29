import { useState } from 'react';
import {
	useCreateDockerRegistry,
	useDeleteDockerRegistry,
	useDockerRegistries,
	useDockerRegistryTypes,
	useExpiringCredentials,
	useHealthCheckAllDockerRegistries,
	useHealthCheckDockerRegistry,
	useLoginAllDockerRegistries,
	useLoginDockerRegistry,
	useRotateDockerRegistryCredentials,
	useSetDefaultDockerRegistry,
	useUpdateDockerRegistry,
} from '../hooks/useDockerRegistries';
import type {
	CreateDockerRegistryRequest,
	DockerRegistry,
	DockerRegistryCredentials,
	DockerRegistryType,
	DockerRegistryTypeInfo,
} from '../lib/types';

function getHealthStatusColor(status: string) {
	switch (status) {
		case 'healthy':
			return {
				bg: 'bg-green-100',
				text: 'text-green-700',
				dot: 'bg-green-500',
			};
		case 'unhealthy':
			return { bg: 'bg-red-100', text: 'text-red-700', dot: 'bg-red-500' };
		default:
			return {
				bg: 'bg-gray-100',
				text: 'text-gray-700',
				dot: 'bg-gray-500',
			};
	}
}

function getRegistryTypeLabel(type: DockerRegistryType) {
	const labels: Record<DockerRegistryType, string> = {
		dockerhub: 'Docker Hub',
		gcr: 'Google Container Registry',
		ecr: 'Amazon ECR',
		acr: 'Azure Container Registry',
		ghcr: 'GitHub Container Registry',
		private: 'Private Registry',
	};
	return labels[type] || type;
}

function formatDate(dateString: string | undefined) {
	if (!dateString) return 'Never';
	return new Date(dateString).toLocaleString();
}

interface RegistryCardProps {
	registry: DockerRegistry;
	onEdit: (registry: DockerRegistry) => void;
	onDelete: (id: string) => void;
	onLogin: (id: string) => void;
	onHealthCheck: (id: string) => void;
	onSetDefault: (id: string) => void;
	onRotateCredentials: (registry: DockerRegistry) => void;
	isProcessing: boolean;
}

function RegistryCard({
	registry,
	onEdit,
	onDelete,
	onLogin,
	onHealthCheck,
	onSetDefault,
	onRotateCredentials,
	isProcessing,
}: RegistryCardProps) {
	const healthColor = getHealthStatusColor(registry.health_status);

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-sm transition-shadow">
			<div className="flex items-start justify-between mb-4">
				<div className="flex items-center gap-3">
					<div className="p-2 bg-indigo-100 rounded-lg">
						<svg
							aria-hidden="true"
							className="w-6 h-6 text-indigo-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
							/>
						</svg>
					</div>
					<div>
						<div className="flex items-center gap-2">
							<h3 className="font-semibold text-gray-900">{registry.name}</h3>
							{registry.is_default && (
								<span className="px-2 py-0.5 text-xs font-medium bg-indigo-100 text-indigo-700 rounded-full">
									Default
								</span>
							)}
							{!registry.enabled && (
								<span className="px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-full">
									Disabled
								</span>
							)}
						</div>
						<p className="text-sm text-gray-500">
							{getRegistryTypeLabel(registry.type)}
						</p>
					</div>
				</div>
				<div className="flex items-center gap-2">
					<span
						className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${healthColor.bg} ${healthColor.text}`}
					>
						<span className={`w-2 h-2 ${healthColor.dot} rounded-full`} />
						{registry.health_status}
					</span>
				</div>
			</div>

			<div className="space-y-2 mb-4">
				<div className="flex items-center text-sm text-gray-600">
					<span className="w-24 text-gray-500">URL:</span>
					<span className="truncate">{registry.url}</span>
				</div>
				<div className="flex items-center text-sm text-gray-600">
					<span className="w-24 text-gray-500">Last Check:</span>
					<span>{formatDate(registry.last_health_check)}</span>
				</div>
				{registry.credentials_expires_at && (
					<div className="flex items-center text-sm text-gray-600">
						<span className="w-24 text-gray-500">Expires:</span>
						<span
							className={
								new Date(registry.credentials_expires_at) < new Date()
									? 'text-red-600 font-medium'
									: ''
							}
						>
							{formatDate(registry.credentials_expires_at)}
						</span>
					</div>
				)}
				{registry.last_health_error && (
					<div className="mt-2 p-2 bg-red-50 rounded text-sm text-red-700">
						{registry.last_health_error}
					</div>
				)}
			</div>

			<div className="flex flex-wrap gap-2 pt-4 border-t border-gray-100">
				<button
					type="button"
					onClick={() => onLogin(registry.id)}
					disabled={isProcessing || !registry.enabled}
					className="px-3 py-1.5 text-sm font-medium text-indigo-700 bg-indigo-50 rounded-lg hover:bg-indigo-100 transition-colors disabled:opacity-50"
				>
					Login
				</button>
				<button
					type="button"
					onClick={() => onHealthCheck(registry.id)}
					disabled={isProcessing}
					className="px-3 py-1.5 text-sm font-medium text-green-700 bg-green-50 rounded-lg hover:bg-green-100 transition-colors disabled:opacity-50"
				>
					Health Check
				</button>
				{!registry.is_default && (
					<button
						type="button"
						onClick={() => onSetDefault(registry.id)}
						disabled={isProcessing}
						className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors disabled:opacity-50"
					>
						Set Default
					</button>
				)}
				<button
					type="button"
					onClick={() => onRotateCredentials(registry)}
					disabled={isProcessing}
					className="px-3 py-1.5 text-sm font-medium text-orange-700 bg-orange-50 rounded-lg hover:bg-orange-100 transition-colors disabled:opacity-50"
				>
					Rotate Credentials
				</button>
				<button
					type="button"
					onClick={() => onEdit(registry)}
					className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
				>
					Edit
				</button>
				<button
					type="button"
					onClick={() => onDelete(registry.id)}
					disabled={isProcessing}
					className="px-3 py-1.5 text-sm font-medium text-red-700 bg-red-50 rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
				>
					Delete
				</button>
			</div>
		</div>
	);
}

interface RegistryFormProps {
	types: DockerRegistryTypeInfo[];
	initialData?: DockerRegistry;
	onSubmit: (data: CreateDockerRegistryRequest) => void;
	onCancel: () => void;
	isSubmitting: boolean;
}

function RegistryForm({
	types,
	initialData,
	onSubmit,
	onCancel,
	isSubmitting,
}: RegistryFormProps) {
	const [name, setName] = useState(initialData?.name ?? '');
	const [type, setType] = useState<DockerRegistryType>(
		initialData?.type ?? 'dockerhub',
	);
	const [url, setUrl] = useState(initialData?.url ?? '');
	const [isDefault, setIsDefault] = useState(initialData?.is_default ?? false);
	const [credentials, setCredentials] = useState<DockerRegistryCredentials>({});

	const selectedType = types.find((t) => t.type === type);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		onSubmit({
			name,
			type,
			url: url || selectedType?.default_url || undefined,
			credentials,
			is_default: isDefault,
		});
	};

	const renderCredentialFields = () => {
		if (!selectedType) return null;

		return selectedType.fields.map((field) => {
			const isPassword =
				field.includes('password') ||
				field.includes('secret') ||
				field.includes('token') ||
				field.includes('key_json');
			const label = field
				.split('_')
				.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
				.join(' ');

			if (field === 'gcr_key_json') {
				return (
					<div key={field}>
						<label className="block text-sm font-medium text-gray-700 mb-1">
							{label}
							<textarea
								value={
									(credentials as Record<string, string | undefined>)[
										field
									] ?? ''
								}
								onChange={(e) =>
									setCredentials({
										...credentials,
										[field]: e.target.value,
									})
								}
								rows={6}
								className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
								placeholder="Paste your GCP service account JSON key here..."
							/>
						</label>
					</div>
				);
			}

			return (
				<div key={field}>
					<label className="block text-sm font-medium text-gray-700 mb-1">
						{label}
						<input
							type={isPassword ? 'password' : 'text'}
							value={
								(credentials as Record<string, string | undefined>)[field] ?? ''
							}
							onChange={(e) =>
								setCredentials({
									...credentials,
									[field]: e.target.value,
								})
							}
							className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</label>
				</div>
			);
		});
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<label className="block text-sm font-medium text-gray-700 mb-1">
					Name
					<input
						type="text"
						value={name}
						onChange={(e) => setName(e.target.value)}
						required
						className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="My Registry"
					/>
				</label>
			</div>

			<div>
				<label className="block text-sm font-medium text-gray-700 mb-1">
					Registry Type
					<select
						value={type}
						onChange={(e) => {
							setType(e.target.value as DockerRegistryType);
							setCredentials({});
							const newType = types.find((t) => t.type === e.target.value);
							if (newType?.default_url) {
								setUrl(newType.default_url);
							}
						}}
						className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						{types.map((t) => (
							<option key={t.type} value={t.type}>
								{t.name}
							</option>
						))}
					</select>
				</label>
				{selectedType && (
					<p className="mt-1 text-sm text-gray-500">
						{selectedType.description}
					</p>
				)}
			</div>

			<div>
				<label className="block text-sm font-medium text-gray-700 mb-1">
					URL
					<input
						type="url"
						value={url}
						onChange={(e) => setUrl(e.target.value)}
						className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder={
							selectedType?.default_url || 'https://registry.example.com'
						}
					/>
				</label>
				{selectedType?.default_url && (
					<p className="mt-1 text-sm text-gray-500">
						Leave empty to use: {selectedType.default_url}
					</p>
				)}
			</div>

			<div className="border-t border-gray-200 pt-4">
				<h4 className="font-medium text-gray-900 mb-3">Credentials</h4>
				<div className="space-y-3">{renderCredentialFields()}</div>
			</div>

			<div className="flex items-center gap-2">
				<input
					type="checkbox"
					id="isDefault"
					checked={isDefault}
					onChange={(e) => setIsDefault(e.target.checked)}
					className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
				/>
				<label htmlFor="isDefault" className="text-sm text-gray-700">
					Set as default registry
				</label>
			</div>

			<div className="flex justify-end gap-3 pt-4">
				<button
					type="button"
					onClick={onCancel}
					className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={isSubmitting}
					className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isSubmitting
						? 'Saving...'
						: initialData
							? 'Update Registry'
							: 'Add Registry'}
				</button>
			</div>
		</form>
	);
}

interface RotateCredentialsModalProps {
	registry: DockerRegistry;
	types: DockerRegistryTypeInfo[];
	onSubmit: (
		credentials: DockerRegistryCredentials,
		expiresAt?: string,
	) => void;
	onCancel: () => void;
	isSubmitting: boolean;
}

function RotateCredentialsModal({
	registry,
	types,
	onSubmit,
	onCancel,
	isSubmitting,
}: RotateCredentialsModalProps) {
	const [credentials, setCredentials] = useState<DockerRegistryCredentials>({});
	const [expiresAt, setExpiresAt] = useState('');

	const selectedType = types.find((t) => t.type === registry.type);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		onSubmit(credentials, expiresAt || undefined);
	};

	const renderCredentialFields = () => {
		if (!selectedType) return null;

		return selectedType.fields.map((field) => {
			const isPassword =
				field.includes('password') ||
				field.includes('secret') ||
				field.includes('token') ||
				field.includes('key_json');
			const label = field
				.split('_')
				.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
				.join(' ');

			if (field === 'gcr_key_json') {
				return (
					<div key={field}>
						<label className="block text-sm font-medium text-gray-700 mb-1">
							{label}
							<textarea
								value={
									(credentials as Record<string, string | undefined>)[
										field
									] ?? ''
								}
								onChange={(e) =>
									setCredentials({
										...credentials,
										[field]: e.target.value,
									})
								}
								rows={6}
								className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
								placeholder="Paste your GCP service account JSON key here..."
							/>
						</label>
					</div>
				);
			}

			return (
				<div key={field}>
					<label className="block text-sm font-medium text-gray-700 mb-1">
						{label}
						<input
							type={isPassword ? 'password' : 'text'}
							value={
								(credentials as Record<string, string | undefined>)[field] ?? ''
							}
							onChange={(e) =>
								setCredentials({
									...credentials,
									[field]: e.target.value,
								})
							}
							className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</label>
				</div>
			);
		});
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						Rotate Credentials - {registry.name}
					</h2>
					<form onSubmit={handleSubmit} className="space-y-4">
						<div className="space-y-3">{renderCredentialFields()}</div>

						<div>
							<label className="block text-sm font-medium text-gray-700 mb-1">
								Expires At (Optional)
								<input
									type="datetime-local"
									value={expiresAt}
									onChange={(e) => setExpiresAt(e.target.value)}
									className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</label>
							<p className="mt-1 text-sm text-gray-500">
								Set a reminder for when these credentials expire
							</p>
						</div>

						<div className="flex justify-end gap-3 pt-4">
							<button
								type="button"
								onClick={onCancel}
								className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
							>
								Cancel
							</button>
							<button
								type="submit"
								disabled={isSubmitting}
								className="px-4 py-2 text-sm font-medium text-white bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors disabled:opacity-50"
							>
								{isSubmitting ? 'Rotating...' : 'Rotate Credentials'}
							</button>
						</div>
					</form>
				</div>
			</div>
		</div>
	);
}

export function DockerRegistries() {
	const [showForm, setShowForm] = useState(false);
	const [editingRegistry, setEditingRegistry] = useState<DockerRegistry | null>(
		null,
	);
	const [rotatingRegistry, setRotatingRegistry] =
		useState<DockerRegistry | null>(null);

	const { data: registries, isLoading, isError } = useDockerRegistries();
	const { data: types } = useDockerRegistryTypes();
	const { data: expiringData } = useExpiringCredentials();

	const createRegistry = useCreateDockerRegistry();
	const updateRegistry = useUpdateDockerRegistry();
	const deleteRegistry = useDeleteDockerRegistry();
	const loginRegistry = useLoginDockerRegistry();
	const loginAllRegistries = useLoginAllDockerRegistries();
	const healthCheckRegistry = useHealthCheckDockerRegistry();
	const healthCheckAllRegistries = useHealthCheckAllDockerRegistries();
	const rotateCredentials = useRotateDockerRegistryCredentials();
	const setDefaultRegistry = useSetDefaultDockerRegistry();

	const isProcessing =
		createRegistry.isPending ||
		updateRegistry.isPending ||
		deleteRegistry.isPending ||
		loginRegistry.isPending ||
		loginAllRegistries.isPending ||
		healthCheckRegistry.isPending ||
		healthCheckAllRegistries.isPending ||
		rotateCredentials.isPending ||
		setDefaultRegistry.isPending;

	const handleCreate = (data: CreateDockerRegistryRequest) => {
		createRegistry.mutate(data, {
			onSuccess: () => {
				setShowForm(false);
			},
		});
	};

	const handleUpdate = (data: CreateDockerRegistryRequest) => {
		if (!editingRegistry) return;
		updateRegistry.mutate(
			{
				id: editingRegistry.id,
				data: {
					name: data.name,
					url: data.url,
					enabled: editingRegistry.enabled,
					is_default: data.is_default,
				},
			},
			{
				onSuccess: () => {
					setEditingRegistry(null);
				},
			},
		);
	};

	const handleDelete = (id: string) => {
		if (window.confirm('Are you sure you want to delete this registry?')) {
			deleteRegistry.mutate(id);
		}
	};

	const handleRotateCredentials = (
		credentials: DockerRegistryCredentials,
		expiresAt?: string,
	) => {
		if (!rotatingRegistry) return;
		rotateCredentials.mutate(
			{
				id: rotatingRegistry.id,
				data: {
					credentials,
					expires_at: expiresAt,
				},
			},
			{
				onSuccess: () => {
					setRotatingRegistry(null);
				},
			},
		);
	};

	const expiringRegistries = expiringData?.registries ?? [];

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">
						Docker Registries
					</h1>
					<p className="text-gray-600 mt-1">
						Manage private Docker registry credentials with encryption and
						auto-login
					</p>
				</div>
				<div className="flex items-center gap-3">
					<button
						type="button"
						onClick={() => healthCheckAllRegistries.mutate()}
						disabled={isProcessing}
						className="px-4 py-2 text-sm font-medium text-green-700 bg-green-50 border border-green-200 rounded-lg hover:bg-green-100 transition-colors disabled:opacity-50"
					>
						Health Check All
					</button>
					<button
						type="button"
						onClick={() => loginAllRegistries.mutate()}
						disabled={isProcessing}
						className="px-4 py-2 text-sm font-medium text-indigo-700 bg-indigo-50 border border-indigo-200 rounded-lg hover:bg-indigo-100 transition-colors disabled:opacity-50"
					>
						Login All
					</button>
					<button
						type="button"
						onClick={() => setShowForm(true)}
						className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Add Registry
					</button>
				</div>
			</div>

			{/* Expiring Credentials Warning */}
			{expiringRegistries.length > 0 && (
				<div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
					<div className="flex items-start gap-3">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-orange-600 flex-shrink-0 mt-0.5"
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
							<h3 className="font-medium text-orange-800">
								Credentials Expiring Soon
							</h3>
							<p className="text-sm text-orange-700 mt-1">
								{expiringRegistries.length} registr
								{expiringRegistries.length === 1 ? 'y has' : 'ies have'}{' '}
								credentials expiring within {expiringData?.warning_days} days:
							</p>
							<ul className="mt-2 space-y-1">
								{expiringRegistries.map((r) => (
									<li key={r.id} className="text-sm text-orange-700">
										<span className="font-medium">{r.name}</span> - expires{' '}
										{formatDate(r.credentials_expires_at)}
									</li>
								))}
							</ul>
						</div>
					</div>
				</div>
			)}

			{/* Add/Edit Form */}
			{(showForm || editingRegistry) && types && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						{editingRegistry ? 'Edit Registry' : 'Add New Registry'}
					</h2>
					<RegistryForm
						types={types}
						initialData={editingRegistry ?? undefined}
						onSubmit={editingRegistry ? handleUpdate : handleCreate}
						onCancel={() => {
							setShowForm(false);
							setEditingRegistry(null);
						}}
						isSubmitting={createRegistry.isPending || updateRegistry.isPending}
					/>
				</div>
			)}

			{/* Registries List */}
			{isError ? (
				<div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
					<p className="text-red-700 font-medium">
						Failed to load Docker registries
					</p>
					<p className="text-red-600 text-sm mt-1">
						Please try refreshing the page
					</p>
				</div>
			) : isLoading ? (
				<div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
					{[1, 2, 3].map((i) => (
						<div
							key={i}
							className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse"
						>
							<div className="flex items-start gap-3">
								<div className="w-10 h-10 bg-gray-200 rounded-lg" />
								<div className="flex-1">
									<div className="h-5 w-32 bg-gray-200 rounded mb-2" />
									<div className="h-4 w-24 bg-gray-200 rounded" />
								</div>
							</div>
						</div>
					))}
				</div>
			) : registries && registries.length > 0 ? (
				<div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
					{registries.map((registry) => (
						<RegistryCard
							key={registry.id}
							registry={registry}
							onEdit={setEditingRegistry}
							onDelete={handleDelete}
							onLogin={(id) => loginRegistry.mutate(id)}
							onHealthCheck={(id) => healthCheckRegistry.mutate(id)}
							onSetDefault={(id) => setDefaultRegistry.mutate(id)}
							onRotateCredentials={setRotatingRegistry}
							isProcessing={isProcessing}
						/>
					))}
				</div>
			) : (
				<div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
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
							d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
						/>
					</svg>
					<h3 className="text-lg font-medium text-gray-900 mb-2">
						No Docker Registries
					</h3>
					<p className="text-gray-500 mb-4">
						Add a Docker registry to enable automatic authentication for image
						pulls and pushes.
					</p>
					<button
						type="button"
						onClick={() => setShowForm(true)}
						className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Add Your First Registry
					</button>
				</div>
			)}

			{/* Rotate Credentials Modal */}
			{rotatingRegistry && types && (
				<RotateCredentialsModal
					registry={rotatingRegistry}
					types={types}
					onSubmit={handleRotateCredentials}
					onCancel={() => setRotatingRegistry(null)}
					isSubmitting={rotateCredentials.isPending}
				/>
			)}
		</div>
	);
}
