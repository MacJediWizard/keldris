import { useQuery } from '@tanstack/react-query';
import { systemHealthApi } from '../lib/api';

export function useSystemHealth() {
	return useQuery({
		queryKey: ['admin', 'health'],
		queryFn: systemHealthApi.getHealth,
		staleTime: 10 * 1000, // 10 seconds
		refetchInterval: 30 * 1000, // Auto-refresh every 30 seconds
	});
}

export function useSystemHealthHistory() {
	return useQuery({
		queryKey: ['admin', 'health', 'history'],
		queryFn: systemHealthApi.getHistory,
		staleTime: 30 * 1000, // 30 seconds
		refetchInterval: 60 * 1000, // Auto-refresh every minute
	});
}
