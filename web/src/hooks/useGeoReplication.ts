import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { geoReplicationApi } from '../lib/api';
import type {
	GeoReplicationCreateRequest,
	GeoReplicationUpdateRequest,
} from '../lib/types';

export function useGeoReplicationRegions() {
	return useQuery({
		queryKey: ['geo-replication', 'regions'],
		queryFn: () => geoReplicationApi.listRegions(),
		staleTime: 24 * 60 * 60 * 1000, // 24 hours - regions don't change often
	});
}

export function useGeoReplicationConfigs() {
	return useQuery({
		queryKey: ['geo-replication', 'configs'],
		queryFn: () => geoReplicationApi.listConfigs(),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useGeoReplicationConfig(id: string) {
	return useQuery({
		queryKey: ['geo-replication', 'configs', id],
		queryFn: () => geoReplicationApi.getConfig(id),
		enabled: !!id,
		staleTime: 30 * 1000,
	});
}

export function useGeoReplicationSummary() {
	return useQuery({
		queryKey: ['geo-replication', 'summary'],
		queryFn: () => geoReplicationApi.getSummary(),
		staleTime: 30 * 1000,
	});
}

export function useGeoReplicationEvents(configId: string) {
	return useQuery({
		queryKey: ['geo-replication', 'configs', configId, 'events'],
		queryFn: () => geoReplicationApi.getEvents(configId),
		enabled: !!configId,
		staleTime: 30 * 1000,
	});
}

export function useRepositoryReplicationStatus(repositoryId: string) {
	return useQuery({
		queryKey: ['geo-replication', 'repositories', repositoryId, 'status'],
		queryFn: () => geoReplicationApi.getRepositoryStatus(repositoryId),
		enabled: !!repositoryId,
		staleTime: 30 * 1000,
	});
}

export function useCreateGeoReplicationConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: GeoReplicationCreateRequest) =>
			geoReplicationApi.createConfig(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['geo-replication'] });
		},
	});
}

export function useUpdateGeoReplicationConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: GeoReplicationUpdateRequest;
		}) => geoReplicationApi.updateConfig(id, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['geo-replication', 'configs', variables.id],
			});
			queryClient.invalidateQueries({ queryKey: ['geo-replication'] });
		},
	});
}

export function useDeleteGeoReplicationConfig() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => geoReplicationApi.deleteConfig(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['geo-replication'] });
		},
	});
}

export function useTriggerReplication() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => geoReplicationApi.triggerReplication(id),
		onSuccess: (_data, configId) => {
			queryClient.invalidateQueries({
				queryKey: ['geo-replication', 'configs', configId],
			});
			queryClient.invalidateQueries({ queryKey: ['geo-replication'] });
		},
	});
}

export function useSetRepositoryRegion() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ repoId, region }: { repoId: string; region: string }) =>
			geoReplicationApi.setRepositoryRegion(repoId, region),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['geo-replication', 'repositories', variables.repoId],
			});
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}
