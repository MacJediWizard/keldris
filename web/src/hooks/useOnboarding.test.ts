import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCompleteOnboardingStep,
	useOnboardingStatus,
	useSkipOnboarding,
} from './useOnboarding';

vi.mock('../lib/api', () => ({
	onboardingApi: {
		getStatus: vi.fn(),
		completeStep: vi.fn(),
		skip: vi.fn(),
	},
}));

import { onboardingApi } from '../lib/api';

const mockStatus = {
	completed: false,
	skipped: false,
	completed_steps: ['create_org'],
	current_step: 'add_repository',
};

describe('useOnboardingStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches onboarding status', async () => {
		vi.mocked(onboardingApi.getStatus).mockResolvedValue(mockStatus);

		const { result } = renderHook(() => useOnboardingStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStatus);
	});
});

describe('useCompleteOnboardingStep', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('completes a step', async () => {
		vi.mocked(onboardingApi.completeStep).mockResolvedValue({
			...mockStatus,
			completed_steps: ['create_org', 'add_repository'],
		});

		const { result } = renderHook(() => useCompleteOnboardingStep(), {
			wrapper: createWrapper(),
		});

		result.current.mutate(
			'add_repository' as Parameters<typeof onboardingApi.completeStep>[0],
		);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSkipOnboarding', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('skips onboarding', async () => {
		vi.mocked(onboardingApi.skip).mockResolvedValue({
			...mockStatus,
			skipped: true,
		});

		const { result } = renderHook(() => useSkipOnboarding(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(onboardingApi.skip).toHaveBeenCalled();
	});
});
