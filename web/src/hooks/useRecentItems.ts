import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { recentItemsApi } from '../lib/api';
import type { RecentItemType, TrackRecentItemRequest } from '../lib/types';

export function useRecentItems(type?: RecentItemType) {
	return useQuery({
		queryKey: ['recentItems', type],
		queryFn: () => recentItemsApi.list(type),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useTrackRecentItem() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: TrackRecentItemRequest) => recentItemsApi.track(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['recentItems'] });
		},
	});
}

export function useDeleteRecentItem() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => recentItemsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['recentItems'] });
		},
	});
}

export function useClearRecentItems() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => recentItemsApi.clearAll(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['recentItems'] });
		},
	});
}
