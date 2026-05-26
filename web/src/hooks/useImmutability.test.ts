import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateImmutabilityLock,
	useExtendImmutabilityLock,
	useImmutabilityLock,
	useImmutabilityLocks,
	useRepositoryImmutabilityLocks,
	useRepositoryImmutabilitySettings,
	useSnapshotImmutabilityStatus,
	useUpdateRepositoryImmutabilitySettings,
} from './useImmutability';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.restoreAllMocks();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('useImmutabilityLocks', () => {
	it('fetches locks', async () => {
		mockFetch({ locks: [] });
		const { result } = renderHook(() => useImmutabilityLocks(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useImmutabilityLock', () => {
	it('fetches a single lock', async () => {
		mockFetch({ id: 'l1' });
		const { result } = renderHook(() => useImmutabilityLock('l1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useImmutabilityLock(''), { wrapper: createWrapper() });
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useRepositoryImmutabilityLocks', () => {
	it('fetches repository locks', async () => {
		mockFetch({ locks: [] });
		const { result } = renderHook(() => useRepositoryImmutabilityLocks('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSnapshotImmutabilityStatus', () => {
	it('fetches snapshot status', async () => {
		mockFetch({ snapshot_id: 's1', locked: false });
		const { result } = renderHook(
			() => useSnapshotImmutabilityStatus('s1', 'r1'),
			{ wrapper: createWrapper() },
		);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when ids are empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useSnapshotImmutabilityStatus('', ''), {
			wrapper: createWrapper(),
		});
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useRepositoryImmutabilitySettings', () => {
	it('fetches settings', async () => {
		mockFetch({ repository_id: 'r1' });
		const { result } = renderHook(
			() => useRepositoryImmutabilitySettings('r1'),
			{ wrapper: createWrapper() },
		);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateImmutabilityLock', () => {
	it('creates a lock', async () => {
		mockFetch({ id: 'l1' });
		const { result } = renderHook(() => useCreateImmutabilityLock(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			snapshot_id: 's1',
			repository_id: 'r1',
		} as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useExtendImmutabilityLock', () => {
	it('extends a lock', async () => {
		mockFetch({ id: 'l1' });
		const { result } = renderHook(() => useExtendImmutabilityLock(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'l1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateRepositoryImmutabilitySettings', () => {
	it('updates settings', async () => {
		mockFetch({ repository_id: 'r1' });
		const { result } = renderHook(
			() => useUpdateRepositoryImmutabilitySettings(),
			{ wrapper: createWrapper() },
		);
		result.current.mutate({ repositoryId: 'r1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
