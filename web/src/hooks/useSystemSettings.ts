import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { orgSettingsApi } from '../lib/api';
import type {
	TestSMTPRequest,
	UpdateOIDCSettingsRequest,
	UpdateSMTPSettingsRequest,
	UpdateSecuritySettingsRequest,
	UpdateStorageDefaultsRequest,
} from '../lib/types';

// Get all system settings
export function useSystemSettings() {
	return useQuery({
		queryKey: ['system-settings'],
		queryFn: () => orgSettingsApi.getAll(),
		staleTime: 60 * 1000, // 1 minute
	});
}

// SMTP Settings
export function useSMTPSettings() {
	return useQuery({
		queryKey: ['system-settings', 'smtp'],
		queryFn: () => orgSettingsApi.getSMTP(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateSMTPSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateSMTPSettingsRequest) =>
			orgSettingsApi.updateSMTP(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

export function useTestSMTP() {
	return useMutation({
		mutationFn: (data: TestSMTPRequest) => orgSettingsApi.testSMTP(data),
	});
}

// OIDC Settings
export function useOIDCSettings() {
	return useQuery({
		queryKey: ['system-settings', 'oidc'],
		queryFn: () => orgSettingsApi.getOIDC(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateOIDCSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateOIDCSettingsRequest) =>
			orgSettingsApi.updateOIDC(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

export function useTestOIDC() {
	return useMutation({
		mutationFn: () => orgSettingsApi.testOIDC(),
	});
}

// Storage Default Settings
export function useStorageDefaultSettings() {
	return useQuery({
		queryKey: ['system-settings', 'storage'],
		queryFn: () => orgSettingsApi.getStorageDefaults(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateStorageDefaultSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateStorageDefaultsRequest) =>
			orgSettingsApi.updateStorageDefaults(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

// Security Settings
export function useSecuritySettings() {
	return useQuery({
		queryKey: ['system-settings', 'security'],
		queryFn: () => orgSettingsApi.getSecurity(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateSecuritySettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateSecuritySettingsRequest) =>
			orgSettingsApi.updateSecurity(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

// Settings Audit Log
export function useSettingsAuditLog(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['system-settings', 'audit-log', limit, offset],
		queryFn: () => orgSettingsApi.getAuditLog(limit, offset),
		staleTime: 30 * 1000,
	});
}
