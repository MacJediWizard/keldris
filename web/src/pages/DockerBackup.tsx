import { useState } from 'react';
import { Button } from '../components/ui/Button';
import { ErrorMessage } from '../components/ui/ErrorMessage';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useAgents } from '../hooks/useAgents';
import {
	useDockerContainers,
	useDockerDaemonStatus,
	useDockerVolumes,
	useTriggerDockerBackup,
} from '../hooks/useDockerBackup';
import { useLocale } from '../hooks/useLocale';
import type { DockerContainer, DockerVolume } from '../lib/types';

function ContainerRow({
	container,
	selected,
	onToggle,
}: {
	container: DockerContainer;
	selected: boolean;
	onToggle: () => void;
}) {
	return (
		<tr className="border-b border-gray-100">
			<td className="px-4 py-3">
				<input
					type="checkbox"
					checked={selected}
					onChange={onToggle}
					className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
				/>
			</td>
			<td className="px-4 py-3 text-sm font-medium text-gray-900">
				{container.name}
			</td>
			<td className="px-4 py-3 text-sm text-gray-500">{container.image}</td>
			<td className="px-4 py-3">
				<span
					className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
						container.state === 'running'
							? 'bg-green-50 text-green-700'
							: 'bg-gray-100 text-gray-600'
					}`}
				>
					{container.state}
				</span>
			</td>
		</tr>
	);
}

function VolumeRow({
	volume,
	selected,
	onToggle,
	formatBytes,
}: {
	volume: DockerVolume;
	selected: boolean;
	onToggle: () => void;
	formatBytes: (bytes: number) => string;
}) {
	return (
		<tr className="border-b border-gray-100">
			<td className="px-4 py-3">
				<input
					type="checkbox"
					checked={selected}
					onChange={onToggle}
					className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
				/>
			</td>
			<td className="px-4 py-3 text-sm font-medium text-gray-900">
				{volume.name}
			</td>
			<td className="px-4 py-3 text-sm text-gray-500">{volume.driver}</td>
			<td className="px-4 py-3 text-sm text-gray-500">
				{formatBytes(volume.size_bytes)}
			</td>
		</tr>
	);
}

export function DockerBackup() {
	const { t, formatBytes } = useLocale();
	const { data: agents, isLoading: agentsLoading } = useAgents();
	const [selectedAgentId, setSelectedAgentId] = useState('');
	const [selectedContainerIds, setSelectedContainerIds] = useState<Set<string>>(
		new Set(),
	);
	const [selectedVolumeNames, setSelectedVolumeNames] = useState<Set<string>>(
		new Set(),
	);
	const [repositoryId, setRepositoryId] = useState('');

	const {
		data: daemonStatus,
		isLoading: statusLoading,
		error: statusError,
	} = useDockerDaemonStatus(selectedAgentId);
	const {
		data: containers,
		isLoading: containersLoading,
		error: containersError,
	} = useDockerContainers(selectedAgentId);
	const {
		data: volumes,
		isLoading: volumesLoading,
		error: volumesError,
	} = useDockerVolumes(selectedAgentId);
	const triggerBackup = useTriggerDockerBackup();

	const handleAgentChange = (agentId: string) => {
		setSelectedAgentId(agentId);
		setSelectedContainerIds(new Set());
		setSelectedVolumeNames(new Set());
	};

	const toggleContainer = (id: string) => {
		setSelectedContainerIds((prev) => {
			const next = new Set(prev);
			if (next.has(id)) {
				next.delete(id);
			} else {
				next.add(id);
			}
			return next;
		});
	};

	const toggleVolume = (name: string) => {
		setSelectedVolumeNames((prev) => {
			const next = new Set(prev);
			if (next.has(name)) {
				next.delete(name);
			} else {
				next.add(name);
			}
			return next;
		});
	};

	const handleBackup = () => {
		if (!selectedAgentId || !repositoryId) return;
		triggerBackup.mutate({
			agent_id: selectedAgentId,
			repository_id: repositoryId,
			container_ids: Array.from(selectedContainerIds),
			volume_names: Array.from(selectedVolumeNames),
		});
	};

	const hasSelection =
		selectedContainerIds.size > 0 || selectedVolumeNames.size > 0;

	return (
		<div className="p-6">
			<div className="mb-6">
				<h1 className="text-2xl font-bold text-gray-900">
					{t('dockerBackup.title')}
				</h1>
				<p className="mt-1 text-sm text-gray-500">
					{t('dockerBackup.subtitle')}
				</p>
			</div>

			{/* Agent selector */}
			<div className="mb-6 grid gap-4 sm:grid-cols-2">
				<div>
					<label
						htmlFor="agent-select"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						{t('dockerBackup.selectAgent')}
					</label>
					<select
						id="agent-select"
						value={selectedAgentId}
						onChange={(e) => handleAgentChange(e.target.value)}
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
						disabled={agentsLoading}
					>
						<option value="">{t('dockerBackup.chooseAgent')}</option>
						{agents?.map((agent) => (
							<option key={agent.id} value={agent.id}>
								{agent.hostname}
							</option>
						))}
					</select>
				</div>
				<div>
					<label
						htmlFor="repo-select"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						{t('dockerBackup.repositoryId')}
					</label>
					<input
						id="repo-select"
						type="text"
						value={repositoryId}
						onChange={(e) => setRepositoryId(e.target.value)}
						placeholder={t('dockerBackup.repositoryIdPlaceholder')}
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
					/>
				</div>
			</div>

			{/* Daemon status */}
			{selectedAgentId && (
				<div className="mb-6">
					{statusLoading && <LoadingSpinner />}
					{statusError && (
						<ErrorMessage message={t('dockerBackup.statusError')} />
					)}
					{daemonStatus && (
						<div
							className={`rounded-lg border p-4 ${daemonStatus.available ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'}`}
						>
							<div className="flex items-center gap-2">
								<span
									className={`inline-block h-2 w-2 rounded-full ${daemonStatus.available ? 'bg-green-500' : 'bg-red-500'}`}
								/>
								<span className="text-sm font-medium">
									{daemonStatus.available
										? t('dockerBackup.daemonRunning')
										: t('dockerBackup.daemonStopped')}
								</span>
								{daemonStatus.version && (
									<span className="text-sm text-gray-500">
										v{daemonStatus.version}
									</span>
								)}
							</div>
						</div>
					)}
				</div>
			)}

			{/* Containers */}
			{selectedAgentId && daemonStatus?.available && (
				<div className="mb-6">
					<h2 className="mb-3 text-lg font-semibold text-gray-900">
						{t('dockerBackup.containers')}
					</h2>
					{containersLoading && <LoadingSpinner />}
					{containersError && (
						<ErrorMessage message={t('dockerBackup.containersError')} />
					)}
					{containers && containers.length > 0 ? (
						<div className="overflow-hidden rounded-lg border border-gray-200">
							<table className="w-full">
								<thead className="bg-gray-50">
									<tr>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('dockerBackup.select')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('common.name')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('dockerBackup.image')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('common.status')}
										</th>
									</tr>
								</thead>
								<tbody>
									{containers.map((container) => (
										<ContainerRow
											key={container.id}
											container={container}
											selected={selectedContainerIds.has(container.id)}
											onToggle={() => toggleContainer(container.id)}
										/>
									))}
								</tbody>
							</table>
						</div>
					) : (
						containers && (
							<p className="text-sm text-gray-500">
								{t('dockerBackup.noContainers')}
							</p>
						)
					)}
				</div>
			)}

			{/* Volumes */}
			{selectedAgentId && daemonStatus?.available && (
				<div className="mb-6">
					<h2 className="mb-3 text-lg font-semibold text-gray-900">
						{t('dockerBackup.volumes')}
					</h2>
					{volumesLoading && <LoadingSpinner />}
					{volumesError && (
						<ErrorMessage message={t('dockerBackup.volumesError')} />
					)}
					{volumes && volumes.length > 0 ? (
						<div className="overflow-hidden rounded-lg border border-gray-200">
							<table className="w-full">
								<thead className="bg-gray-50">
									<tr>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('dockerBackup.select')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('common.name')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('dockerBackup.driver')}
										</th>
										<th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
											{t('dockerBackup.size')}
										</th>
									</tr>
								</thead>
								<tbody>
									{volumes.map((volume) => (
										<VolumeRow
											key={volume.name}
											volume={volume}
											selected={selectedVolumeNames.has(volume.name)}
											onToggle={() => toggleVolume(volume.name)}
											formatBytes={formatBytes}
										/>
									))}
								</tbody>
							</table>
						</div>
					) : (
						volumes && (
							<p className="text-sm text-gray-500">
								{t('dockerBackup.noVolumes')}
							</p>
						)
					)}
				</div>
			)}

			{/* Backup button */}
			{selectedAgentId && daemonStatus?.available && (
				<div className="flex items-center gap-4">
					<Button
						onClick={handleBackup}
						disabled={!hasSelection || !repositoryId || triggerBackup.isPending}
						loading={triggerBackup.isPending}
					>
						{t('dockerBackup.triggerBackup')}
					</Button>
					{triggerBackup.isSuccess && (
						<span className="text-sm text-green-600">
							{t('dockerBackup.backupQueued')}
						</span>
					)}
					{triggerBackup.isError && (
						<span className="text-sm text-red-600">
							{t('dockerBackup.backupError')}
						</span>
					)}
				</div>
			)}
		</div>
	);
}

export default DockerBackup;
