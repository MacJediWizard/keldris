import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { authApi } from '../lib/api';
import type {
	CustomerLoginRequest,
	CustomerRegisterRequest,
} from '../lib/types';

export function useMe() {
	return useQuery({
		queryKey: ['auth', 'me'],
		queryFn: authApi.me,
		retry: false,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useLogin() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CustomerLoginRequest) => authApi.login(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['auth'] });
		},
	});
}

export function useRegister() {
	return useMutation({
		mutationFn: (data: CustomerRegisterRequest) => authApi.register(data),
	});
}

export function useLogout() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: authApi.logout,
		onSuccess: () => {
			queryClient.clear();
			window.location.href = '/login';
		},
	});
}

export function useChangePassword() {
	return useMutation({
		mutationFn: ({
			currentPassword,
			newPassword,
		}: {
			currentPassword: string;
			newPassword: string;
		}) => authApi.changePassword(currentPassword, newPassword),
	});
}
