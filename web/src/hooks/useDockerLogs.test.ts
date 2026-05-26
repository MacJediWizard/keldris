import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useApplyDockerLogRetention,
	useDeleteDockerLogBackup,
	useDockerLogBackup,
	useDockerLogBackups,
	useDockerLogBackupsByAgent,
	useDockerLogBackupsByContainer,
	useDockerLogDownload,
	useDockerLogSettings,
	useDockerLogStorageStats,
	useDockerLogView,
	useUpdateDockerLogSettings,
} from './useDockerLogs';

vi.mock('../lib/api', () => ({
	dockerLogsApi: {
		list: vi.fn(),
		get: vi.fn(),
		view: vi.fn(),
		download: vi.fn(),
		delete: vi.fn(),
		getSettings: vi.fn(),
		updateSettings: vi.fn(),
		listByAgent: vi.fn(),
		listByContainer: vi.fn(),
		getStorageStats: vi.fn(),
		applyRetention: vi.fn(),
	},
}));

import { dockerLogsApi } from '../lib/api';

describe('useDockerLogBackups', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists docker log backups', async () => {
		vi.mocked(dockerLogsApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useDockerLogBackups(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.list).toHaveBeenCalledWith(undefined);
	});
});

describe('useDockerLogBackup', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a backup', async () => {
		vi.mocked(dockerLogsApi.get).mockResolvedValue({ id: 'b-1' });

		const { result } = renderHook(() => useDockerLogBackup('b-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.get).toHaveBeenCalledWith('b-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDockerLogBackup(''), { wrapper: createWrapper() });
		expect(dockerLogsApi.get).not.toHaveBeenCalled();
	});
});

describe('useDockerLogView', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('views backup contents', async () => {
		vi.mocked(dockerLogsApi.view).mockResolvedValue({ lines: [] });

		const { result } = renderHook(() => useDockerLogView('b-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.view).toHaveBeenCalledWith('b-1', 0, 1000);
	});
});

describe('useDockerLogDownload', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('downloads a backup', async () => {
		const blob = new Blob(['x']);
		vi.mocked(dockerLogsApi.download).mockResolvedValue(blob);
		const createObjectURL = vi.fn().mockReturnValue('blob:url');
		const revokeObjectURL = vi.fn();
		vi.stubGlobal('URL', { createObjectURL, revokeObjectURL });

		const { result } = renderHook(() => useDockerLogDownload(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'b-1', format: 'json' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.download).toHaveBeenCalledWith('b-1', 'json');

		vi.unstubAllGlobals();
	});
});

describe('useDeleteDockerLogBackup', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a backup', async () => {
		vi.mocked(dockerLogsApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteDockerLogBackup(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('b-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.delete).toHaveBeenCalledWith('b-1');
	});
});

describe('useDockerLogSettings', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches settings for an agent', async () => {
		vi.mocked(dockerLogsApi.getSettings).mockResolvedValue({});

		const { result } = renderHook(() => useDockerLogSettings('a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.getSettings).toHaveBeenCalledWith('a-1');
	});
});

describe('useUpdateDockerLogSettings', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates settings', async () => {
		vi.mocked(dockerLogsApi.updateSettings).mockResolvedValue({});

		const { result } = renderHook(() => useUpdateDockerLogSettings(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			agentId: 'a-1',
			settings: { enabled: true } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.updateSettings).toHaveBeenCalledWith('a-1', {
			enabled: true,
		});
	});
});

describe('useDockerLogBackupsByAgent', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backups by agent', async () => {
		vi.mocked(dockerLogsApi.listByAgent).mockResolvedValue([]);

		const { result } = renderHook(() => useDockerLogBackupsByAgent('a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.listByAgent).toHaveBeenCalledWith('a-1');
	});
});

describe('useDockerLogBackupsByContainer', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backups by container', async () => {
		vi.mocked(dockerLogsApi.listByContainer).mockResolvedValue([]);

		const { result } = renderHook(
			() => useDockerLogBackupsByContainer('a-1', 'c-1'),
			{ wrapper: createWrapper() },
		);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.listByContainer).toHaveBeenCalledWith('a-1', 'c-1');
	});
});

describe('useDockerLogStorageStats', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches storage stats', async () => {
		vi.mocked(dockerLogsApi.getStorageStats).mockResolvedValue({});

		const { result } = renderHook(() => useDockerLogStorageStats('a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.getStorageStats).toHaveBeenCalledWith('a-1');
	});
});

describe('useApplyDockerLogRetention', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('applies retention', async () => {
		vi.mocked(dockerLogsApi.applyRetention).mockResolvedValue({});

		const { result } = renderHook(() => useApplyDockerLogRetention(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ agentId: 'a-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerLogsApi.applyRetention).toHaveBeenCalledWith('a-1', undefined);
	});
});
