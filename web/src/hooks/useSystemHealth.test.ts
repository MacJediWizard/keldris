import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useSystemHealth, useSystemHealthHistory } from './useSystemHealth';

vi.mock('../lib/api', () => ({
	systemHealthApi: {
		getHealth: vi.fn(),
		getHistory: vi.fn(),
	},
}));

import { systemHealthApi } from '../lib/api';

describe('useSystemHealth', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches system health', async () => {
		vi.mocked(systemHealthApi.getHealth).mockResolvedValue({ status: 'ok' });

		const { result } = renderHook(() => useSystemHealth(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ status: 'ok' });
		expect(systemHealthApi.getHealth).toHaveBeenCalledOnce();
	});

	it('handles error state', async () => {
		vi.mocked(systemHealthApi.getHealth).mockRejectedValue(
			new Error('Network error'),
		);

		const { result } = renderHook(() => useSystemHealth(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error).toBeDefined();
	});
});

describe('useSystemHealthHistory', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches history', async () => {
		vi.mocked(systemHealthApi.getHistory).mockResolvedValue([]);

		const { result } = renderHook(() => useSystemHealthHistory(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(systemHealthApi.getHistory).toHaveBeenCalledOnce();
	});
});
