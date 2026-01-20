import { useQuery } from '@tanstack/react-query';
import { statsApi } from '../lib/api';

export function useStorageStatsSummary() {
	return useQuery({
		queryKey: ['storage-stats', 'summary'],
		queryFn: statsApi.getSummary,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useStorageGrowth(days = 30) {
	return useQuery({
		queryKey: ['storage-stats', 'growth', days],
		queryFn: () => statsApi.getGrowth(days),
		staleTime: 60 * 1000,
	});
}

export function useRepositoryStatsList() {
	return useQuery({
		queryKey: ['storage-stats', 'repositories'],
		queryFn: statsApi.listRepositoryStats,
		staleTime: 60 * 1000,
	});
}

export function useRepositoryStats(id: string) {
	return useQuery({
		queryKey: ['storage-stats', 'repositories', id],
		queryFn: () => statsApi.getRepositoryStats(id),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}

export function useRepositoryGrowth(id: string, days = 30) {
	return useQuery({
		queryKey: ['storage-stats', 'repositories', id, 'growth', days],
		queryFn: () => statsApi.getRepositoryGrowth(id, days),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}

export function useRepositoryHistory(id: string, limit = 30) {
	return useQuery({
		queryKey: ['storage-stats', 'repositories', id, 'history', limit],
		queryFn: () => statsApi.getRepositoryHistory(id, limit),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}
