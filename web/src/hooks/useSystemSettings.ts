import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { systemSettingsApi } from '../lib/api';
import type {
	UpdateOIDCSettingsRequest,
	UpdateSecuritySettingsRequest,
	UpdateSMTPSettingsRequest,
	UpdateStorageDefaultsRequest,
	TestSMTPRequest,
} from '../lib/types';

// Get all system settings
export function useSystemSettings() {
	return useQuery({
		queryKey: ['system-settings'],
		queryFn: () => systemSettingsApi.getAll(),
		staleTime: 60 * 1000, // 1 minute
	});
}

// SMTP Settings
export function useSMTPSettings() {
	return useQuery({
		queryKey: ['system-settings', 'smtp'],
		queryFn: () => systemSettingsApi.getSMTP(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateSMTPSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateSMTPSettingsRequest) =>
			systemSettingsApi.updateSMTP(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

export function useTestSMTP() {
	return useMutation({
		mutationFn: (data: TestSMTPRequest) => systemSettingsApi.testSMTP(data),
	});
}

// OIDC Settings
export function useOIDCSettings() {
	return useQuery({
		queryKey: ['system-settings', 'oidc'],
		queryFn: () => systemSettingsApi.getOIDC(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateOIDCSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateOIDCSettingsRequest) =>
			systemSettingsApi.updateOIDC(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

export function useTestOIDC() {
	return useMutation({
		mutationFn: () => systemSettingsApi.testOIDC(),
	});
}

// Storage Default Settings
export function useStorageDefaultSettings() {
	return useQuery({
		queryKey: ['system-settings', 'storage'],
		queryFn: () => systemSettingsApi.getStorageDefaults(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateStorageDefaultSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateStorageDefaultsRequest) =>
			systemSettingsApi.updateStorageDefaults(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

// Security Settings
export function useSecuritySettings() {
	return useQuery({
		queryKey: ['system-settings', 'security'],
		queryFn: () => systemSettingsApi.getSecurity(),
		staleTime: 60 * 1000,
	});
}

export function useUpdateSecuritySettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: UpdateSecuritySettingsRequest) =>
			systemSettingsApi.updateSecurity(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['system-settings'] });
		},
	});
}

// Settings Audit Log
export function useSettingsAuditLog(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['system-settings', 'audit-log', limit, offset],
		queryFn: () => systemSettingsApi.getAuditLog(limit, offset),
		staleTime: 30 * 1000,
	});
}
