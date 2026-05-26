import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateRegistrationCode,
	useDeleteRegistrationCode,
	usePendingRegistrations,
} from './useAgentRegistration';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('usePendingRegistrations', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches pending registrations', async () => {
		mockFetch({ registrations: [] });

		const { result } = renderHook(() => usePendingRegistrations(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([]);
	});
});

describe('useCreateRegistrationCode', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a registration code', async () => {
		mockFetch({ code: 'abc123' });

		const { result } = renderHook(() => useCreateRegistrationCode(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ expires_in_hours: 24 } as Parameters<
			typeof result.current.mutate
		>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteRegistrationCode', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a registration code', async () => {
		mockFetch({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteRegistrationCode(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('code-id');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
