import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dockerLogsApi } from '../lib/api';
import type {
	DockerLogBackupStatus,
	DockerLogSettingsUpdate,
} from '../lib/types';

// List all docker log backups
export function useDockerLogBackups(status?: DockerLogBackupStatus) {
	return useQuery({
		queryKey: ['docker-log-backups', status],
		queryFn: () => dockerLogsApi.list(status),
		staleTime: 30 * 1000, // 30 seconds
	});
}

// Get a specific backup
export function useDockerLogBackup(id: string) {
	return useQuery({
		queryKey: ['docker-log-backup', id],
		queryFn: () => dockerLogsApi.get(id),
		enabled: !!id,
	});
}

// View backup contents
export function useDockerLogView(
	id: string,
	offset = 0,
	limit = 1000,
	enabled = true,
) {
	return useQuery({
		queryKey: ['docker-log-view', id, offset, limit],
		queryFn: () => dockerLogsApi.view(id, offset, limit),
		enabled: !!id && enabled,
		staleTime: 60 * 1000, // 1 minute
	});
}

// Download backup
export function useDockerLogDownload() {
	return useMutation({
		mutationFn: ({
			id,
			format,
		}: {
			id: string;
			format: 'json' | 'csv' | 'raw';
		}) => dockerLogsApi.download(id, format),
		onSuccess: (blob, { id, format }) => {
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			const ext = format === 'raw' ? 'log' : format;
			a.download = `docker_logs_${id}_${new Date().toISOString().split('T')[0]}.${ext}`;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}

// Delete a backup
export function useDeleteDockerLogBackup() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => dockerLogsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-log-backups'] });
		},
	});
}

// Get settings for an agent
export function useDockerLogSettings(agentId: string) {
	return useQuery({
		queryKey: ['docker-log-settings', agentId],
		queryFn: () => dockerLogsApi.getSettings(agentId),
		enabled: !!agentId,
	});
}

// Update settings for an agent
export function useUpdateDockerLogSettings() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			agentId,
			settings,
		}: {
			agentId: string;
			settings: DockerLogSettingsUpdate;
		}) => dockerLogsApi.updateSettings(agentId, settings),
		onSuccess: (_, { agentId }) => {
			queryClient.invalidateQueries({
				queryKey: ['docker-log-settings', agentId],
			});
		},
	});
}

// List backups by agent
export function useDockerLogBackupsByAgent(agentId: string) {
	return useQuery({
		queryKey: ['docker-log-backups', 'agent', agentId],
		queryFn: () => dockerLogsApi.listByAgent(agentId),
		enabled: !!agentId,
		staleTime: 30 * 1000,
	});
}

// List backups by container
export function useDockerLogBackupsByContainer(
	agentId: string,
	containerId: string,
) {
	return useQuery({
		queryKey: ['docker-log-backups', 'container', agentId, containerId],
		queryFn: () => dockerLogsApi.listByContainer(agentId, containerId),
		enabled: !!agentId && !!containerId,
		staleTime: 30 * 1000,
	});
}

// Get storage stats for an agent
export function useDockerLogStorageStats(agentId: string) {
	return useQuery({
		queryKey: ['docker-log-stats', agentId],
		queryFn: () => dockerLogsApi.getStorageStats(agentId),
		enabled: !!agentId,
		staleTime: 60 * 1000,
	});
}

// Apply retention policy
export function useApplyDockerLogRetention() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			agentId,
			containerId,
		}: {
			agentId: string;
			containerId?: string;
		}) => dockerLogsApi.applyRetention(agentId, containerId),
		onSuccess: (_, { agentId }) => {
			queryClient.invalidateQueries({ queryKey: ['docker-log-backups'] });
			queryClient.invalidateQueries({
				queryKey: ['docker-log-stats', agentId],
			});
		},
	});
}
