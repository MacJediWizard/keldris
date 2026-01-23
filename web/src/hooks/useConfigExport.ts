import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { configExportApi, templatesApi } from '../lib/api';
import type {
	CreateTemplateRequest,
	ExportBundleRequest,
	ExportFormat,
	ImportConfigRequest,
	ImportResult,
	UpdateTemplateRequest,
	UseTemplateRequest,
} from '../lib/types';

// Export hooks
export function useExportAgent() {
	return useMutation({
		mutationFn: ({ id, format }: { id: string; format?: ExportFormat }) =>
			configExportApi.exportAgent(id, format),
	});
}

export function useExportSchedule() {
	return useMutation({
		mutationFn: ({ id, format }: { id: string; format?: ExportFormat }) =>
			configExportApi.exportSchedule(id, format),
	});
}

export function useExportRepository() {
	return useMutation({
		mutationFn: ({ id, format }: { id: string; format?: ExportFormat }) =>
			configExportApi.exportRepository(id, format),
	});
}

export function useExportBundle() {
	return useMutation({
		mutationFn: (data: ExportBundleRequest) =>
			configExportApi.exportBundle(data),
	});
}

// Import hooks
export function useImportConfig() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: ImportConfigRequest) =>
			configExportApi.importConfig(data),
		onSuccess: (result: ImportResult) => {
			if (result.success) {
				// Invalidate relevant queries based on what was imported
				if (result.imported.agent_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['agents'] });
				}
				if (result.imported.schedule_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['schedules'] });
				}
				if (result.imported.repository_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['repositories'] });
				}
			}
		},
	});
}

export function useValidateImport() {
	return useMutation({
		mutationFn: (data: { config: string; format?: ExportFormat }) =>
			configExportApi.validateImport(data),
	});
}

// Template hooks
export function useTemplates() {
	return useQuery({
		queryKey: ['templates'],
		queryFn: () => templatesApi.list(),
	});
}

export function useTemplate(id: string) {
	return useQuery({
		queryKey: ['templates', id],
		queryFn: () => templatesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateTemplate() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateTemplateRequest) => templatesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['templates'] });
		},
	});
}

export function useUpdateTemplate() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateTemplateRequest }) =>
			templatesApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['templates'] });
		},
	});
}

export function useDeleteTemplate() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => templatesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['templates'] });
		},
	});
}

export function useUseTemplate() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, data }: { id: string; data?: UseTemplateRequest }) =>
			templatesApi.use(id, data),
		onSuccess: (result: ImportResult) => {
			if (result.success) {
				// Invalidate relevant queries based on what was imported
				if (result.imported.agent_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['agents'] });
				}
				if (result.imported.schedule_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['schedules'] });
				}
				if (result.imported.repository_count > 0) {
					queryClient.invalidateQueries({ queryKey: ['repositories'] });
				}
				// Update templates to reflect usage
				queryClient.invalidateQueries({ queryKey: ['templates'] });
			}
		},
	});
}

// Helper function to download exported config as file
export function downloadExport(
	content: string,
	filename: string,
	format: ExportFormat = 'json',
) {
	const blob = new Blob([content], {
		type: format === 'yaml' ? 'application/x-yaml' : 'application/json',
	});
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = `${filename}.${format}`;
	document.body.appendChild(link);
	link.click();
	document.body.removeChild(link);
	URL.revokeObjectURL(url);
}
