import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useClearServerLogs,
	useExportServerLogsCsv,
	useExportServerLogsJson,
	useServerLogComponents,
	useServerLogs,
} from './useServerLogs';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

function mockFetchBlob(blob: Blob, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		blob: () => Promise.resolve(blob),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useServerLogs', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches server logs without filter', async () => {
		const fetchFn = mockFetch({ logs: [] });

		const { result } = renderHook(() => useServerLogs(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/logs',
			expect.any(Object),
		);
	});

	it('fetches server logs with filter', async () => {
		const fetchFn = mockFetch({ logs: [] });

		const { result } = renderHook(
			() => useServerLogs({ level: 'error', limit: 10 }),
			{ wrapper: createWrapper() },
		);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/logs?level=error&limit=10',
			expect.any(Object),
		);
	});
});

describe('useServerLogComponents', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches server log components', async () => {
		mockFetch({ components: ['api', 'auth'] });

		const { result } = renderHook(() => useServerLogComponents(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(['api', 'auth']);
	});
});

describe('useExportServerLogsCsv', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(window.URL, 'createObjectURL').mockReturnValue('blob:mock');
		vi.spyOn(window.URL, 'revokeObjectURL').mockImplementation(() => {});
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('exports server logs as CSV', async () => {
		const blob = new Blob(['csv-data']);
		const fetchFn = mockFetchBlob(blob);

		const { result } = renderHook(() => useExportServerLogsCsv(), {
			wrapper: createWrapper(),
		});

		result.current.mutate(undefined);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/logs/export/csv',
			expect.any(Object),
		);
	});
});

describe('useExportServerLogsJson', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(window.URL, 'createObjectURL').mockReturnValue('blob:mock');
		vi.spyOn(window.URL, 'revokeObjectURL').mockImplementation(() => {});
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('exports server logs as JSON', async () => {
		const blob = new Blob(['json-data']);
		const fetchFn = mockFetchBlob(blob);

		const { result } = renderHook(() => useExportServerLogsJson(), {
			wrapper: createWrapper(),
		});

		result.current.mutate(undefined);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/logs/export/json',
			expect.any(Object),
		);
	});
});

describe('useClearServerLogs', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('clears server logs', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useClearServerLogs(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalled();
	});
});
