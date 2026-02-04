import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { brandingApi } from '../lib/api';
import type { UpdateBrandingSettingsRequest } from '../lib/types';

// Get branding settings (requires auth)
export function useBrandingSettings() {
	return useQuery({
		queryKey: ['branding'],
		queryFn: () => brandingApi.get(),
		staleTime: 60 * 1000, // 1 minute
	});
}

// Update branding settings
export function useUpdateBrandingSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateBrandingSettingsRequest) =>
			brandingApi.update(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['branding'] });
		},
	});
}

// Get public branding settings (no auth required, for login page)
export function usePublicBrandingSettings(orgSlug: string) {
	return useQuery({
		queryKey: ['branding', 'public', orgSlug],
		queryFn: () => brandingApi.getPublic(orgSlug),
		staleTime: 5 * 60 * 1000, // 5 minutes
		enabled: !!orgSlug,
	});
}
