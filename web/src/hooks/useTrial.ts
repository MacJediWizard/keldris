import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { trialApi } from '../lib/api';
import type {
	ConvertTrialRequest,
	ExtendTrialRequest,
	StartTrialRequest,
} from '../lib/types';

// Get current trial status
export function useTrialStatus() {
	return useQuery({
		queryKey: ['trial-status'],
		queryFn: () => trialApi.getStatus(),
		staleTime: 5 * 60 * 1000, // 5 minutes
		retry: 1,
	});
}

// Start a new trial
export function useStartTrial() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: StartTrialRequest) => trialApi.startTrial(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['trial-status'] });
		},
	});
}

// Get available Pro features
export function useProFeatures() {
	return useQuery({
		queryKey: ['trial-features'],
		queryFn: () => trialApi.getFeatures(),
		staleTime: 10 * 60 * 1000, // 10 minutes
	});
}

// Get trial activity log
export function useTrialActivity(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['trial-activity', limit, offset],
		queryFn: () => trialApi.getActivity(limit, offset),
		staleTime: 60 * 1000, // 1 minute
	});
}

// Extend trial (admin/superuser only)
export function useExtendTrial() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: ExtendTrialRequest) => trialApi.extendTrial(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['trial-status'] });
			queryClient.invalidateQueries({ queryKey: ['trial-extensions'] });
		},
	});
}

// Convert trial to paid
export function useConvertTrial() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: ConvertTrialRequest) => trialApi.convertTrial(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['trial-status'] });
			queryClient.invalidateQueries({ queryKey: ['trial-features'] });
		},
	});
}

// Get extension history
export function useTrialExtensions() {
	return useQuery({
		queryKey: ['trial-extensions'],
		queryFn: () => trialApi.getExtensions(),
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}
