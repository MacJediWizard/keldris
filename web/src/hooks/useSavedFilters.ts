import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { savedFiltersApi } from '../lib/api';
import type {
	CreateSavedFilterRequest,
	UpdateSavedFilterRequest,
} from '../lib/types';

export function useSavedFilters(entityType?: string) {
	return useQuery({
		queryKey: ['savedFilters', entityType],
		queryFn: () => savedFiltersApi.list(entityType),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useSavedFilter(id: string) {
	return useQuery({
		queryKey: ['savedFilters', 'detail', id],
		queryFn: () => savedFiltersApi.get(id),
		enabled: !!id,
	});
}

export function useCreateSavedFilter() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateSavedFilterRequest) =>
			savedFiltersApi.create(data),
		onSuccess: (filter) => {
			queryClient.invalidateQueries({
				queryKey: ['savedFilters', filter.entity_type],
			});
			queryClient.invalidateQueries({ queryKey: ['savedFilters', undefined] });
		},
	});
}

export function useUpdateSavedFilter() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateSavedFilterRequest }) =>
			savedFiltersApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['savedFilters'] });
		},
	});
}

export function useDeleteSavedFilter() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => savedFiltersApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['savedFilters'] });
		},
	});
}
