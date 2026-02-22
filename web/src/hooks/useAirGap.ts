import { useQuery } from '@tanstack/react-query';
import { airGapApi } from '../lib/api';

export function useAirGapStatus() {
	return useQuery({
		queryKey: ['airgap-status'],
		queryFn: airGapApi.getStatus,
		staleTime: 60 * 1000,
	});
}
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

const API_BASE = '/api/v1';

// Types

export interface AirGapStatus {
	airgap_mode: boolean;
	disable_update_checker: boolean;
	disable_telemetry: boolean;
	disable_external_links: boolean;
	offline_docs_version?: string;
	license_valid: boolean;
}

export interface LicenseStatus {
	valid: boolean;
	type?: 'community' | 'pro' | 'enterprise';
	organization?: string;
	expires_at?: string;
	days_until_expiry?: number;
	airgap_mode: boolean;
	features?: string[];
	max_agents?: number;
	error?: string;
	metadata?: Record<string, string>;
}

export interface RenewalRequest {
	license_id: string;
	organization: string;
	email: string;
	current_type: string;
	hardware_id: string;
	requested_at: string;
	expires_at: string;
}

export interface UpdatePackage {
	filename: string;
	path: string;
	size: number;
	version?: string;
	created_at: string;
}

export interface UpdatePackagesResponse {
	packages: UpdatePackage[];
	count: number;
}

export interface DocSection {
	id: string;
	title: string;
	path: string;
	children?: DocSection[];
}

export interface DocumentationIndex {
	version: string;
	built_at: string;
	sections: DocSection[];
	search_index?: Record<string, string>;
}

// API functions

async function fetchAirGapStatus(): Promise<AirGapStatus> {
	const res = await fetch(`${API_BASE}/public/airgap/status`);
	if (!res.ok) {
		throw new Error('Failed to fetch air-gap status');
	}
	return res.json();
}

async function fetchLicenseStatus(): Promise<LicenseStatus> {
	const res = await fetch(`${API_BASE}/airgap/license`, {
		credentials: 'include',
	});
	if (!res.ok) {
		throw new Error('Failed to fetch license status');
	}
	return res.json();
}

async function uploadLicense(licenseData: string): Promise<LicenseStatus> {
	const res = await fetch(`${API_BASE}/airgap/license`, {
		method: 'POST',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ license: licenseData }),
	});
	if (!res.ok) {
		const error = await res.json();
		throw new Error(error.error || 'Failed to upload license');
	}
	return res.json();
}

async function fetchRenewalRequest(): Promise<Blob> {
	const res = await fetch(`${API_BASE}/airgap/license/renewal-request`, {
		credentials: 'include',
	});
	if (!res.ok) {
		throw new Error('Failed to generate renewal request');
	}
	return res.blob();
}

async function updateRevocationList(data: string): Promise<void> {
	const res = await fetch(`${API_BASE}/airgap/revocations`, {
		method: 'POST',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: data,
	});
	if (!res.ok) {
		const error = await res.json();
		throw new Error(error.error || 'Failed to update revocation list');
	}
}

async function fetchUpdatePackages(): Promise<UpdatePackagesResponse> {
	const res = await fetch(`${API_BASE}/airgap/updates`, {
		credentials: 'include',
	});
	if (!res.ok) {
		throw new Error('Failed to fetch update packages');
	}
	return res.json();
}

async function applyUpdate(filename: string): Promise<void> {
	const res = await fetch(`${API_BASE}/airgap/updates/${filename}/apply`, {
		method: 'POST',
		credentials: 'include',
	});
	if (!res.ok) {
		const error = await res.json();
		throw new Error(error.error || 'Failed to apply update');
	}
}

async function fetchDocumentationIndex(): Promise<DocumentationIndex> {
	const res = await fetch(`${API_BASE}/airgap/docs`, {
		credentials: 'include',
	});
	if (!res.ok) {
		throw new Error('Failed to fetch documentation index');
	}
	return res.json();
}

// Hooks

/**
 * Hook to get air-gap mode status (public, no auth required).
 * Use this to conditionally disable features that require network access.
 */
export function useAirGapStatus() {
	return useQuery({
		queryKey: ['airgap-status'],
		queryFn: fetchAirGapStatus,
		staleTime: 5 * 60 * 1000, // 5 minutes
		retry: 1,
	});
}

/**
 * Hook to get detailed license status (requires auth).
 */
export function useLicenseStatus() {
	return useQuery({
		queryKey: ['license-status'],
		queryFn: fetchLicenseStatus,
		staleTime: 60 * 1000, // 1 minute
	});
}

/**
 * Hook to upload a new license file.
 */
export function useUploadLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: uploadLicense,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['license-status'] });
			queryClient.invalidateQueries({ queryKey: ['airgap-status'] });
		},
	});
}

/**
 * Hook to generate and download a license renewal request.
 */
export function useDownloadRenewalRequest() {
	return useMutation({
		mutationFn: async () => {
			const blob = await fetchRenewalRequest();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'license-renewal-request.json';
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		},
	});
}

/**
 * Hook to update the revocation list.
 */
export function useUpdateRevocationList() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: updateRevocationList,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['license-status'] });
		},
	});
}

/**
 * Hook to list available offline update packages.
 */
export function useUpdatePackages() {
	return useQuery({
		queryKey: ['update-packages'],
		queryFn: fetchUpdatePackages,
		staleTime: 30 * 1000, // 30 seconds
	});
}

/**
 * Hook to apply an offline update package.
 */
export function useApplyUpdate() {
	return useMutation({
		mutationFn: applyUpdate,
	});
}

/**
 * Hook to get the offline documentation index.
 */
export function useDocumentationIndex() {
	return useQuery({
		queryKey: ['documentation-index'],
		queryFn: fetchDocumentationIndex,
		staleTime: 10 * 60 * 1000, // 10 minutes
	});
}

/**
 * Convenience hook that combines air-gap status with computed values.
 */
export function useAirGap() {
	const { data: status, isLoading, error } = useAirGapStatus();

	return {
		isLoading,
		error,
		isAirGapMode: status?.airgap_mode ?? false,
		disableUpdateChecker: status?.disable_update_checker ?? false,
		disableTelemetry: status?.disable_telemetry ?? false,
		disableExternalLinks: status?.disable_external_links ?? false,
		offlineDocsVersion: status?.offline_docs_version,
		licenseValid: status?.license_valid ?? false,

		// Helper to check if external links should be disabled
		shouldBlockExternalLink: (url: string) => {
			if (!status?.disable_external_links) return false;
			try {
				const parsed = new URL(url, window.location.origin);
				return parsed.origin !== window.location.origin;
			} catch {
				return false;
			}
		},
	};
}
