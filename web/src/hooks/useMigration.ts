import { useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchApi } from '../lib/api';
import type {
	MigrationExportRequest,
	MigrationImportRequest,
	MigrationImportResult,
	MigrationValidationResult,
} from '../lib/types';

// Migration Export API
export function useGenerateExportKey() {
	return useMutation({
		mutationFn: () =>
			fetchApi<{ key: string }>('/migration/export/generate-key', {
				method: 'POST',
			}),
	});
}

export function useMigrationExport() {
	return useMutation({
		mutationFn: async (data: MigrationExportRequest): Promise<Blob> => {
			const response = await fetch('/api/v1/migration/export', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data),
			});
			if (!response.ok) {
				const errorData = await response
					.json()
					.catch(() => ({ error: 'Export failed' }));
				throw new Error(errorData.error);
			}
			return response.blob();
		},
	});
}

export function useValidateMigrationImport() {
	return useMutation({
		mutationFn: (data: { data: string; decryption_key?: string }) =>
			fetchApi<MigrationValidationResult>('/migration/import/validate', {
				method: 'POST',
				body: JSON.stringify(data),
			}),
	});
}

export function useMigrationImport() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: MigrationImportRequest) =>
			fetchApi<MigrationImportResult>('/migration/import', {
				method: 'POST',
				body: JSON.stringify(data),
			}),
		onSuccess: (result) => {
			if (result.success && !result.dry_run) {
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
