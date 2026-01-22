import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { authApi } from '../lib/api';
import type { UpdateUserPreferencesRequest } from '../lib/types';

export function useMe() {
	return useQuery({
		queryKey: ['auth', 'me'],
		queryFn: authApi.me,
		retry: false,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useLogout() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: authApi.logout,
		onSuccess: () => {
			queryClient.clear();
			window.location.href = authApi.getLoginUrl();
		},
	});
}

export function useUpdatePreferences() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateUserPreferencesRequest) =>
			authApi.updatePreferences(data),
		onSuccess: (user) => {
			queryClient.setQueryData(['auth', 'me'], user);
		},
	});
}
