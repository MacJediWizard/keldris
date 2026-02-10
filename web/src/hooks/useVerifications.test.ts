import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useVerificationStatus,
	useVerifications,
	useVerification,
	useTriggerVerification,
	useVerificationSchedules,
	useCreateVerificationSchedule,
	useDeleteVerificationSchedule,
} from './useVerifications';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	verificationsApi: {
		getStatus: vi.fn(),
		listByRepository: vi.fn(),
		get: vi.fn(),
		trigger: vi.fn(),
		listSchedules: vi.fn(),
		createSchedule: vi.fn(),
		getSchedule: vi.fn(),
		updateSchedule: vi.fn(),
		deleteSchedule: vi.fn(),
	},
}));

import { verificationsApi } from '../lib/api';

describe('useVerificationStatus', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches status', async () => {
		vi.mocked(verificationsApi.getStatus).mockResolvedValue({ status: 'verified' });
		const { result } = renderHook(() => useVerificationStatus('r1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useVerificationStatus(''), { wrapper: createWrapper() });
		expect(verificationsApi.getStatus).not.toHaveBeenCalled();
	});
});

describe('useVerifications', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches verifications', async () => {
		vi.mocked(verificationsApi.listByRepository).mockResolvedValue([]);
		const { result } = renderHook(() => useVerifications('r1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useVerification', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a verification', async () => {
		vi.mocked(verificationsApi.get).mockResolvedValue({ id: 'v1' });
		const { result } = renderHook(() => useVerification('v1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useVerification(''), { wrapper: createWrapper() });
		expect(verificationsApi.get).not.toHaveBeenCalled();
	});
});

describe('useTriggerVerification', () => {
	beforeEach(() => vi.clearAllMocks());

	it('triggers verification', async () => {
		vi.mocked(verificationsApi.trigger).mockResolvedValue({ id: 'v1' });
		const { result } = renderHook(() => useTriggerVerification(), { wrapper: createWrapper() });
		result.current.mutate({ repoId: 'r1', data: { type: 'full' } as Parameters<typeof verificationsApi.trigger>[1] });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useVerificationSchedules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches schedules', async () => {
		vi.mocked(verificationsApi.listSchedules).mockResolvedValue([]);
		const { result } = renderHook(() => useVerificationSchedules('r1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateVerificationSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a schedule', async () => {
		vi.mocked(verificationsApi.createSchedule).mockResolvedValue({ id: 'vs1' });
		const { result } = renderHook(() => useCreateVerificationSchedule(), { wrapper: createWrapper() });
		result.current.mutate({ repoId: 'r1', data: { cron: '0 0 * * 0' } as Parameters<typeof verificationsApi.createSchedule>[1] });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteVerificationSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a schedule', async () => {
		vi.mocked(verificationsApi.deleteSchedule).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteVerificationSchedule(), { wrapper: createWrapper() });
		result.current.mutate('vs1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
