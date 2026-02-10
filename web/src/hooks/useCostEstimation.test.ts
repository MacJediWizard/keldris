import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCostAlert,
	useCostAlerts,
	useCostForecast,
	useCostHistory,
	useCostSummary,
	useCreateCostAlert,
	useCreatePricing,
	useDefaultPricing,
	useDeleteCostAlert,
	useDeletePricing,
	usePricing,
	useRepositoryCost,
	useRepositoryCosts,
} from './useCostEstimation';

vi.mock('../lib/api', () => ({
	costsApi: {
		getSummary: vi.fn(),
		listRepositoryCosts: vi.fn(),
		getRepositoryCost: vi.fn(),
		getForecast: vi.fn(),
		getHistory: vi.fn(),
	},
	pricingApi: {
		list: vi.fn(),
		getDefaults: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
	costAlertsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
}));

import { costAlertsApi, costsApi, pricingApi } from '../lib/api';

describe('useCostSummary', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches cost summary', async () => {
		vi.mocked(costsApi.getSummary).mockResolvedValue({ total: 100 });
		const { result } = renderHook(() => useCostSummary(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRepositoryCosts', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches repository costs', async () => {
		vi.mocked(costsApi.listRepositoryCosts).mockResolvedValue({ costs: [] });
		const { result } = renderHook(() => useRepositoryCosts(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRepositoryCost', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches a repo cost', async () => {
		vi.mocked(costsApi.getRepositoryCost).mockResolvedValue({ cost: 10 });
		const { result } = renderHook(() => useRepositoryCost('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
	it('does not fetch when id is empty', () => {
		renderHook(() => useRepositoryCost(''), { wrapper: createWrapper() });
		expect(costsApi.getRepositoryCost).not.toHaveBeenCalled();
	});
});

describe('useCostForecast', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches forecast', async () => {
		vi.mocked(costsApi.getForecast).mockResolvedValue({ forecast: [] });
		const { result } = renderHook(() => useCostForecast(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(costsApi.getForecast).toHaveBeenCalledWith(30);
	});
});

describe('useCostHistory', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches history', async () => {
		vi.mocked(costsApi.getHistory).mockResolvedValue({ history: [] });
		const { result } = renderHook(() => useCostHistory(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('usePricing', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches pricing', async () => {
		vi.mocked(pricingApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => usePricing(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDefaultPricing', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches default pricing', async () => {
		vi.mocked(pricingApi.getDefaults).mockResolvedValue({ defaults: [] });
		const { result } = renderHook(() => useDefaultPricing(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreatePricing', () => {
	beforeEach(() => vi.clearAllMocks());
	it('creates pricing', async () => {
		vi.mocked(pricingApi.create).mockResolvedValue({ id: 'p1' });
		const { result } = renderHook(() => useCreatePricing(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ type: 's3', price_per_gb: 0.023 } as Parameters<
			typeof pricingApi.create
		>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeletePricing', () => {
	beforeEach(() => vi.clearAllMocks());
	it('deletes pricing', async () => {
		vi.mocked(pricingApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeletePricing(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('p1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCostAlerts', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches cost alerts', async () => {
		vi.mocked(costAlertsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useCostAlerts(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCostAlert', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches a cost alert', async () => {
		vi.mocked(costAlertsApi.get).mockResolvedValue({ id: 'ca1' });
		const { result } = renderHook(() => useCostAlert('ca1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
	it('does not fetch when id is empty', () => {
		renderHook(() => useCostAlert(''), { wrapper: createWrapper() });
		expect(costAlertsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateCostAlert', () => {
	beforeEach(() => vi.clearAllMocks());
	it('creates a cost alert', async () => {
		vi.mocked(costAlertsApi.create).mockResolvedValue({ id: 'ca1' });
		const { result } = renderHook(() => useCreateCostAlert(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ threshold: 100 } as Parameters<
			typeof costAlertsApi.create
		>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteCostAlert', () => {
	beforeEach(() => vi.clearAllMocks());
	it('deletes a cost alert', async () => {
		vi.mocked(costAlertsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteCostAlert(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('ca1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
