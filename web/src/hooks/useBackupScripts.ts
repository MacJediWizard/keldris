import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { backupScriptsApi } from '../lib/api';
import type {
	CreateBackupScriptRequest,
	UpdateBackupScriptRequest,
} from '../lib/types';

export function useBackupScripts(scheduleId: string) {
	return useQuery({
		queryKey: ['backupScripts', scheduleId],
		queryFn: () => backupScriptsApi.list(scheduleId),
		enabled: !!scheduleId,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useBackupScript(scheduleId: string, id: string) {
	return useQuery({
		queryKey: ['backupScripts', scheduleId, id],
		queryFn: () => backupScriptsApi.get(scheduleId, id),
		enabled: !!scheduleId && !!id,
	});
}

export function useCreateBackupScript() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			scheduleId,
			data,
		}: {
			scheduleId: string;
			data: CreateBackupScriptRequest;
		}) => backupScriptsApi.create(scheduleId, data),
		onSuccess: (_, { scheduleId }) => {
			queryClient.invalidateQueries({
				queryKey: ['backupScripts', scheduleId],
			});
		},
	});
}

export function useUpdateBackupScript() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			scheduleId,
			id,
			data,
		}: {
			scheduleId: string;
			id: string;
			data: UpdateBackupScriptRequest;
		}) => backupScriptsApi.update(scheduleId, id, data),
		onSuccess: (_, { scheduleId, id }) => {
			queryClient.invalidateQueries({
				queryKey: ['backupScripts', scheduleId],
			});
			queryClient.invalidateQueries({
				queryKey: ['backupScripts', scheduleId, id],
			});
		},
	});
}

export function useDeleteBackupScript() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ scheduleId, id }: { scheduleId: string; id: string }) =>
			backupScriptsApi.delete(scheduleId, id),
		onSuccess: (_, { scheduleId }) => {
			queryClient.invalidateQueries({
				queryKey: ['backupScripts', scheduleId],
			});
		},
	});
}
