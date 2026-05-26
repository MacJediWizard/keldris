import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateGeoReplicationConfig,
	useDeleteGeoReplicationConfig,
	useGeoReplicationConfig,
	useGeoReplicationConfigs,
	useGeoReplicationEvents,
	useGeoReplicationRegions,
	useGeoReplicationSummary,
	useRepositoryReplicationStatus,
	useSetRepositoryRegion,
	useTriggerReplication,
	useUpdateGeoReplicationConfig,
} from './useGeoReplication';

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

describe('useGeoReplicationRegions', () => {
	it('fetches regions', async () => {
		mockFetch({ regions: [], pairs: [] });
		const { result } = renderHook(() => useGeoReplicationRegions(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useGeoReplicationConfigs', () => {
	it('fetches configs', async () => {
		mockFetch({ configs: [] });
		const { result } = renderHook(() => useGeoReplicationConfigs(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useGeoReplicationConfig', () => {
	it('fetches a single config', async () => {
		mockFetch({ id: 'c1' });
		const { result } = renderHook(() => useGeoReplicationConfig('c1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useGeoReplicationConfig(''), {
			wrapper: createWrapper(),
		});
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useGeoReplicationSummary', () => {
	it('fetches summary', async () => {
		mockFetch({ total: 0 });
		const { result } = renderHook(() => useGeoReplicationSummary(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useGeoReplicationEvents', () => {
	it('fetches events', async () => {
		mockFetch({ events: [] });
		const { result } = renderHook(() => useGeoReplicationEvents('c1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRepositoryReplicationStatus', () => {
	it('fetches repository status', async () => {
		mockFetch({ repository_id: 'r1' });
		const { result } = renderHook(() => useRepositoryReplicationStatus('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateGeoReplicationConfig', () => {
	it('creates a config', async () => {
		mockFetch({ id: 'c1' });
		const { result } = renderHook(() => useCreateGeoReplicationConfig(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ name: 'test' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateGeoReplicationConfig', () => {
	it('updates a config', async () => {
		mockFetch({ id: 'c1' });
		const { result } = renderHook(() => useUpdateGeoReplicationConfig(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'c1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteGeoReplicationConfig', () => {
	it('deletes a config', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteGeoReplicationConfig(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('c1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useTriggerReplication', () => {
	it('triggers replication', async () => {
		mockFetch({ message: 'ok' });
		const { result } = renderHook(() => useTriggerReplication(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('c1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSetRepositoryRegion', () => {
	it('sets a repository region', async () => {
		mockFetch({ message: 'ok' });
		const { result } = renderHook(() => useSetRepositoryRegion(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ repoId: 'r1', region: 'us-east-1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
