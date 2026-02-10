import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateReportSchedule,
	useDeleteReportSchedule,
	usePreviewReport,
	useReportHistory,
	useReportHistoryEntry,
	useReportSchedule,
	useReportSchedules,
	useSendReport,
} from './useReports';

vi.mock('../lib/api', () => ({
	reportsApi: {
		listSchedules: vi.fn(),
		getSchedule: vi.fn(),
		createSchedule: vi.fn(),
		updateSchedule: vi.fn(),
		deleteSchedule: vi.fn(),
		sendReport: vi.fn(),
		previewReport: vi.fn(),
		listHistory: vi.fn(),
		getHistory: vi.fn(),
	},
}));

import { reportsApi } from '../lib/api';

describe('useReportSchedules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches report schedules', async () => {
		vi.mocked(reportsApi.listSchedules).mockResolvedValue([]);
		const { result } = renderHook(() => useReportSchedules(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useReportSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a schedule', async () => {
		vi.mocked(reportsApi.getSchedule).mockResolvedValue({ id: 'rs1' });
		const { result } = renderHook(() => useReportSchedule('rs1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useReportSchedule(''), { wrapper: createWrapper() });
		expect(reportsApi.getSchedule).not.toHaveBeenCalled();
	});
});

describe('useCreateReportSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a schedule', async () => {
		vi.mocked(reportsApi.createSchedule).mockResolvedValue({ id: 'rs1' });
		const { result } = renderHook(() => useCreateReportSchedule(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ name: 'Weekly Report' } as Parameters<
			typeof reportsApi.createSchedule
		>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteReportSchedule', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a schedule', async () => {
		vi.mocked(reportsApi.deleteSchedule).mockResolvedValue({
			message: 'Deleted',
		});
		const { result } = renderHook(() => useDeleteReportSchedule(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('rs1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSendReport', () => {
	beforeEach(() => vi.clearAllMocks());

	it('sends a report', async () => {
		vi.mocked(reportsApi.sendReport).mockResolvedValue({ message: 'Sent' });
		const { result } = renderHook(() => useSendReport(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'rs1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('usePreviewReport', () => {
	beforeEach(() => vi.clearAllMocks());

	it('previews a report', async () => {
		vi.mocked(reportsApi.previewReport).mockResolvedValue({
			html: '<p>Report</p>',
		});
		const { result } = renderHook(() => usePreviewReport(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			frequency: 'weekly' as Parameters<typeof reportsApi.previewReport>[0],
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useReportHistory', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches history', async () => {
		vi.mocked(reportsApi.listHistory).mockResolvedValue([]);
		const { result } = renderHook(() => useReportHistory(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useReportHistoryEntry', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a history entry', async () => {
		vi.mocked(reportsApi.getHistory).mockResolvedValue({ id: 'rh1' });
		const { result } = renderHook(() => useReportHistoryEntry('rh1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useReportHistoryEntry(''), { wrapper: createWrapper() });
		expect(reportsApi.getHistory).not.toHaveBeenCalled();
	});
});
