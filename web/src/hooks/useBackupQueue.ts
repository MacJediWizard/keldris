import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { backupQueueApi, concurrencyApi } from '../lib/api';
import type { UpdateConcurrencyRequest } from '../lib/types';

export function useBackupQueue() {
	return useQuery({
		queryKey: ['backup-queue'],
		queryFn: () => backupQueueApi.list(),
		staleTime: 10 * 1000,
		refetchInterval: 30 * 1000, // Refresh every 30 seconds
	});
}

export function useBackupQueueSummary() {
	return useQuery({
		queryKey: ['backup-queue', 'summary'],
		queryFn: () => backupQueueApi.getSummary(),
		staleTime: 10 * 1000,
		refetchInterval: 30 * 1000,
	});
}

export function useCancelQueuedBackup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => backupQueueApi.cancel(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['backup-queue'] });
		},
	});
}

export function useOrgConcurrency(orgId: string) {
	return useQuery({
		queryKey: ['concurrency', 'org', orgId],
		queryFn: () => concurrencyApi.getOrgConcurrency(orgId),
		enabled: !!orgId,
		staleTime: 30 * 1000,
	});
}

export function useUpdateOrgConcurrency() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			data,
		}: { orgId: string; data: UpdateConcurrencyRequest }) =>
			concurrencyApi.updateOrgConcurrency(orgId, data),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['concurrency', 'org', orgId],
			});
			queryClient.invalidateQueries({ queryKey: ['backup-queue'] });
		},
	});
}

export function useAgentConcurrency(agentId: string) {
	return useQuery({
		queryKey: ['concurrency', 'agent', agentId],
		queryFn: () => concurrencyApi.getAgentConcurrency(agentId),
		enabled: !!agentId,
		staleTime: 30 * 1000,
	});
}

export function useUpdateAgentConcurrency() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			agentId,
			data,
		}: { agentId: string; data: UpdateConcurrencyRequest }) =>
			concurrencyApi.updateAgentConcurrency(agentId, data),
		onSuccess: (_, { agentId }) => {
			queryClient.invalidateQueries({
				queryKey: ['concurrency', 'agent', agentId],
			});
			queryClient.invalidateQueries({ queryKey: ['backup-queue'] });
		},
	});
}
