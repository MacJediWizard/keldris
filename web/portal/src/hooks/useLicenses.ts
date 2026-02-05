import { useQuery } from '@tanstack/react-query';
import { licensesApi } from '../lib/api';

export function useLicenses() {
	return useQuery({
		queryKey: ['licenses'],
		queryFn: () => licensesApi.list(),
	});
}

export function useLicense(id: string) {
	return useQuery({
		queryKey: ['licenses', id],
		queryFn: () => licensesApi.get(id),
		enabled: !!id,
	});
}

export function useLicenseDownload(id: string) {
	return useQuery({
		queryKey: ['licenses', id, 'download'],
		queryFn: () => licensesApi.download(id),
		enabled: false, // Only fetch on demand
	});
}
