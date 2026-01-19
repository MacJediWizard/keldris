import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { authApi } from '../lib/api';

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
