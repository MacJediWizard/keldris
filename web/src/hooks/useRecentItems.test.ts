import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useClearRecentItems,
	useDeleteRecentItem,
	useRecentItems,
	useTrackRecentItem,
} from './useRecentItems';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useRecentItems', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches recent items without filter', async () => {
		const fetchFn = mockFetch({ items: [{ id: 'item-1' }] });

		const { result } = renderHook(() => useRecentItems(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'item-1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/recent-items',
			expect.any(Object),
		);
	});

	it('fetches recent items filtered by type', async () => {
		const fetchFn = mockFetch({ items: [] });

		const { result } = renderHook(() => useRecentItems('agent'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/recent-items?type=agent',
			expect.any(Object),
		);
	});
});

describe('useTrackRecentItem', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('tracks a recent item', async () => {
		const fetchFn = mockFetch({ id: 'item-1' });

		const { result } = renderHook(() => useTrackRecentItem(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			item_type: 'agent',
			item_id: 'agent-1',
			item_name: 'host',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/recent-items',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useDeleteRecentItem', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a recent item', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useDeleteRecentItem(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('item-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/recent-items/item-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useClearRecentItems', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('clears all recent items', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useClearRecentItems(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/recent-items',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});
