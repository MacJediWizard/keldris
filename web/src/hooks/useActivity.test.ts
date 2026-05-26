import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActivity,
	useActivityCategories,
	useActivityCount,
	useActivitySearch,
	useRecentActivity,
} from './useActivity';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useActivity', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches activity events', async () => {
		mockFetch({ events: [], total: 0 });

		const { result } = renderHook(() => useActivity(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([]);
	});
});

describe('useRecentActivity', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches recent activity', async () => {
		mockFetch({ events: [] });

		const { result } = renderHook(() => useRecentActivity(10), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useActivityCount', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches activity count', async () => {
		mockFetch({ count: 42 });

		const { result } = renderHook(() => useActivityCount(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(42);
	});
});

describe('useActivityCategories', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches categories', async () => {
		mockFetch({ categories: [] });

		const { result } = renderHook(() => useActivityCategories(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useActivitySearch', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('searches activity when query is non-empty', async () => {
		mockFetch({ events: [] });

		const { result } = renderHook(() => useActivitySearch('test'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when query is empty', () => {
		const fetchFn = mockFetch({ events: [] });

		renderHook(() => useActivitySearch(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});
