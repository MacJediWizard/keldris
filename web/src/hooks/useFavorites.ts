import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { favoritesApi } from '../lib/api';
import type { CreateFavoriteRequest, FavoriteEntityType } from '../lib/types';

export function useFavorites(entityType?: FavoriteEntityType) {
	return useQuery({
		queryKey: ['favorites', entityType],
		queryFn: () => favoritesApi.list(entityType),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useAddFavorite() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateFavoriteRequest) => favoritesApi.create(data),
		onSuccess: (favorite) => {
			queryClient.invalidateQueries({
				queryKey: ['favorites', favorite.entity_type],
			});
			queryClient.invalidateQueries({ queryKey: ['favorites', undefined] });
		},
	});
}

export function useRemoveFavorite() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			entityType,
			entityId,
		}: {
			entityType: FavoriteEntityType;
			entityId: string;
		}) => favoritesApi.delete(entityType, entityId),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['favorites'] });
		},
	});
}

// Hook to check if a specific entity is favorited
export function useIsFavorite(
	entityType: FavoriteEntityType,
	entityId: string,
) {
	const { data: favorites } = useFavorites(entityType);
	return favorites?.some((f) => f.entity_id === entityId) ?? false;
}

// Hook to get favorite IDs for a specific entity type
export function useFavoriteIds(entityType: FavoriteEntityType) {
	const { data: favorites } = useFavorites(entityType);
	return new Set(favorites?.map((f) => f.entity_id) ?? []);
}
