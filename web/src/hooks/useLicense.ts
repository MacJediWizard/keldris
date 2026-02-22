import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { licenseApi } from '../lib/api';
import type { StartTrialRequest } from '../lib/types';

export function useLicense() {
	return useQuery({
		queryKey: ['license'],
		queryFn: licenseApi.getInfo,
		staleTime: 5 * 60 * 1000,
	});
}

export function useActivateLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (licenseKey: string) => licenseApi.activate(licenseKey),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['license'] });
		},
	});
}

export function useDeactivateLicense() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => licenseApi.deactivate(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['license'] });
		},
	});
}

export function usePricingPlans() {
	return useQuery({
		queryKey: ['pricing-plans'],
		queryFn: licenseApi.getPlans,
		staleTime: 30 * 60 * 1000, // 30 minutes
	});
}

export function useStartTrial() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: StartTrialRequest) => licenseApi.startTrial(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['license'] });
		},
	});
}

export function useCheckTrial(email: string) {
	return useQuery({
		queryKey: ['trial-check', email],
		queryFn: () => licenseApi.checkTrial(email),
		enabled: !!email,
		staleTime: 5 * 60 * 1000,
	});
}
