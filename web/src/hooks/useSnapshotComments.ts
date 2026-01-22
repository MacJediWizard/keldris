import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { snapshotCommentsApi } from '../lib/api';
import type { CreateSnapshotCommentRequest } from '../lib/types';

export function useSnapshotComments(snapshotId: string) {
	return useQuery({
		queryKey: ['snapshots', snapshotId, 'comments'],
		queryFn: () => snapshotCommentsApi.list(snapshotId),
		enabled: !!snapshotId,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useCreateSnapshotComment(snapshotId: string) {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateSnapshotCommentRequest) =>
			snapshotCommentsApi.create(snapshotId, data),
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: ['snapshots', snapshotId, 'comments'],
			});
		},
	});
}

export function useDeleteSnapshotComment(snapshotId: string) {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (commentId: string) => snapshotCommentsApi.delete(commentId),
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: ['snapshots', snapshotId, 'comments'],
			});
		},
	});
}
