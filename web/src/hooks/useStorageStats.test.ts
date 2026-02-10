import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useRepositoryGrowth,
	useRepositoryHistory,
	useRepositoryStats,
	useRepositoryStatsList,
	useStorageGrowth,
	useStorageStatsSummary,
} from './useStorageStats';

vi.mock('../lib/api', () => ({
	statsApi: {
		getSummary: vi.fn(),
		getGrowth: vi.fn(),
		listRepositoryStats: vi.fn(),
		getRepositoryStats: vi.fn(),
		getRepositoryGrowth: vi.fn(),
		getRepositoryHistory: vi.fn(),
	},
}));

import { statsApi } from '../lib/api';

describe('useStorageStatsSummary', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches storage summary', async () => {
		vi.mocked(statsApi.getSummary).mockResolvedValue({ total_size: 1024 });
		const { result } = renderHook(() => useStorageStatsSummary(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useStorageGrowth', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches growth with default days', async () => {
		vi.mocked(statsApi.getGrowth).mockResolvedValue([]);
		const { result } = renderHook(() => useStorageGrowth(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(statsApi.getGrowth).toHaveBeenCalledWith(30);
	});

	it('fetches growth with custom days', async () => {
		vi.mocked(statsApi.getGrowth).mockResolvedValue([]);
		const { result } = renderHook(() => useStorageGrowth(7), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(statsApi.getGrowth).toHaveBeenCalledWith(7);
	});
});

describe('useRepositoryStatsList', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches repository stats list', async () => {
		vi.mocked(statsApi.listRepositoryStats).mockResolvedValue([]);
		const { result } = renderHook(() => useRepositoryStatsList(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRepositoryStats', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches repo stats', async () => {
		vi.mocked(statsApi.getRepositoryStats).mockResolvedValue({ id: 'r1' });
		const { result } = renderHook(() => useRepositoryStats('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useRepositoryStats(''), { wrapper: createWrapper() });
		expect(statsApi.getRepositoryStats).not.toHaveBeenCalled();
	});
});

describe('useRepositoryGrowth', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches repo growth', async () => {
		vi.mocked(statsApi.getRepositoryGrowth).mockResolvedValue({ growth: [] });
		const { result } = renderHook(() => useRepositoryGrowth('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(statsApi.getRepositoryGrowth).toHaveBeenCalledWith('r1', 30);
	});
});

describe('useRepositoryHistory', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches repo history', async () => {
		vi.mocked(statsApi.getRepositoryHistory).mockResolvedValue({ history: [] });
		const { result } = renderHook(() => useRepositoryHistory('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(statsApi.getRepositoryHistory).toHaveBeenCalledWith('r1', 30);
	});
});
