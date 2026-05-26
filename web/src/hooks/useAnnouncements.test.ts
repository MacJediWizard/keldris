import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActiveAnnouncements,
	useAnnouncement,
	useAnnouncements,
	useCreateAnnouncement,
	useDeleteAnnouncement,
	useDismissAnnouncement,
	useUpdateAnnouncement,
} from './useAnnouncements';

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

describe('useAnnouncements', () => {
	it('fetches announcements list', async () => {
		mockFetch({ announcements: [{ id: 'a1', title: 'Hello' }] });

		const { result } = renderHook(() => useAnnouncements(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'a1', title: 'Hello' }]);
	});
});

describe('useAnnouncement', () => {
	it('fetches single announcement', async () => {
		mockFetch({ id: 'a1', title: 'Hello' });

		const { result } = renderHook(() => useAnnouncement('a1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 'a1', title: 'Hello' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useAnnouncement(''), { wrapper: createWrapper() });

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useActiveAnnouncements', () => {
	it('fetches active announcements', async () => {
		mockFetch({ announcements: [] });

		const { result } = renderHook(() => useActiveAnnouncements(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([]);
	});
});

describe('useCreateAnnouncement', () => {
	it('creates announcement', async () => {
		mockFetch({ id: 'a1', title: 'New' });

		const { result } = renderHook(() => useCreateAnnouncement(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ title: 'New' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateAnnouncement', () => {
	it('updates announcement', async () => {
		mockFetch({ id: 'a1', title: 'Updated' });

		const { result } = renderHook(() => useUpdateAnnouncement(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'a1', data: { title: 'Updated' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteAnnouncement', () => {
	it('deletes announcement', async () => {
		mockFetch({ message: 'deleted' });

		const { result } = renderHook(() => useDeleteAnnouncement(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('a1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDismissAnnouncement', () => {
	it('dismisses announcement', async () => {
		mockFetch({ message: 'dismissed' });

		const { result } = renderHook(() => useDismissAnnouncement(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('a1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
