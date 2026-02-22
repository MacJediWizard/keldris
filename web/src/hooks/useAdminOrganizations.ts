import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminOrganizationsApi } from '../lib/api';
import type {
	AdminCreateOrgRequest,
	AdminOrgSettings,
	TransferOwnershipRequest,
} from '../lib/types';

export function useAdminOrganizations(params?: {
	search?: string;
	limit?: number;
	offset?: number;
}) {
	return useQuery({
		queryKey: ['admin', 'organizations', params],
		queryFn: () => adminOrganizationsApi.list(params),
	});
}

export function useAdminOrganization(id: string) {
	return useQuery({
		queryKey: ['admin', 'organizations', id],
		queryFn: () => adminOrganizationsApi.get(id),
		enabled: !!id,
	});
}

export function useAdminOrgUsageStats(id: string) {
	return useQuery({
		queryKey: ['admin', 'organizations', id, 'usage'],
		queryFn: () => adminOrganizationsApi.getUsageStats(id),
		enabled: !!id,
	});
}

export function useAdminCreateOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: AdminCreateOrgRequest) =>
			adminOrganizationsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'organizations'] });
		},
	});
}

export function useAdminUpdateOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: AdminOrgSettings }) =>
			adminOrganizationsApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'organizations'] });
			queryClient.invalidateQueries({
				queryKey: ['admin', 'organizations', id],
			});
		},
	});
}

export function useAdminDeleteOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => adminOrganizationsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'organizations'] });
		},
	});
}

export function useAdminTransferOwnership() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			data,
		}: {
			orgId: string;
			data: TransferOwnershipRequest;
		}) => adminOrganizationsApi.transferOwnership(orgId, data),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({ queryKey: ['admin', 'organizations'] });
			queryClient.invalidateQueries({
				queryKey: ['admin', 'organizations', orgId],
			});
		},
	});
}
