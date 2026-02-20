import { useQuery } from '@tanstack/react-query';
import { versionApi } from '../lib/api';

export function useVersion() {
	return useQuery({
		queryKey: ['version'],
		queryFn: versionApi.get,
		staleTime: 60 * 60 * 1000,
	});
}
