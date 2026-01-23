import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { snapshotMountsApi, snapshotsApi } from '../lib/api';
import type { MountSnapshotRequest } from '../lib/types';

export function useSnapshotMounts(agentId?: string) {
	return useQuery({
		queryKey: ['snapshot-mounts', agentId],
		queryFn: () => snapshotMountsApi.list(agentId),
		staleTime: 10 * 1000, // 10 seconds - mounts can change frequently
		refetchInterval: 30 * 1000, // Poll every 30 seconds
	});
}

export function useSnapshotMount(snapshotId: string, agentId: string) {
	return useQuery({
		queryKey: ['snapshot-mounts', snapshotId, agentId],
		queryFn: () => snapshotsApi.getMount(snapshotId, agentId),
		enabled: !!snapshotId && !!agentId,
		staleTime: 10 * 1000, // 10 seconds
		refetchInterval: 15 * 1000, // Poll every 15 seconds while viewing
		retry: false, // Don't retry if mount doesn't exist
	});
}

export function useMountSnapshot() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			snapshotId,
			data,
		}: {
			snapshotId: string;
			data: MountSnapshotRequest;
		}) => snapshotsApi.mount(snapshotId, data),
		onSuccess: (_data, variables) => {
			// Invalidate mounts list
			queryClient.invalidateQueries({ queryKey: ['snapshot-mounts'] });
			// Invalidate specific mount status
			queryClient.invalidateQueries({
				queryKey: [
					'snapshot-mounts',
					variables.snapshotId,
					variables.data.agent_id,
				],
			});
		},
	});
}

export function useUnmountSnapshot() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			snapshotId,
			agentId,
		}: {
			snapshotId: string;
			agentId: string;
		}) => snapshotsApi.unmount(snapshotId, agentId),
		onSuccess: (_data, variables) => {
			// Invalidate mounts list
			queryClient.invalidateQueries({ queryKey: ['snapshot-mounts'] });
			// Invalidate specific mount status
			queryClient.invalidateQueries({
				queryKey: ['snapshot-mounts', variables.snapshotId, variables.agentId],
			});
		},
	});
}
