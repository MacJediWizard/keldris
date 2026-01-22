import { useQuery } from '@tanstack/react-query';
import { fileHistoryApi } from '../lib/api';
import type { FileHistoryParams } from '../lib/types';

export function useFileHistory(params: FileHistoryParams | null) {
	return useQuery({
		queryKey: [
			'fileHistory',
			params?.path,
			params?.agent_id,
			params?.repository_id,
		],
		queryFn: () => {
			if (!params) {
				throw new Error('File history params required');
			}
			return fileHistoryApi.getHistory(params);
		},
		enabled: !!params?.path && !!params?.agent_id && !!params?.repository_id,
		staleTime: 60 * 1000, // 1 minute (file history doesn't change often)
	});
}
