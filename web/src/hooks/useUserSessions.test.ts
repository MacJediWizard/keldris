import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useRevokeAllSessions,
	useRevokeSession,
	useUserSessions,
} from './useUserSessions';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useUserSessions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches user sessions', async () => {
		const fetchFn = mockFetch({ sessions: [{ id: 's-1' }] });

		const { result } = renderHook(() => useUserSessions(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 's-1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/user-sessions',
			expect.any(Object),
		);
	});
});

describe('useRevokeSession', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('revokes a session', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useRevokeSession(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('s-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/user-sessions/s-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useRevokeAllSessions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('revokes all sessions', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useRevokeAllSessions(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/user-sessions',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});
