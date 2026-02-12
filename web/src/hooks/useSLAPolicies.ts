import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { slaPoliciesApi } from '../lib/api';
import type {
	CreateSLAPolicyRequest,
	UpdateSLAPolicyRequest,
} from '../lib/types';

export function useSLAPolicies() {
	return useQuery({
		queryKey: ['slaPolicies'],
		queryFn: slaPoliciesApi.list,
		staleTime: 30 * 1000,
	});
}

export function useSLAPolicy(id: string) {
	return useQuery({
		queryKey: ['slaPolicies', id],
		queryFn: () => slaPoliciesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateSLAPolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateSLAPolicyRequest) =>
			slaPoliciesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['slaPolicies'] });
		},
	});
}

export function useUpdateSLAPolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateSLAPolicyRequest }) =>
			slaPoliciesApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['slaPolicies'] });
		},
	});
}

export function useDeleteSLAPolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => slaPoliciesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['slaPolicies'] });
		},
	});
}

export function useSLAPolicyStatus(id: string) {
	return useQuery({
		queryKey: ['slaPolicies', id, 'status'],
		queryFn: () => slaPoliciesApi.getStatus(id),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}

export function useSLAPolicyHistory(id: string) {
	return useQuery({
		queryKey: ['slaPolicies', id, 'history'],
		queryFn: () => slaPoliciesApi.getHistory(id),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}
