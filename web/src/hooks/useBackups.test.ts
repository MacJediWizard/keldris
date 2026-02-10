import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useBackup, useBackups } from './useBackups';

vi.mock('../lib/api', () => ({
	backupsApi: {
		list: vi.fn(),
		get: vi.fn(),
	},
}));

import { backupsApi } from '../lib/api';

const mockBackups = [
	{
		id: 'backup-1',
		agent_id: 'agent-1',
		schedule_id: 'sched-1',
		status: 'completed',
		started_at: '2024-06-15T12:00:00Z',
		completed_at: '2024-06-15T12:30:00Z',
	},
	{
		id: 'backup-2',
		agent_id: 'agent-1',
		schedule_id: 'sched-1',
		status: 'failed',
		started_at: '2024-06-14T12:00:00Z',
		completed_at: '2024-06-14T12:15:00Z',
	},
];

describe('useBackups', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backups list', async () => {
		vi.mocked(backupsApi.list).mockResolvedValue(mockBackups);

		const { result } = renderHook(() => useBackups(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockBackups);
	});

	it('fetches backups with filter', async () => {
		vi.mocked(backupsApi.list).mockResolvedValue([mockBackups[0]]);

		const filter = { agent_id: 'agent-1', status: 'completed' };
		const { result } = renderHook(() => useBackups(filter), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupsApi.list).toHaveBeenCalledWith(filter);
	});
});

describe('useBackup', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single backup', async () => {
		vi.mocked(backupsApi.get).mockResolvedValue(mockBackups[0]);

		const { result } = renderHook(() => useBackup('backup-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupsApi.get).toHaveBeenCalledWith('backup-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useBackup(''), {
			wrapper: createWrapper(),
		});

		expect(backupsApi.get).not.toHaveBeenCalled();
	});
});
