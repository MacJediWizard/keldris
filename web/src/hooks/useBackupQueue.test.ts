import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAgentConcurrency,
	useBackupQueue,
	useBackupQueueSummary,
	useCancelQueuedBackup,
	useOrgConcurrency,
	useUpdateAgentConcurrency,
	useUpdateOrgConcurrency,
} from './useBackupQueue';

vi.mock('../lib/api', () => ({
	backupQueueApi: {
		list: vi.fn(),
		getSummary: vi.fn(),
		cancel: vi.fn(),
	},
	concurrencyApi: {
		getOrgConcurrency: vi.fn(),
		updateOrgConcurrency: vi.fn(),
		getAgentConcurrency: vi.fn(),
		updateAgentConcurrency: vi.fn(),
	},
}));

import { backupQueueApi, concurrencyApi } from '../lib/api';

describe('useBackupQueue', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches the queue', async () => {
		vi.mocked(backupQueueApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useBackupQueue(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupQueueApi.list).toHaveBeenCalledOnce();
	});
});

describe('useBackupQueueSummary', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches summary', async () => {
		vi.mocked(backupQueueApi.getSummary).mockResolvedValue({ queued: 0 });

		const { result } = renderHook(() => useBackupQueueSummary(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupQueueApi.getSummary).toHaveBeenCalledOnce();
	});
});

describe('useCancelQueuedBackup', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('cancels a queued backup', async () => {
		vi.mocked(backupQueueApi.cancel).mockResolvedValue({ message: 'Canceled' });

		const { result } = renderHook(() => useCancelQueuedBackup(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('q-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupQueueApi.cancel).toHaveBeenCalledWith('q-1');
	});
});

describe('useOrgConcurrency', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches org concurrency', async () => {
		vi.mocked(concurrencyApi.getOrgConcurrency).mockResolvedValue({ max: 5 });

		const { result } = renderHook(() => useOrgConcurrency('org-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(concurrencyApi.getOrgConcurrency).toHaveBeenCalledWith('org-1');
	});

	it('does not fetch when orgId is empty', () => {
		renderHook(() => useOrgConcurrency(''), { wrapper: createWrapper() });
		expect(concurrencyApi.getOrgConcurrency).not.toHaveBeenCalled();
	});
});

describe('useUpdateOrgConcurrency', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates org concurrency', async () => {
		vi.mocked(concurrencyApi.updateOrgConcurrency).mockResolvedValue({
			max: 10,
		});

		const { result } = renderHook(() => useUpdateOrgConcurrency(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ orgId: 'org-1', data: { max: 10 } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(concurrencyApi.updateOrgConcurrency).toHaveBeenCalledWith('org-1', {
			max: 10,
		});
	});
});

describe('useAgentConcurrency', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches agent concurrency', async () => {
		vi.mocked(concurrencyApi.getAgentConcurrency).mockResolvedValue({ max: 3 });

		const { result } = renderHook(() => useAgentConcurrency('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(concurrencyApi.getAgentConcurrency).toHaveBeenCalledWith('agent-1');
	});
});

describe('useUpdateAgentConcurrency', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates agent concurrency', async () => {
		vi.mocked(concurrencyApi.updateAgentConcurrency).mockResolvedValue({
			max: 4,
		});

		const { result } = renderHook(() => useUpdateAgentConcurrency(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ agentId: 'agent-1', data: { max: 4 } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(concurrencyApi.updateAgentConcurrency).toHaveBeenCalledWith(
			'agent-1',
			{ max: 4 },
		);
	});
});
