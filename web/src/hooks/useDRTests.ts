import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { drTestsApi } from '../lib/api';
import type { RunDRTestRequest } from '../lib/types';

export function useDRTests(params?: { runbook_id?: string; status?: string }) {
	return useQuery({
		queryKey: ['dr-tests', params],
		queryFn: () => drTestsApi.list(params),
		staleTime: 30 * 1000,
	});
}

export function useDRTest(id: string) {
	return useQuery({
		queryKey: ['dr-tests', id],
		queryFn: () => drTestsApi.get(id),
		enabled: !!id,
	});
}

export function useRunDRTest() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: RunDRTestRequest) => drTestsApi.run(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['dr-tests'] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useCancelDRTest() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, notes }: { id: string; notes?: string }) =>
			drTestsApi.cancel(id, notes),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['dr-tests'] });
			queryClient.invalidateQueries({ queryKey: ['dr-tests', id] });
		},
	});
}
