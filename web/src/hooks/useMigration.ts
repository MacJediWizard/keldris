import { useMutation, useQueryClient } from '@tanstack/react-query';
import type {
	MigrationExportRequest,
	MigrationImportRequest,
	MigrationImportResult,
	MigrationValidationResult,
} from '../lib/types';

const API_BASE = '/api/v1';

class ApiError extends Error {
	constructor(
		public status: number,
		message: string,
	) {
		super(message);
		this.name = 'ApiError';
	}
}

// Migration Export API
export function useGenerateExportKey() {
	return useMutation({
		mutationFn: async (): Promise<{ key: string }> => {
			const response = await fetch(`${API_BASE}/migration/export/generate-key`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json',
				},
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Failed to generate key' }));
				throw new ApiError(response.status, errorData.error);
			}
			return response.json();
		},
	});
}

export function useMigrationExport() {
	return useMutation({
		mutationFn: async (data: MigrationExportRequest): Promise<Blob> => {
			const response = await fetch(`${API_BASE}/migration/export`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(data),
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Export failed' }));
				throw new ApiError(response.status, errorData.error);
			}
			return response.blob();
		},
	});
}

export function useValidateMigrationImport() {
	return useMutation({
		mutationFn: async (data: {
			data: string;
			decryption_key?: string;
		}): Promise<MigrationValidationResult> => {
			const response = await fetch(`${API_BASE}/migration/import/validate`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(data),
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Validation failed' }));
				throw new ApiError(response.status, errorData.error);
			}
			return response.json();
		},
	});
}

export function useMigrationImport() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async (
			data: MigrationImportRequest,
		): Promise<MigrationImportResult> => {
			const response = await fetch(`${API_BASE}/migration/import`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(data),
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Import failed' }));
				throw new ApiError(response.status, errorData.error);
			}
			return response.json();
		},
		onSuccess: (result) => {
			if (result.success && !result.dry_run) {
				// Invalidate all relevant queries after successful import
				queryClient.invalidateQueries({ queryKey: ['organizations'] });
				queryClient.invalidateQueries({ queryKey: ['users'] });
				queryClient.invalidateQueries({ queryKey: ['agents'] });
				queryClient.invalidateQueries({ queryKey: ['repositories'] });
				queryClient.invalidateQueries({ queryKey: ['schedules'] });
				queryClient.invalidateQueries({ queryKey: ['policies'] });
			}
		},
	});
}

// Helper function to download exported data as file
export function downloadMigrationExport(blob: Blob, encrypted: boolean) {
	const timestamp = new Date().toISOString().slice(0, 19).replace(/[:-]/g, '');
	const extension = encrypted ? 'encrypted' : 'json';
	const filename = `keldris-migration-${timestamp}.${extension}`;

	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = filename;
	document.body.appendChild(link);
	link.click();
	document.body.removeChild(link);
	URL.revokeObjectURL(url);
}

// Helper function to read file as text
export function readFileAsText(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = () => resolve(reader.result as string);
		reader.onerror = () => reject(new Error('Failed to read file'));
		reader.readAsText(file);
	});
}
