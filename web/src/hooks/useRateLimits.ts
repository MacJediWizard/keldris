import { useQuery, useQueryClient } from '@tanstack/react-query';
import { rateLimitsApi } from '../lib/api';

export function useRateLimitDashboard() {
	const queryClient = useQueryClient();

	const { data, error, isLoading } = useQuery({
		queryKey: ['admin', 'rate-limits'],
		queryFn: () => rateLimitsApi.getDashboardStats(),
		refetchInterval: 10000, // Refresh every 10 seconds
		staleTime: 5000,
	});

	const refresh = () => {
		queryClient.invalidateQueries({ queryKey: ['admin', 'rate-limits'] });
	};

	return {
		stats: data,
		isLoading,
		error,
		refresh,
	};
}
