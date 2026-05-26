import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateLifecyclePolicy,
	useDeleteLifecyclePolicy,
	useLifecycleDeletions,
	useLifecycleDryRun,
	useLifecyclePolicies,
	useLifecyclePolicy,
	useLifecyclePreview,
	useOrgLifecycleDeletions,
	useUpdateLifecyclePolicy,
} from './useLifecyclePolicies';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useLifecyclePolicies', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches lifecycle policies', async () => {
		const fetchFn = mockFetch({ policies: [{ id: 'lp-1', name: 'daily' }] });

		const { result } = renderHook(() => useLifecyclePolicies(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'lp-1', name: 'daily' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useLifecyclePolicy', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single lifecycle policy', async () => {
		const mockPolicy = { id: 'lp-1', name: 'monthly' };
		mockFetch(mockPolicy);

		const { result } = renderHook(() => useLifecyclePolicy('lp-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockPolicy);
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useLifecyclePolicy(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateLifecyclePolicy', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a lifecycle policy', async () => {
		const fetchFn = mockFetch({ id: 'lp-new' });

		const { result } = renderHook(() => useCreateLifecyclePolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'weekly' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useUpdateLifecyclePolicy', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates a lifecycle policy', async () => {
		const fetchFn = mockFetch({ id: 'lp-1' });

		const { result } = renderHook(() => useUpdateLifecyclePolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'lp-1', data: { name: 'renamed' } });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/lp-1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useDeleteLifecyclePolicy', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a lifecycle policy', async () => {
		const fetchFn = mockFetch({ message: 'deleted' });

		const { result } = renderHook(() => useDeleteLifecyclePolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('lp-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/lp-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useLifecycleDryRun', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('runs dry-run for a policy', async () => {
		const fetchFn = mockFetch({ deletions: [] });

		const { result } = renderHook(() => useLifecycleDryRun(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('lp-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/lp-1/dry-run',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useLifecyclePreview', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('previews a lifecycle policy', async () => {
		const fetchFn = mockFetch({ deletions: [] });

		const { result } = renderHook(() => useLifecyclePreview(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			retention_days: 30,
		} as Parameters<typeof result.current.mutate>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/preview',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useLifecycleDeletions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches deletions for a policy', async () => {
		const fetchFn = mockFetch({ events: [{ id: 'e1' }] });

		const { result } = renderHook(() => useLifecycleDeletions('lp-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'e1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/lp-1/deletions',
			expect.objectContaining({ credentials: 'include' }),
		);
	});

	it('does not fetch when policyId is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useLifecycleDeletions(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useOrgLifecycleDeletions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches org-wide deletions', async () => {
		const fetchFn = mockFetch({ events: [] });

		const { result } = renderHook(() => useOrgLifecycleDeletions(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/lifecycle-policies/deletions',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});
