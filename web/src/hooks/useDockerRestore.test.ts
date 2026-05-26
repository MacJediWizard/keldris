import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCancelDockerRestore,
	useContainersInSnapshot,
	useCreateDockerRestore,
	useDockerRestore,
	useDockerRestorePreview,
	useDockerRestoreProgress,
	useDockerRestores,
	useVolumesInSnapshot,
} from './useDockerRestore';

vi.mock('../lib/api', () => ({
	dockerRestoresApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		preview: vi.fn(),
		getProgress: vi.fn(),
		listContainers: vi.fn(),
		listVolumes: vi.fn(),
		cancel: vi.fn(),
	},
}));

import { dockerRestoresApi } from '../lib/api';

describe('useDockerRestores', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists docker restores', async () => {
		vi.mocked(dockerRestoresApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useDockerRestores(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.list).toHaveBeenCalledWith(undefined);
	});
});

describe('useDockerRestore', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a restore', async () => {
		vi.mocked(dockerRestoresApi.get).mockResolvedValue({
			id: 'r-1',
			status: 'complete',
		});

		const { result } = renderHook(() => useDockerRestore('r-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.get).toHaveBeenCalledWith('r-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDockerRestore(''), { wrapper: createWrapper() });
		expect(dockerRestoresApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateDockerRestore', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a restore', async () => {
		vi.mocked(dockerRestoresApi.create).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateDockerRestore(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ snapshot_id: 's-1' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.create).toHaveBeenCalled();
	});
});

describe('useDockerRestorePreview', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('previews a restore', async () => {
		vi.mocked(dockerRestoresApi.preview).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useDockerRestorePreview(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ snapshot_id: 's-1' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDockerRestoreProgress', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches restore progress', async () => {
		vi.mocked(dockerRestoresApi.getProgress).mockResolvedValue({ percent: 50 });

		const { result } = renderHook(() => useDockerRestoreProgress('r-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.getProgress).toHaveBeenCalledWith('r-1');
	});

	it('does not fetch when disabled', () => {
		renderHook(() => useDockerRestoreProgress('r-1', false), {
			wrapper: createWrapper(),
		});
		expect(dockerRestoresApi.getProgress).not.toHaveBeenCalled();
	});
});

describe('useContainersInSnapshot', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists containers in snapshot', async () => {
		vi.mocked(dockerRestoresApi.listContainers).mockResolvedValue([]);

		const { result } = renderHook(() => useContainersInSnapshot('s-1', 'a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.listContainers).toHaveBeenCalledWith('s-1', 'a-1');
	});
});

describe('useVolumesInSnapshot', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists volumes in snapshot', async () => {
		vi.mocked(dockerRestoresApi.listVolumes).mockResolvedValue([]);

		const { result } = renderHook(() => useVolumesInSnapshot('s-1', 'a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.listVolumes).toHaveBeenCalledWith('s-1', 'a-1');
	});
});

describe('useCancelDockerRestore', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('cancels a restore', async () => {
		vi.mocked(dockerRestoresApi.cancel).mockResolvedValue({
			message: 'Canceled',
		});

		const { result } = renderHook(() => useCancelDockerRestore(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRestoresApi.cancel).toHaveBeenCalledWith('r-1');
	});
});
