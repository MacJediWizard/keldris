import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dockerBackupApi } from '../lib/api';
import type { DockerBackupRequest } from '../lib/types';

export function useDockerContainers(agentId: string) {
	return useQuery({
		queryKey: ['docker', 'containers', agentId],
		queryFn: () => dockerBackupApi.listContainers(agentId),
		enabled: !!agentId,
		staleTime: 30 * 1000,
	});
}

export function useDockerVolumes(agentId: string) {
	return useQuery({
		queryKey: ['docker', 'volumes', agentId],
		queryFn: () => dockerBackupApi.listVolumes(agentId),
		enabled: !!agentId,
		staleTime: 30 * 1000,
	});
}

export function useDockerDaemonStatus(agentId: string) {
	return useQuery({
		queryKey: ['docker', 'status', agentId],
		queryFn: () => dockerBackupApi.getDaemonStatus(agentId),
		enabled: !!agentId,
		staleTime: 30 * 1000,
	});
}

export function useTriggerDockerBackup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: DockerBackupRequest) =>
			dockerBackupApi.triggerBackup(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker'] });
			queryClient.invalidateQueries({ queryKey: ['backups'] });
		},
	});
}
