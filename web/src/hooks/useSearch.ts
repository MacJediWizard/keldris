import { useQuery } from '@tanstack/react-query';
import { searchApi } from '../lib/api';
import type { SearchFilter } from '../lib/types';

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
