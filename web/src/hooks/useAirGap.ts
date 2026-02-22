import { useQuery } from '@tanstack/react-query';
import { airGapApi } from '../lib/api';

export function useAirGapStatus() {
	return useQuery({
		queryKey: ['airgap-status'],
		queryFn: airGapApi.getStatus,
		staleTime: 60 * 1000,
	});
}
