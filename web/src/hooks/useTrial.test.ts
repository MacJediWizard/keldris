import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useConvertTrial,
	useExtendTrial,
	useProFeatures,
	useStartTrial,
	useTrialActivity,
	useTrialExtensions,
	useTrialStatus,
} from './useTrial';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useTrialStatus', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches trial status', async () => {
		const fetchFn = mockFetch({ active: true });

		const { result } = renderHook(() => useTrialStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true), {
			timeout: 5000,
		});
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/status',
			expect.any(Object),
		);
	});
});

describe('useStartTrial', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('starts a trial', async () => {
		const fetchFn = mockFetch({ active: true });

		const { result } = renderHook(() => useStartTrial(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ tier: 'pro' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/start',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useProFeatures', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches pro features', async () => {
		const fetchFn = mockFetch({ features: [] });

		const { result } = renderHook(() => useProFeatures(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/features',
			expect.any(Object),
		);
	});
});

describe('useTrialActivity', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches trial activity with defaults', async () => {
		const fetchFn = mockFetch({ activities: [] });

		const { result } = renderHook(() => useTrialActivity(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/activity?limit=50&offset=0',
			expect.any(Object),
		);
	});
});

describe('useExtendTrial', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('extends a trial', async () => {
		const fetchFn = mockFetch({ id: 'ext-1' });

		const { result } = renderHook(() => useExtendTrial(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ days: 7, reason: 'request' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/extend',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useConvertTrial', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('converts a trial', async () => {
		const fetchFn = mockFetch({ active: false });

		const { result } = renderHook(() => useConvertTrial(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ license_key: 'KEY-XYZ' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/convert',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useTrialExtensions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches trial extensions', async () => {
		const fetchFn = mockFetch({ extensions: [] });

		const { result } = renderHook(() => useTrialExtensions(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/trial/extensions',
			expect.any(Object),
		);
	});
});
