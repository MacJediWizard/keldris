import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { policiesApi } from '../lib/api';
import type {
	ApplyPolicyRequest,
	CreatePolicyRequest,
	UpdatePolicyRequest,
} from '../lib/types';

export function usePolicies() {
	return useQuery({
		queryKey: ['policies'],
		queryFn: () => policiesApi.list(),
		staleTime: 30 * 1000,
	});
}

export function usePolicy(id: string) {
	return useQuery({
		queryKey: ['policies', id],
		queryFn: () => policiesApi.get(id),
		enabled: !!id,
	});
}

export function usePolicySchedules(id: string) {
	return useQuery({
		queryKey: ['policies', id, 'schedules'],
		queryFn: () => policiesApi.listSchedules(id),
		enabled: !!id,
	});
}

export function useCreatePolicy() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreatePolicyRequest) => policiesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['policies'] });
		},
	});
}

export function useUpdatePolicy() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdatePolicyRequest }) =>
			policiesApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['policies'] });
			queryClient.invalidateQueries({ queryKey: ['policies', id] });
		},
	});
}

export function useDeletePolicy() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => policiesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['policies'] });
		},
	});
}

export function useApplyPolicy() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: ApplyPolicyRequest }) =>
			policiesApi.apply(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({ queryKey: ['policies'] });
		},
	});
}
