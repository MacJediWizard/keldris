import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { containerHooksApi } from '../lib/api';
import type {
	CreateContainerBackupHookRequest,
	UpdateContainerBackupHookRequest,
} from '../lib/types';

export function useContainerHooks(scheduleId: string) {
	return useQuery({
		queryKey: ['containerHooks', scheduleId],
		queryFn: () => containerHooksApi.list(scheduleId),
		enabled: !!scheduleId,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useContainerHook(scheduleId: string, id: string) {
	return useQuery({
		queryKey: ['containerHooks', scheduleId, id],
		queryFn: () => containerHooksApi.get(scheduleId, id),
		enabled: !!scheduleId && !!id,
	});
}

export function useCreateContainerHook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			scheduleId,
			data,
		}: {
			scheduleId: string;
			data: CreateContainerBackupHookRequest;
		}) => containerHooksApi.create(scheduleId, data),
		onSuccess: (_, { scheduleId }) => {
			queryClient.invalidateQueries({
				queryKey: ['containerHooks', scheduleId],
			});
		},
	});
}

export function useUpdateContainerHook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			scheduleId,
			id,
			data,
		}: {
			scheduleId: string;
			id: string;
			data: UpdateContainerBackupHookRequest;
		}) => containerHooksApi.update(scheduleId, id, data),
		onSuccess: (_, { scheduleId, id }) => {
			queryClient.invalidateQueries({
				queryKey: ['containerHooks', scheduleId],
			});
			queryClient.invalidateQueries({
				queryKey: ['containerHooks', scheduleId, id],
			});
		},
	});
}

export function useDeleteContainerHook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ scheduleId, id }: { scheduleId: string; id: string }) =>
			containerHooksApi.delete(scheduleId, id),
		onSuccess: (_, { scheduleId }) => {
			queryClient.invalidateQueries({
				queryKey: ['containerHooks', scheduleId],
			});
		},
	});
}

export function useContainerHookTemplates() {
	return useQuery({
		queryKey: ['containerHookTemplates'],
		queryFn: () => containerHooksApi.listTemplates(),
		staleTime: 5 * 60 * 1000, // 5 minutes - templates don't change often
	});
}

export function useContainerHookExecutions(backupId: string) {
	return useQuery({
		queryKey: ['containerHookExecutions', backupId],
		queryFn: () => containerHooksApi.listExecutions(backupId),
		enabled: !!backupId,
		staleTime: 30 * 1000, // 30 seconds
	});
}
