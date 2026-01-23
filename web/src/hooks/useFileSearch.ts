import { useQuery } from '@tanstack/react-query';
import { fileSearchApi } from '../lib/api';
import type { FileSearchParams, FileSearchResponse } from '../lib/types';

const emptyResponse: FileSearchResponse = {
	query: '',
	agent_id: '',
	repository_id: '',
	total_count: 0,
	snapshot_count: 0,
	snapshots: [],
};

export function useFileSearch(params: FileSearchParams | null) {
	return useQuery({
		queryKey: ['file-search', params],
		queryFn: () => {
			if (!params || !params.q || !params.agent_id || !params.repository_id) {
				return emptyResponse;
			}
			return fileSearchApi.search(params);
		},
		enabled: !!params?.q && !!params?.agent_id && !!params?.repository_id,
		staleTime: 60 * 1000, // 1 minute
	});
}
