import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentsApi, schedulesApi } from '../lib/api';
import type { AgentLogFilter, CreateAgentRequest } from '../lib/types';

export function useAgents() {
	return useQuery({
		queryKey: ['agents'],
		queryFn: agentsApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAgent(id: string) {
	return useQuery({
		queryKey: ['agents', id],
		queryFn: () => agentsApi.get(id),
		enabled: !!id,
	});
}

export function useCreateAgent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateAgentRequest) => agentsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}

export function useDeleteAgent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => agentsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}

export function useRotateAgentApiKey() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => agentsApi.rotateApiKey(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}

export function useRevokeAgentApiKey() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => agentsApi.revokeApiKey(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}

export function useAgentStats(id: string) {
	return useQuery({
		queryKey: ['agents', id, 'stats'],
		queryFn: () => agentsApi.getStats(id),
		enabled: !!id,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAgentBackups(id: string) {
	return useQuery({
		queryKey: ['agents', id, 'backups'],
		queryFn: () => agentsApi.getBackups(id),
		enabled: !!id,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAgentSchedules(id: string) {
	return useQuery({
		queryKey: ['agents', id, 'schedules'],
		queryFn: () => agentsApi.getSchedules(id),
		enabled: !!id,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useRunSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => schedulesApi.run(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
			queryClient.invalidateQueries({ queryKey: ['backups'] });
		},
	});
}

export function useAgentHealthHistory(id: string, limit = 100) {
	return useQuery({
		queryKey: ['agents', id, 'health-history', limit],
		queryFn: () => agentsApi.getHealthHistory(id, limit),
		enabled: !!id,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useFleetHealth() {
	return useQuery({
		queryKey: ['agents', 'fleet-health'],
		queryFn: () => agentsApi.getFleetHealth(),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAgentLogs(id: string, filter?: AgentLogFilter) {
	return useQuery({
		queryKey: ['agents', id, 'logs', filter],
		queryFn: () => agentsApi.getLogs(id, filter),
		enabled: !!id,
		staleTime: 5 * 1000, // 5 seconds for more real-time updates
		refetchInterval: 10 * 1000, // Poll every 10 seconds for new logs
	});
}
