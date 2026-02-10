import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	usePolicies,
	usePolicy,
	usePolicySchedules,
	useCreatePolicy,
	useDeletePolicy,
	useApplyPolicy,
} from './usePolicies';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	policiesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		listSchedules: vi.fn(),
		apply: vi.fn(),
	},
}));

import { policiesApi } from '../lib/api';

describe('usePolicies', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches policies', async () => {
		vi.mocked(policiesApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => usePolicies(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('usePolicy', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a policy', async () => {
		vi.mocked(policiesApi.get).mockResolvedValue({ id: 'p1', name: 'Default' });
		const { result } = renderHook(() => usePolicy('p1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => usePolicy(''), { wrapper: createWrapper() });
		expect(policiesApi.get).not.toHaveBeenCalled();
	});
});

describe('usePolicySchedules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches schedules for a policy', async () => {
		vi.mocked(policiesApi.listSchedules).mockResolvedValue([]);
		const { result } = renderHook(() => usePolicySchedules('p1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreatePolicy', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a policy', async () => {
		vi.mocked(policiesApi.create).mockResolvedValue({ id: 'p1' });
		const { result } = renderHook(() => useCreatePolicy(), { wrapper: createWrapper() });
		result.current.mutate({ name: 'New Policy' } as Parameters<typeof policiesApi.create>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeletePolicy', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a policy', async () => {
		vi.mocked(policiesApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeletePolicy(), { wrapper: createWrapper() });
		result.current.mutate('p1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useApplyPolicy', () => {
	beforeEach(() => vi.clearAllMocks());

	it('applies a policy', async () => {
		vi.mocked(policiesApi.apply).mockResolvedValue({ applied_count: 3 });
		const { result } = renderHook(() => useApplyPolicy(), { wrapper: createWrapper() });
		result.current.mutate({ id: 'p1', data: { schedule_ids: ['s1'] } });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
