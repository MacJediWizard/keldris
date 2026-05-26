import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAdminCreateOrganization,
	useAdminDeleteOrganization,
	useAdminOrgUsageStats,
	useAdminOrganization,
	useAdminOrganizations,
	useAdminTransferOwnership,
	useAdminUpdateOrganization,
} from './useAdminOrganizations';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useAdminOrganizations', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches admin organizations', async () => {
		mockFetch({ organizations: [], total: 0 });

		const { result } = renderHook(() => useAdminOrganizations(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ organizations: [], total: 0 });
	});
});

describe('useAdminOrganization', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single organization', async () => {
		mockFetch({ id: 'org1', name: 'Test' });

		const { result } = renderHook(() => useAdminOrganization('org1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useAdminOrganization(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useAdminOrgUsageStats', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches usage stats', async () => {
		mockFetch({ agent_count: 5 });

		const { result } = renderHook(() => useAdminOrgUsageStats('org1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAdminCreateOrganization', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates an organization', async () => {
		mockFetch({ id: 'org1', name: 'New' });

		const { result } = renderHook(() => useAdminCreateOrganization(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'New' } as Parameters<
			typeof result.current.mutate
		>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAdminUpdateOrganization', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates an organization', async () => {
		mockFetch({ id: 'org1', name: 'Updated' });

		const { result } = renderHook(() => useAdminUpdateOrganization(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			id: 'org1',
			data: {} as Parameters<typeof result.current.mutate>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAdminDeleteOrganization', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes an organization', async () => {
		mockFetch({ message: 'Deleted' });

		const { result } = renderHook(() => useAdminDeleteOrganization(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('org1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAdminTransferOwnership', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('transfers ownership', async () => {
		mockFetch({ message: 'Transferred' });

		const { result } = renderHook(() => useAdminTransferOwnership(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			orgId: 'org1',
			data: { new_owner_id: 'user1' },
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
