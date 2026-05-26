import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useFileSearch } from './useFileSearch';

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

describe('useFileSearch', () => {
	it('fetches file search results', async () => {
		mockFetch({
			query: 'foo',
			agent_id: 'a1',
			repository_id: 'r1',
			total_count: 0,
			snapshot_count: 0,
			snapshots: [],
		});
		const { result } = renderHook(
			() =>
				useFileSearch({
					q: 'foo',
					agent_id: 'a1',
					repository_id: 'r1',
				}),
			{ wrapper: createWrapper() },
		);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when params are null', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useFileSearch(null), { wrapper: createWrapper() });
		expect(fetchFn).not.toHaveBeenCalled();
	});
});
