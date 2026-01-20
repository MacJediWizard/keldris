import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { tagsApi } from '../lib/api';
import type {
	AssignTagsRequest,
	CreateTagRequest,
	UpdateTagRequest,
} from '../lib/types';

export function useTags() {
	return useQuery({
		queryKey: ['tags'],
		queryFn: () => tagsApi.list(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useTag(id: string) {
	return useQuery({
		queryKey: ['tags', id],
		queryFn: () => tagsApi.get(id),
		enabled: !!id,
	});
}

export function useCreateTag() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateTagRequest) => tagsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['tags'] });
		},
	});
}

export function useUpdateTag() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateTagRequest }) =>
			tagsApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['tags'] });
		},
	});
}

export function useDeleteTag() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => tagsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['tags'] });
		},
	});
}

export function useBackupTags(backupId: string) {
	return useQuery({
		queryKey: ['backups', backupId, 'tags'],
		queryFn: () => tagsApi.getBackupTags(backupId),
		enabled: !!backupId,
	});
}

export function useSetBackupTags() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			backupId,
			data,
		}: {
			backupId: string;
			data: AssignTagsRequest;
		}) => tagsApi.setBackupTags(backupId, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['backups', variables.backupId, 'tags'],
			});
			queryClient.invalidateQueries({ queryKey: ['backups'] });
		},
	});
}
