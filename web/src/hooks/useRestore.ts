import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { restoresApi } from '../lib/api';
import type { CreateRestoreRequest } from '../lib/types';

export interface RestoresFilter {
	agent_id?: string;
	status?: string;
}

export function useRestores(filter?: RestoresFilter) {
	return useQuery({
		queryKey: ['restores', filter],
		queryFn: () => restoresApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useRestore(id: string) {
	return useQuery({
		queryKey: ['restores', id],
		queryFn: () => restoresApi.get(id),
		enabled: !!id,
		refetchInterval: (query) => {
			// Refetch every 5 seconds if restore is in progress
			const data = query.state.data;
			if (data?.status === 'pending' || data?.status === 'running') {
				return 5000;
			}
			return false;
		},
	});
}

export function useCreateRestore() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateRestoreRequest) => restoresApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['restores'] });
		},
	});
}
