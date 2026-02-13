import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { brandingApi } from '../lib/api';
import type { UpdateBrandingRequest } from '../lib/types';

export function useBranding() {
	return useQuery({
		queryKey: ['branding'],
		queryFn: () => brandingApi.get(),
		staleTime: 60 * 1000,
		retry: (failureCount, error) => {
			// Don't retry on 402 (feature not available)
			if (error instanceof Error && 'status' in error) {
				const status = (error as { status: number }).status;
				if (status === 402) return false;
			}
			return failureCount < 3;
		},
	});
}

export function useUpdateBranding() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateBrandingRequest) => brandingApi.update(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['branding'] });
		},
	});
}

export function useResetBranding() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => brandingApi.reset(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['branding'] });
		},
	});
}
