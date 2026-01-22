import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ssoGroupMappingsApi } from '../lib/api';
import type {
	CreateSSOGroupMappingRequest,
	UpdateSSOGroupMappingRequest,
	UpdateSSOSettingsRequest,
} from '../lib/types';

// SSO Group Mapping hooks
export function useSSOGroupMappings(orgId: string) {
	return useQuery({
		queryKey: ['organizations', orgId, 'sso-group-mappings'],
		queryFn: () => ssoGroupMappingsApi.list(orgId),
		enabled: !!orgId,
	});
}

export function useSSOGroupMapping(orgId: string, id: string) {
	return useQuery({
		queryKey: ['organizations', orgId, 'sso-group-mappings', id],
		queryFn: () => ssoGroupMappingsApi.get(orgId, id),
		enabled: !!orgId && !!id,
	});
}

export function useCreateSSOGroupMapping() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			data,
		}: {
			orgId: string;
			data: CreateSSOGroupMappingRequest;
		}) => ssoGroupMappingsApi.create(orgId, data),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'sso-group-mappings'],
			});
		},
	});
}

export function useUpdateSSOGroupMapping() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			id,
			data,
		}: {
			orgId: string;
			id: string;
			data: UpdateSSOGroupMappingRequest;
		}) => ssoGroupMappingsApi.update(orgId, id, data),
		onSuccess: (_, { orgId, id }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'sso-group-mappings'],
			});
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'sso-group-mappings', id],
			});
		},
	});
}

export function useDeleteSSOGroupMapping() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ orgId, id }: { orgId: string; id: string }) =>
			ssoGroupMappingsApi.delete(orgId, id),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'sso-group-mappings'],
			});
		},
	});
}

// SSO Settings hooks
export function useSSOSettings(orgId: string) {
	return useQuery({
		queryKey: ['organizations', orgId, 'sso-settings'],
		queryFn: () => ssoGroupMappingsApi.getSettings(orgId),
		enabled: !!orgId,
	});
}

export function useUpdateSSOSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			data,
		}: {
			orgId: string;
			data: UpdateSSOSettingsRequest;
		}) => ssoGroupMappingsApi.updateSettings(orgId, data),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'sso-settings'],
			});
		},
	});
}

// User SSO Groups hook
export function useUserSSOGroups(userId: string) {
	return useQuery({
		queryKey: ['users', userId, 'sso-groups'],
		queryFn: () => ssoGroupMappingsApi.getUserSSOGroups(userId),
		enabled: !!userId,
	});
}
