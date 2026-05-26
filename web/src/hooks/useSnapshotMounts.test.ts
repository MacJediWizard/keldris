import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useMountSnapshot,
	useSnapshotMount,
	useSnapshotMounts,
	useUnmountSnapshot,
} from './useSnapshotMounts';

vi.mock('../lib/api', () => ({
	snapshotMountsApi: {
		list: vi.fn(),
	},
	snapshotsApi: {
		getMount: vi.fn(),
		mount: vi.fn(),
		unmount: vi.fn(),
	},
}));

import { snapshotMountsApi, snapshotsApi } from '../lib/api';

describe('useSnapshotMounts', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists snapshot mounts', async () => {
		vi.mocked(snapshotMountsApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useSnapshotMounts(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotMountsApi.list).toHaveBeenCalledWith(undefined);
	});

	it('passes agentId', async () => {
		vi.mocked(snapshotMountsApi.list).mockResolvedValue([]);

		renderHook(() => useSnapshotMounts('a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => {
			expect(snapshotMountsApi.list).toHaveBeenCalledWith('a-1');
		});
	});
});

describe('useSnapshotMount', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a mount', async () => {
		vi.mocked(snapshotsApi.getMount).mockResolvedValue({ id: 'm-1' });

		const { result } = renderHook(() => useSnapshotMount('s-1', 'a-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotsApi.getMount).toHaveBeenCalledWith('s-1', 'a-1');
	});

	it('does not fetch when ids are empty', () => {
		renderHook(() => useSnapshotMount('', ''), { wrapper: createWrapper() });
		expect(snapshotsApi.getMount).not.toHaveBeenCalled();
	});
});

describe('useMountSnapshot', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('mounts a snapshot', async () => {
		vi.mocked(snapshotsApi.mount).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useMountSnapshot(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			snapshotId: 's-1',
			data: { agent_id: 'a-1' } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotsApi.mount).toHaveBeenCalledWith('s-1', { agent_id: 'a-1' });
	});
});

describe('useUnmountSnapshot', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('unmounts a snapshot', async () => {
		vi.mocked(snapshotsApi.unmount).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useUnmountSnapshot(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ snapshotId: 's-1', agentId: 'a-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotsApi.unmount).toHaveBeenCalledWith('s-1', 'a-1');
	});
});
