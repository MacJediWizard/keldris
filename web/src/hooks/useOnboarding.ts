import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { onboardingApi } from '../lib/api';
import type { OnboardingStep } from '../lib/types';

export function useOnboardingStatus() {
	return useQuery({
		queryKey: ['onboarding', 'status'],
		queryFn: onboardingApi.getStatus,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useCompleteOnboardingStep() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (step: OnboardingStep) => onboardingApi.completeStep(step),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['onboarding'] });
		},
	});
}

export function useSkipOnboarding() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => onboardingApi.skip(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['onboarding'] });
		},
	});
}
