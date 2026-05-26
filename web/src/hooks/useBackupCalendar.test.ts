import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useBackupCalendar } from './useBackupCalendar';

vi.mock('../lib/api', () => ({
	backupsApi: {
		getCalendar: vi.fn(),
	},
}));

import { backupsApi } from '../lib/api';

describe('useBackupCalendar', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches calendar data for given month', async () => {
		const mockData = { days: [], month: '2025-01' };
		vi.mocked(backupsApi.getCalendar).mockResolvedValue(mockData);

		const { result } = renderHook(
			() => useBackupCalendar({ month: '2025-01' }),
			{ wrapper: createWrapper() },
		);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockData);
		expect(backupsApi.getCalendar).toHaveBeenCalledWith('2025-01');
	});

	it('handles error state', async () => {
		vi.mocked(backupsApi.getCalendar).mockRejectedValue(
			new Error('Network error'),
		);

		const { result } = renderHook(
			() => useBackupCalendar({ month: '2025-01' }),
			{ wrapper: createWrapper() },
		);

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error).toBeDefined();
	});
});
