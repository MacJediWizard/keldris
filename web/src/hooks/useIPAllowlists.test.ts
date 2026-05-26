import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateIPAllowlist,
	useDeleteIPAllowlist,
	useIPAllowlist,
	useIPAllowlistSettings,
	useIPAllowlists,
	useIPBlockedAttempts,
	useUpdateIPAllowlist,
	useUpdateIPAllowlistSettings,
} from './useIPAllowlists';

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

describe('useIPAllowlists', () => {
	it('fetches allowlists', async () => {
		mockFetch({ allowlists: [] });
		const { result } = renderHook(() => useIPAllowlists(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useIPAllowlist', () => {
	it('fetches a single allowlist', async () => {
		mockFetch({ id: 'a1' });
		const { result } = renderHook(() => useIPAllowlist('a1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useIPAllowlist(''), { wrapper: createWrapper() });
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateIPAllowlist', () => {
	it('creates an allowlist', async () => {
		mockFetch({ id: 'a1' });
		const { result } = renderHook(() => useCreateIPAllowlist(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ cidr: '10.0.0.0/8' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateIPAllowlist', () => {
	it('updates an allowlist', async () => {
		mockFetch({ id: 'a1' });
		const { result } = renderHook(() => useUpdateIPAllowlist(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'a1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteIPAllowlist', () => {
	it('deletes an allowlist', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteIPAllowlist(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('a1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useIPAllowlistSettings', () => {
	it('fetches settings', async () => {
		mockFetch({ enabled: true });
		const { result } = renderHook(() => useIPAllowlistSettings(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateIPAllowlistSettings', () => {
	it('updates settings', async () => {
		mockFetch({ enabled: true });
		const { result } = renderHook(() => useUpdateIPAllowlistSettings(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ enabled: true } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useIPBlockedAttempts', () => {
	it('fetches blocked attempts', async () => {
		mockFetch({ attempts: [] });
		const { result } = renderHook(() => useIPBlockedAttempts(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
