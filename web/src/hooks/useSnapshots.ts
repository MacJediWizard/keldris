import { useQuery } from '@tanstack/react-query';
import { snapshotsApi } from '../lib/api';

export interface SnapshotsFilter {
	agent_id?: string;
	repository_id?: string;
}

export function useSnapshots(filter?: SnapshotsFilter) {
	return useQuery({
		queryKey: ['snapshots', filter],
		queryFn: () => snapshotsApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useSnapshot(id: string) {
	return useQuery({
		queryKey: ['snapshots', id],
		queryFn: () => snapshotsApi.get(id),
		enabled: !!id,
	});
}

export function useSnapshotFiles(snapshotId: string, path?: string) {
	return useQuery({
		queryKey: ['snapshots', snapshotId, 'files', path ?? ''],
		queryFn: () => snapshotsApi.listFiles(snapshotId, path),
		enabled: !!snapshotId,
		staleTime: 60 * 1000, // 1 minute (files don't change)
	});
}

export function useSnapshotCompare(snapshotId1: string, snapshotId2: string) {
	return useQuery({
		queryKey: ['snapshots', 'compare', snapshotId1, snapshotId2],
		queryFn: () => snapshotsApi.compare(snapshotId1, snapshotId2),
		enabled: !!snapshotId1 && !!snapshotId2,
		staleTime: 60 * 1000, // 1 minute (comparison results don't change)
	});
}
