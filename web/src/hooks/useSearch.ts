import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { searchApi } from '../lib/api';
import type { SaveRecentSearchRequest, SearchFilter } from '../lib/types';

export function useSearch(filter: SearchFilter | null) {
	return useQuery({
		queryKey: ['search', filter],
		queryFn: () => {
			if (!filter || !filter.q) {
				return { results: [], query: '', total: 0 };
			}
			return searchApi.search(filter);
		},
		enabled: !!filter?.q,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useGroupedSearch(filter: SearchFilter | null) {
	return useQuery({
		queryKey: ['search', 'grouped', filter],
		queryFn: () => {
			if (!filter || !filter.q) {
				return {
					agents: [],
					backups: [],
					snapshots: [],
					schedules: [],
					repositories: [],
					query: '',
					total: 0,
				};
			}
			return searchApi.searchGrouped(filter);
		},
		enabled: !!filter?.q,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useSearchSuggestions(query: string, enabled = true) {
	return useQuery({
		queryKey: ['search', 'suggestions', query],
		queryFn: () => {
			if (!query || query.length < 2) {
				return { suggestions: [] };
			}
			return searchApi.getSuggestions(query);
		},
		enabled: enabled && query.length >= 2,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useRecentSearches(limit = 10) {
	return useQuery({
		queryKey: ['search', 'recent', limit],
		queryFn: () => searchApi.getRecentSearches(limit),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useSaveRecentSearch() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: SaveRecentSearchRequest) =>
			searchApi.saveRecentSearch(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['search', 'recent'] });
		},
	});
}

export function useDeleteRecentSearch() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => searchApi.deleteRecentSearch(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['search', 'recent'] });
		},
	});
}

export function useClearRecentSearches() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => searchApi.clearRecentSearches(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['search', 'recent'] });
		},
	});
}
