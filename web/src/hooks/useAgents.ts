import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentsApi } from '../lib/api';
import type { CreateAgentRequest } from '../lib/types';

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
