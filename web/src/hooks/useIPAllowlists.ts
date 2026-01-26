import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ipAllowlistsApi } from '../lib/api';
import type {
	CreateIPAllowlistRequest,
	UpdateIPAllowlistRequest,
	UpdateIPAllowlistSettingsRequest,
} from '../lib/types';

export function useIPAllowlists() {
	return useQuery({
		queryKey: ['ip-allowlists'],
		queryFn: () => ipAllowlistsApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useIPAllowlist(id: string) {
	return useQuery({
		queryKey: ['ip-allowlists', id],
		queryFn: () => ipAllowlistsApi.get(id),
		enabled: !!id,
	});
}

export function useCreateIPAllowlist() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateIPAllowlistRequest) =>
			ipAllowlistsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['ip-allowlists'] });
		},
	});
}

export function useUpdateIPAllowlist() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateIPAllowlistRequest }) =>
			ipAllowlistsApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['ip-allowlists'] });
			queryClient.invalidateQueries({ queryKey: ['ip-allowlists', id] });
		},
	});
}

export function useDeleteIPAllowlist() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => ipAllowlistsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['ip-allowlists'] });
		},
	});
}

export function useIPAllowlistSettings() {
	return useQuery({
		queryKey: ['ip-allowlist-settings'],
		queryFn: () => ipAllowlistsApi.getSettings(),
		staleTime: 30 * 1000,
	});
}

export function useUpdateIPAllowlistSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateIPAllowlistSettingsRequest) =>
			ipAllowlistsApi.updateSettings(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['ip-allowlist-settings'] });
		},
	});
}

export function useIPBlockedAttempts(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['ip-blocked-attempts', limit, offset],
		queryFn: () => ipAllowlistsApi.listBlockedAttempts(limit, offset),
		staleTime: 30 * 1000,
	});
}
