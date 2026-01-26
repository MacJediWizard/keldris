import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { passwordApi, passwordPoliciesApi } from '../lib/api';
import type {
	ChangePasswordRequest,
	UpdatePasswordPolicyRequest,
} from '../lib/types';

export function usePasswordPolicy() {
	return useQuery({
		queryKey: ['passwordPolicy'],
		queryFn: () => passwordPoliciesApi.get(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function usePasswordRequirements() {
	return useQuery({
		queryKey: ['passwordPolicy', 'requirements'],
		queryFn: () => passwordPoliciesApi.getRequirements(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useUpdatePasswordPolicy() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdatePasswordPolicyRequest) =>
			passwordPoliciesApi.update(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['passwordPolicy'] });
		},
	});
}

export function useValidatePassword() {
	return useMutation({
		mutationFn: (password: string) =>
			passwordPoliciesApi.validatePassword(password),
	});
}

export function useChangePassword() {
	return useMutation({
		mutationFn: (data: ChangePasswordRequest) =>
			passwordApi.changePassword(data),
	});
}

export function usePasswordExpiration() {
	return useQuery({
		queryKey: ['passwordExpiration'],
		queryFn: () => passwordApi.getExpiration(),
		staleTime: 5 * 60 * 1000, // 5 minutes
		refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
	});
}
