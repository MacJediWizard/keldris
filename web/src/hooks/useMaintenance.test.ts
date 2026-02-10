import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useMaintenanceWindows,
	useMaintenanceWindow,
	useActiveMaintenance,
	useCreateMaintenanceWindow,
	useDeleteMaintenanceWindow,
} from './useMaintenance';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	maintenanceApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		getActive: vi.fn(),
	},
}));

import { maintenanceApi } from '../lib/api';

describe('useMaintenanceWindows', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches maintenance windows', async () => {
		vi.mocked(maintenanceApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useMaintenanceWindows(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useMaintenanceWindow', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a single window', async () => {
		vi.mocked(maintenanceApi.get).mockResolvedValue({ id: 'mw1' });
		const { result } = renderHook(() => useMaintenanceWindow('mw1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useMaintenanceWindow(''), { wrapper: createWrapper() });
		expect(maintenanceApi.get).not.toHaveBeenCalled();
	});
});

describe('useActiveMaintenance', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches active maintenance', async () => {
		vi.mocked(maintenanceApi.getActive).mockResolvedValue({ active: null, upcoming: null });
		const { result } = renderHook(() => useActiveMaintenance(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateMaintenanceWindow', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a window', async () => {
		vi.mocked(maintenanceApi.create).mockResolvedValue({ id: 'mw1' });
		const { result } = renderHook(() => useCreateMaintenanceWindow(), { wrapper: createWrapper() });
		result.current.mutate({ title: 'Planned Maintenance' } as Parameters<typeof maintenanceApi.create>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteMaintenanceWindow', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a window', async () => {
		vi.mocked(maintenanceApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteMaintenanceWindow(), { wrapper: createWrapper() });
		result.current.mutate('mw1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
