import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useCreateRestore, useRestore, useRestores } from './useRestore';

vi.mock('../lib/api', () => ({
	restoresApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
	},
}));

import { restoresApi } from '../lib/api';

describe('useRestores', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches restores', async () => {
		vi.mocked(restoresApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useRestores(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('fetches with filter', async () => {
		vi.mocked(restoresApi.list).mockResolvedValue([]);
		const filter = { agent_id: 'a1', status: 'completed' };
		const { result } = renderHook(() => useRestores(filter), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(restoresApi.list).toHaveBeenCalledWith(filter);
	});
});

describe('useRestore', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a restore', async () => {
		vi.mocked(restoresApi.get).mockResolvedValue({
			id: 'r1',
			status: 'completed',
		});
		const { result } = renderHook(() => useRestore('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useRestore(''), { wrapper: createWrapper() });
		expect(restoresApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateRestore', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a restore', async () => {
		vi.mocked(restoresApi.create).mockResolvedValue({ id: 'r1' });
		const { result } = renderHook(() => useCreateRestore(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			snapshot_id: 's1',
			target_path: '/restore',
		} as Parameters<typeof restoresApi.create>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
