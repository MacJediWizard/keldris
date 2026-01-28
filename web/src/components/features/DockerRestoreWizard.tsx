import { useState } from 'react';
import { useAgents } from '../../hooks/useAgents';
import {
	useContainersInSnapshot,
	useCreateDockerRestore,
	useDockerRestorePreview,
	useVolumesInSnapshot,
} from '../../hooks/useDockerRestore';
import { useSnapshots } from '../../hooks/useSnapshots';
import type {
	Agent,
	DockerContainer,
	DockerRestoreConflict,
	DockerRestorePlan,
	DockerRestoreTarget,
	DockerVolume,
	Snapshot,
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

type WizardStep = 'source' | 'selection' | 'preview' | 'options' | 'restoring';

interface DockerRestoreWizardProps {
	isOpen: boolean;
	onClose: () => void;
	onSuccess: (restoreId: string) => void;
	preSelectedSnapshotId?: string;
	preSelectedAgentId?: string;
}

export function DockerRestoreWizard({
	isOpen,
	onClose,
	onSuccess,
	preSelectedSnapshotId,
	preSelectedAgentId,
}: DockerRestoreWizardProps) {
	const [step, setStep] = useState<WizardStep>('source');
	const [error, setError] = useState<string | null>(null);

	// Source selection state
	const [selectedAgentId, setSelectedAgentId] = useState(
		preSelectedAgentId || '',
	);
	const [selectedSnapshotId, setSelectedSnapshotId] = useState(
		preSelectedSnapshotId || '',
	);
	const [selectedRepositoryId, setSelectedRepositoryId] = useState('');

	// Container/Volume selection state
	const [restoreType, setRestoreType] = useState<'container' | 'volume'>(
		'container',
	);
	const [selectedContainer, setSelectedContainer] =
		useState<DockerContainer | null>(null);
	const [selectedVolume, setSelectedVolume] = useState<DockerVolume | null>(
		null,
	);

	// Preview state
	const [preview, setPreview] = useState<DockerRestorePlan | null>(null);

	// Options state
	const [newContainerName, setNewContainerName] = useState('');
	const [newVolumeName, setNewVolumeName] = useState('');
	const [targetType, setTargetType] = useState<'local' | 'remote'>('local');
	const [remoteHost, setRemoteHost] = useState('');
	const [remoteCertPath, setRemoteCertPath] = useState('');
	const [remoteTlsVerify, setRemoteTlsVerify] = useState(true);
	const [overwriteExisting, setOverwriteExisting] = useState(false);
	const [startAfterRestore, setStartAfterRestore] = useState(true);
	const [verifyStart, setVerifyStart] = useState(true);

	// Hooks
	const { data: agents } = useAgents();
	const { data: snapshots } = useSnapshots();
	const { data: containers } = useContainersInSnapshot(
		selectedSnapshotId,
		selectedAgentId,
		!!selectedSnapshotId && !!selectedAgentId,
	);
	const { data: volumes } = useVolumesInSnapshot(
		selectedSnapshotId,
		selectedAgentId,
		!!selectedSnapshotId && !!selectedAgentId,
	);
	const previewMutation = useDockerRestorePreview();
	const createRestore = useCreateDockerRestore();

	const handlePreview = async () => {
		setError(null);
		try {
			const target: DockerRestoreTarget | undefined =
				targetType === 'remote'
					? {
							type: 'remote',
							host: remoteHost,
							cert_path: remoteCertPath || undefined,
							tls_verify: remoteTlsVerify,
						}
					: { type: 'local' };

			const result = await previewMutation.mutateAsync({
				snapshot_id: selectedSnapshotId,
				agent_id: selectedAgentId,
				repository_id: selectedRepositoryId,
				container_name:
					restoreType === 'container' ? selectedContainer?.name : undefined,
				volume_name:
					restoreType === 'volume' ? selectedVolume?.name : undefined,
				target,
			});

			setPreview(result);
			setStep('preview');
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to generate preview',
			);
		}
	};

	const handleRestore = async () => {
		setError(null);
		setStep('restoring');
		try {
			const target: DockerRestoreTarget | undefined =
				targetType === 'remote'
					? {
							type: 'remote',
							host: remoteHost,
							cert_path: remoteCertPath || undefined,
							tls_verify: remoteTlsVerify,
						}
					: { type: 'local' };

			const result = await createRestore.mutateAsync({
				snapshot_id: selectedSnapshotId,
				agent_id: selectedAgentId,
				repository_id: selectedRepositoryId,
				container_name:
					restoreType === 'container' ? selectedContainer?.name : undefined,
				volume_name:
					restoreType === 'volume' ? selectedVolume?.name : undefined,
				new_container_name: newContainerName || undefined,
				new_volume_name: newVolumeName || undefined,
				target,
				overwrite_existing: overwriteExisting,
				start_after_restore: startAfterRestore,
				verify_start: verifyStart,
			});

			onSuccess(result.id);
			resetForm();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to start Docker restore',
			);
			setStep('options');
		}
	};

	const resetForm = () => {
		setStep('source');
		setError(null);
		setSelectedAgentId(preSelectedAgentId || '');
		setSelectedSnapshotId(preSelectedSnapshotId || '');
		setSelectedRepositoryId('');
		setRestoreType('container');
		setSelectedContainer(null);
		setSelectedVolume(null);
		setPreview(null);
		setNewContainerName('');
		setNewVolumeName('');
		setTargetType('local');
		setRemoteHost('');
		setRemoteCertPath('');
		setRemoteTlsVerify(true);
		setOverwriteExisting(false);
		setStartAfterRestore(true);
		setVerifyStart(true);
	};

	const handleClose = () => {
		resetForm();
		onClose();
	};

	if (!isOpen) return null;

	const renderSourceStep = () => (
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
							Docker Container/Volume Restore
						</p>
						<p className="text-sm text-blue-700 mt-1">
							Restore Docker containers and volumes from a backup snapshot.
							Select the source snapshot and agent.
						</p>
					</div>
				</div>
			</div>

			<div>
				<label
					htmlFor="restore-agent"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Agent <span className="text-red-500">*</span>
				</label>
				<select
					id="restore-agent"
					value={selectedAgentId}
					onChange={(e) => {
						setSelectedAgentId(e.target.value);
						setSelectedSnapshotId('');
					}}
					className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				>
					<option value="">Select an agent...</option>
					{agents?.map((agent: Agent) => (
						<option key={agent.id} value={agent.id}>
							{agent.hostname}
						</option>
					))}
				</select>
			</div>

			{selectedAgentId && (
				<div>
					<label
						htmlFor="restore-snapshot"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						Snapshot <span className="text-red-500">*</span>
					</label>
					<select
						id="restore-snapshot"
						value={selectedSnapshotId}
						onChange={(e) => {
							const snapshot = snapshots?.find(
								(s: Snapshot) => s.id === e.target.value,
							);
							setSelectedSnapshotId(e.target.value);
							setSelectedRepositoryId(snapshot?.repository_id || '');
						}}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="">Select a snapshot...</option>
						{snapshots
							?.filter((s: Snapshot) => s.agent_id === selectedAgentId)
							.map((snapshot: Snapshot) => (
								<option key={snapshot.id} value={snapshot.id}>
									{snapshot.short_id} -{' '}
									{new Date(snapshot.time).toLocaleString()}
								</option>
							))}
					</select>
				</div>
			)}
		</div>
	);

	const renderSelectionStep = () => (
		<div className="space-y-4">
			<div>
				<span className="block text-sm font-medium text-gray-700 mb-2">
					What do you want to restore?
				</span>
				<div className="flex gap-4">
					<label className="flex items-center gap-2">
						<input
							type="radio"
							name="restore-type"
							value="container"
							checked={restoreType === 'container'}
							onChange={() => {
								setRestoreType('container');
								setSelectedVolume(null);
							}}
							className="text-indigo-600 focus:ring-indigo-500"
						/>
						<span className="text-sm text-gray-700">
							Container (with volumes)
						</span>
					</label>
					<label className="flex items-center gap-2">
						<input
							type="radio"
							name="restore-type"
							value="volume"
							checked={restoreType === 'volume'}
							onChange={() => {
								setRestoreType('volume');
								setSelectedContainer(null);
							}}
							className="text-indigo-600 focus:ring-indigo-500"
						/>
						<span className="text-sm text-gray-700">Volume only</span>
					</label>
				</div>
			</div>

			{restoreType === 'container' && (
				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Select Container
					</h4>
					{containers && containers.length > 0 ? (
						<div className="border border-gray-200 rounded-lg divide-y divide-gray-200 max-h-64 overflow-y-auto">
							{containers.map((container: DockerContainer) => (
								<label
									key={container.id}
									className={`flex items-center gap-3 p-3 cursor-pointer hover:bg-gray-50 ${
										selectedContainer?.id === container.id ? 'bg-indigo-50' : ''
									}`}
								>
									<input
										type="radio"
										name="container"
										checked={selectedContainer?.id === container.id}
										onChange={() => setSelectedContainer(container)}
										className="text-indigo-600 focus:ring-indigo-500"
									/>
									<div className="flex-1 min-w-0">
										<p className="text-sm font-medium text-gray-900">
											{container.name}
										</p>
										<p className="text-xs text-gray-500 truncate">
											{container.image}
										</p>
										{container.volumes && container.volumes.length > 0 && (
											<p className="text-xs text-gray-400 mt-1">
												{container.volumes.length} volume(s)
											</p>
										)}
									</div>
								</label>
							))}
						</div>
					) : (
						<div className="text-center py-8 bg-gray-50 rounded-lg">
							<p className="text-sm text-gray-500">
								No containers found in this snapshot
							</p>
						</div>
					)}
				</div>
			)}

			{restoreType === 'volume' && (
				<div>
					<h4 className="text-sm font-medium text-gray-700 mb-2">
						Select Volume
					</h4>
					{volumes && volumes.length > 0 ? (
						<div className="border border-gray-200 rounded-lg divide-y divide-gray-200 max-h-64 overflow-y-auto">
							{volumes.map((volume: DockerVolume) => (
								<label
									key={volume.name}
									className={`flex items-center gap-3 p-3 cursor-pointer hover:bg-gray-50 ${
										selectedVolume?.name === volume.name ? 'bg-indigo-50' : ''
									}`}
								>
									<input
										type="radio"
										name="volume"
										checked={selectedVolume?.name === volume.name}
										onChange={() => setSelectedVolume(volume)}
										className="text-indigo-600 focus:ring-indigo-500"
									/>
									<div className="flex-1 min-w-0">
										<p className="text-sm font-medium text-gray-900">
											{volume.name}
										</p>
										<p className="text-xs text-gray-500">
											{volume.driver} - {formatBytes(volume.size_bytes)}
										</p>
									</div>
								</label>
							))}
						</div>
					) : (
						<div className="text-center py-8 bg-gray-50 rounded-lg">
							<p className="text-sm text-gray-500">
								No volumes found in this snapshot
							</p>
						</div>
					)}
				</div>
			)}
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
								Restore Plan Generated
							</p>
							<p className="text-sm text-green-700 mt-1">
								Review the restore plan below before proceeding.
							</p>
						</div>
					</div>
				</div>

				{preview.container && (
					<div className="bg-gray-50 rounded-lg p-4">
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Container
						</h4>
						<p className="text-sm text-gray-900">{preview.container.name}</p>
						<p className="text-xs text-gray-500">{preview.container.image}</p>
					</div>
				)}

				{preview.volumes.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 mb-2">
							Volumes to Restore
						</h4>
						<div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
							{preview.volumes.map((vol) => (
								<div key={vol.name} className="p-3">
									<p className="text-sm font-medium text-gray-900">
										{vol.name}
									</p>
									<p className="text-xs text-gray-500">
										{vol.driver} - {formatBytes(vol.size_bytes)}
									</p>
								</div>
							))}
						</div>
					</div>
				)}

				<div className="grid grid-cols-2 gap-4">
					<div className="bg-gray-50 rounded-lg p-4">
						<p className="text-sm text-gray-500">Total Size</p>
						<p className="text-xl font-bold text-gray-900">
							{formatBytes(preview.total_size_bytes)}
						</p>
					</div>
					<div className="bg-gray-50 rounded-lg p-4">
						<p className="text-sm text-gray-500">Volumes</p>
						<p className="text-xl font-bold text-gray-900">
							{preview.volumes.length}
						</p>
					</div>
				</div>

				{preview.conflicts.length > 0 && (
					<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
						<h4 className="text-sm font-medium text-amber-800 mb-2">
							Conflicts Detected
						</h4>
						<ul className="text-sm text-amber-700 space-y-1">
							{preview.conflicts.map((conflict: DockerRestoreConflict) => (
								<li key={`${conflict.type}-${conflict.name}`}>
									{conflict.type}: {conflict.description}
								</li>
							))}
						</ul>
					</div>
				)}
			</div>
		);
	};

	const renderOptionsStep = () => (
		<div className="space-y-4">
			{restoreType === 'container' && selectedContainer && (
				<FormField
					label="New Container Name"
					id="new-container-name"
					value={newContainerName}
					onChange={setNewContainerName}
					placeholder={selectedContainer.name}
					helpText="Leave empty to use the original name"
				/>
			)}

			{restoreType === 'volume' && selectedVolume && (
				<FormField
					label="New Volume Name"
					id="new-volume-name"
					value={newVolumeName}
					onChange={setNewVolumeName}
					placeholder={selectedVolume.name}
					helpText="Leave empty to use the original name"
				/>
			)}

			<div>
				<span className="block text-sm font-medium text-gray-700 mb-2">
					Restore Target
				</span>
				<div className="flex gap-4">
					<label className="flex items-center gap-2">
						<input
							type="radio"
							name="target-type"
							value="local"
							checked={targetType === 'local'}
							onChange={() => setTargetType('local')}
							className="text-indigo-600 focus:ring-indigo-500"
						/>
						<span className="text-sm text-gray-700">Local Docker Host</span>
					</label>
					<label className="flex items-center gap-2">
						<input
							type="radio"
							name="target-type"
							value="remote"
							checked={targetType === 'remote'}
							onChange={() => setTargetType('remote')}
							className="text-indigo-600 focus:ring-indigo-500"
						/>
						<span className="text-sm text-gray-700">Remote Docker Host</span>
					</label>
				</div>
			</div>

			{targetType === 'remote' && (
				<div className="space-y-4 pl-4 border-l-2 border-indigo-200">
					<FormField
						label="Docker Host URL"
						id="remote-host"
						value={remoteHost}
						onChange={setRemoteHost}
						placeholder="tcp://192.168.1.100:2376"
						required
					/>
					<FormField
						label="TLS Certificate Path"
						id="remote-cert-path"
						value={remoteCertPath}
						onChange={setRemoteCertPath}
						placeholder="/path/to/certs"
						helpText="Path to directory containing ca.pem, cert.pem, and key.pem"
					/>
					<div className="flex items-center gap-2">
						<input
							type="checkbox"
							id="remote-tls-verify"
							checked={remoteTlsVerify}
							onChange={(e) => setRemoteTlsVerify(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						<label
							htmlFor="remote-tls-verify"
							className="text-sm text-gray-700"
						>
							Verify TLS certificate
						</label>
					</div>
				</div>
			)}

			<hr className="my-4" />

			<div className="space-y-3">
				<div className="flex items-center gap-2">
					<input
						type="checkbox"
						id="overwrite-existing"
						checked={overwriteExisting}
						onChange={(e) => setOverwriteExisting(e.target.checked)}
						className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
					/>
					<label htmlFor="overwrite-existing" className="text-sm text-gray-700">
						Overwrite existing containers/volumes
					</label>
				</div>

				{restoreType === 'container' && (
					<>
						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="start-after-restore"
								checked={startAfterRestore}
								onChange={(e) => setStartAfterRestore(e.target.checked)}
								className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
							/>
							<label
								htmlFor="start-after-restore"
								className="text-sm text-gray-700"
							>
								Start container after restore
							</label>
						</div>

						{startAfterRestore && (
							<div className="flex items-center gap-2 pl-6">
								<input
									type="checkbox"
									id="verify-start"
									checked={verifyStart}
									onChange={(e) => setVerifyStart(e.target.checked)}
									className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
								/>
								<label htmlFor="verify-start" className="text-sm text-gray-700">
									Verify container starts successfully
								</label>
							</div>
						)}
					</>
				)}
			</div>
		</div>
	);

	const renderRestoringStep = () => (
		<div className="py-8 text-center">
			<div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4" />
			<p className="text-gray-600">Starting Docker restore...</p>
			<p className="text-sm text-gray-500 mt-2">This may take a moment</p>
		</div>
	);

	const renderStepContent = () => {
		switch (step) {
			case 'source':
				return renderSourceStep();
			case 'selection':
				return renderSelectionStep();
			case 'preview':
				return renderPreviewStep();
			case 'options':
				return renderOptionsStep();
			case 'restoring':
				return renderRestoringStep();
			default:
				return null;
		}
	};

	const getStepTitle = () => {
		switch (step) {
			case 'source':
				return 'Select Source';
			case 'selection':
				return 'Select Container/Volume';
			case 'preview':
				return 'Preview Restore';
			case 'options':
				return 'Configure Options';
			case 'restoring':
				return 'Restoring...';
			default:
				return 'Docker Restore';
		}
	};

	const isNextDisabled = () => {
		if (step === 'source') {
			return !selectedAgentId || !selectedSnapshotId;
		}
		if (step === 'selection') {
			return (
				(restoreType === 'container' && !selectedContainer) ||
				(restoreType === 'volume' && !selectedVolume)
			);
		}
		if (step === 'options' && targetType === 'remote') {
			return !remoteHost;
		}
		return false;
	};

	const handleNext = () => {
		if (step === 'source') {
			setStep('selection');
		} else if (step === 'selection') {
			handlePreview();
		} else if (step === 'preview') {
			setStep('options');
		} else if (step === 'options') {
			handleRestore();
		}
	};

	const handleBack = () => {
		if (step === 'selection') {
			setStep('source');
		} else if (step === 'preview') {
			setStep('selection');
		} else if (step === 'options') {
			setStep('preview');
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900">
						{getStepTitle()}
					</h3>
					{step !== 'restoring' && (
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
				{step !== 'restoring' && (
					<div className="flex items-center gap-2 mb-6">
						{['source', 'selection', 'preview', 'options'].map((s, i) => (
							<div key={s} className="flex items-center">
								<div
									className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
										step === s
											? 'bg-indigo-600 text-white'
											: ['selection', 'preview', 'options'].indexOf(step) > i
												? 'bg-indigo-100 text-indigo-600'
												: 'bg-gray-100 text-gray-400'
									}`}
								>
									{i + 1}
								</div>
								{i < 3 && (
									<div
										className={`w-8 h-0.5 mx-1 ${
											['selection', 'preview', 'options'].indexOf(step) > i
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

				{step !== 'restoring' && (
					<div className="flex justify-between mt-6">
						<button
							type="button"
							onClick={() => {
								if (step === 'source') handleClose();
								else handleBack();
							}}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							{step === 'source' ? 'Cancel' : 'Back'}
						</button>
						<button
							type="button"
							onClick={handleNext}
							disabled={
								isNextDisabled() ||
								previewMutation.isPending ||
								createRestore.isPending
							}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
						>
							{previewMutation.isPending
								? 'Loading...'
								: step === 'options'
									? 'Start Restore'
									: 'Continue'}
						</button>
					</div>
				)}
			</div>
		</div>
	);
}
