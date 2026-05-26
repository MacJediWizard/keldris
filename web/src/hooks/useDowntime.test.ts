import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActiveDowntime,
	useCreateDowntimeAlert,
	useCreateDowntimeEvent,
	useDeleteDowntimeAlert,
	useDeleteDowntimeEvent,
	useDowntimeAlert,
	useDowntimeAlerts,
	useDowntimeEvent,
	useDowntimeEvents,
	useMonthlyUptimeReport,
	useRefreshUptimeBadges,
	useResolveDowntimeEvent,
	useUpdateDowntimeAlert,
	useUpdateDowntimeEvent,
	useUptimeBadges,
	useUptimeSummary,
} from './useDowntime';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.restoreAllMocks();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('useDowntimeEvents', () => {
	it('fetches downtime events', async () => {
		mockFetch({ events: [] });
		const { result } = renderHook(() => useDowntimeEvents(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useActiveDowntime', () => {
	it('fetches active downtime', async () => {
		mockFetch({ events: [] });
		const { result } = renderHook(() => useActiveDowntime(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUptimeSummary', () => {
	it('fetches uptime summary', async () => {
		mockFetch({ uptime_percentage: 99.9 });
		const { result } = renderHook(() => useUptimeSummary(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDowntimeEvent', () => {
	it('fetches a single event', async () => {
		mockFetch({ id: 'e1' });
		const { result } = renderHook(() => useDowntimeEvent('e1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useDowntimeEvent(''), { wrapper: createWrapper() });
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateDowntimeEvent', () => {
	it('creates a downtime event', async () => {
		mockFetch({ id: 'e1' });
		const { result } = renderHook(() => useCreateDowntimeEvent(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ agent_id: 'a1' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateDowntimeEvent', () => {
	it('updates an event', async () => {
		mockFetch({ id: 'e1' });
		const { result } = renderHook(() => useUpdateDowntimeEvent(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'e1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useResolveDowntimeEvent', () => {
	it('resolves an event', async () => {
		mockFetch({ id: 'e1' });
		const { result } = renderHook(() => useResolveDowntimeEvent(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'e1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteDowntimeEvent', () => {
	it('deletes an event', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteDowntimeEvent(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('e1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUptimeBadges', () => {
	it('fetches uptime badges', async () => {
		mockFetch({ badges: [] });
		const { result } = renderHook(() => useUptimeBadges(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRefreshUptimeBadges', () => {
	it('refreshes uptime badges', async () => {
		mockFetch({ message: 'ok' });
		const { result } = renderHook(() => useRefreshUptimeBadges(), {
			wrapper: createWrapper(),
		});
		result.current.mutate();
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useMonthlyUptimeReport', () => {
	it('fetches a monthly report', async () => {
		mockFetch({ year: 2025, month: 1 });
		const { result } = renderHook(() => useMonthlyUptimeReport(2025, 1), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch with invalid month', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useMonthlyUptimeReport(2025, 13), {
			wrapper: createWrapper(),
		});
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useDowntimeAlerts', () => {
	it('fetches alerts', async () => {
		mockFetch({ alerts: [] });
		const { result } = renderHook(() => useDowntimeAlerts(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDowntimeAlert', () => {
	it('fetches a single alert', async () => {
		mockFetch({ id: 'al1' });
		const { result } = renderHook(() => useDowntimeAlert('al1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateDowntimeAlert', () => {
	it('creates an alert', async () => {
		mockFetch({ id: 'al1' });
		const { result } = renderHook(() => useCreateDowntimeAlert(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ name: 'test' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateDowntimeAlert', () => {
	it('updates an alert', async () => {
		mockFetch({ id: 'al1' });
		const { result } = renderHook(() => useUpdateDowntimeAlert(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'al1', data: {} as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteDowntimeAlert', () => {
	it('deletes an alert', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteDowntimeAlert(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('al1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
