import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useDRRunbooks,
	useDRRunbook,
	useDRStatus,
	useCreateDRRunbook,
	useDeleteDRRunbook,
	useActivateDRRunbook,
	useArchiveDRRunbook,
	useRenderDRRunbook,
	useGenerateDRRunbook,
	useDRTestSchedules,
	useCreateDRTestSchedule,
} from './useDRRunbooks';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	drRunbooksApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		activate: vi.fn(),
		archive: vi.fn(),
		render: vi.fn(),
		generateFromSchedule: vi.fn(),
		getStatus: vi.fn(),
		listTestSchedules: vi.fn(),
		createTestSchedule: vi.fn(),
	},
}));

import { drRunbooksApi } from '../lib/api';

describe('useDRRunbooks', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches runbooks', async () => {
		vi.mocked(drRunbooksApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useDRRunbooks(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a runbook', async () => {
		vi.mocked(drRunbooksApi.get).mockResolvedValue({ id: 'rb1' });
		const { result } = renderHook(() => useDRRunbook('rb1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDRRunbook(''), { wrapper: createWrapper() });
		expect(drRunbooksApi.get).not.toHaveBeenCalled();
	});
});

describe('useDRStatus', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches DR status', async () => {
		vi.mocked(drRunbooksApi.getStatus).mockResolvedValue({ overall_status: 'healthy' });
		const { result } = renderHook(() => useDRStatus(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a runbook', async () => {
		vi.mocked(drRunbooksApi.create).mockResolvedValue({ id: 'rb1' });
		const { result } = renderHook(() => useCreateDRRunbook(), { wrapper: createWrapper() });
		result.current.mutate({ name: 'DR Plan' } as Parameters<typeof drRunbooksApi.create>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a runbook', async () => {
		vi.mocked(drRunbooksApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteDRRunbook(), { wrapper: createWrapper() });
		result.current.mutate('rb1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useActivateDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('activates a runbook', async () => {
		vi.mocked(drRunbooksApi.activate).mockResolvedValue({ id: 'rb1', status: 'active' });
		const { result } = renderHook(() => useActivateDRRunbook(), { wrapper: createWrapper() });
		result.current.mutate('rb1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useArchiveDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('archives a runbook', async () => {
		vi.mocked(drRunbooksApi.archive).mockResolvedValue({ id: 'rb1', status: 'archived' });
		const { result } = renderHook(() => useArchiveDRRunbook(), { wrapper: createWrapper() });
		result.current.mutate('rb1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRenderDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders a runbook', async () => {
		vi.mocked(drRunbooksApi.render).mockResolvedValue({ html: '<p>Plan</p>' });
		const { result } = renderHook(() => useRenderDRRunbook('rb1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not render when id is empty', () => {
		renderHook(() => useRenderDRRunbook(''), { wrapper: createWrapper() });
		expect(drRunbooksApi.render).not.toHaveBeenCalled();
	});
});

describe('useGenerateDRRunbook', () => {
	beforeEach(() => vi.clearAllMocks());

	it('generates a runbook', async () => {
		vi.mocked(drRunbooksApi.generateFromSchedule).mockResolvedValue({ id: 'rb1' });
		const { result } = renderHook(() => useGenerateDRRunbook(), { wrapper: createWrapper() });
		result.current.mutate('sched-1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDRTestSchedules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches test schedules', async () => {
		vi.mocked(drRunbooksApi.listTestSchedules).mockResolvedValue([]);
		const { result } = renderHook(() => useDRTestSchedules('rb1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDRTestSchedules(''), { wrapper: createWrapper() });
		expect(drRunbooksApi.listTestSchedules).not.toHaveBeenCalled();
	});
});

describe('useCreateDRTestSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a test schedule', async () => {
		vi.mocked(drRunbooksApi.createTestSchedule).mockResolvedValue({ id: 'ts1' });
		const { result } = renderHook(() => useCreateDRTestSchedule(), { wrapper: createWrapper() });
		result.current.mutate({ runbookId: 'rb1', data: { cron: '0 0 * * *' } as Parameters<typeof drRunbooksApi.createTestSchedule>[1] });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
