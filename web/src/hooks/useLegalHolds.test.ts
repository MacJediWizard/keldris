import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateLegalHold,
	useDeleteLegalHold,
	useLegalHold,
	useLegalHolds,
} from './useLegalHolds';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useLegalHolds', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches legal holds', async () => {
		const fetchFn = mockFetch({
			legal_holds: [{ id: 'h1', snapshot_id: 's1', reason: 'audit' }],
		});

		const { result } = renderHook(() => useLegalHolds(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([
			{ id: 'h1', snapshot_id: 's1', reason: 'audit' },
		]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/legal-holds',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useLegalHold', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single legal hold for a snapshot', async () => {
		const mockHold = { id: 'h1', snapshot_id: 'snap-1', reason: 'legal' };
		const fetchFn = mockFetch(mockHold);

		const { result } = renderHook(() => useLegalHold('snap-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockHold);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/snapshots/snap-1/hold',
			expect.objectContaining({ credentials: 'include' }),
		);
	});

	it('does not fetch when snapshotId is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useLegalHold(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateLegalHold', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a legal hold', async () => {
		const fetchFn = mockFetch({ id: 'h1', snapshot_id: 'snap-1' });

		const { result } = renderHook(() => useCreateLegalHold(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			snapshotId: 'snap-1',
			data: { reason: 'investigation' },
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/snapshots/snap-1/hold',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useDeleteLegalHold', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a legal hold', async () => {
		const fetchFn = mockFetch({ message: 'deleted' });

		const { result } = renderHook(() => useDeleteLegalHold(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('snap-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/snapshots/snap-1/hold',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});
