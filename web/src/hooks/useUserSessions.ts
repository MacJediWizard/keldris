import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { userSessionsApi } from '../lib/api';

export function useUserSessions() {
	return useQuery({
		queryKey: ['userSessions'],
		queryFn: () => userSessionsApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useRevokeSession() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => userSessionsApi.revoke(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['userSessions'] });
		},
	});
}

export function useRevokeAllSessions() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => userSessionsApi.revokeAll(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['userSessions'] });
		},
	});
}
