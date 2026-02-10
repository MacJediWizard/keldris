import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useBackupScript,
	useBackupScripts,
	useCreateBackupScript,
	useDeleteBackupScript,
} from './useBackupScripts';

vi.mock('../lib/api', () => ({
	backupScriptsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
}));

import { backupScriptsApi } from '../lib/api';

describe('useBackupScripts', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches scripts for a schedule', async () => {
		vi.mocked(backupScriptsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useBackupScripts('sched-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupScriptsApi.list).toHaveBeenCalledWith('sched-1');
	});

	it('does not fetch when scheduleId is empty', () => {
		renderHook(() => useBackupScripts(''), { wrapper: createWrapper() });
		expect(backupScriptsApi.list).not.toHaveBeenCalled();
	});
});

describe('useBackupScript', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a single script', async () => {
		vi.mocked(backupScriptsApi.get).mockResolvedValue({ id: 'bs1' });
		const { result } = renderHook(() => useBackupScript('sched-1', 'bs1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when either id is empty', () => {
		renderHook(() => useBackupScript('', 'bs1'), { wrapper: createWrapper() });
		expect(backupScriptsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateBackupScript', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a script', async () => {
		vi.mocked(backupScriptsApi.create).mockResolvedValue({ id: 'bs1' });
		const { result } = renderHook(() => useCreateBackupScript(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			scheduleId: 'sched-1',
			data: { type: 'pre_backup', script: 'echo hello' } as Parameters<
				typeof backupScriptsApi.create
			>[1],
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteBackupScript', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a script', async () => {
		vi.mocked(backupScriptsApi.delete).mockResolvedValue({
			message: 'Deleted',
		});
		const { result } = renderHook(() => useDeleteBackupScript(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ scheduleId: 'sched-1', id: 'bs1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
