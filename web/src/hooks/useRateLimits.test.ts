import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useBlockedRequests,
	useCreateIPBan,
	useCreateRateLimitConfig,
	useDeleteIPBan,
	useDeleteRateLimitConfig,
	useIPBans,
	useRateLimitConfig,
	useRateLimitConfigs,
	useRateLimitDashboard,
	useRateLimitStats,
	useUpdateRateLimitConfig,
} from './useRateLimits';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useRateLimitDashboard', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches dashboard stats', async () => {
		const stats = { total: 5 };
		const fetchFn = mockFetch(stats);

		const { result } = renderHook(() => useRateLimitDashboard(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.stats).toEqual(stats));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/rate-limits',
			expect.any(Object),
		);
	});
});

describe('useRateLimitConfigs', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches rate limit configs', async () => {
		const fetchFn = mockFetch({ configs: [{ id: 'rl-1' }] });

		const { result } = renderHook(() => useRateLimitConfigs(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'rl-1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/rate-limit-configs',
			expect.any(Object),
		);
	});
});

describe('useRateLimitConfig', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single rate limit config', async () => {
		mockFetch({ id: 'rl-1' });

		const { result } = renderHook(() => useRateLimitConfig('rl-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 'rl-1' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useRateLimitConfig(''), { wrapper: createWrapper() });

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useRateLimitStats', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches rate limit stats', async () => {
		mockFetch({ total: 3 });

		const { result } = renderHook(() => useRateLimitStats(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ total: 3 });
	});
});

describe('useBlockedRequests', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches blocked requests', async () => {
		mockFetch({ blocked: [] });

		const { result } = renderHook(() => useBlockedRequests(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ blocked: [] });
	});
});

describe('useCreateRateLimitConfig', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a rate limit config', async () => {
		const fetchFn = mockFetch({ id: 'new-rl' });

		const { result } = renderHook(() => useCreateRateLimitConfig(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'test',
			endpoint_pattern: '/api/x',
			method: 'GET',
			max_requests: 10,
			window_seconds: 60,
			enabled: true,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/rate-limit-configs',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useUpdateRateLimitConfig', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates a rate limit config', async () => {
		const fetchFn = mockFetch({ id: 'rl-1' });

		const { result } = renderHook(() => useUpdateRateLimitConfig(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'rl-1', data: { enabled: false } });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/rate-limit-configs/rl-1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useDeleteRateLimitConfig', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a rate limit config', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useDeleteRateLimitConfig(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('rl-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/rate-limit-configs/rl-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useIPBans', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches IP bans', async () => {
		mockFetch({ bans: [{ id: 'ban-1' }] });

		const { result } = renderHook(() => useIPBans(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'ban-1' }]);
	});
});

describe('useCreateIPBan', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates an IP ban', async () => {
		const fetchFn = mockFetch({ id: 'ban-1' });

		const { result } = renderHook(() => useCreateIPBan(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ ip_address: '1.2.3.4', reason: 'spam' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/ip-bans',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useDeleteIPBan', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes an IP ban', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useDeleteIPBan(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('ban-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/ip-bans/ban-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});
