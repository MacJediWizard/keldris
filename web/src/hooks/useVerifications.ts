import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { verificationsApi } from '../lib/api';
import type {
	CreateVerificationScheduleRequest,
	TriggerVerificationRequest,
	UpdateVerificationScheduleRequest,
	VerificationType,
} from '../lib/types';

export function useVerificationStatus(repoId: string) {
	return useQuery({
		queryKey: ['verification-status', repoId],
		queryFn: () => verificationsApi.getStatus(repoId),
		enabled: !!repoId,
		staleTime: 30 * 1000, // 30 seconds
		refetchInterval: 60 * 1000, // Refresh every minute
		retry: (failureCount, error) => {
			// Don't retry on 404 (verification not configured)
			if (
				error instanceof Error &&
				'status' in error &&
				(error as { status: number }).status === 404
			)
				return false;
			return failureCount < 3;
		},
	});
}

export function useVerifications(repoId: string) {
	return useQuery({
		queryKey: ['verifications', repoId],
		queryFn: () => verificationsApi.listByRepository(repoId),
		enabled: !!repoId,
		staleTime: 30 * 1000,
	});
}

export function useVerification(id: string) {
	return useQuery({
		queryKey: ['verifications', 'detail', id],
		queryFn: () => verificationsApi.get(id),
		enabled: !!id,
	});
}

export function useTriggerVerification() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			repoId,
			type,
		}: {
			repoId: string;
			type: VerificationType;
		}) => {
			const data: TriggerVerificationRequest = { type };
			return verificationsApi.trigger(repoId, data);
		},
		onSuccess: (_, { repoId }) => {
			queryClient.invalidateQueries({ queryKey: ['verifications', repoId] });
			queryClient.invalidateQueries({
				queryKey: ['verification-status', repoId],
			});
		},
	});
}

export function useVerificationSchedules(repoId: string) {
	return useQuery({
		queryKey: ['verification-schedules', repoId],
		queryFn: () => verificationsApi.listSchedules(repoId),
		enabled: !!repoId,
		staleTime: 60 * 1000,
	});
}

export function useCreateVerificationSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			repoId,
			data,
		}: {
			repoId: string;
			data: CreateVerificationScheduleRequest;
		}) => verificationsApi.createSchedule(repoId, data),
		onSuccess: (_, { repoId }) => {
			queryClient.invalidateQueries({
				queryKey: ['verification-schedules', repoId],
			});
			queryClient.invalidateQueries({
				queryKey: ['verification-status', repoId],
			});
		},
	});
}

export function useUpdateVerificationSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			repoId: string;
			data: UpdateVerificationScheduleRequest;
		}) => verificationsApi.updateSchedule(id, data),
		onSuccess: (_, { repoId }) => {
			queryClient.invalidateQueries({
				queryKey: ['verification-schedules', repoId],
			});
			queryClient.invalidateQueries({
				queryKey: ['verification-status', repoId],
			});
		},
	});
}

export function useDeleteVerificationSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id }: { id: string; repoId: string }) =>
			verificationsApi.deleteSchedule(id),
		onSuccess: (_, { repoId }) => {
			queryClient.invalidateQueries({
				queryKey: ['verification-schedules', repoId],
			});
			queryClient.invalidateQueries({
				queryKey: ['verification-status', repoId],
			});
		},
	});
}
