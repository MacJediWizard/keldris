import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { setupApi } from '../lib/api';
import type {
	ActivateLicenseRequest,
	CreateFirstOrgRequest,
	CreateSuperuserRequest,
	OIDCSettings,
	SMTPSettings,
	StartTrialRequest,
} from '../lib/types';

export function useSetupStatus() {
	return useQuery({
		queryKey: ['setup', 'status'],
		queryFn: setupApi.getStatus,
		staleTime: 30 * 1000, // 30 seconds
		retry: false,
	});
}

export function useTestDatabase() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: setupApi.testDatabase,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useCreateSuperuser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateSuperuserRequest) =>
			setupApi.createSuperuser(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useConfigureSMTP() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: SMTPSettings) => setupApi.configureSMTP(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useSkipSMTP() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: setupApi.skipSMTP,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useConfigureOIDC() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: OIDCSettings) => setupApi.configureOIDC(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useSkipOIDC() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: setupApi.skipOIDC,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useActivateLicense() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: ActivateLicenseRequest) =>
			setupApi.activateLicense(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useStartTrial() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: StartTrialRequest) => setupApi.startTrial(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useCreateFirstOrganization() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateFirstOrgRequest) =>
			setupApi.createOrganization(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup'] });
		},
	});
}

export function useCompleteSetup() {
	return useMutation({
		mutationFn: setupApi.completeSetup,
		onSuccess: (data) => {
			// Full page reload to reinitialize app after setup
			window.location.href = data.redirect || '/';
		},
	});
}

// Superuser re-run hooks
export function useRerunStatus() {
	return useQuery({
		queryKey: ['setup', 'rerun'],
		queryFn: setupApi.getRerunStatus,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useRerunConfigureSMTP() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: SMTPSettings) => setupApi.rerunConfigureSMTP(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup', 'rerun'] });
		},
	});
}

export function useRerunConfigureOIDC() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: OIDCSettings) => setupApi.rerunConfigureOIDC(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup', 'rerun'] });
		},
	});
}

export function useRerunUpdateLicense() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: ActivateLicenseRequest) =>
			setupApi.rerunUpdateLicense(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['setup', 'rerun'] });
		},
	});
}
