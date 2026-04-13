import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchApi, fetchBlob } from "../lib/api";

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
	type?: "community" | "pro" | "enterprise";
	organization?: string;
	expires_at?: string;
	days_until_expiry?: number;
	airgap_mode: boolean;
	features?: string[];
	max_agents?: number;
	error?: string;
	metadata?: Record<string, string>;
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

// The public airgap status endpoint doesn't require auth and needs
// special 404 handling, so it uses raw fetch instead of the api client.
async function fetchAirGapStatus(): Promise<AirGapStatus> {
	const res = await fetch("/api/v1/public/airgap/status");
	if (!res.ok) {
		if (res.status === 404) {
			return {
				airgap_mode: false,
				disable_update_checker: false,
				disable_telemetry: false,
				disable_external_links: false,
				license_valid: true,
			};
		}
		throw new Error("Failed to fetch air-gap status");
	}
	return res.json();
}

// Hooks

export function useAirGapStatus() {
	return useQuery({
		queryKey: ["airgap-status"],
		queryFn: fetchAirGapStatus,
		staleTime: 5 * 60 * 1000,
		retry: 1,
	});
}

export function useLicenseStatus() {
	return useQuery({
		queryKey: ["license-status"],
		queryFn: () => fetchApi<LicenseStatus>("/airgap/license"),
		staleTime: 60 * 1000,
	});
}

export function useUploadLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (licenseData: string) =>
			fetchApi<LicenseStatus>("/airgap/license", {
				method: "POST",
				body: JSON.stringify({ license: licenseData }),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["license-status"] });
			queryClient.invalidateQueries({ queryKey: ["airgap-status"] });
		},
	});
}

export function useDownloadRenewalRequest() {
	return useMutation({
		mutationFn: async () => {
			const blob = await fetchBlob("/airgap/license/renewal-request");
			const url = URL.createObjectURL(blob);
			const a = document.createElement("a");
			a.href = url;
			a.download = "license-renewal-request.json";
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		},
	});
}

export function useUpdateRevocationList() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: string) =>
			fetchApi<void>("/airgap/revocations", {
				method: "POST",
				body: data,
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["license-status"] });
		},
	});
}

export function useUpdatePackages() {
	return useQuery({
		queryKey: ["update-packages"],
		queryFn: () => fetchApi<UpdatePackagesResponse>("/airgap/updates"),
		staleTime: 30 * 1000,
	});
}

export function useApplyUpdate() {
	return useMutation({
		mutationFn: (filename: string) =>
			fetchApi<void>(`/airgap/updates/${filename}/apply`, {
				method: "POST",
			}),
	});
}

export function useDocumentationIndex() {
	return useQuery({
		queryKey: ["documentation-index"],
		queryFn: () => fetchApi<DocumentationIndex>("/airgap/docs"),
		staleTime: 10 * 60 * 1000,
	});
}

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
