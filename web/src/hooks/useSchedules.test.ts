import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateSchedule,
	useDeleteSchedule,
	useReplicationStatus,
	useRunSchedule,
	useSchedule,
	useSchedules,
} from './useSchedules';

vi.mock('../lib/api', () => ({
	schedulesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		run: vi.fn(),
		getReplicationStatus: vi.fn(),
	},
}));

import { schedulesApi } from '../lib/api';

const mockSchedules = [
	{
		id: 'sched-1',
		name: 'Daily Backup',
		cron: '0 0 * * *',
		enabled: true,
		agent_id: 'agent-1',
	},
];

describe('useSchedules', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches all schedules', async () => {
		vi.mocked(schedulesApi.list).mockResolvedValue(mockSchedules);

		const { result } = renderHook(() => useSchedules(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockSchedules);
	});

	it('fetches schedules filtered by agent', async () => {
		vi.mocked(schedulesApi.list).mockResolvedValue(mockSchedules);

		const { result } = renderHook(() => useSchedules('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(schedulesApi.list).toHaveBeenCalledWith('agent-1');
	});
});

describe('useSchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single schedule', async () => {
		vi.mocked(schedulesApi.get).mockResolvedValue(mockSchedules[0]);

		const { result } = renderHook(() => useSchedule('sched-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(schedulesApi.get).toHaveBeenCalledWith('sched-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSchedule(''), {
			wrapper: createWrapper(),
		});
		expect(schedulesApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateSchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a schedule', async () => {
		vi.mocked(schedulesApi.create).mockResolvedValue(mockSchedules[0]);

		const { result } = renderHook(() => useCreateSchedule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'Daily Backup',
			cron: '0 0 * * *',
			agent_id: 'agent-1',
		} as Parameters<typeof schedulesApi.create>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteSchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a schedule', async () => {
		vi.mocked(schedulesApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteSchedule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('sched-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(schedulesApi.delete).toHaveBeenCalledWith('sched-1');
	});
});

describe('useRunSchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('runs a schedule', async () => {
		vi.mocked(schedulesApi.run).mockResolvedValue({
			backup_id: 'backup-1',
			message: 'Started',
		});

		const { result } = renderHook(() => useRunSchedule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('sched-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(schedulesApi.run).toHaveBeenCalledWith('sched-1');
	});
});

describe('useReplicationStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches replication status', async () => {
		vi.mocked(schedulesApi.getReplicationStatus).mockResolvedValue([]);

		const { result } = renderHook(() => useReplicationStatus('sched-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(schedulesApi.getReplicationStatus).toHaveBeenCalledWith('sched-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useReplicationStatus(''), {
			wrapper: createWrapper(),
		});
		expect(schedulesApi.getReplicationStatus).not.toHaveBeenCalled();
	});
});
