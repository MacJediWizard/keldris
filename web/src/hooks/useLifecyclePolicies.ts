import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { lifecyclePoliciesApi } from '../lib/api';
import type {
	CreateLifecyclePolicyRequest,
	LifecycleDryRunRequest,
	UpdateLifecyclePolicyRequest,
} from '../lib/types';

export function useLifecyclePolicies() {
	return useQuery({
		queryKey: ['lifecycle-policies'],
		queryFn: () => lifecyclePoliciesApi.list(),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useLifecyclePolicy(id: string) {
	return useQuery({
		queryKey: ['lifecycle-policies', id],
		queryFn: () => lifecyclePoliciesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateLifecyclePolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateLifecyclePolicyRequest) =>
			lifecyclePoliciesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: ['lifecycle-policies'],
			});
		},
	});
}

export function useUpdateLifecyclePolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateLifecyclePolicyRequest;
		}) => lifecyclePoliciesApi.update(id, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['lifecycle-policies'],
			});
			queryClient.invalidateQueries({
				queryKey: ['lifecycle-policies', variables.id],
			});
		},
	});
}

export function useDeleteLifecyclePolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => lifecyclePoliciesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: ['lifecycle-policies'],
			});
		},
	});
}

export function useLifecycleDryRun() {
	return useMutation({
		mutationFn: (id: string) => lifecyclePoliciesApi.dryRun(id),
	});
}

export function useLifecyclePreview() {
	return useMutation({
		mutationFn: (data: LifecycleDryRunRequest) =>
			lifecyclePoliciesApi.preview(data),
	});
}

export function useLifecycleDeletions(policyId: string, limit?: number) {
	return useQuery({
		queryKey: ['lifecycle-deletions', policyId, limit],
		queryFn: () => lifecyclePoliciesApi.listDeletions(policyId, limit),
		enabled: !!policyId,
	});
}

export function useOrgLifecycleDeletions(limit?: number) {
	return useQuery({
		queryKey: ['lifecycle-deletions', 'org', limit],
		queryFn: () => lifecyclePoliciesApi.listOrgDeletions(limit),
	});
}
