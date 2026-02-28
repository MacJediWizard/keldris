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

interface AuthStatus {
	oidc_enabled: boolean;
	password_enabled: boolean;
}

export function useAuthStatus() {
	return useQuery({
		queryKey: ['auth', 'status'],
		queryFn: async (): Promise<AuthStatus> => {
			const response = await fetch('/auth/status', {
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
			});
			if (!response.ok) {
				throw new Error('Failed to fetch auth status');
			}
			return response.json();
		},
		staleTime: 5 * 60 * 1000,
	});
}

interface PasswordLoginRequest {
	email: string;
	password: string;
}

export function usePasswordLogin() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async (data: PasswordLoginRequest) => {
			const response = await fetch('/auth/login/password', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data),
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Login failed' }));
				throw new Error(errorData.error || errorData.message || 'Login failed');
			}
			return response.json();
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
		},
	});
}
