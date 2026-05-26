import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateSavedFilter,
	useDeleteSavedFilter,
	useSavedFilter,
	useSavedFilters,
	useUpdateSavedFilter,
} from './useSavedFilters';

vi.mock('../lib/api', () => ({
	savedFiltersApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
}));

import { savedFiltersApi } from '../lib/api';

describe('useSavedFilters', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists saved filters', async () => {
		vi.mocked(savedFiltersApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useSavedFilters(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(savedFiltersApi.list).toHaveBeenCalledWith(undefined);
	});

	it('passes entityType', async () => {
		vi.mocked(savedFiltersApi.list).mockResolvedValue([]);

		renderHook(() => useSavedFilters('backup'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => {
			expect(savedFiltersApi.list).toHaveBeenCalledWith('backup');
		});
	});
});

describe('useSavedFilter', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a filter', async () => {
		vi.mocked(savedFiltersApi.get).mockResolvedValue({ id: 'f-1' });

		const { result } = renderHook(() => useSavedFilter('f-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(savedFiltersApi.get).toHaveBeenCalledWith('f-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSavedFilter(''), { wrapper: createWrapper() });
		expect(savedFiltersApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateSavedFilter', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a filter', async () => {
		vi.mocked(savedFiltersApi.create).mockResolvedValue({
			id: 'new',
			entity_type: 'backup',
		});

		const { result } = renderHook(() => useCreateSavedFilter(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'f', entity_type: 'backup' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(savedFiltersApi.create).toHaveBeenCalled();
	});
});

describe('useUpdateSavedFilter', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a filter', async () => {
		vi.mocked(savedFiltersApi.update).mockResolvedValue({ id: 'f-1' });

		const { result } = renderHook(() => useUpdateSavedFilter(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'f-1', data: { name: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(savedFiltersApi.update).toHaveBeenCalledWith('f-1', { name: 'x' });
	});
});

describe('useDeleteSavedFilter', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a filter', async () => {
		vi.mocked(savedFiltersApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteSavedFilter(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('f-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(savedFiltersApi.delete).toHaveBeenCalledWith('f-1');
	});
});
