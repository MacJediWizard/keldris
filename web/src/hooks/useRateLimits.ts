import { useQuery, useQueryClient } from '@tanstack/react-query';
import { rateLimitsApi } from '../lib/api';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ipBansApi, rateLimitConfigsApi, rateLimitsApi } from '../lib/api';
import type {
	CreateIPBanRequest,
	CreateRateLimitConfigRequest,
	UpdateRateLimitConfigRequest,
} from '../lib/types';

// Dashboard stats hook (from main)
export function useRateLimitDashboard() {
	const queryClient = useQueryClient();

	const { data, error, isLoading } = useQuery({
		queryKey: ['admin', 'rate-limits'],
		queryFn: () => rateLimitsApi.getDashboardStats(),
		refetchInterval: 10000, // Refresh every 10 seconds
		staleTime: 5000,
	});

	const refresh = () => {
		queryClient.invalidateQueries({ queryKey: ['admin', 'rate-limits'] });
	};

	return {
		stats: data,
		isLoading,
		error,
		refresh,
	};
}

// Rate Limit Config CRUD hooks
export function useRateLimitConfigs() {
	return useQuery({
		queryKey: ['rate-limit-configs'],
		queryFn: () => rateLimitConfigsApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useRateLimitConfig(id: string) {
	return useQuery({
		queryKey: ['rate-limit-configs', id],
		queryFn: () => rateLimitConfigsApi.get(id),
		enabled: !!id,
	});
}

export function useRateLimitStats() {
	return useQuery({
		queryKey: ['rate-limit-stats'],
		queryFn: () => rateLimitConfigsApi.getStats(),
		staleTime: 30 * 1000,
		refetchInterval: 60 * 1000, // Refresh every minute
	});
}

export function useBlockedRequests() {
	return useQuery({
		queryKey: ['blocked-requests'],
		queryFn: () => rateLimitConfigsApi.listBlocked(),
		staleTime: 30 * 1000,
	});
}

export function useCreateRateLimitConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateRateLimitConfigRequest) =>
			rateLimitConfigsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rate-limit-configs'] });
		},
	});
}

export function useUpdateRateLimitConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateRateLimitConfigRequest }) =>
			rateLimitConfigsApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['rate-limit-configs'] });
			queryClient.invalidateQueries({ queryKey: ['rate-limit-configs', id] });
		},
	});
}

export function useDeleteRateLimitConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => rateLimitConfigsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rate-limit-configs'] });
		},
	});
}

// IP Bans hooks

export function useIPBans() {
	return useQuery({
		queryKey: ['ip-bans'],
		queryFn: () => ipBansApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useCreateIPBan() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateIPBanRequest) => ipBansApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['ip-bans'] });
			queryClient.invalidateQueries({ queryKey: ['rate-limit-stats'] });
		},
	});
}

export function useDeleteIPBan() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => ipBansApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['ip-bans'] });
		},
	});
}
