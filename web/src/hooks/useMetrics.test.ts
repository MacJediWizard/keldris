import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useBackupDurationTrend,
	useBackupSuccessRates,
	useDailyBackupStats,
	useDashboardStats,
	useStorageGrowthTrend,
} from './useMetrics';

vi.mock('../lib/api', () => ({
	metricsApi: {
		getDashboardStats: vi.fn(),
		getBackupSuccessRates: vi.fn(),
		getStorageGrowthTrend: vi.fn(),
		getBackupDurationTrend: vi.fn(),
		getDailyBackupStats: vi.fn(),
	},
}));

import { metricsApi } from '../lib/api';

describe('useDashboardStats', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches dashboard stats', async () => {
		const mockStats = {
			total_agents: 10,
			active_agents: 8,
			total_backups: 500,
			total_storage_bytes: 1024 * 1024 * 1024,
		};
		vi.mocked(metricsApi.getDashboardStats).mockResolvedValue(mockStats);

		const { result } = renderHook(() => useDashboardStats(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStats);
	});
});

describe('useBackupSuccessRates', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches success rates', async () => {
		const mockRates = {
			rate_7d: { total: 50, successful: 48, rate: 96 },
			rate_30d: { total: 200, successful: 190, rate: 95 },
		};
		vi.mocked(metricsApi.getBackupSuccessRates).mockResolvedValue(mockRates);

		const { result } = renderHook(() => useBackupSuccessRates(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockRates);
	});
});

describe('useStorageGrowthTrend', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches storage growth with default days', async () => {
		vi.mocked(metricsApi.getStorageGrowthTrend).mockResolvedValue([]);

		const { result } = renderHook(() => useStorageGrowthTrend(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(metricsApi.getStorageGrowthTrend).toHaveBeenCalledWith(30);
	});

	it('fetches storage growth with custom days', async () => {
		vi.mocked(metricsApi.getStorageGrowthTrend).mockResolvedValue([]);

		const { result } = renderHook(() => useStorageGrowthTrend(7), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(metricsApi.getStorageGrowthTrend).toHaveBeenCalledWith(7);
	});
});

describe('useBackupDurationTrend', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backup duration trend', async () => {
		vi.mocked(metricsApi.getBackupDurationTrend).mockResolvedValue([]);

		const { result } = renderHook(() => useBackupDurationTrend(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(metricsApi.getBackupDurationTrend).toHaveBeenCalledWith(30);
	});
});

describe('useDailyBackupStats', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches daily backup stats', async () => {
		vi.mocked(metricsApi.getDailyBackupStats).mockResolvedValue([]);

		const { result } = renderHook(() => useDailyBackupStats(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(metricsApi.getDailyBackupStats).toHaveBeenCalledWith(30);
	});
});
