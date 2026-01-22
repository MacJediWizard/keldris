import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentRegistrationApi } from '../lib/api';
import type { CreateRegistrationCodeRequest } from '../lib/types';

export function usePendingRegistrations() {
	return useQuery({
		queryKey: ['agent-registration-codes'],
		queryFn: agentRegistrationApi.listPending,
		staleTime: 30 * 1000, // 30 seconds
		refetchInterval: 30 * 1000, // Auto-refresh every 30 seconds to show expirations
	});
}

export function useCreateRegistrationCode() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateRegistrationCodeRequest) =>
			agentRegistrationApi.createCode(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agent-registration-codes'] });
		},
	});
}

export function useDeleteRegistrationCode() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => agentRegistrationApi.deleteCode(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agent-registration-codes'] });
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}
