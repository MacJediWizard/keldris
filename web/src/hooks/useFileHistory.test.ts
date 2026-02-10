import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { useFileHistory } from './useFileHistory';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	fileHistoryApi: {
		getHistory: vi.fn(),
	},
}));

import { fileHistoryApi } from '../lib/api';

describe('useFileHistory', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches file history', async () => {
		vi.mocked(fileHistoryApi.getHistory).mockResolvedValue({ versions: [] });
		const params = { path: '/etc/config', agent_id: 'a1', repository_id: 'r1' };
		const { result } = renderHook(() => useFileHistory(params), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fileHistoryApi.getHistory).toHaveBeenCalledWith(params);
	});

	it('does not fetch when params are incomplete', () => {
		renderHook(() => useFileHistory({ path: '', agent_id: 'a1', repository_id: 'r1' }), {
			wrapper: createWrapper(),
		});
		expect(fileHistoryApi.getHistory).not.toHaveBeenCalled();
	});
});
