import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dockerRestoresApi } from '../lib/api';
import type {
	CreateDockerRestoreRequest,
	DockerRestorePreviewRequest,
} from '../lib/types';

export interface DockerRestoresFilter {
	agent_id?: string;
	status?: string;
}

export function useDockerRestores(filter?: DockerRestoresFilter) {
	return useQuery({
		queryKey: ['docker-restores', filter],
		queryFn: () => dockerRestoresApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useDockerRestore(id: string) {
	return useQuery({
		queryKey: ['docker-restores', id],
		queryFn: () => dockerRestoresApi.get(id),
		enabled: !!id,
		refetchInterval: (query) => {
			// Refetch every 5 seconds if restore is in progress
			const data = query.state.data;
			if (
				data?.status === 'pending' ||
				data?.status === 'preparing' ||
				data?.status === 'restoring_volumes' ||
				data?.status === 'creating_container' ||
				data?.status === 'starting' ||
				data?.status === 'verifying'
			) {
				return 5000;
			}
			return false;
		},
	});
}

export function useCreateDockerRestore() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDockerRestoreRequest) =>
			dockerRestoresApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-restores'] });
		},
	});
}

export function useDockerRestorePreview() {
	return useMutation({
		mutationFn: (data: DockerRestorePreviewRequest) =>
			dockerRestoresApi.preview(data),
	});
}

export function useDockerRestoreProgress(id: string, enabled = true) {
	return useQuery({
		queryKey: ['docker-restores', id, 'progress'],
		queryFn: () => dockerRestoresApi.getProgress(id),
		enabled: enabled && !!id,
		refetchInterval: 2000, // Refetch every 2 seconds for live progress
		staleTime: 1000,
	});
}

export function useContainersInSnapshot(
	snapshotId: string,
	agentId: string,
	enabled = true,
) {
	return useQuery({
		queryKey: ['docker-restores', 'containers', snapshotId, agentId],
		queryFn: () => dockerRestoresApi.listContainers(snapshotId, agentId),
		enabled: enabled && !!snapshotId && !!agentId,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useVolumesInSnapshot(
	snapshotId: string,
	agentId: string,
	enabled = true,
) {
	return useQuery({
		queryKey: ['docker-restores', 'volumes', snapshotId, agentId],
		queryFn: () => dockerRestoresApi.listVolumes(snapshotId, agentId),
		enabled: enabled && !!snapshotId && !!agentId,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useCancelDockerRestore() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerRestoresApi.cancel(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-restores'] });
		},
	});
}
