import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useSearch } from './useSearch';

vi.mock('../lib/api', () => ({
	searchApi: {
		search: vi.fn(),
	},
}));

import { searchApi } from '../lib/api';

describe('useSearch', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('searches with a valid filter', async () => {
		const mockResponse = {
			results: [{ id: '1', type: 'agent', name: 'test' }],
			query: 'test',
			total: 1,
		};
		vi.mocked(searchApi.search).mockResolvedValue(mockResponse);

		const { result } = renderHook(() => useSearch({ q: 'test' }), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(searchApi.search).toHaveBeenCalledWith({ q: 'test' });
	});

	it('does not search when filter is null', () => {
		renderHook(() => useSearch(null), {
			wrapper: createWrapper(),
		});

		expect(searchApi.search).not.toHaveBeenCalled();
	});

	it('does not search when query is empty', () => {
		renderHook(() => useSearch({ q: '' }), {
			wrapper: createWrapper(),
		});

		expect(searchApi.search).not.toHaveBeenCalled();
	});
});
