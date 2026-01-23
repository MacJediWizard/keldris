import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { legalHoldsApi } from '../lib/api';
import type { CreateLegalHoldRequest } from '../lib/types';

export function useLegalHolds() {
	return useQuery({
		queryKey: ['legal-holds'],
		queryFn: () => legalHoldsApi.list(),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useLegalHold(snapshotId: string) {
	return useQuery({
		queryKey: ['legal-holds', snapshotId],
		queryFn: () => legalHoldsApi.get(snapshotId),
		enabled: !!snapshotId,
		retry: false, // Don't retry on 404
	});
}

export function useCreateLegalHold() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			snapshotId,
			data,
		}: {
			snapshotId: string;
			data: CreateLegalHoldRequest;
		}) => legalHoldsApi.create(snapshotId, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['legal-holds'],
			});
			queryClient.invalidateQueries({
				queryKey: ['legal-holds', variables.snapshotId],
			});
			queryClient.invalidateQueries({
				queryKey: ['snapshots'],
			});
		},
	});
}

export function useDeleteLegalHold() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (snapshotId: string) => legalHoldsApi.delete(snapshotId),
		onSuccess: (_data, snapshotId) => {
			queryClient.invalidateQueries({
				queryKey: ['legal-holds'],
			});
			queryClient.invalidateQueries({
				queryKey: ['legal-holds', snapshotId],
			});
			queryClient.invalidateQueries({
				queryKey: ['snapshots'],
			});
		},
	});
}
