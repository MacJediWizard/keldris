import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCancelDRTest,
	useDRTest,
	useDRTests,
	useRunDRTest,
} from './useDRTests';

vi.mock('../lib/api', () => ({
	drTestsApi: {
		list: vi.fn(),
		get: vi.fn(),
		run: vi.fn(),
		cancel: vi.fn(),
	},
}));

import { drTestsApi } from '../lib/api';

describe('useDRTests', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches DR tests', async () => {
		vi.mocked(drTestsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useDRTests(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('fetches with params', async () => {
		vi.mocked(drTestsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useDRTests({ runbook_id: 'rb1' }), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(drTestsApi.list).toHaveBeenCalledWith({ runbook_id: 'rb1' });
	});
});

describe('useDRTest', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a test', async () => {
		vi.mocked(drTestsApi.get).mockResolvedValue({ id: 'dt1' });
		const { result } = renderHook(() => useDRTest('dt1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDRTest(''), { wrapper: createWrapper() });
		expect(drTestsApi.get).not.toHaveBeenCalled();
	});
});

describe('useRunDRTest', () => {
	beforeEach(() => vi.clearAllMocks());

	it('runs a DR test', async () => {
		vi.mocked(drTestsApi.run).mockResolvedValue({ id: 'dt1' });
		const { result } = renderHook(() => useRunDRTest(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ runbook_id: 'rb1' } as Parameters<
			typeof drTestsApi.run
		>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCancelDRTest', () => {
	beforeEach(() => vi.clearAllMocks());

	it('cancels a DR test', async () => {
		vi.mocked(drTestsApi.cancel).mockResolvedValue({
			id: 'dt1',
			status: 'cancelled',
		});
		const { result } = renderHook(() => useCancelDRTest(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'dt1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
