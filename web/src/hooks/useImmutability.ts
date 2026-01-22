import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { immutabilityApi } from '../lib/api';
import type {
	CreateImmutabilityLockRequest,
	ExtendImmutabilityLockRequest,
	UpdateRepositoryImmutabilitySettingsRequest,
} from '../lib/types';

export function useImmutabilityLocks() {
	return useQuery({
		queryKey: ['immutability-locks'],
		queryFn: () => immutabilityApi.listLocks(),
		staleTime: 30 * 1000,
	});
}

export function useImmutabilityLock(id: string) {
	return useQuery({
		queryKey: ['immutability-locks', id],
		queryFn: () => immutabilityApi.getLock(id),
		enabled: !!id,
	});
}

export function useRepositoryImmutabilityLocks(repositoryId: string) {
	return useQuery({
		queryKey: ['immutability-locks', 'repository', repositoryId],
		queryFn: () => immutabilityApi.listRepositoryLocks(repositoryId),
		enabled: !!repositoryId,
		staleTime: 30 * 1000,
	});
}

export function useSnapshotImmutabilityStatus(snapshotId: string, repositoryId: string) {
	return useQuery({
		queryKey: ['immutability-status', snapshotId, repositoryId],
		queryFn: () => immutabilityApi.getSnapshotStatus(snapshotId, repositoryId),
		enabled: !!snapshotId && !!repositoryId,
		staleTime: 60 * 1000,
	});
}

export function useRepositoryImmutabilitySettings(repositoryId: string) {
	return useQuery({
		queryKey: ['immutability-settings', repositoryId],
		queryFn: () => immutabilityApi.getRepositorySettings(repositoryId),
		enabled: !!repositoryId,
		staleTime: 60 * 1000,
	});
}

export function useCreateImmutabilityLock() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateImmutabilityLockRequest) =>
			immutabilityApi.createLock(data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: ['immutability-locks'] });
			queryClient.invalidateQueries({
				queryKey: ['immutability-locks', 'repository', variables.repository_id],
			});
			queryClient.invalidateQueries({
				queryKey: ['immutability-status', variables.snapshot_id],
			});
			queryClient.invalidateQueries({ queryKey: ['snapshots'] });
		},
	});
}

export function useExtendImmutabilityLock() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: ExtendImmutabilityLockRequest }) =>
			immutabilityApi.extendLock(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['immutability-locks'] });
			queryClient.invalidateQueries({ queryKey: ['snapshots'] });
		},
	});
}

export function useUpdateRepositoryImmutabilitySettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			repositoryId,
			data,
		}: {
			repositoryId: string;
			data: UpdateRepositoryImmutabilitySettingsRequest;
		}) => immutabilityApi.updateRepositorySettings(repositoryId, data),
		onSuccess: (_, { repositoryId }) => {
			queryClient.invalidateQueries({
				queryKey: ['immutability-settings', repositoryId],
			});
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}
