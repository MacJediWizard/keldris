import { useQuery } from '@tanstack/react-query';
import { licenseApi } from '../lib/api';

export function useLicense() {
	return useQuery({
		queryKey: ['license'],
		queryFn: licenseApi.getInfo,
		staleTime: 5 * 60 * 1000,
	});
}
