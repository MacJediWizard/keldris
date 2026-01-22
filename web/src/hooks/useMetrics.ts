import { useQuery } from '@tanstack/react-query';
import { metricsApi } from '../lib/api';

export function useDashboardStats() {
	return useQuery({
		queryKey: ['metrics', 'dashboard'],
		queryFn: metricsApi.getDashboardStats,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useBackupSuccessRates() {
	return useQuery({
		queryKey: ['metrics', 'success-rates'],
		queryFn: metricsApi.getBackupSuccessRates,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useStorageGrowthTrend(days = 30) {
	return useQuery({
		queryKey: ['metrics', 'storage-growth', days],
		queryFn: () => metricsApi.getStorageGrowthTrend(days),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useBackupDurationTrend(days = 30) {
	return useQuery({
		queryKey: ['metrics', 'backup-duration', days],
		queryFn: () => metricsApi.getBackupDurationTrend(days),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useDailyBackupStats(days = 30) {
	return useQuery({
		queryKey: ['metrics', 'daily-backups', days],
		queryFn: () => metricsApi.getDailyBackupStats(days),
		staleTime: 60 * 1000, // 1 minute
	});
}
