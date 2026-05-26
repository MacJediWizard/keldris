import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAddFavorite,
	useFavoriteIds,
	useFavorites,
	useIsFavorite,
	useRemoveFavorite,
} from './useFavorites';

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

describe('useFavorites', () => {
	it('fetches favorites', async () => {
		mockFetch({
			favorites: [{ id: 'f1', entity_type: 'agent', entity_id: 'a1' }],
		});
		const { result } = renderHook(() => useFavorites(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAddFavorite', () => {
	it('adds a favorite', async () => {
		mockFetch({ id: 'f1', entity_type: 'agent', entity_id: 'a1' });
		const { result } = renderHook(() => useAddFavorite(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ entity_type: 'agent', entity_id: 'a1' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRemoveFavorite', () => {
	it('removes a favorite', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useRemoveFavorite(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ entityType: 'agent' as never, entityId: 'a1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useIsFavorite', () => {
	it('returns false when entity is not favorited', async () => {
		mockFetch({ favorites: [] });
		const { result } = renderHook(() => useIsFavorite('agent' as never, 'a1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current).toBe(false));
	});
});

describe('useFavoriteIds', () => {
	it('returns a Set of favorite ids', async () => {
		mockFetch({
			favorites: [{ id: 'f1', entity_type: 'agent', entity_id: 'a1' }],
		});
		const { result } = renderHook(() => useFavoriteIds('agent' as never), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.has('a1')).toBe(true));
	});
});
