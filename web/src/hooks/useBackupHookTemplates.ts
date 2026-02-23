import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { backupHookTemplatesApi } from '../lib/api';
import type {
	ApplyBackupHookTemplateRequest,
	CreateBackupHookTemplateRequest,
	UpdateBackupHookTemplateRequest,
} from '../lib/types';

export function useBackupHookTemplates(params?: {
	service_type?: string;
	visibility?: string;
	tag?: string;
}) {
	return useQuery({
		queryKey: ['backupHookTemplates', params],
		queryFn: () => backupHookTemplatesApi.list(params),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useBuiltInBackupHookTemplates() {
	return useQuery({
		queryKey: ['backupHookTemplates', 'built-in'],
		queryFn: () => backupHookTemplatesApi.listBuiltIn(),
		staleTime: 5 * 60 * 1000, // 5 minutes - built-in templates don't change often
	});
}

export function useBackupHookTemplate(id: string) {
	return useQuery({
		queryKey: ['backupHookTemplates', id],
		queryFn: () => backupHookTemplatesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateBackupHookTemplate() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateBackupHookTemplateRequest) =>
			backupHookTemplatesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['backupHookTemplates'] });
		},
	});
}

export function useUpdateBackupHookTemplate() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateBackupHookTemplateRequest;
		}) => backupHookTemplatesApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['backupHookTemplates'] });
			queryClient.invalidateQueries({ queryKey: ['backupHookTemplates', id] });
		},
	});
}

export function useDeleteBackupHookTemplate() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => backupHookTemplatesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['backupHookTemplates'] });
		},
	});
}

export function useApplyBackupHookTemplate() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			templateId,
			data,
		}: {
			templateId: string;
			data: ApplyBackupHookTemplateRequest;
		}) => backupHookTemplatesApi.apply(templateId, data),
		onSuccess: (_, { data }) => {
			// Invalidate backup scripts for the schedule
			queryClient.invalidateQueries({
				queryKey: ['backupScripts', data.schedule_id],
			});
		},
	});
}
