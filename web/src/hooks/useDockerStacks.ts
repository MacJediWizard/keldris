import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dockerStacksApi } from '../lib/api';
import type {
	CreateDockerStackRequest,
	DiscoverDockerStacksRequest,
	RestoreDockerStackRequest,
	TriggerDockerStackBackupRequest,
	UpdateDockerStackRequest,
} from '../lib/types';

export function useDockerStacks(agentId?: string) {
	return useQuery({
		queryKey: ['docker-stacks', { agentId }],
		queryFn: () => dockerStacksApi.list(agentId),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useDockerStack(id: string) {
	return useQuery({
		queryKey: ['docker-stacks', id],
		queryFn: () => dockerStacksApi.get(id),
		enabled: !!id,
	});
}

export function useCreateDockerStack() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDockerStackRequest) =>
			dockerStacksApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-stacks'] });
		},
	});
}

export function useUpdateDockerStack() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateDockerStackRequest }) =>
			dockerStacksApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['docker-stacks'] });
			queryClient.invalidateQueries({ queryKey: ['docker-stacks', id] });
		},
	});
}

export function useDeleteDockerStack() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerStacksApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-stacks'] });
		},
	});
}

export function useTriggerDockerStackBackup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data?: TriggerDockerStackBackupRequest;
		}) => dockerStacksApi.triggerBackup(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['docker-stacks', id] });
			queryClient.invalidateQueries({
				queryKey: ['docker-stack-backups', { stackId: id }],
			});
		},
	});
}

export function useDockerStackBackups(stackId: string) {
	return useQuery({
		queryKey: ['docker-stack-backups', { stackId }],
		queryFn: () => dockerStacksApi.listBackups(stackId),
		enabled: !!stackId,
		staleTime: 30 * 1000,
	});
}

export function useDockerStackBackup(id: string) {
	return useQuery({
		queryKey: ['docker-stack-backups', id],
		queryFn: () => dockerStacksApi.getBackup(id),
		enabled: !!id,
	});
}

export function useDeleteDockerStackBackup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerStacksApi.deleteBackup(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-stack-backups'] });
		},
	});
}

export function useRestoreDockerStack() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			backupId,
			data,
		}: {
			backupId: string;
			data: RestoreDockerStackRequest;
		}) => dockerStacksApi.restoreBackup(backupId, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-stack-restores'] });
		},
	});
}

export function useDockerStackRestore(id: string) {
	return useQuery({
		queryKey: ['docker-stack-restores', id],
		queryFn: () => dockerStacksApi.getRestore(id),
		enabled: !!id,
		refetchInterval: (query) => {
			// Poll every 2 seconds while restore is in progress
			const status = query.state.data?.status;
			if (status === 'pending' || status === 'running') {
				return 2000;
			}
			return false;
		},
	});
}

export function useDiscoverDockerStacks() {
	return useMutation({
		mutationFn: (data: DiscoverDockerStacksRequest) =>
			dockerStacksApi.discoverStacks(data),
	});
}
