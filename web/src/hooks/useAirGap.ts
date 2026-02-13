import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { airGapApi } from '../lib/api';

export function useAirGapStatus() {
	return useQuery({
		queryKey: ['airgap-status'],
		queryFn: airGapApi.getStatus,
		staleTime: 60 * 1000,
	});
}

export function useUploadLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (licenseData: ArrayBuffer) =>
			airGapApi.uploadLicense(licenseData),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['airgap-status'] });
		},
	});
}
