import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { licensesApi } from '../lib/api';
import type {
	CreateLicenseKeyRequest,
	UpdateLicenseRequest,
} from '../lib/types';

export function useCurrentLicense() {
	return useQuery({
		queryKey: ['licenses', 'current'],
		queryFn: () => licensesApi.getCurrent(),
		staleTime: 60 * 1000,
	});
}

export function useLicenseWarnings() {
	return useQuery({
		queryKey: ['licenses', 'warnings'],
		queryFn: async () => {
			try {
				return await licensesApi.getWarnings();
			} catch {
				return { warnings: { limits: [] } };
			}
		},
		refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
		staleTime: 60 * 1000,
	});
}

export function useLicenseHistory(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['licenses', 'history', { limit, offset }],
		queryFn: () => licensesApi.getHistory(limit, offset),
		staleTime: 30 * 1000,
	});
}

export function useValidateLicense() {
	return useMutation({
		mutationFn: (key: string) => licensesApi.validate(key),
	});
}

export function useActivateLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateLicenseKeyRequest) => licensesApi.activate(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['licenses'] });
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
		},
	});
}

export function useLicensePurchaseUrl() {
	return useQuery({
		queryKey: ['licenses', 'purchase-url'],
		queryFn: async () => {
			try {
				return await licensesApi.getPurchaseUrl();
			} catch {
				return { url: '' };
			}
		},
		staleTime: 5 * 60 * 1000,
	});
}

// Admin hooks
export function useAdminLicenses(params?: {
	org_id?: string;
	tier?: string;
	status?: string;
	limit?: number;
	offset?: number;
}) {
	return useQuery({
		queryKey: ['admin', 'licenses', params],
		queryFn: () => licensesApi.adminList(params),
		staleTime: 30 * 1000,
	});
}

export function useAdminLicense(id: string) {
	return useQuery({
		queryKey: ['admin', 'licenses', id],
		queryFn: () => licensesApi.adminGet(id),
		enabled: !!id,
	});
}

export function useAdminUpdateLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateLicenseRequest }) =>
			licensesApi.adminUpdate(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'licenses'] });
			queryClient.invalidateQueries({ queryKey: ['admin', 'licenses', id] });
			queryClient.invalidateQueries({ queryKey: ['licenses'] });
		},
	});
}

export function useAdminRevokeLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => licensesApi.adminRevoke(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'licenses'] });
			queryClient.invalidateQueries({ queryKey: ['licenses'] });
		},
	});
}
